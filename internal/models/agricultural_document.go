package models

import (
	"time"

	"github.com/google/uuid"
)

type AgriculturalDocument struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Crop      string    `gorm:"size:100;index" json:"crop"`
	Category  string    `gorm:"size:100;not null;index" json:"category"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Language  string    `gorm:"size:20;not null;default:english" json:"language"`
	Source    string    `gorm:"type:text" json:"source"`
	Reviewed  bool      `gorm:"not null;default:false" json:"reviewed"`
	CreatedAt time.Time `json:"created_at"`
}

func (d *AgriculturalDocument) BeforeCreate(_ interface{}) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
