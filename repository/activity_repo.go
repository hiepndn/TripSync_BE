package repository

import (
	"context"
	"time"
	"tripsync-backend/dto"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type ActivityRepository interface {
	GetActivitiesByGroup(ctx context.Context, groupID int, userID int) ([]dto.ActivityResponse, error)
	CreateWithGroupDateUpdate(ctx context.Context, activity *models.Activity, needsDateUpdate bool, newStartDate, newEndDate time.Time) error
	ToggleVote(ctx context.Context, activityID int, userID int) (bool, error)
	UpdateStatus(ctx context.Context, activityID uint, status string) error
	Create(ctx context.Context, activity *models.Activity) error
	GetByID(id int) (*models.Activity, error)
	Update(activity *models.Activity) error
	Delete(id int) error
	DeletePendingAIActivities(groupID uint) error
}

type activityRepositoryImpl struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepositoryImpl{db: db}
}

func (r *activityRepositoryImpl) GetActivitiesByGroup(ctx context.Context, groupID int, userID int) ([]dto.ActivityResponse, error) {
	var activities []dto.ActivityResponse

	query := `
       SELECT 
          a.id, a.name, a.type, a.location, a.description, a.start_time, a.end_time, 
          a.group_id, a.status, a.created_by, a.lat, a.lng, a.place_id,
          a.is_ai_generated, a.estimated_cost, a.currency, a.image_url, a.rating, a.external_link,
          (SELECT COUNT(*) FROM activity_votes v WHERE v.activity_id = a.id) AS vote_count,
          EXISTS(SELECT 1 FROM activity_votes v WHERE v.activity_id = a.id AND v.user_id = ?) AS has_voted
       FROM activities a
       WHERE a.group_id = ? AND a.deleted_at IS NULL
       ORDER BY a.start_time ASC
    `

	// Chú ý thứ tự tham số: userID cho dấu ? đầu tiên, groupID cho dấu ? thứ hai
	err := r.db.WithContext(ctx).Raw(query, userID, groupID).Scan(&activities).Error
	if err != nil {
		return nil, err
	}

	return activities, nil
}

func (r *activityRepositoryImpl) CreateWithGroupDateUpdate(ctx context.Context, activity *models.Activity, needsDateUpdate bool, newStartDate, newEndDate time.Time) error {
	// Sử dụng db.Transaction của GORM
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 1. Nếu cần nới rộng ngày, update bảng GROUPS
		if needsDateUpdate {
			err := tx.Model(&models.Group{}).Where("id = ?", activity.GroupID).Updates(map[string]interface{}{
				"start_date": newStartDate,
				"end_date":   newEndDate,
			}).Error
			if err != nil {
				return err // Trả về err -> GORM tự động Rollback
			}
		}

		// 2. Insert Activity mới vào bảng ACTIVITIES
		if err := tx.Create(activity).Error; err != nil {
			return err // Trả về err -> GORM tự động Rollback
		}

		// Trả về nil -> GORM tự động Commit transaction
		return nil
	})
}

func (r *activityRepositoryImpl) ToggleVote(ctx context.Context, activityID int, userID int) (bool, error) {
	var vote models.ActivityVote

	// Tìm xem user này đã vote cho activity này chưa
	err := r.db.WithContext(ctx).Where("activity_id = ? AND user_id = ?", activityID, userID).First(&vote).Error

	if err != nil {
		// Nếu chưa tìm thấy record -> Tiến hành thêm Vote
		if err == gorm.ErrRecordNotFound {
			newVote := models.ActivityVote{
				ActivityID: uint(activityID),
				UserID:     uint(userID),
				VoteType:   "UPVOTE",
			}
			createErr := r.db.WithContext(ctx).Create(&newVote).Error
			return true, createErr
		}
		// Lỗi khác (DB sập, v.v.)
		return false, err
	}

	// Nếu đã tìm thấy record -> Tiến hành xóa (Hủy vote)
	deleteErr := r.db.WithContext(ctx).Delete(&vote).Error
	return false, deleteErr
}

func (r *activityRepositoryImpl) UpdateStatus(ctx context.Context, activityID uint, status string) error {
	// Dùng Model và Update của GORM để chỉ cập nhật 1 cột status
	err := r.db.WithContext(ctx).
		Model(&models.Activity{}).
		Where("id = ?", activityID).
		Update("status", status).Error

	return err
}

func (r *activityRepositoryImpl) Create(ctx context.Context, activity *models.Activity) error {
	// Dùng tx.Create đơn giản của GORM để insert 1 record
	return r.db.WithContext(ctx).Create(activity).Error
}

func (r *activityRepositoryImpl) GetByID(id int) (*models.Activity, error) {
	var activity models.Activity
	if err := r.db.First(&activity, id).Error; err != nil {
		return nil, err
	}
	return &activity, nil
}

func (r *activityRepositoryImpl) Update(activity *models.Activity) error {
	return r.db.Save(activity).Error
}

func (r *activityRepositoryImpl) Delete(id int) error {
	// Dùng Transaction để xóa Vote trước, xóa Activity sau cho an toàn
	tx := r.db.Begin()

	// 1. Xóa tất cả các vote liên quan
	if err := tx.Where("activity_id = ?", id).Delete(&models.ActivityVote{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. Xóa activity
	if err := tx.Where("id = ?", id).Delete(&models.Activity{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *activityRepositoryImpl) DeletePendingAIActivities(groupID uint) error {
	return r.db.Unscoped().
		Where("group_id = ? AND status = ? AND is_ai_generated = ?", groupID, "PENDING", true).
		Delete(&models.Activity{}).Error
}
