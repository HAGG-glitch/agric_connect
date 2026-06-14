package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/agriconnect-ai/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WeatherRepository interface {
	GetCache(ctx context.Context, district string) (map[string]interface{}, *time.Time, error)
	SetCache(ctx context.Context, district string, data map[string]interface{}) error
}

type weatherRepository struct {
	db *gorm.DB
}

func NewWeatherRepository(db *gorm.DB) WeatherRepository {
	return &weatherRepository{db: db}
}

func (r *weatherRepository) GetCache(ctx context.Context, district string) (map[string]interface{}, *time.Time, error) {
	var cache models.WeatherCache
	err := r.db.WithContext(ctx).First(&cache, "district = ?", district).Error
	if err != nil {
		return nil, nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(cache.Response, &data); err != nil {
		return nil, nil, err
	}
	return data, &cache.FetchedAt, nil
}

func (r *weatherRepository) SetCache(ctx context.Context, district string, data map[string]interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	cache := models.WeatherCache{
		District:  district,
		Response:  raw,
		FetchedAt: time.Now().UTC(),
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "district"}},
			DoUpdates: clause.AssignmentColumns([]string{"response", "fetched_at"}),
		}).
		Create(&cache).Error
}
