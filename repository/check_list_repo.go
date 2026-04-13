package repository

import (
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type ChecklistRepository interface {
	CreateItem(item *models.ChecklistItem) error
	GetItemsByGroup(groupID uint) ([]models.ChecklistItem, error)
	GetItemByID(itemID uint, groupID uint) (*models.ChecklistItem, error)
	UpdateItem(item *models.ChecklistItem) error
	DeleteItem(itemID uint, groupID uint) error
}

type checklistRepository struct {
	db *gorm.DB
}

func NewChecklistRepository(db *gorm.DB) ChecklistRepository {
	return &checklistRepository{db: db}
}

func (r *checklistRepository) CreateItem(item *models.ChecklistItem) error {
	return r.db.Create(item).Error
}

func (r *checklistRepository) GetItemsByGroup(groupID uint) ([]models.ChecklistItem, error) {
	var items []models.ChecklistItem
	// Preload Assignee (người được giao) và CompletedBy (người check xong) để FE lấy Avatar
	err := r.db.
		Preload("Assignee").
		Preload("CompletedBy").
		Where("group_id = ?", groupID).
		Order("created_at asc").
		Find(&items).Error
	return items, err
}

func (r *checklistRepository) GetItemByID(itemID uint, groupID uint) (*models.ChecklistItem, error) {
	var item models.ChecklistItem
	err := r.db.Where("id = ? AND group_id = ?", itemID, groupID).First(&item).Error
	return &item, err
}

func (r *checklistRepository) UpdateItem(item *models.ChecklistItem) error {
	return r.db.Save(item).Error
}

func (r *checklistRepository) DeleteItem(itemID uint, groupID uint) error {
	return r.db.Where("id = ? AND group_id = ?", itemID, groupID).Delete(&models.ChecklistItem{}).Error
}
