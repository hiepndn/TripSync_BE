package controllers

import (
	"net/http"
	"strconv"
	"tripsync-backend/dto"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

type ActivityController struct {
	useCase usecase.ActivityUseCase
}

func NewActivityController(u usecase.ActivityUseCase) *ActivityController {
	return &ActivityController{useCase: u}
}

func (c *ActivityController) GetActivities(ctx *gin.Context) {
	// 1. Lấy group_id từ URL
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	// 2. Lấy user_id từ context (do Middleware Auth set vào)
	userIDVal, _ := ctx.Get("user_id")
	userID := int(userIDVal.(float64))

	// 3. Gọi UseCase
	activities, err := c.useCase.GetGroupActivities(ctx.Request.Context(), groupID, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. Trả về Response
	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    activities,
	})
}

func (c *ActivityController) CreateActivity(ctx *gin.Context) {
	// 1. Lấy group_id từ URL (ví dụ: /groups/1/activities)
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	// 2. Lấy user_id từ Context (Giả sử Middleware Auth của bạn đã set "user_id")
	// Chỗ này tùy cách bạn viết Middleware, thường sẽ lưu float64 (nếu dùng JWT parse ra) hoặc string
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64)) // Ép kiểu cẩn thận tùy theo token của bạn nhé

	// 3. Hứng và Validate Payload từ Client
	var req dto.CreateActivityReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Dữ liệu đầu vào không hợp lệ",
			"details": err.Error(),
		})
		return
	}

	err = c.useCase.CreateActivity(ctx.Request.Context(), groupID, userID, req)
	if err != nil {
		// Ở đây nếu xịn hơn có thể check err là NotFound hay InternalServerError để trả status code cho chuẩn
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 5. Trả về thành công
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Thêm hoạt động thành công!",
	})
}

func (c *ActivityController) ToggleVote(ctx *gin.Context) {
	// 1. Lấy activity_id từ URL
	activityIDStr := ctx.Param("activity_id")
	activityID, err := strconv.Atoi(activityIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID hoạt động không hợp lệ"})
		return
	}

	// 2. Lấy user_id từ token (đã qua Auth Middleware)
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64)) // Ép kiểu tùy setup JWT của bạn

	// 3. Gọi UseCase
	isAdded, err := c.useCase.ToggleActivityVote(ctx.Request.Context(), activityID, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi xử lý bình chọn"})
		return
	}

	// 4. Trả về Response
	msg := "Đã bỏ bình chọn"
	if isAdded {
		msg = "Đã bình chọn thành công"
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":  msg,
		"is_added": isAdded, // Frontend dùng cờ này để update lại state UI (xanh lá/xám)
	})
}

func (c *ActivityController) FinalizeActivity(ctx *gin.Context) {
	// 1. Lấy thông tin từ URL
	groupIDStr := ctx.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)

	activityIDStr := ctx.Param("activity_id")
	activityID, _ := strconv.Atoi(activityIDStr)

	// 2. Lấy UserID từ Token (Middleware Auth)
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	// 3. Gọi UseCase thực thi
	err := c.useCase.FinalizeActivity(ctx.Request.Context(), uint(groupID), uint(activityID), uint(userID))
	if err != nil {
		// Trả về lỗi 403 Forbidden nếu không phải Admin
		if err.Error() == "chỉ Admin mới có quyền chốt hoạt động vào lịch chính thức" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		// Trả về 500 cho các lỗi DB khác
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: " + err.Error()})
		return
	}

	// 4. Trả về thành công
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Đã chốt hoạt động vào lịch chính thức",
		"status":  "APPROVED",
	})
}

func (c *ActivityController) UpdateActivity(ctx *gin.Context) {
	groupID, _ := strconv.Atoi(ctx.Param("id")) // param :id của group
	activityID, _ := strconv.Atoi(ctx.Param("activity_id"))
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	var req dto.UpdateActivityReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if err := c.useCase.UpdateActivity(userID, groupID, activityID, req); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật hoạt động thành công",
	})
}

func (c *ActivityController) DeleteActivity(ctx *gin.Context) {
	groupID, _ := strconv.Atoi(ctx.Param("id"))
	activityID, _ := strconv.Atoi(ctx.Param("activity_id"))
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	if err := c.useCase.DeleteActivity(userID, groupID, activityID); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Xóa hoạt động thành công",
	})
}

func (c *ActivityController) RateActivity(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	activityIDStr := ctx.Param("activity_id")
	activityID, err := strconv.Atoi(activityIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity ID"})
		return
	}

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	var req dto.RateActivityReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Rating must be between 1 and 5"})
		return
	}

	if err := c.useCase.RateActivity(ctx.Request.Context(), groupID, activityID, userID, req.Rating); err != nil {
		if err.Error() == "bạn không phải là thành viên của nhóm này" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Đánh giá thành công"})
}

func (c *ActivityController) GetSuggestions(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	activityType := ctx.Query("type")
	location := ctx.Query("location")

	suggestions, err := c.useCase.GetSuggestions(ctx.Request.Context(), groupID, activityType, location)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "type must be one of HOTEL, ATTRACTION, RESTAURANT, CAMPING" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		if errMsg == "location query parameter is required" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "success", "data": suggestions})
}
