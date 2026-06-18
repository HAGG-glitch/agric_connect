package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MarketPrice struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Commodity  string     `gorm:"size:100;not null;index"`
	MarketName string     `gorm:"size:200;not null"`
	District   string     `gorm:"size:100;not null;index"`
	Price      float64    `gorm:"type:numeric(12,2);not null"`
	Currency   string     `gorm:"size:10;not null;default:SLE"`
	Unit       string     `gorm:"size:50;not null"`
	Source     string     `gorm:"size:200"`
	IsVerified bool       `gorm:"not null;default:false"`
	CreatedBy  *uuid.UUID `gorm:"type:uuid"`
	CreatedAt  time.Time  `gorm:"not null;default:now()"`
	UpdatedAt  time.Time  `gorm:"not null;default:now()"`
}

func (MarketPrice) TableName() string { return "market_prices" }

func (m *MarketPrice) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
