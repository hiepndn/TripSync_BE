package controllers

import (
	"net/http"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authUC usecase.AuthUseCase
}

func NewAuthController(uc usecase.AuthUseCase) *AuthController {
	return &AuthController{authUC: uc}
}

// LoginRequest là cái khay để hứng data từ React ném sang
// Cái tag `binding` cực kỳ xịn, nó giúp Gin tự động kiểm tra định dạng email và độ dài mật khẩu luôn!
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type RegisterRequest struct {
	FullName string `json:"fullName" binding:"required"` // Khớp với name bên React
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type UpdateProfileRequest struct {
	FullName  string `json:"fullName"`
	AvatarURL string `json:"avatarUrl"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required,min=6"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// helper: lấy userID từ context do JWT Middleware set
func getUserIDFromContext(c *gin.Context) (uint, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	floatID, ok := userIDVal.(float64)
	if !ok {
		return 0, false
	}
	return uint(floatID), true
}

// Login là hàm xử lý chính cho endpoint /api/auth/login
func (ac *AuthController) Login(c *gin.Context) {
	var req LoginRequest

	// 1. Đọc dữ liệu JSON từ request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Dữ liệu không hợp lệ. Vui lòng kiểm tra lại định dạng email/mật khẩu!",
		})
		return
	}

	// 2. Giao việc cho Đầu bếp UseCase (Xóa code giả cầy đi, dùng hàng thật)
	token, err := ac.authUC.Login(req.Email, req.Password)

	// 3. Nếu Đầu bếp báo lỗi (sai email, sai pass...)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   err.Error(), // Lấy đúng câu chửi từ UseCase trả về cho React
		})
		return
	}

	// 4. Thành công mỹ mãn! Trả Token thật về cho React
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đăng nhập thành công!",
		"token":   token,
	})
}

func (ac *AuthController) Register(c *gin.Context) {
	var req RegisterRequest

	// 1. Kiểm tra dữ liệu đầu vào
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Thiếu thông tin hoặc sai định dạng!",
		})
		return
	}

	// 2. Gọi Đầu bếp (UseCase) để thực hiện Đăng ký (lưu vào Database)
	err := ac.authUC.Register(req.FullName, req.Email, req.Password)

	// 3. Nếu UseCase báo lỗi (ví dụ: Email đã tồn tại)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(), // Trả cái câu lỗi từ UseCase về cho React hiển thị
		})
		return
	}

	// 4. THÀNH CÔNG: Trả về JSON báo tin vui cho React! 🎉
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đăng ký thành công! Chào mừng bạn đến với TripSync.",
	})
}

func (ac *AuthController) GetMe(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin xác thực"})
		return
	}

	user, err := ac.authUC.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Thành công",
		"data":    user,
	})
}

// UpdateProfile xử lý PUT /api/auth/me
func (ac *AuthController) UpdateProfile(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không xác định được người dùng"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	updatedUser, err := ac.authUC.UpdateProfile(userID, req.FullName, req.AvatarURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cập nhật hồ sơ thành công!",
		"data":    updatedUser,
	})
}

// ChangePassword xử lý PUT /api/auth/me/password
func (ac *AuthController) ChangePassword(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không xác định được người dùng"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập đủ mật khẩu cũ và mật khẩu mới (tối thiểu 6 ký tự)"})
		return
	}

	if err := ac.authUC.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đổi mật khẩu thành công!",
	})
}
