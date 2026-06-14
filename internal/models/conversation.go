package models

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Title             string     `gorm:"size:200;not null" json:"title"`
	PreferredLanguage string     `gorm:"size:20;not null;default:english" json:"preferred_language"`
	District          string     `gorm:"size:100" json:"district"`
	Crop              string     `gorm:"size:100" json:"crop"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `gorm:"index" json:"updated_at"`
	Messages          []Message  `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

func (Conversation) TableName() string {
	return "ai_conversations"
}

func (c *Conversation) BeforeCreate(_ interface{}) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
