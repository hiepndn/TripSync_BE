package usecase

import (
	"testing"
	"tripsync-backend/models"
)

// ============================================================
// TEST DATA — 6 người, 4 khoản chi chéo nhau
// ============================================================
func makeBenchData() ([]models.Expense, []models.ExpenseSplit) {
	expenses := []models.Expense{
		{PayerID: 1, Amount: 300000, Currency: "VND"},
		{PayerID: 2, Amount: 150000, Currency: "VND"},
		{PayerID: 3, Amount: 450000, Currency: "VND"},
		{PayerID: 4, Amount: 200000, Currency: "VND"},
	}
	// Gán ID thủ công để splits có thể tham chiếu
	expenses[0].ID = 1
	expenses[1].ID = 2
	expenses[2].ID = 3
	expenses[3].ID = 4

	splits := []models.ExpenseSplit{
		// Khoản 1 (300k): chia đều 3 người (1,2,3)
		{ExpenseID: 1, UserID: 1, AmountOwed: 100000},
		{ExpenseID: 1, UserID: 2, AmountOwed: 100000},
		{ExpenseID: 1, UserID: 3, AmountOwed: 100000},
		// Khoản 2 (150k): chia đều 3 người (2,4,5)
		{ExpenseID: 2, UserID: 2, AmountOwed: 50000},
		{ExpenseID: 2, UserID: 4, AmountOwed: 50000},
		{ExpenseID: 2, UserID: 5, AmountOwed: 50000},
		// Khoản 3 (450k): chia đều 6 người
		{ExpenseID: 3, UserID: 1, AmountOwed: 75000},
		{ExpenseID: 3, UserID: 2, AmountOwed: 75000},
		{ExpenseID: 3, UserID: 3, AmountOwed: 75000},
		{ExpenseID: 3, UserID: 4, AmountOwed: 75000},
		{ExpenseID: 3, UserID: 5, AmountOwed: 75000},
		{ExpenseID: 3, UserID: 6, AmountOwed: 75000},
		// Khoản 4 (200k): chia đều 4 người (1,3,5,6)
		{ExpenseID: 4, UserID: 1, AmountOwed: 50000},
		{ExpenseID: 4, UserID: 3, AmountOwed: 50000},
		{ExpenseID: 4, UserID: 5, AmountOwed: 50000},
		{ExpenseID: 4, UserID: 6, AmountOwed: 50000},
	}
	return expenses, splits
}

// ============================================================
// UNIT TEST — Kiểm tra tính đúng đắn (sau khi thanh toán mọi người về 0)
// ============================================================
func TestAllAlgorithmsBalance(t *testing.T) {
	expenses, splits := makeBenchData()

	algos := []DebtAlgorithm{
		&GreedyAlgorithm{},
		&NaiveAlgorithm{},
		&BacktrackingAlgorithm{},
	}

	for _, algo := range algos {
		t.Run(algo.Name(), func(t *testing.T) {
			settlements := algo.Calculate(expenses, splits)

			// Bắt đầu từ net balance gốc
			finalBal := computeBalances(expenses, splits)

			// Áp dụng từng giao dịch:
			// - Người trả (FromUserID) tăng balance (họ đã trả tiền ra)
			// - Người nhận (ToUserID) giảm balance (họ đã nhận tiền về)
			for _, s := range settlements {
				finalBal[s.FromUserID] += s.Amount
				finalBal[s.ToUserID] -= s.Amount
			}

			// Sau khi áp dụng tất cả giao dịch, mọi người phải về 0
			allZero := true
			for userID, bal := range finalBal {
				if bal > 1 || bal < -1 { // Sai số 1 VND do làm tròn
					t.Errorf("[%s] user %d còn dư %.2f sau khi thanh toán", algo.Name(), userID, bal)
					allZero = false
				}
			}
			if allZero {
				t.Logf("[%s] ✅ Tất cả về 0 — %d giao dịch", algo.Name(), len(settlements))
			}
		})
	}
}

// ============================================================
// BENCHMARK — So sánh tốc độ giữa 3 thuật toán
// ============================================================
func BenchmarkGreedy(b *testing.B) {
	expenses, splits := makeBenchData()
	algo := &GreedyAlgorithm{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		algo.Calculate(expenses, splits)
	}
}

func BenchmarkNaive(b *testing.B) {
	expenses, splits := makeBenchData()
	algo := &NaiveAlgorithm{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		algo.Calculate(expenses, splits)
	}
}

func BenchmarkBacktracking(b *testing.B) {
	expenses, splits := makeBenchData()
	algo := &BacktrackingAlgorithm{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		algo.Calculate(expenses, splits)
	}
}
