package diagnosis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, d *CropDiagnosis) error
	FindByID(ctx context.Context, id uuid.UUID) (*CropDiagnosis, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]CropDiagnosis, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	Update(ctx context.Context, d *CropDiagnosis) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, d *CropDiagnosis) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*CropDiagnosis, error) {
	var d CropDiagnosis
	err := r.db.WithContext(ctx).First(&d, "id = ?", id).Error
	if err != nil {
		return nil, fmt.Errorf("diagnosis not found: %w", err)
	}
	return &d, nil
}

func (r *repository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]CropDiagnosis, error) {
	var diags []CropDiagnosis
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&diags).Error
	return diags, err
}

func (r *repository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&CropDiagnosis{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *repository) Update(ctx context.Context, d *CropDiagnosis) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&CropDiagnosis{}, "id = ?", id).Error
}
