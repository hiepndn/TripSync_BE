package controllers

import (
	"net/http"
	"strconv"
	"tripsync-backend/dto"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

type ChecklistController struct {
	useCase usecase.ChecklistUseCase
}

func NewChecklistController(useCase usecase.ChecklistUseCase) *ChecklistController {
	return &ChecklistController{useCase: useCase}
}

// 1. TẠO CÔNG VIỆC MỚI
func (c *ChecklistController) CreateItem(ctx *gin.Context) {
	groupID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	var req dto.CreateChecklistItemReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	goCtx := ctx.Request.Context()
	item, err := c.useCase.CreateItem(goCtx, uint(groupID), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Đã thêm công việc", "data": item})
}

// 2. LẤY DANH SÁCH CÔNG VIỆC CỦA NHÓM
func (c *ChecklistController) GetItems(ctx *gin.Context) {
	groupID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	goCtx := ctx.Request.Context()
	items, err := c.useCase.GetItemsByGroup(goCtx, uint(groupID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": items})
}

// 3. ĐÁNH DẤU XONG / CHƯA XONG
func (c *ChecklistController) ToggleComplete(ctx *gin.Context) {
	groupID, _ := strconv.Atoi(ctx.Param("id"))
	itemID, _ := strconv.Atoi(ctx.Param("itemId"))

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy user"})
		return
	}
	userID := int(userIDVal.(float64))

	goCtx := ctx.Request.Context()
	if err := c.useCase.ToggleComplete(goCtx, uint(itemID), uint(groupID), uint(userID)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Cập nhật trạng thái thành công"})
}

// 4. PHÂN CÔNG NGƯỜI PHỤ TRÁCH
func (c *ChecklistController) AssignMember(ctx *gin.Context) {
	groupID, _ := strconv.Atoi(ctx.Param("id"))
	itemID, _ := strconv.Atoi(ctx.Param("itemId"))

	var req dto.AssignMemberReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	goCtx := ctx.Request.Context()
	if err := c.useCase.AssignMember(goCtx, uint(itemID), uint(groupID), req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Phân công thành công"})
}

// 5. XÓA CÔNG VIỆC
func (c *ChecklistController) DeleteItem(ctx *gin.Context) {
	groupID, _ := strconv.Atoi(ctx.Param("id"))
	itemID, _ := strconv.Atoi(ctx.Param("itemId"))

	goCtx := ctx.Request.Context()
	if err := c.useCase.DeleteItem(goCtx, uint(itemID), uint(groupID)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Đã xóa công việc"})
}
