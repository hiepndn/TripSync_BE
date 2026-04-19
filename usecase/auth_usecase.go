package usecase

import (
	"errors"
	"fmt"
	"os"
	"time"
	"tripsync-backend/models"
	"tripsync-backend/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase interface {
	Login(email, password string) (string, error)
	Register(fullName, email, password string) error
	GetProfile(id uint) (*models.User, error)
	UpdateProfile(userID uint, fullName, avatarURL string) (*models.User, error)
	ChangePassword(userID uint, oldPassword, newPassword string) error
}

type authUseCase struct {
	userRepo repository.UserRepository
}

func NewAuthUseCase(repo repository.UserRepository) AuthUseCase {
	return &authUseCase{userRepo: repo}
}

func (u *authUseCase) Login(email, password string) (string, error) {
	user, err := u.userRepo.FindByEmail(email)
	if err != nil {
		return "", errors.New("không tìm thấy tài khoản với email này")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("mật khẩu không chính xác")
	}

	secretKey := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		fmt.Println("❌ Tạch ở bước tạo Token:", err) // Log 6
		return "", errors.New("lỗi khi tạo token đăng nhập")
	}

	return tokenString, nil
}

func (u *authUseCase) Register(fullName, email, password string) error {
	// Kiểm tra xem email đã tồn tại chưa
	_, err := u.userRepo.FindByEmail(email)
	if err == nil {
		return errors.New("email này đã được sử dụng")
	}

	// Băm mật khẩu (Hash)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("lỗi khi mã hóa mật khẩu")
	}

	// Lưu xuống DB
	newUser := &models.User{
		FullName: fullName,
		Email:    email,
		Password: string(hashedPassword),
	}

	return u.userRepo.CreateUser(newUser)
}

func (u *authUseCase) GetProfile(id uint) (*models.User, error) {
	// Gọi xuống Repo để lấy data
	user, err := u.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *authUseCase) UpdateProfile(userID uint, fullName, avatarURL string) (*models.User, error) {
	// Cập nhật vào database
	if err := u.userRepo.UpdateProfile(userID, fullName, avatarURL); err != nil {
		return nil, errors.New("lỗi khi cập nhật hồ sơ")
	}
	// Trả lại profile mới nhất
	return u.userRepo.FindByID(userID)
}

func (u *authUseCase) ChangePassword(userID uint, oldPassword, newPassword string) error {
	// Lấy user kèm password hash để xác thực mật khẩu cũ
	user, err := u.userRepo.FindByIDWithPassword(userID)
	if err != nil {
		return errors.New("không tìm thấy tài khoản")
	}

	// So sánh mật khẩu cũ
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("mật khẩu cũ không chính xác")
	}

	// Hash mật khẩu mới
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("lỗi khi mã hóa mật khẩu mới")
	}

	return u.userRepo.UpdatePassword(userID, string(hashed))
}
