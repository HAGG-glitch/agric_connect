package models

import (
	"time"

	"gorm.io/datatypes"
)

type WeatherCache struct {
	District  string         `gorm:"primaryKey;size:100" json:"district"`
	Response  datatypes.JSON `gorm:"type:jsonb;not null" json:"response"`
	FetchedAt time.Time      `gorm:"not null" json:"fetched_at"`
}

func (WeatherCache) TableName() string {
	return "weather_cache"
}
