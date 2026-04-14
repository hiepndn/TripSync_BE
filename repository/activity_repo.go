package repository

import (
	"context"
	"strings"
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
	DeleteAllActivities(ctx context.Context, groupID uint) error
	UpsertRating(ctx context.Context, activityID uint, userID uint, rating int) error
	GetRatingContext(ctx context.Context) (*dto.RatingContext, error)
	GetSuggestions(ctx context.Context, groupID uint, activityType string, location string, routeDestinations string) ([]dto.SuggestionResponse, error)
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
          EXISTS(SELECT 1 FROM activity_votes v WHERE v.activity_id = a.id AND v.user_id = ?) AS has_voted,
          COALESCE(ROUND(AVG(ar.rating)::numeric, 1), 0) AS average_user_rating,
          COALESCE((SELECT ar2.rating FROM activity_ratings ar2 WHERE ar2.activity_id = a.id AND ar2.user_id = ?), 0) AS my_rating
       FROM activities a
       LEFT JOIN activity_ratings ar ON ar.activity_id = a.id
       WHERE a.group_id = ? AND a.deleted_at IS NULL
       GROUP BY a.id
       ORDER BY a.start_time ASC
    `

	err := r.db.WithContext(ctx).Raw(query, userID, userID, groupID).Scan(&activities).Error
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

func (r *activityRepositoryImpl) DeleteAllActivities(ctx context.Context, groupID uint) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("group_id = ?", groupID).
		Delete(&models.Activity{}).Error
}

func (r *activityRepositoryImpl) UpsertRating(ctx context.Context, activityID uint, userID uint, rating int) error {
	query := `
		INSERT INTO activity_ratings (activity_id, user_id, rating, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
		ON CONFLICT (activity_id, user_id)
		DO UPDATE SET rating = EXCLUDED.rating, updated_at = NOW()
	`
	return r.db.WithContext(ctx).Exec(query, activityID, userID, rating).Error
}

func (r *activityRepositoryImpl) GetRatingContext(ctx context.Context) (*dto.RatingContext, error) {
	type row struct {
		Name      string  `gorm:"column:name"`
		Type      string  `gorm:"column:type"`
		Location  string  `gorm:"column:location"`
		AvgRating float64 `gorm:"column:avg_rating"`
	}

	highlyQuery := `
		SELECT a.name, a.type, a.location, ROUND(AVG(ar.rating)::numeric, 1) AS avg_rating
		FROM activities a
		JOIN activity_ratings ar ON ar.activity_id = a.id
		GROUP BY a.id, a.name, a.type, a.location
		HAVING AVG(ar.rating) >= 4.0
		ORDER BY avg_rating DESC
		LIMIT 10
	`
	poorlyQuery := `
		SELECT a.name, a.type, a.location, ROUND(AVG(ar.rating)::numeric, 1) AS avg_rating
		FROM activities a
		JOIN activity_ratings ar ON ar.activity_id = a.id
		GROUP BY a.id, a.name, a.type, a.location
		HAVING AVG(ar.rating) <= 2.0
		ORDER BY avg_rating ASC
		LIMIT 10
	`

	var highlyRows []row
	if err := r.db.WithContext(ctx).Raw(highlyQuery).Scan(&highlyRows).Error; err != nil {
		return nil, err
	}

	var poorlyRows []row
	if err := r.db.WithContext(ctx).Raw(poorlyQuery).Scan(&poorlyRows).Error; err != nil {
		return nil, err
	}

	result := &dto.RatingContext{}
	for _, row := range highlyRows {
		result.HighlyRated = append(result.HighlyRated, dto.RatingContextItem{
			Name:              row.Name,
			Type:              row.Type,
			Location:          row.Location,
			AverageUserRating: row.AvgRating,
		})
	}
	for _, row := range poorlyRows {
		result.PoorlyRated = append(result.PoorlyRated, dto.RatingContextItem{
			Name:              row.Name,
			Type:              row.Type,
			Location:          row.Location,
			AverageUserRating: row.AvgRating,
		})
	}

	return result, nil
}

func (r *activityRepositoryImpl) GetSuggestions(ctx context.Context, groupID uint, activityType string, location string, routeDestinations string) ([]dto.SuggestionResponse, error) {
	locationPattern := "%" + location + "%"
	upperType := strings.ToUpper(activityType)

	// Tách route_destinations thành các từ khóa để match linh hoạt
	// VD: "Đà Nẵng, Hội An" → tìm group khác có route_destinations chứa "Đà Nẵng" HOẶC "Hội An"
	// Dùng cách đơn giản: lấy từng destination và OR chúng lại
	// Nhưng để đơn giản và hiệu quả, dùng ANY keyword từ route_destinations của group hiện tại
	// Cách: split bằng dấu phẩy, lấy từng phần, build điều kiện OR

	// Build danh sách pattern từ route_destinations
	var routePatterns []string
	for _, dest := range strings.Split(routeDestinations, ",") {
		dest = strings.TrimSpace(dest)
		if dest != "" {
			routePatterns = append(routePatterns, "%"+dest+"%")
		}
	}

	// Nếu không có route_destinations, chỉ filter theo location của activity
	if len(routePatterns) == 0 {
		routePatterns = []string{locationPattern}
	}

	// Build OR conditions cho route_destinations
	routeConditions := ""
	routeArgs := []interface{}{}
	for i, p := range routePatterns {
		if i > 0 {
			routeConditions += " OR "
		}
		routeConditions += "g.route_destinations ILIKE ?"
		routeArgs = append(routeArgs, p)
	}

	ratedQuery := `
		SELECT
			a.id, a.name, a.type, a.location, a.description,
			a.estimated_cost, a.currency, a.image_url, a.rating, a.external_link,
			a.group_id, a.status, a.created_by, a.lat, a.lng, a.place_id,
			a.is_ai_generated, a.start_time, a.end_time,
			ROUND(AVG(ar.rating)::numeric, 1) AS average_user_rating
		FROM activities a
		JOIN activity_ratings ar ON ar.activity_id = a.id
		JOIN groups g ON g.id = a.group_id
		WHERE UPPER(a.type) = ?
		  AND a.location ILIKE ?
		  AND (` + routeConditions + `)
		  AND a.group_id != ?
		  AND a.deleted_at IS NULL
		  AND g.deleted_at IS NULL
		GROUP BY a.id
		ORDER BY average_user_rating DESC
		LIMIT 5
	`

	args := []interface{}{upperType, locationPattern}
	args = append(args, routeArgs...)
	args = append(args, groupID)

	var ratedResults []dto.SuggestionResponse
	if err := r.db.WithContext(ctx).Raw(ratedQuery, args...).Scan(&ratedResults).Error; err != nil {
		return nil, err
	}

	return ratedResults, nil
}
