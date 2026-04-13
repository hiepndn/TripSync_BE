package repository

import (
	// Nhớ đổi tên module "TripSync_BE" cho đúng với máy bạn nha

	"errors"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

// Tạo một Interface để định nghĩa các chức năng của Thủ kho (Giúp sau này dễ test)
type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	CreateUser(user *models.User) error
	FindByID(id uint) (*models.User, error)
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
