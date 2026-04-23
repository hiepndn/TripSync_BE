package usecase

import (
	"errors"
	"math"
	"tripsync-backend/dto"
	"tripsync-backend/models"
	"tripsync-backend/repository"
)

type ExpenseUseCase interface {
	CalculateOptimalDebts(groupID uint) ([]dto.DebtSettlement, error)
	CreateExpense(groupID uint, payerID uint, req dto.CreateExpenseReq) error
	GetExpenseSummary(groupID uint, userID uint) (dto.ExpenseSummaryRes, error)
	GetExpenseList(groupID uint) ([]models.Expense, error)
}

type expenseUseCase struct {
	expenseRepo repository.ExpenseRepository
}

func NewExpenseUseCase(expenseRepo repository.ExpenseRepository) ExpenseUseCase {
	return &expenseUseCase{
		expenseRepo: expenseRepo,
	}
}

// Struct nội bộ phục vụ tính toán thuật toán
type UserBalance struct {
	UserID uint
	Amount float64
}

// 🌟 TÍNH CÔNG NỢ — dùng thuật toán được cấu hình bởi ActiveAlgorithm
// Để đổi thuật toán: thay giá trị ActiveAlgorithm trong debt_algorithm.go
func (u *expenseUseCase) CalculateOptimalDebts(groupID uint) ([]dto.DebtSettlement, error) {
	expenses, splits, err := u.expenseRepo.GetAllByGroup(groupID)
	if err != nil {
		return nil, err
	}

	algo := GetAlgorithm(ActiveAlgorithm)
	settlements := algo.Calculate(expenses, splits)
	return settlements, nil
}

func (u *expenseUseCase) CreateExpense(groupID uint, payerID uint, req dto.CreateExpenseReq) error {
	// 1. Validate cốt lõi: Tổng tiền từng người nợ phải BẰNG tổng hóa đơn
	var totalSplit float64
	for _, s := range req.Splits {
		totalSplit += s.AmountOwed
	}

	// Dùng sai số 0.01 để so sánh float (tránh lỗi làm tròn số của máy tính)
	if math.Abs(totalSplit-req.Amount) > 0.01 {
		return errors.New("tổng tiền chia cho các thành viên không khớp với tổng hóa đơn")
	}

	// 2. Map dữ liệu từ DTO sang Model Expense
	expense := &models.Expense{
		GroupID:     groupID,
		PayerID:     payerID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		SplitType:   req.SplitType,
	}

	// 3. Map dữ liệu từ DTO sang mảng Model ExpenseSplit
	var splits []models.ExpenseSplit
	for _, s := range req.Splits {
		splits = append(splits, models.ExpenseSplit{
			UserID:     s.UserID,
			AmountOwed: s.AmountOwed,
		})
	}

	// 4. Gọi Repo lưu vào Database (đã dùng Transaction an toàn ở repo)
	return u.expenseRepo.CreateExpense(expense, splits)
}

// 🌟 LẤY THỐNG KÊ CHI TIÊU CỦA 1 MẢNH GHÉP (TỔNG CHI, ĐÃ CHI, CÒN NỢ)
func (u *expenseUseCase) GetExpenseSummary(groupID uint, userID uint) (dto.ExpenseSummaryRes, error) {
	expenses, splits, err := u.expenseRepo.GetAllByGroup(groupID)
	if err != nil {
		return dto.ExpenseSummaryRes{}, err
	}

	var totalGroupSpent float64
	var userPaid float64
	var userTotalSplit float64
	currency := "VND"

	if len(expenses) > 0 {
		currency = expenses[0].Currency
	}

	for _, exp := range expenses {
		// Chỉ cộng vào tổng chi tiêu nhóm nếu KHÔNG phải khoản chuyển tiền thanh toán nợ
		if exp.SplitType != "SETTLEMENT" {
			totalGroupSpent += exp.Amount
		}
		// Nhưng số tiền user đã chi (từ túi) thì vẫn cộng bình thường
		if exp.PayerID == userID {
			userPaid += exp.Amount
		}
	}

	for _, split := range splits {
		if split.UserID == userID {
			userTotalSplit += split.AmountOwed
		}
	}

	// Nợ = Tổng bị chia - Tổng đã ứng trả
	userDebt := userTotalSplit - userPaid
	if userDebt < 0 {
		userDebt = 0 // Nếu trả nhiều hơn nợ tức là chủ nợ, không bị nợ ai cả
	}

	return dto.ExpenseSummaryRes{
		TotalGroupSpent: totalGroupSpent,
		UserPaid:        userPaid,
		UserDebt:        userDebt,
		Currency:        currency,
	}, nil
}

// GetExpenseList trả về danh sách khoản chi thực (không gồm SETTLEMENT) để hiển thị lịch sử
func (u *expenseUseCase) GetExpenseList(groupID uint) ([]models.Expense, error) {
	return u.expenseRepo.GetListByGroup(groupID)
}
