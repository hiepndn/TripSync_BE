package repository

import (
	// Nhớ đổi tên module "TripSync_BE" cho đúng với máy bạn nha

	"errors"
	"fmt"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

// Tạo một Interface để định nghĩa các chức năng của Thủ kho (Giúp sau này dễ test)
type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	CreateUser(user *models.User) error
	FindByID(id uint) (*models.User, error)
	FindByIDWithPassword(id uint) (*models.User, error)
	UpdateProfile(userID uint, fullName, avatar string) error
	UpdatePassword(userID uint, hashedPassword string) error
	// Admin
	GetAllUsers(page, pageSize int, search string) ([]models.User, int64, error)
	GetTotalUsers() (int64, error)
	GetNewUsersToday() (int64, error)
	UpdateUserRole(userID uint, role string) error
	DeleteUser(userID uint) error
	GetUsersByDay(days int) ([]map[string]interface{}, error)
	GetGrowthData(period int) ([]map[string]interface{}, error)
}

type userRepository struct {
	db *gorm.DB
}

// Hàm khởi tạo Thủ kho, truyền cái Database (gorm.DB) vào cho ổng giữ chìa khóa
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db}
}

// Hàm đi tìm User dựa vào Email
func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User

	// Bảo GORM: "Ê, tìm cho tao 1 dòng đầu tiên (First) có email khớp với biến email"
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err // Nếu không thấy hoặc lỗi mạng thì trả về lỗi
	}

	return &user, nil // Tìm thấy thì đưa data đây!
}

func (r *userRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error // GORM tự động tạo record mới
}

func (r *userRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	// Chỉ select đúng những trường cần thiết để tối ưu hiệu năng
	err := r.db.Select("id, full_name, email, avatar").Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("không tìm thấy người dùng")
		}
		return nil, err
	}
	return &user, nil
}

// FindByIDWithPassword lấy user kèm password hash để xác thực khi đổi mật khẩu
func (r *userRepository) FindByIDWithPassword(id uint) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("không tìm thấy người dùng")
		}
		return nil, err
	}
	return &user, nil
}

// UpdateProfile cập nhật tên hiển thị và URL avatar
func (r *userRepository) UpdateProfile(userID uint, fullName, avatar string) error {
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
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
}

// UpdatePassword lưu mật khẩu đã được hash
func (r *userRepository) UpdatePassword(userID uint, hashedPassword string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("password", hashedPassword).Error
}

func (r *userRepository) GetAllUsers(page, pageSize int, search string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.Model(&models.User{})
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

func (r *userRepository) GetTotalUsers() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Count(&count).Error
	return count, err
}

func (r *userRepository) GetNewUsersToday() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).
		Where("DATE(created_at) = CURRENT_DATE").
		Count(&count).Error
	return count, err
}

func (r *userRepository) UpdateUserRole(userID uint, role string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("role", role).Error
}

func (r *userRepository) DeleteUser(userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Activity votes của user (junction table, không có DeletedAt → hard delete)
		if err := tx.Where("user_id = ?", userID).Delete(&models.ActivityVote{}).Error; err != nil {
			return err
		}

		// 2. Activity ratings của user
		if err := tx.Where("user_id = ?", userID).Delete(&models.ActivityRating{}).Error; err != nil {
			return err
		}

		// 3. Nullify created_by trên activities
		if err := tx.Model(&models.Activity{}).Where("created_by = ?", userID).Update("created_by", nil).Error; err != nil {
			return err
		}

		// 4. Expense splits của user
		if err := tx.Where("user_id = ?", userID).Delete(&models.ExpenseSplit{}).Error; err != nil {
			return err
		}

		// 5. Group favorites của user
		if err := tx.Where("user_id = ?", userID).Delete(&models.GroupFavorite{}).Error; err != nil {
			return err
		}

		// 6. Group members của user
		if err := tx.Where("user_id = ?", userID).Delete(&models.GroupMember{}).Error; err != nil {
			return err
		}

		// 7. Soft delete user (set deleted_at, giữ lại record)
		return tx.Delete(&models.User{}, userID).Error
	})
}

func (r *userRepository) GetUsersByDay(days int) ([]map[string]interface{}, error) {
	type result struct {
		Day   string `gorm:"column:day"`
		Count int    `gorm:"column:count"`
	}
	var rows []result
	err := r.db.Raw(`
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

// GetGrowthData trả về số lượng user tích lũy theo thời gian
// period=0: all time (group by month), period=1: hôm nay (group by hour)
// period=30/90/180: group by day, period=365: group by month
func (r *userRepository) GetGrowthData(period int) ([]map[string]interface{}, error) {
	type row struct {
		Label string  `gorm:"column:label"`
		Total float64 `gorm:"column:total"`
	}
	var rows []row

	var query string
	switch {
	case period == 1:
		// Hôm nay — group by hour
		query = `
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'HH24:00') AS label,
			       COUNT(*) AS total
			FROM users
			WHERE DATE(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh') = CURRENT_DATE
			GROUP BY label ORDER BY label`
	case period <= 90:
		// 30 hoặc 90 ngày — group by day
		query = fmt.Sprintf(`
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'DD/MM') AS label,
			       COUNT(*) AS total
			FROM users
			WHERE created_at >= NOW() - INTERVAL '%d days'
			GROUP BY label ORDER BY MIN(created_at)`, period)
	case period == 0:
		// All time — group by month
		query = `
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'MM/YYYY') AS label,
			       COUNT(*) AS total
			FROM users
			GROUP BY label ORDER BY MIN(created_at)`
	default:
		// 180, 365 — group by month
		query = fmt.Sprintf(`
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'MM/YYYY') AS label,
			       COUNT(*) AS total
			FROM users
			WHERE created_at >= NOW() - INTERVAL '%d days'
			GROUP BY label ORDER BY MIN(created_at)`, period)
	}

	if err := r.db.Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}

	// Tính cumulative (tổng tích lũy)
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
