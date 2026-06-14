package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

type Message struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index" json:"conversation_id"`
	Role           string    `gorm:"size:20;not null" json:"role"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	Language       string    `gorm:"size:20" json:"language,omitempty"`
	Model          string    `gorm:"size:150" json:"model,omitempty"`
	InputTokens    int       `json:"input_tokens,omitempty"`
	OutputTokens   int       `json:"output_tokens,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (Message) TableName() string {
	return "ai_messages"
}

func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
