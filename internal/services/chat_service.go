package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	aiPkg "github.com/agriconnect-ai/internal/ai"
	"github.com/agriconnect-ai/internal/models"
	"github.com/agriconnect-ai/internal/repositories"
	"github.com/google/uuid"
)

type SendMessageResult struct {
	Message *models.Message
	Sources []string
}

type ChatService interface {
	CreateConversation(ctx context.Context, userID uuid.UUID, language, district, crop string) (*models.Conversation, error)
	ListConversations(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error)
	GetConversation(ctx context.Context, id, userID uuid.UUID) (*models.Conversation, error)
	DeleteConversation(ctx context.Context, id, userID uuid.UUID) error
	SendMessage(ctx context.Context, conversationID, userID uuid.UUID, text string, orchestrator *aiPkg.Orchestrator, maxCtx int) (*SendMessageResult, error)
	SendMessageStream(ctx context.Context, conversationID, userID uuid.UUID, text string, orchestrator *aiPkg.Orchestrator, maxCtx int, tokenCh chan<- string, statusCh chan<- string) (*SendMessageResult, error)
}

type chatService struct {
	convRepo repositories.ConversationRepository
	msgRepo  repositories.MessageRepository
	model    string
}

func NewChatService(convRepo repositories.ConversationRepository, msgRepo repositories.MessageRepository, model string) ChatService {
	return &chatService{convRepo: convRepo, msgRepo: msgRepo, model: model}
}

func (s *chatService) CreateConversation(ctx context.Context, userID uuid.UUID, language, district, crop string) (*models.Conversation, error) {
	if language == "" {
		language = "english"
	}
	conv := &models.Conversation{
		ID:                uuid.New(),
		UserID:            userID,
		Title:             "New agricultural conversation",
		PreferredLanguage: language,
		District:          district,
		Crop:              crop,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, fmt.Errorf("creating conversation: %w", err)
	}
	return conv, nil
}

func (s *chatService) ListConversations(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error) {
	return s.convRepo.FindByUserID(ctx, userID)
}

func (s *chatService) GetConversation(ctx context.Context, id, userID uuid.UUID) (*models.Conversation, error) {
	conv, err := s.convRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if conv.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}
	return conv, nil
}

func (s *chatService) DeleteConversation(ctx context.Context, id, userID uuid.UUID) error {
	conv, err := s.convRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if conv.UserID != userID {
		return fmt.Errorf("access denied")
	}
	return s.convRepo.Delete(ctx, id)
}

func (s *chatService) SendMessage(ctx context.Context, conversationID, userID uuid.UUID, text string, orchestrator *aiPkg.Orchestrator, maxCtx int) (*SendMessageResult, error) {
	conv, err := s.convRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	userMsg := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           models.RoleUser,
		Content:        text,
		Language:       conv.PreferredLanguage,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("saving user message: %w", err)
	}

	// Get recent messages for context
	recentMsgs, err := s.msgRepo.FindByConversationID(ctx, conversationID, maxCtx)
	if err != nil {
		return nil, fmt.Errorf("loading context: %w", err)
	}

	aiMessages := buildAIMessages(recentMsgs)
	result, err := orchestrator.Run(ctx, aiMessages, conv.PreferredLanguage, conv.District, conv.Crop)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	assistantMsg := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           models.RoleAssistant,
		Content:        result.Content,
		Language:       conv.PreferredLanguage,
		Model:          s.model,
		InputTokens:    result.InputTokens,
		OutputTokens:   result.OutputTokens,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.msgRepo.Create(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("saving assistant message: %w", err)
	}

	// Update conversation title after first question
	if conv.Title == "New agricultural conversation" && len(recentMsgs) <= 2 {
		conv.Title = generateTitle(text)
		conv.UpdatedAt = time.Now().UTC()
		_ = s.convRepo.Update(ctx, conv)
	} else {
		conv.UpdatedAt = time.Now().UTC()
		_ = s.convRepo.Update(ctx, conv)
	}

	return &SendMessageResult{Message: assistantMsg, Sources: result.Sources}, nil
}

func (s *chatService) SendMessageStream(ctx context.Context, conversationID, userID uuid.UUID, text string, orchestrator *aiPkg.Orchestrator, maxCtx int, tokenCh chan<- string, statusCh chan<- string) (*SendMessageResult, error) {
	conv, err := s.convRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	userMsg := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           models.RoleUser,
		Content:        text,
		Language:       conv.PreferredLanguage,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("saving user message: %w", err)
	}

	recentMsgs, err := s.msgRepo.FindByConversationID(ctx, conversationID, maxCtx)
	if err != nil {
		return nil, fmt.Errorf("loading context: %w", err)
	}

	aiMessages := buildAIMessages(recentMsgs)
	result, err := orchestrator.RunStream(ctx, aiMessages, conv.PreferredLanguage, conv.District, conv.Crop, tokenCh, statusCh)
	if err != nil {
		return nil, err
	}

	assistantMsg := &models.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           models.RoleAssistant,
		Content:        result.Content,
		Language:       conv.PreferredLanguage,
		Model:          s.model,
		InputTokens:    result.InputTokens,
		OutputTokens:   result.OutputTokens,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.msgRepo.Create(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("saving assistant message: %w", err)
	}

	if conv.Title == "New agricultural conversation" && len(recentMsgs) <= 2 {
		conv.Title = generateTitle(text)
		conv.UpdatedAt = time.Now().UTC()
	} else {
		conv.UpdatedAt = time.Now().UTC()
	}
	_ = s.convRepo.Update(ctx, conv)

	return &SendMessageResult{Message: assistantMsg, Sources: result.Sources}, nil
}

func buildAIMessages(msgs []models.Message) []aiPkg.Message {
	var out []aiPkg.Message
	for _, m := range msgs {
		if m.Role == models.RoleSystem {
			continue
		}
		out = append(out, aiPkg.Message{Role: m.Role, Content: m.Content})
	}
	return out
}

func generateTitle(question string) string {
	words := strings.Fields(question)
	if len(words) > 8 {
		words = words[:8]
	}
	title := strings.Join(words, " ")
	if len(title) > 80 {
		title = title[:80]
	}
	return title + "..."
}
