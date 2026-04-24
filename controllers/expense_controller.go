package controllers

import (
	"net/http"
	"strconv"
	"tripsync-backend/dto"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

type ExpenseController struct {
	useCase usecase.ExpenseUseCase
}

func NewExpenseController(useCase usecase.ExpenseUseCase) *ExpenseController {
	return &ExpenseController{useCase: useCase}
}

func (c *ExpenseController) CreateExpense(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	payerID := int(userIDVal.(float64))

	var req dto.CreateExpenseReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu hóa đơn không hợp lệ: " + err.Error()})
		return
	}

	goCtx := ctx.Request.Context()
	if err := c.useCase.CreateExpense(goCtx, uint(groupID), uint(payerID), req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Đã thêm khoản chi tiêu thành công",
	})
}

// GetOptimalDebts tính toán công nợ tối ưu (Smart Bill Splitter)
func (c *ExpenseController) GetOptimalDebts(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	goCtx := ctx.Request.Context()
	settlements, err := c.useCase.CalculateOptimalDebts(goCtx, uint(groupID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tính toán công nợ: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Tính toán công nợ thành công",
		"data":    settlements,
	})
}

// GetExpenseSummary thống kê chi tiêu của 1 thành viên
func (c *ExpenseController) GetExpenseSummary(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	goCtx := ctx.Request.Context()
	summary, err := c.useCase.GetExpenseSummary(goCtx, uint(groupID), uint(userID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Lấy thống kê thành công",
		"data":    summary,
	})
}

// SettleDebt chốt sổ / thanh toán công nợ
func (c *ExpenseController) SettleDebt(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	var req dto.SettleDebtReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu thanh toán không hợp lệ: " + err.Error()})
		return
	}

	settleExp := dto.CreateExpenseReq{
		Amount:      req.Amount,
		Currency:    "VND",
		Description: "Thanh toán công nợ",
		SplitType:   "SETTLEMENT",
		Splits: []dto.ExpenseSplitReq{
			{
				UserID:     req.ToUserID,
				AmountOwed: req.Amount,
			},
		},
	}

	goCtx := ctx.Request.Context()
	if err := c.useCase.CreateExpense(goCtx, uint(groupID), req.FromUserID, settleExp); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi ghi nhận thanh toán: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Đã ghi nhận thanh toán thành công",
	})
}

// GetExpenseList lấy danh sách lịch sử khoản chi
func (c *ExpenseController) GetExpenseList(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	goCtx := ctx.Request.Context()
	expenses, err := c.useCase.GetExpenseList(goCtx, uint(groupID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy lịch sử chi tiêu: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Lấy lịch sử chi tiêu thành công",
		"data":    expenses,
	})
}
