package repository

import (
	"context"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type ChecklistRepository interface {
	CreateItem(ctx context.Context, item *models.ChecklistItem) error
	GetItemsByGroup(ctx context.Context, groupID uint) ([]models.ChecklistItem, error)
	GetItemByID(ctx context.Context, itemID uint, groupID uint) (*models.ChecklistItem, error)
	UpdateItem(ctx context.Context, item *models.ChecklistItem) error
	DeleteItem(ctx context.Context, itemID uint, groupID uint) error
}

type checklistRepository struct {
	db *gorm.DB
}

func NewChecklistRepository(db *gorm.DB) ChecklistRepository {
	return &checklistRepository{db: db}
}

func (r *checklistRepository) CreateItem(ctx context.Context, item *models.ChecklistItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *checklistRepository) GetItemsByGroup(ctx context.Context, groupID uint) ([]models.ChecklistItem, error) {
	var items []models.ChecklistItem
	err := r.db.WithContext(ctx).
		Preload("Assignee").
		Preload("CompletedBy").
		Where("group_id = ?", groupID).
		Order("created_at asc").
		Find(&items).Error
	return items, err
}

func (r *checklistRepository) GetItemByID(ctx context.Context, itemID uint, groupID uint) (*models.ChecklistItem, error) {
	var item models.ChecklistItem
	err := r.db.WithContext(ctx).Where("id = ? AND group_id = ?", itemID, groupID).First(&item).Error
	return &item, err
}

func (r *checklistRepository) UpdateItem(ctx context.Context, item *models.ChecklistItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *checklistRepository) DeleteItem(ctx context.Context, itemID uint, groupID uint) error {
	return r.db.WithContext(ctx).Where("id = ? AND group_id = ?", itemID, groupID).Delete(&models.ChecklistItem{}).Error
}
