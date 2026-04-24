package repository

import (
	"context"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type DocumentRepository interface {
	Create(ctx context.Context, doc *models.Document) error
	GetByGroupID(ctx context.Context, groupID uint) ([]models.Document, error)
	GetByID(ctx context.Context, id uint) (*models.Document, error)
	Delete(ctx context.Context, id uint) error
}

type documentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) Create(ctx context.Context, doc *models.Document) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *documentRepository) GetByGroupID(ctx context.Context, groupID uint) ([]models.Document, error) {
	var docs []models.Document
	err := r.db.WithContext(ctx).Preload("UploadedBy").
		Where("group_id = ?", groupID).
		Order("created_at DESC").
		Find(&docs).Error
	return docs, err
}

func (r *documentRepository) GetByID(ctx context.Context, id uint) (*models.Document, error) {
	var doc models.Document
	err := r.db.WithContext(ctx).First(&doc, id).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *documentRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Document{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
