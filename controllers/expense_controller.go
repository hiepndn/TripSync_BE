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
	// 1. Lấy GroupID từ URL
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	// 2. Lấy UserID (người trả tiền/người tạo hóa đơn) từ Token
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	payerID := int(userIDVal.(float64))

	// 3. Bind JSON body
	var req dto.CreateExpenseReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu hóa đơn không hợp lệ: " + err.Error()})
		return
	}

	// 4. Gọi UseCase
	if err := c.useCase.CreateExpense(uint(groupID), uint(payerID), req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 5. Trả kết quả
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Đã thêm khoản chi tiêu thành công",
	})
}

// ==========================================
// TÍNH TOÁN CÔNG NỢ TỐI ƯU (SMART BILL SPLITTER)
// ==========================================
func (c *ExpenseController) GetOptimalDebts(ctx *gin.Context) {
	// 1. Lấy GroupID từ URL
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	// 2. Gọi UseCase để chạy Thuật toán Tham lam
	settlements, err := c.useCase.CalculateOptimalDebts(uint(groupID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tính toán công nợ: " + err.Error()})
		return
	}

	// 3. Trả về mảng các giao dịch (hướng dẫn chuyển tiền)
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Tính toán công nợ thành công",
		"data":    settlements, // Mảng rỗng nếu mọi người đã hòa tiền
	})
}

// ==========================================
// THỐNG KÊ CHI TIÊU CỦA 1 THÀNH VIÊN
// ==========================================
func (c *ExpenseController) GetExpenseSummary(ctx *gin.Context) {
	// 1. Lấy GroupID từ URL
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	// 2. Lấy UserID từ Token
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := int(userIDVal.(float64))

	// 3. Gọi UseCase
	summary, err := c.useCase.GetExpenseSummary(uint(groupID), uint(userID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống: " + err.Error()})
		return
	}

	// 4. Trả kết quả
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Lấy thống kê thành công",
		"data":    summary,
	})
}

// ==========================================
// CHỐT SỔ / THANH TOÁN CÔNG NỢ
// ==========================================
func (c *ExpenseController) SettleDebt(ctx *gin.Context) {
	// 1. Lấy GroupID từ URL
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	// 2. Bind JSON body
	var req dto.SettleDebtReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu thanh toán không hợp lệ: " + err.Error()})
		return
	}

	// 3. Tạo một CreateExpenseReq (Loại SETTLEMENT) đại diện cho việc thanh toán
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

	// 4. Payer chính là người chuyển tiền (FromUserID)
	if err := c.useCase.CreateExpense(uint(groupID), req.FromUserID, settleExp); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi ghi nhận thanh toán: " + err.Error()})
		return
	}

	// 5. Trả kết quả
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Đã ghi nhận thanh toán thành công",
	})
}

// ==========================================
// DANH SÁCH LỊCH SỬ KHOẢN CHI
// ==========================================
func (c *ExpenseController) GetExpenseList(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	expenses, err := c.useCase.GetExpenseList(uint(groupID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy lịch sử chi tiêu: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Lấy lịch sử chi tiêu thành công",
		"data":    expenses,
	})
}
