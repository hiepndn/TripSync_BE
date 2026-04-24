package usecase

import (
	"context"
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
	Login(ctx context.Context, email, password string) (string, error)
	Register(ctx context.Context, fullName, email, password string) error
	GetProfile(ctx context.Context, id uint) (*models.User, error)
	UpdateProfile(ctx context.Context, userID uint, fullName, avatarURL string) (*models.User, error)
	ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error
}

type authUseCase struct {
	userRepo repository.UserRepository
}

func NewAuthUseCase(repo repository.UserRepository) AuthUseCase {
	return &authUseCase{userRepo: repo}
}

func (u *authUseCase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := u.userRepo.FindByEmail(ctx, email)
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
		fmt.Println("❌ Tạch ở bước tạo Token:", err)
		return "", errors.New("lỗi khi tạo token đăng nhập")
	}

	return tokenString, nil
}

func (u *authUseCase) Register(ctx context.Context, fullName, email, password string) error {
	_, err := u.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return errors.New("email này đã được sử dụng")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("lỗi khi mã hóa mật khẩu")
	}

	newUser := &models.User{
		FullName: fullName,
		Email:    email,
		Password: string(hashedPassword),
	}

	return u.userRepo.CreateUser(ctx, newUser)
}

func (u *authUseCase) GetProfile(ctx context.Context, id uint) (*models.User, error) {
	user, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u *authUseCase) UpdateProfile(ctx context.Context, userID uint, fullName, avatarURL string) (*models.User, error) {
	if err := u.userRepo.UpdateProfile(ctx, userID, fullName, avatarURL); err != nil {
		return nil, errors.New("lỗi khi cập nhật hồ sơ")
	}
	return u.userRepo.FindByID(ctx, userID)
}

func (u *authUseCase) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	user, err := u.userRepo.FindByIDWithPassword(ctx, userID)
	if err != nil {
		return errors.New("không tìm thấy tài khoản")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("mật khẩu cũ không chính xác")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("lỗi khi mã hóa mật khẩu mới")
	}

	return u.userRepo.UpdatePassword(ctx, userID, string(hashed))
}
