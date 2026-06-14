package tests

import (
	"context"
	"testing"

	"github.com/agriconnect-ai/internal/models"
	"github.com/agriconnect-ai/internal/services"
	"github.com/google/uuid"
)

type mockConversationRepo struct {
	convs map[uuid.UUID]*models.Conversation
}

func (m *mockConversationRepo) Create(_ context.Context, conv *models.Conversation) error {
	m.convs[conv.ID] = conv
	return nil
}

func (m *mockConversationRepo) FindByID(_ context.Context, id uuid.UUID) (*models.Conversation, error) {
	if conv, ok := m.convs[id]; ok {
		return conv, nil
	}
	return nil, nil
}

func (m *mockConversationRepo) FindByUserID(_ context.Context, _ uuid.UUID) ([]models.Conversation, error) {
	return nil, nil
}

func (m *mockConversationRepo) Update(_ context.Context, conv *models.Conversation) error {
	m.convs[conv.ID] = conv
	return nil
}

func (m *mockConversationRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.convs, id)
	return nil
}

type mockMessageRepo struct{}

func (m *mockMessageRepo) Create(_ context.Context, _ *models.Message) error {
	return nil
}

func (m *mockMessageRepo) FindByConversationID(_ context.Context, _ uuid.UUID, limit int) ([]models.Message, error) {
	return []models.Message{}, nil
}

func TestConversationLifecycle(t *testing.T) {
	convRepo := &mockConversationRepo{convs: make(map[uuid.UUID]*models.Conversation)}
	msgRepo := &mockMessageRepo{}
	svc := services.NewChatService(convRepo, msgRepo, "test-model")

	userID := uuid.New()
	conv, err := svc.CreateConversation(context.Background(), userID, "english", "Bo", "Cassava")
	if err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}

	if conv.PreferredLanguage != "english" {
		t.Errorf("expected english, got %s", conv.PreferredLanguage)
	}
	if conv.District != "Bo" {
		t.Errorf("expected Bo, got %s", conv.District)
	}
	if conv.Crop != "Cassava" {
		t.Errorf("expected Cassava, got %s", conv.Crop)
	}

	// Ownership check
	_, err = svc.GetConversation(context.Background(), conv.ID, uuid.New())
	if err == nil || err.Error() != "access denied" {
		t.Errorf("expected access denied, got %v", err)
	}

	// Valid ownership
	loaded, err := svc.GetConversation(context.Background(), conv.ID, userID)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}
	if loaded.ID != conv.ID {
		t.Errorf("conversation ID mismatch")
	}

	// Delete with wrong owner
	err = svc.DeleteConversation(context.Background(), conv.ID, uuid.New())
	if err == nil || err.Error() != "access denied" {
		t.Errorf("expected access denied, got %v", err)
	}

	// Delete with correct owner
	err = svc.DeleteConversation(context.Background(), conv.ID, userID)
	if err != nil {
		t.Fatalf("DeleteConversation failed: %v", err)
	}
}

func TestConversationDefaultLanguage(t *testing.T) {
	convRepo := &mockConversationRepo{convs: make(map[uuid.UUID]*models.Conversation)}
	msgRepo := &mockMessageRepo{}
	svc := services.NewChatService(convRepo, msgRepo, "test-model")

	conv, err := svc.CreateConversation(context.Background(), uuid.New(), "", "", "")
	if err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}

	if conv.PreferredLanguage != "english" {
		t.Errorf("expected default english, got %s", conv.PreferredLanguage)
	}
}
