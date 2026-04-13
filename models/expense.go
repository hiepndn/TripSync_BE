package models

type SplitType string

const (
	SplitEqual      SplitType = "EQUAL"
	SplitPercentage SplitType = "PERCENTAGE"
	SplitExact      SplitType = "EXACT"
	SplitSettlement SplitType = "SETTLEMENT"
)

type Expense struct {
	BaseModel
	Description string  `gorm:"not null" json:"description"`
	Amount      float64 `gorm:"not null" json:"amount"`
	Currency    string  `gorm:"default:'VND'" json:"currency"`

	GroupID uint `gorm:"not null" json:"group_id"`
	// PayerID: ID người móc túi trả tiền thật sự cho cả nhóm
	// Không dùng association "Payer User" để tránh GORM tự thêm bảng — JOIN thủ công khi cần full_name
	PayerID uint `gorm:"not null" json:"payer_id"`

	SplitType SplitType `gorm:"default:'EQUAL'" json:"split_type"`

	// Splits: phần tiêu thụ của TỪNG NGƯỜI (kể cả người trả)
	// VD: A trả 300k cho 3 người → Splits = [A:100k, B:100k, C:100k]
	// Greedy Algorithm dùng: Net(A) = +300k (đã trả) - 100k (phần của A) = +200k chủ nợ
	Splits []ExpenseSplit `gorm:"foreignKey:ExpenseID;constraint:OnDelete:CASCADE;" json:"splits"`
}

type ExpenseSplit struct {
	ExpenseID  uint    `gorm:"primaryKey" json:"expense_id"`
	UserID     uint    `gorm:"primaryKey" json:"user_id"`
	// AmountOwed: phần tiêu thụ của user này trong khoản chi (KHÔNG phải số tiền phải chuyển khoản)
	// Greedy Algorithm tính số phải chuyển sau khi đã net balance với tổng đã trả
	AmountOwed float64 `gorm:"not null" json:"amount_owed"`
}
