package repositories

import (
	"context"

	"github.com/agriconnect-ai/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *models.Message) error
	FindByConversationID(ctx context.Context, conversationID uuid.UUID, limit int) ([]models.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, msg *models.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *messageRepository) FindByConversationID(ctx context.Context, conversationID uuid.UUID, limit int) ([]models.Message, error) {
	var messages []models.Message
	query := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&messages).Error
	return messages, err
}
