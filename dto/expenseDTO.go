package dto

import "tripsync-backend/models"

type ExpenseSplitReq struct {
	UserID     uint    `json:"user_id" binding:"required"`
	AmountOwed float64 `json:"amount_owed" binding:"required,min=0"`
}

type CreateExpenseReq struct {
	Amount      float64           `json:"amount" binding:"required,gt=0"`
	Currency    string            `json:"currency" binding:"required"`
	Description string            `json:"description" binding:"required"`
	SplitType   models.SplitType  `json:"split_type" binding:"required"` // EQUAL, EXACT, PERCENTAGES...
	Splits      []ExpenseSplitReq `json:"splits" binding:"required,min=1"`
}

type DebtSettlement struct {
	FromUserID uint    `json:"from_user_id"`
	ToUserID   uint    `json:"to_user_id"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
}

type ExpenseSummaryRes struct {
	TotalGroupSpent float64 `json:"total_group_spent"`
	UserPaid        float64 `json:"user_paid"`
	UserDebt        float64 `json:"user_debt"`
	Currency        string  `json:"currency"`
}

type SettleDebtReq struct {
	FromUserID uint    `json:"from_user_id" binding:"required"`
	ToUserID   uint    `json:"to_user_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"required,gt=0"`
}
