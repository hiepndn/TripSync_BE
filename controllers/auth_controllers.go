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

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type RegisterRequest struct {
	FullName string `json:"fullName" binding:"required"`
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

func (ac *AuthController) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Dữ liệu không hợp lệ. Vui lòng kiểm tra lại định dạng email/mật khẩu!",
		})
		return
	}

	goCtx := c.Request.Context()
	token, err := ac.authUC.Login(goCtx, req.Email, req.Password)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đăng nhập thành công!",
		"token":   token,
	})
}

func (ac *AuthController) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Thiếu thông tin hoặc sai định dạng!",
		})
		return
	}

	goCtx := c.Request.Context()
	err := ac.authUC.Register(goCtx, req.FullName, req.Email, req.Password)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

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

	goCtx := c.Request.Context()
	user, err := ac.authUC.GetProfile(goCtx, userID)
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

	goCtx := c.Request.Context()
	updatedUser, err := ac.authUC.UpdateProfile(goCtx, userID, req.FullName, req.AvatarURL)
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

	goCtx := c.Request.Context()
	if err := ac.authUC.ChangePassword(goCtx, userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đổi mật khẩu thành công!",
	})
}
