package repository

import (
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type ExpenseRepository interface {
	// Tạo 1 khoản chi tiêu mới kèm theo các bản ghi chia tiền (dùng Transaction)
	CreateExpense(expense *models.Expense, splits []models.ExpenseSplit) error

	// Lấy toàn bộ chi tiêu và chi tiết chia tiền của 1 nhóm (Phục vụ thuật toán chia tiền)
	GetAllByGroup(groupID uint) ([]models.Expense, []models.ExpenseSplit, error)

	// Lấy danh sách khoản chi (bỏ SETTLEMENT) để hiển thị lịch sử cho FE
	GetListByGroup(groupID uint) ([]models.Expense, error)
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
func (r *expenseRepository) CreateExpense(expense *models.Expense, splits []models.ExpenseSplit) error {
	// Bắt đầu 1 Transaction để đảm bảo tính toàn vẹn (ACID)
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Lưu hóa đơn tổng vào bảng expenses
		if err := tx.Create(expense).Error; err != nil {
			return err // Lỗi -> Tự động Rollback
		}

		// 2. Gán ID của hóa đơn vừa tạo cho từng bản ghi split
		for i := range splits {
			splits[i].ExpenseID = expense.ID
		}

		// 3. Lưu toàn bộ mảng splits vào bảng expense_splits (Batch Insert)
		if err := tx.Create(&splits).Error; err != nil {
			return err // Lỗi -> Tự động Rollback cả hóa đơn ở bước 1
		}

		return nil // Thành công -> Tự động Commit
	})
}

// ==========================================
// 2. LẤY TOÀN BỘ DATA CỦA NHÓM CHO THUẬT TOÁN
// ==========================================
func (r *expenseRepository) GetAllByGroup(groupID uint) ([]models.Expense, []models.ExpenseSplit, error) {
	var expenses []models.Expense
	var splits []models.ExpenseSplit

	// 1. Lấy tất cả hóa đơn thuộc về Group này
	if err := r.db.Where("group_id = ?", groupID).Find(&expenses).Error; err != nil {
		return nil, nil, err
	}

	// Nếu nhóm chưa có khoản chi nào thì return rỗng luôn cho nhanh
	if len(expenses) == 0 {
		return expenses, splits, nil
	}

	// 2. Gom tất cả Expense ID lại thành 1 mảng
	var expenseIDs []uint
	for _, exp := range expenses {
		expenseIDs = append(expenseIDs, exp.ID)
	}

	// 3. Lấy tất cả các bản ghi chia tiền thuộc về những hóa đơn trên (Dùng IN query)
	if err := r.db.Where("expense_id IN ?", expenseIDs).Find(&splits).Error; err != nil {
		return nil, nil, err
	}

	return expenses, splits, nil
}

// ==========================================
// 3. LẤY LỊCH SỬ CHI TIÊU (bỏ SETTLEMENT)
// ==========================================
func (r *expenseRepository) GetListByGroup(groupID uint) ([]models.Expense, error) {
	var expenses []models.Expense
	err := r.db.
		Where("group_id = ? AND split_type != ?", groupID, models.SplitSettlement).
		Order("created_at DESC").
		Find(&expenses).Error
	return expenses, err
}
