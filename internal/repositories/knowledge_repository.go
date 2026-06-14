package repositories

import (
	"context"
	"strings"

	"github.com/agriconnect-ai/internal/models"
	"gorm.io/gorm"
)

type KnowledgeRepository interface {
	Search(ctx context.Context, crop, category string, limit int) ([]models.AgriculturalDocument, error)
	SeedDocuments(ctx context.Context, docs []models.AgriculturalDocument) error
}

type knowledgeRepository struct {
	db *gorm.DB
}

func NewKnowledgeRepository(db *gorm.DB) KnowledgeRepository {
	return &knowledgeRepository{db: db}
}

func (r *knowledgeRepository) Search(ctx context.Context, crop, category string, limit int) ([]models.AgriculturalDocument, error) {
	var docs []models.AgriculturalDocument
	query := r.db.WithContext(ctx).Model(&models.AgriculturalDocument{})

	conditions := []string{}
	args := []interface{}{}

	if crop != "" {
		conditions = append(conditions, "LOWER(crop) = ?")
		args = append(args, strings.ToLower(crop))
	}
	if category != "" {
		conditions = append(conditions, "LOWER(category) = ?")
		args = append(args, strings.ToLower(category))
	}

	if len(conditions) > 0 {
		query = query.Where(strings.Join(conditions, " AND "), args...)
	}

	// Score: crop+category match first, then category only
	query = query.Order("reviewed DESC, created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&docs).Error
	return docs, err
}

func (r *knowledgeRepository) SeedDocuments(ctx context.Context, docs []models.AgriculturalDocument) error {
	for i := range docs {
		var existing models.AgriculturalDocument
		err := r.db.WithContext(ctx).
			Where("title = ? AND crop = ?", docs[i].Title, docs[i].Crop).
			First(&existing).Error
		if err != nil {
			// Not found, create
			if err := r.db.WithContext(ctx).Create(&docs[i]).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
