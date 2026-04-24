package repository

import (
	"context"
	"errors"
	"fmt"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByIDWithPassword(ctx context.Context, id uint) (*models.User, error)
	UpdateProfile(ctx context.Context, userID uint, fullName, avatar string) error
	UpdatePassword(ctx context.Context, userID uint, hashedPassword string) error
	// Admin
	GetAllUsers(ctx context.Context, page, pageSize int, search string) ([]models.User, int64, error)
	GetTotalUsers(ctx context.Context) (int64, error)
	GetNewUsersToday(ctx context.Context) (int64, error)
	UpdateUserRole(ctx context.Context, userID uint, role string) error
	DeleteUser(ctx context.Context, userID uint) error
	GetUsersByDay(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetGrowthData(ctx context.Context, period int) ([]map[string]interface{}, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db}
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Select("id, full_name, email, avatar").Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("không tìm thấy người dùng")
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByIDWithPassword(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("không tìm thấy người dùng")
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, userID uint, fullName, avatar string) error {
	updates := map[string]interface{}{}
	if fullName != "" {
		updates["full_name"] = fullName
	}
	if avatar != "" {
		updates["avatar"] = avatar
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *userRepository) UpdatePassword(ctx context.Context, userID uint, hashedPassword string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("password", hashedPassword).Error
}

func (r *userRepository) GetAllUsers(ctx context.Context, page, pageSize int, search string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.WithContext(ctx).Model(&models.User{})
	if search != "" {
		query = query.Where("full_name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)
	err := query.Select("id, full_name, email, avatar, role, created_at, updated_at").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error

	return users, total, err
}

func (r *userRepository) GetTotalUsers(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}

func (r *userRepository) GetNewUsersToday(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("DATE(created_at) = CURRENT_DATE").
		Count(&count).Error
	return count, err
}

func (r *userRepository) UpdateUserRole(ctx context.Context, userID uint, role string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("role", role).Error
}

func (r *userRepository) DeleteUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.ActivityVote{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.ActivityRating{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Activity{}).Where("created_by = ?", userID).Update("created_by", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.ExpenseSplit{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.GroupFavorite{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.GroupMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.User{}, userID).Error
	})
}

func (r *userRepository) GetUsersByDay(ctx context.Context, days int) ([]map[string]interface{}, error) {
	type result struct {
		Day   string `gorm:"column:day"`
		Count int    `gorm:"column:count"`
	}
	var rows []result
	err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'DD/MM') AS day, COUNT(*) AS count
		FROM users
		WHERE created_at >= NOW() - INTERVAL '? days'
		GROUP BY day
		ORDER BY MIN(created_at)
	`, days).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(rows))
	for i, r := range rows {
		out[i] = map[string]interface{}{"day": r.Day, "count": r.Count}
	}
	return out, nil
}

func (r *userRepository) GetGrowthData(ctx context.Context, period int) ([]map[string]interface{}, error) {
	type row struct {
		Label string  `gorm:"column:label"`
		Total float64 `gorm:"column:total"`
	}
	var rows []row

	var query string
	switch {
	case period == 1:
		query = `
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'HH24:00') AS label,
			       COUNT(*) AS total
			FROM users
			WHERE DATE(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh') = CURRENT_DATE
			GROUP BY label ORDER BY label`
	case period <= 90:
		query = fmt.Sprintf(`
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'DD/MM') AS label,
			       COUNT(*) AS total
			FROM users
			WHERE created_at >= NOW() - INTERVAL '%d days'
			GROUP BY label ORDER BY MIN(created_at)`, period)
	case period == 0:
		query = `
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'MM/YYYY') AS label,
			       COUNT(*) AS total
			FROM users
			GROUP BY label ORDER BY MIN(created_at)`
	default:
		query = fmt.Sprintf(`
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'MM/YYYY') AS label,
			       COUNT(*) AS total
			FROM users
			WHERE created_at >= NOW() - INTERVAL '%d days'
			GROUP BY label ORDER BY MIN(created_at)`, period)
	}

	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]map[string]interface{}, len(rows))
	var cumulative float64
	for i, row := range rows {
		cumulative += row.Total
		out[i] = map[string]interface{}{
			"label": row.Label,
			"count": row.Total,
			"total": cumulative,
		}
	}
	return out, nil
}
