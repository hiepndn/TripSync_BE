package controllers

import (
	"net/http"
	"os"
	"strconv"
	"tripsync-backend/models"
	"tripsync-backend/repository"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AdminController struct {
	userRepo  repository.UserRepository
	groupRepo repository.GroupRepository
}

func NewAdminController(userRepo repository.UserRepository, groupRepo repository.GroupRepository) *AdminController {
	return &AdminController{userRepo: userRepo, groupRepo: groupRepo}
}

// POST /api/admin/seed — tạo hoặc nâng cấp tài khoản SUPERADMIN
func (ac *AdminController) SeedAdmin(c *gin.Context) {
	var body struct {
		SeedKey  string `json:"seed_key" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		FullName string `json:"full_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu thông tin"})
		return
	}

	expectedKey := os.Getenv("ADMIN_SEED_KEY")
	if expectedKey == "" || body.SeedKey != expectedKey {
		c.JSON(http.StatusForbidden, gin.H{"error": "Seed key không hợp lệ"})
		return
	}

	goCtx := c.Request.Context()

	user, err := ac.userRepo.FindByEmail(goCtx, body.Email)
	if err != nil {
		hashedPw, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi mã hóa mật khẩu"})
			return
		}
		newUser := &models.User{
			Email:    body.Email,
			Password: string(hashedPw),
			FullName: body.FullName,
			Role:     "SUPERADMIN",
		}
		if err := ac.userRepo.CreateUser(goCtx, newUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo tài khoản: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã tạo tài khoản SUPERADMIN mới", "email": body.Email})
		return
	}

	if err := ac.userRepo.UpdateUserRole(goCtx, user.ID, "SUPERADMIN"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã nâng cấp tài khoản lên SUPERADMIN", "email": body.Email})
}

func (ac *AdminController) GetStats(c *gin.Context) {
	goCtx := c.Request.Context()
	totalUsers, _ := ac.userRepo.GetTotalUsers(goCtx)
	newUsersToday, _ := ac.userRepo.GetNewUsersToday(goCtx)
	totalGroups, _ := ac.groupRepo.GetTotalGroups(goCtx)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_users":     totalUsers,
			"new_users_today": newUsersToday,
			"total_groups":    totalGroups,
		},
	})
}

// GET /api/admin/stats/chart — data cho chart 7 ngày gần nhất
func (ac *AdminController) GetChartData(c *gin.Context) {
	goCtx := c.Request.Context()
	usersByDay, err := ac.userRepo.GetUsersByDay(goCtx, 7)
	if err != nil {
		usersByDay = []map[string]interface{}{}
	}
	groupsByDay, err := ac.groupRepo.GetGroupsByDay(goCtx, 7)
	if err != nil {
		groupsByDay = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users_by_day":  usersByDay,
			"groups_by_day": groupsByDay,
		},
	})
}

// GET /api/admin/stats/growth?period=30&entity=users
func (ac *AdminController) GetGrowthChart(c *gin.Context) {
	periodStr := c.DefaultQuery("period", "30")
	entity := c.DefaultQuery("entity", "users")
	period, _ := strconv.Atoi(periodStr)

	goCtx := c.Request.Context()
	var data []map[string]interface{}
	var err error

	switch entity {
	case "groups":
		data, err = ac.groupRepo.GetGrowthData(goCtx, period)
	default:
		data, err = ac.userRepo.GetGrowthData(goCtx, period)
	}

	if err != nil || data == nil {
		data = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// GET /api/admin/users?page=1&page_size=10&search=
func (ac *AdminController) GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	goCtx := c.Request.Context()
	users, total, err := ac.userRepo.GetAllUsers(goCtx, page, pageSize, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách người dùng"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users":     users,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// PUT /api/admin/users/:id/role
func (ac *AdminController) UpdateUserRole(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID không hợp lệ"})
		return
	}

	var body struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	validRoles := map[string]bool{"USER": true, "SUPERADMIN": true}
	if !validRoles[body.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role không hợp lệ"})
		return
	}

	goCtx := c.Request.Context()
	if err := ac.userRepo.UpdateUserRole(goCtx, uint(userID), body.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi cập nhật role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Cập nhật role thành công"})
}

// DELETE /api/admin/users/:id
func (ac *AdminController) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID không hợp lệ"})
		return
	}

	selfIDVal, _ := c.Get("user_id")
	selfID := uint(selfIDVal.(float64))
	if uint(userID) == selfID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể xóa tài khoản của chính mình"})
		return
	}

	goCtx := c.Request.Context()
	if err := ac.userRepo.DeleteUser(goCtx, uint(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi xóa người dùng"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã xóa người dùng"})
}

// GET /api/admin/groups?page=1&page_size=10&search=
func (ac *AdminController) GetGroups(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	goCtx := c.Request.Context()
	groups, total, err := ac.groupRepo.GetAllGroups(goCtx, page, pageSize, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách nhóm"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"groups":    groups,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// DELETE /api/admin/groups/:id
func (ac *AdminController) DeleteGroup(c *gin.Context) {
	groupIDStr := c.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID không hợp lệ"})
		return
	}

	goCtx := c.Request.Context()
	if err := ac.groupRepo.AdminDeleteGroup(goCtx, uint(groupID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi xóa nhóm"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã xóa nhóm"})
}
