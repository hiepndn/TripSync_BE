package repository

import (
	"context"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type ExpenseRepository interface {
	// Tạo 1 khoản chi tiêu mới kèm theo các bản ghi chia tiền (dùng Transaction)
	CreateExpense(ctx context.Context, expense *models.Expense, splits []models.ExpenseSplit) error

	// Lấy toàn bộ chi tiêu và chi tiết chia tiền của 1 nhóm (Phục vụ thuật toán chia tiền)
	GetAllByGroup(ctx context.Context, groupID uint) ([]models.Expense, []models.ExpenseSplit, error)

	// Lấy danh sách khoản chi (bỏ SETTLEMENT) để hiển thị lịch sử cho FE
	GetListByGroup(ctx context.Context, groupID uint) ([]models.Expense, error)
}

type expenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) ExpenseRepository {
	return &expenseRepository{db: db}
}

// ==========================================
// 1. TẠO KHOẢN CHI (CÓ DÙNG TRANSACTION)
// ==========================================
func (r *expenseRepository) CreateExpense(ctx context.Context, expense *models.Expense, splits []models.ExpenseSplit) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(expense).Error; err != nil {
			return err
		}

		for i := range splits {
			splits[i].ExpenseID = expense.ID
		}

		if err := tx.Create(&splits).Error; err != nil {
			return err
		}

		return nil
	})
}

// ==========================================
// 2. LẤY TOÀN BỘ DATA CỦA NHÓM CHO THUẬT TOÁN
// ==========================================
func (r *expenseRepository) GetAllByGroup(ctx context.Context, groupID uint) ([]models.Expense, []models.ExpenseSplit, error) {
	var expenses []models.Expense
	var splits []models.ExpenseSplit

	if err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&expenses).Error; err != nil {
		return nil, nil, err
	}

	if len(expenses) == 0 {
		return expenses, splits, nil
	}

	var expenseIDs []uint
	for _, exp := range expenses {
		expenseIDs = append(expenseIDs, exp.ID)
	}

	if err := r.db.WithContext(ctx).Where("expense_id IN ?", expenseIDs).Find(&splits).Error; err != nil {
		return nil, nil, err
	}

	return expenses, splits, nil
}

// ==========================================
// 3. LẤY LỊCH SỬ CHI TIÊU (bỏ SETTLEMENT)
// ==========================================
func (r *expenseRepository) GetListByGroup(ctx context.Context, groupID uint) ([]models.Expense, error) {
	var expenses []models.Expense
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND split_type != ?", groupID, models.SplitSettlement).
		Order("created_at DESC").
		Find(&expenses).Error
	return expenses, err
}
