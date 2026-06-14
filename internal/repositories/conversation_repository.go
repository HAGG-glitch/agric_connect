package repositories

import (
	"context"
	"fmt"

	"github.com/agriconnect-ai/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConversationRepository interface {
	Create(ctx context.Context, conv *models.Conversation) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error)
	Update(ctx context.Context, conv *models.Conversation) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type conversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepository{db: db}
}

func (r *conversationRepository) Create(ctx context.Context, conv *models.Conversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

func (r *conversationRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	err := r.db.WithContext(ctx).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		First(&conv, "id = ?", id).Error
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}
	return &conv, nil
}

func (r *conversationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error) {
	var convs []models.Conversation
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&convs).Error
	return convs, err
}

func (r *conversationRepository) Update(ctx context.Context, conv *models.Conversation) error {
	return r.db.WithContext(ctx).Save(conv).Error
}

func (r *conversationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Conversation{}, "id = ?", id).Error
}
