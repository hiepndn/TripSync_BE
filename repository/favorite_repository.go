package repository

import (
	"context"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type FavoriteRepository interface {
	Toggle(ctx context.Context, userID uint, groupID uint) (isFavorited bool, err error)
	GetFavoritesByUser(ctx context.Context, userID uint) ([]models.GroupFavorite, error)
	IsFavorited(ctx context.Context, userID uint, groupID uint) (bool, error)
}

type favoriteRepository struct {
	db *gorm.DB
}

func NewFavoriteRepository(db *gorm.DB) FavoriteRepository {
	return &favoriteRepository{db: db}
}

// Toggle thêm hoặc xóa favorite, trả về trạng thái sau khi toggle
func (r *favoriteRepository) Toggle(ctx context.Context, userID uint, groupID uint) (bool, error) {
	var existing models.GroupFavorite
	err := r.db.WithContext(ctx).Where("user_id = ? AND group_id = ?", userID, groupID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		fav := models.GroupFavorite{UserID: userID, GroupID: groupID}
		if err := r.db.WithContext(ctx).Create(&fav).Error; err != nil {
			return false, err
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}

	if err := r.db.WithContext(ctx).Where("user_id = ? AND group_id = ?", userID, groupID).Delete(&models.GroupFavorite{}).Error; err != nil {
		return false, err
	}
	return false, nil
}

func (r *favoriteRepository) GetFavoritesByUser(ctx context.Context, userID uint) ([]models.GroupFavorite, error) {
	var favorites []models.GroupFavorite
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Preload("Group").
		Find(&favorites).Error
	return favorites, err
}

func (r *favoriteRepository) IsFavorited(ctx context.Context, userID uint, groupID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.GroupFavorite{}).
		Where("user_id = ? AND group_id = ?", userID, groupID).
		Count(&count).Error
	return count > 0, err
}
