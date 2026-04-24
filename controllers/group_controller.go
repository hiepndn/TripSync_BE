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

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Thiếu thông tin hoặc sai định dạng"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Không tìm thấy thông tin xác thực"})
		return
	}
	uid := uint(userID.(float64))

	goCtx := c.Request.Context()
	newGroup, err := gc.groupUC.CreateGroup(goCtx, req, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

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

	goCtx := c.Request.Context()
	groups, err := gc.groupUC.GetUserGroupsWithRole(goCtx, uid)
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

	userID, _ := c.Get("user_id")
	uid := uint(userID.(float64))

	goCtx := c.Request.Context()
	joinedGroup, err := gc.groupUC.JoinGroupByCode(goCtx, req.InviteCode, uid)
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

	goCtx := c.Request.Context()
	group, role, members, err := gc.groupUC.GetGroupDetail(goCtx, groupID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Thành công",
		"data": gin.H{
			"group_info": group,
			"my_role":    role,
			"members":    members,
		},
	})
}

func (c *GroupController) RegenerateAI(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)

	userIDVal, _ := ctx.Get("user_id")
	userID := int(userIDVal.(float64))

	goCtx := ctx.Request.Context()
	if err := c.groupUC.RegenerateAIActivities(goCtx, uint(groupID), uint(userID)); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Đang khởi tạo lại lịch trình bằng AI...",
	})
}

func (gc *GroupController) UpdateGroup(c *gin.Context) {
	idParam := c.Param("id")
	var groupID uint
	fmt.Sscanf(idParam, "%d", &groupID)

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Chưa đăng nhập"})
		return
	}
	userID := uint(userIDVal.(float64))

	var req dto.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Dữ liệu không hợp lệ"})
		return
	}

	goCtx := c.Request.Context()
	updated, err := gc.groupUC.UpdateGroup(goCtx, groupID, userID, req)
	if err != nil {
		status := http.StatusInternalServerError
		msg := err.Error()
		if msg == "chỉ Admin mới có quyền chỉnh sửa thông tin nhóm" {
			status = http.StatusForbidden
		} else if msg == "ngày kết thúc không được trước ngày bắt đầu" {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"success": false, "error": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Cập nhật thành công!", "data": updated})
}

func (gc *GroupController) KickMember(c *gin.Context) {
	idParam := c.Param("id")
	userIDParam := c.Param("user_id")
	var groupID, targetUserID uint
	fmt.Sscanf(idParam, "%d", &groupID)
	fmt.Sscanf(userIDParam, "%d", &targetUserID)

	requestingIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Chưa đăng nhập"})
		return
	}
	requestingID := uint(requestingIDVal.(float64))

	goCtx := c.Request.Context()
	if err := gc.groupUC.RemoveMember(goCtx, groupID, targetUserID, requestingID); err != nil {
		status := http.StatusInternalServerError
		msg := err.Error()
		switch msg {
		case "không thể tự xóa chính mình khỏi nhóm":
			status = http.StatusBadRequest
		case "chỉ Admin mới có quyền xóa thành viên":
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"success": false, "error": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã xóa thành viên khỏi nhóm"})
}

func (gc *GroupController) DeleteGroup(c *gin.Context) {
	idParam := c.Param("id")
	var groupID uint
	fmt.Sscanf(idParam, "%d", &groupID)

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Chưa đăng nhập"})
		return
	}
	userID := uint(userIDVal.(float64))

	goCtx := c.Request.Context()
	if err := gc.groupUC.DeleteGroup(goCtx, groupID, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "chỉ Admin mới có quyền xóa nhóm" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã xóa nhóm thành công"})
}

func (gc *GroupController) UpdateVisibility(c *gin.Context) {
	idParam := c.Param("id")
	var groupID uint
	fmt.Sscanf(idParam, "%d", &groupID)

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Chưa đăng nhập"})
		return
	}
	userID := uint(userIDVal.(float64))

	var req dto.UpdateVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Dữ liệu không hợp lệ"})
		return
	}

	goCtx := c.Request.Context()
	if err := gc.groupUC.UpdateVisibility(goCtx, groupID, userID, req.IsPublic); err != nil {
		msg := err.Error()
		status := http.StatusInternalServerError
		switch msg {
		case "chỉ Admin mới có quyền thay đổi chế độ công khai":
			status = http.StatusForbidden
		case "không tìm thấy nhóm":
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"success": false, "error": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Cập nhật chế độ công khai thành công"})
}

func (gc *GroupController) GetPublicGroups(c *gin.Context) {
	goCtx := c.Request.Context()
	groups, err := gc.groupUC.GetPublicGroups(goCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Lỗi khi lấy danh sách nhóm công khai"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    groups,
	})
}

func (gc *GroupController) GetPublicGroupDetail(c *gin.Context) {
	idParam := c.Param("id")
	var groupID uint
	fmt.Sscanf(idParam, "%d", &groupID)

	goCtx := c.Request.Context()
	result, err := gc.groupUC.GetPublicGroupDetail(goCtx, groupID)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "not_found":
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Không tìm thấy nhóm"})
		case "forbidden":
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Nhóm này không công khai"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Lỗi hệ thống"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
