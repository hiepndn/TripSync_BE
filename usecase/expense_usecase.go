package usecase

import (
	"errors"
	"math"
	"sort"
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

// 🌟 THUẬT TOÁN GREEDY: TỐI GIẢN HÓA CÔNG NỢ
func (u *expenseUseCase) CalculateOptimalDebts(groupID uint) ([]dto.DebtSettlement, error) {
	// 1. Lấy toàn bộ Expenses và Splits của nhóm từ DB
	// (Giả định ông sẽ viết hàm GetAllByGroup trong ExpenseRepository)
	expenses, splits, err := u.expenseRepo.GetAllByGroup(groupID)
	if err != nil {
		return nil, err
	}

	// ==========================================
	// BƯỚC 1: TÍNH NET BALANCE CHO TỪNG USER
	// ==========================================
	balances := make(map[uint]float64)

	// Cộng tiền người thanh toán (Họ trả hộ nên số dư tăng)
	for _, exp := range expenses {
		balances[exp.PayerID] += exp.Amount
	}

	// Trừ tiền người nợ (Bị bổ đầu nên số dư giảm)
	for _, split := range splits {
		balances[split.UserID] -= split.AmountOwed
	}

	// ==========================================
	// BƯỚC 2: PHÂN NHÓM CHỦ NỢ & CON NỢ
	// ==========================================
	var creditors []UserBalance // Người cần thu tiền về
	var debtors []UserBalance   // Người phải xì tiền ra

	for userID, amount := range balances {
		// Dùng 0.01 để tránh sai số dấu phẩy động float64 trong Go
		if amount > 0.01 {
			creditors = append(creditors, UserBalance{UserID: userID, Amount: amount})
		} else if amount < -0.01 {
			debtors = append(debtors, UserBalance{UserID: userID, Amount: -amount}) // Đổi số âm thành dương để dễ tính
		}
	}

	// TỐI ƯU HÓA: Sắp xếp giảm dần để "Cục nợ to" đập vào "Cục thu to" trước
	sort.Slice(creditors, func(i, j int) bool { return creditors[i].Amount > creditors[j].Amount })
	sort.Slice(debtors, func(i, j int) bool { return debtors[i].Amount > debtors[j].Amount })

	// ==========================================
	// BƯỚC 3: KHỚP NỢ (GREEDY MATCHING)
	// ==========================================
	var settlements []dto.DebtSettlement
	i, j := 0, 0 // i chạy cho debtors, j chạy cho creditors

	// Lấy đơn vị tiền tệ chung của nhóm (Lấy từ hóa đơn đầu tiên)
	currency := "VND"
	if len(expenses) > 0 {
		currency = expenses[0].Currency
	}

	for i < len(debtors) && j < len(creditors) {
		debt := debtors[i].Amount
		cred := creditors[j].Amount

		// Số tiền giao dịch là con số nhỏ hơn giữa Nợ và Thu
		settleAmount := math.Min(debt, cred)

		// Ghi nhận chỉ thị chuyển tiền vào kết quả
		settlements = append(settlements, dto.DebtSettlement{
			FromUserID: debtors[i].UserID,
			ToUserID:   creditors[j].UserID,
			Amount:     math.Round(settleAmount), // Làm tròn số tiền
			Currency:   currency,
		})

		// Trừ số tiền vừa thanh toán khỏi số dư của cả 2 bên
		debtors[i].Amount -= settleAmount
		creditors[j].Amount -= settleAmount

		// Ai hết nợ/thu (về 0) thì tiến con trỏ nhảy sang người tiếp theo
		if debtors[i].Amount < 0.01 {
			i++
		}
		if creditors[j].Amount < 0.01 {
			j++
		}
	}

	// BƯỚC 4: TRẢ KẾT QUẢ CHO FRONTEND
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
