package repository

import (
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type DocumentRepository interface {
	Create(doc *models.Document) error
	GetByGroupID(groupID uint) ([]models.Document, error)
	GetByID(id uint) (*models.Document, error)
	Delete(id uint) error
}

type documentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) Create(doc *models.Document) error {
	return r.db.Create(doc).Error
}

func (r *documentRepository) GetByGroupID(groupID uint) ([]models.Document, error) {
	var docs []models.Document
	err := r.db.Preload("UploadedBy").
		Where("group_id = ?", groupID).
		Order("created_at DESC").
		Find(&docs).Error
	return docs, err
}

func (r *documentRepository) GetByID(id uint) (*models.Document, error) {
	var doc models.Document
	err := r.db.First(&doc, id).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *documentRepository) Delete(id uint) error {
	result := r.db.Delete(&models.Document{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
