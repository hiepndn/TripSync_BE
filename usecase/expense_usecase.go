package usecase

import (
	"context"
	"errors"
	"math"
	"tripsync-backend/dto"
	"tripsync-backend/models"
	"tripsync-backend/repository"
)

type ExpenseUseCase interface {
	CalculateOptimalDebts(ctx context.Context, groupID uint) ([]dto.DebtSettlement, error)
	CreateExpense(ctx context.Context, groupID uint, payerID uint, req dto.CreateExpenseReq) error
	GetExpenseSummary(ctx context.Context, groupID uint, userID uint) (dto.ExpenseSummaryRes, error)
	GetExpenseList(ctx context.Context, groupID uint) ([]models.Expense, error)
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

// CalculateOptimalDebts tính công nợ dùng thuật toán được cấu hình bởi ActiveAlgorithm
func (u *expenseUseCase) CalculateOptimalDebts(ctx context.Context, groupID uint) ([]dto.DebtSettlement, error) {
	expenses, splits, err := u.expenseRepo.GetAllByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	algo := GetAlgorithm(ActiveAlgorithm)
	settlements := algo.Calculate(expenses, splits)
	return settlements, nil
}

func (u *expenseUseCase) CreateExpense(ctx context.Context, groupID uint, payerID uint, req dto.CreateExpenseReq) error {
	var totalSplit float64
	for _, s := range req.Splits {
		totalSplit += s.AmountOwed
	}

	if math.Abs(totalSplit-req.Amount) > 0.01 {
		return errors.New("tổng tiền chia cho các thành viên không khớp với tổng hóa đơn")
	}

	expense := &models.Expense{
		GroupID:     groupID,
		PayerID:     payerID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		SplitType:   req.SplitType,
	}

	var splits []models.ExpenseSplit
	for _, s := range req.Splits {
		splits = append(splits, models.ExpenseSplit{
			UserID:     s.UserID,
			AmountOwed: s.AmountOwed,
		})
	}

	return u.expenseRepo.CreateExpense(ctx, expense, splits)
}

// GetExpenseSummary lấy thống kê chi tiêu của 1 thành viên (tổng chi, đã chi, còn nợ)
func (u *expenseUseCase) GetExpenseSummary(ctx context.Context, groupID uint, userID uint) (dto.ExpenseSummaryRes, error) {
	expenses, splits, err := u.expenseRepo.GetAllByGroup(ctx, groupID)
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
		if exp.SplitType != "SETTLEMENT" {
			totalGroupSpent += exp.Amount
		}
		if exp.PayerID == userID {
			userPaid += exp.Amount
		}
	}

	for _, split := range splits {
		if split.UserID == userID {
			userTotalSplit += split.AmountOwed
		}
	}

	userDebt := userTotalSplit - userPaid
	if userDebt < 0 {
		userDebt = 0
	}

	return dto.ExpenseSummaryRes{
		TotalGroupSpent: totalGroupSpent,
		UserPaid:        userPaid,
		UserDebt:        userDebt,
		Currency:        currency,
	}, nil
}

// GetExpenseList trả về danh sách khoản chi thực (không gồm SETTLEMENT) để hiển thị lịch sử
func (u *expenseUseCase) GetExpenseList(ctx context.Context, groupID uint) ([]models.Expense, error) {
	return u.expenseRepo.GetListByGroup(ctx, groupID)
}
