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
	// Giả sử Middleware JWT của bác đã nhét userID vào context với key là "userID"
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin xác thực"})
		return
	}

	var userID uint
	if floatID, ok := userIDVal.(float64); ok {
		userID = uint(floatID)
	} else {
		// Đề phòng trường hợp lúc tạo token bác set user_id là string
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi định dạng ID người dùng trong token"})
		return
	}

	// Gọi UseCase
	user, err := ac.authUC.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Trả về Frontend
	c.JSON(http.StatusOK, gin.H{
		"message": "Thành công",
		"data":    user,
	})
}
