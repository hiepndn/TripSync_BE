package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"tripsync-backend/dto"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

type GroupController struct {
	groupUC usecase.GroupUseCase
}

func NewGroupController(uc usecase.GroupUseCase) *GroupController {
	return &GroupController{groupUC: uc}
}

type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	StartDate   string `json:"start_date" binding:"required"`
	EndDate     string `json:"end_date" binding:"required"`
}

func (gc *GroupController) CreateGroup(c *gin.Context) {
	var req dto.CreateGroupRequest

	// 1. Bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Thiếu thông tin hoặc sai định dạng"})
		return
	}

	// 2. Lấy userID từ Middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Không tìm thấy thông tin xác thực"})
		return
	}
	// Ép kiểu cho chuẩn (phụ thuộc vào jwt claims của bạn, thường là float64)
	uid := uint(userID.(float64))

	// 3. Gọi UseCase
	newGroup, err := gc.groupUC.CreateGroup(req, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 4. Trả về thành công
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tạo chuyến đi thành công!",
		"data":    newGroup,
	})
}

func (gc *GroupController) GetGroups(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Chưa đăng nhập"})
		return
	}
	uid := uint(userID.(float64))

	groups, err := gc.groupUC.GetUserGroups(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Lỗi khi lấy danh sách nhóm"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    groups,
	})
}

type JoinGroupRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

func (gc *GroupController) JoinGroup(c *gin.Context) {
	var req JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Vui lòng nhập mã mời"})
		return
	}

	// Lấy ID người dùng từ Middleware
	userID, _ := c.Get("user_id")
	uid := uint(userID.(float64))

	// Gọi UseCase xử lý
	joinedGroup, err := gc.groupUC.JoinGroupByCode(req.InviteCode, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tham gia nhóm thành công!",
		"data":    joinedGroup,
	})
}

func (gc *GroupController) GetDetail(c *gin.Context) {
	idParam := c.Param("id")
	var groupID uint
	fmt.Sscanf(idParam, "%d", &groupID)

	userIDVal, _ := c.Get("user_id")
	userID := uint(userIDVal.(float64))

	// Hứng thêm biến members từ UseCase
	group, role, members, err := gc.groupUC.GetGroupDetail(groupID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Trả JSON
	c.JSON(http.StatusOK, gin.H{
		"message": "Thành công",
		"data": gin.H{
			"group_info": group,
			"my_role":    role,
			"members":    members, // Đầy đủ Avatar, Name, Role của cả nhóm!
		},
	})
}

func (c *GroupController) RegenerateAI(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)

	// Ép kiểu float64 y như ông hay làm
	userIDVal, _ := ctx.Get("user_id")
	userID := int(userIDVal.(float64))

	if err := c.groupUC.RegenerateAIActivities(uint(groupID), uint(userID)); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Đang khởi tạo lại lịch trình bằng AI...",
	})
}
