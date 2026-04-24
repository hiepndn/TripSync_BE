package usecase

import (
	"fmt"
	"math"
	"sort"
	"tripsync-backend/dto"
	"tripsync-backend/models"
)

// ============================================================
// STRATEGY PATTERN — ĐỔI THUẬT TOÁN CHỈ CẦN ĐỔI 1 DÒNG
// ============================================================

// DebtAlgorithm là interface chung cho mọi thuật toán tính công nợ.
// Để thêm thuật toán mới: implement interface này, đăng ký vào AlgorithmRegistry.
type DebtAlgorithm interface {
	// Name trả về tên định danh của thuật toán (dùng cho log/benchmark)
	Name() string
	// Calculate nhận vào expenses + splits, trả về danh sách giao dịch cần thực hiện
	Calculate(expenses []models.Expense, splits []models.ExpenseSplit) []dto.DebtSettlement
}

// ============================================================
// REGISTRY — ĐĂNG KÝ & LẤY THUẬT TOÁN THEO TÊN
// ============================================================

var algorithmRegistry = map[string]DebtAlgorithm{
	"greedy":       &GreedyAlgorithm{},
	"naive":        &NaiveAlgorithm{},
	"backtracking": &BacktrackingAlgorithm{},
}

// GetAlgorithm trả về thuật toán theo tên. Fallback về Greedy nếu không tìm thấy.
func GetAlgorithm(name string) DebtAlgorithm {
	if algo, ok := algorithmRegistry[name]; ok {
		return algo
	}
	return algorithmRegistry["greedy"]
}

// ActiveAlgorithm là thuật toán đang được dùng trong production.
// ĐỂ ĐỔI THUẬT TOÁN: chỉ cần thay "greedy" thành "naive" hoặc "backtracking"
const ActiveAlgorithm = "greedy"

// ============================================================
// HELPER DÙNG CHUNG: TÍNH NET BALANCE
// ============================================================

// computeBalances tính số dư thuần của từng user từ expenses + splits.
// Balance > 0: chủ nợ (cần thu tiền)
// Balance < 0: con nợ (phải trả tiền)
func computeBalances(expenses []models.Expense, splits []models.ExpenseSplit) map[uint]float64 {
	balances := make(map[uint]float64)
	for _, exp := range expenses {
		balances[exp.PayerID] += exp.Amount
	}
	for _, split := range splits {
		balances[split.UserID] -= split.AmountOwed
	}
	return balances
}

// getCurrency lấy đơn vị tiền tệ từ danh sách expenses, fallback về VND.
func getCurrency(expenses []models.Expense) string {
	if len(expenses) > 0 {
		return expenses[0].Currency
	}
	return "VND"
}

// ============================================================
// THUẬT TOÁN 1: GREEDY (THAM LAM) — ĐÃ CÓ, REFACTOR VÀO ĐÂY
// ============================================================

// GreedyAlgorithm tối giản số lượng giao dịch bằng cách khớp
// con nợ lớn nhất với chủ nợ lớn nhất. Độ phức tạp: O(n log n).
type GreedyAlgorithm struct{}

func (g *GreedyAlgorithm) Name() string { return "greedy" }

func (g *GreedyAlgorithm) Calculate(expenses []models.Expense, splits []models.ExpenseSplit) []dto.DebtSettlement {
	balances := computeBalances(expenses, splits)
	currency := getCurrency(expenses)

	var creditors, debtors []UserBalance
	for userID, amount := range balances {
		if amount > 0.01 {
			creditors = append(creditors, UserBalance{UserID: userID, Amount: amount})
		} else if amount < -0.01 {
			debtors = append(debtors, UserBalance{UserID: userID, Amount: -amount})
		}
	}

	// Sắp xếp giảm dần để "cục nợ to" đập vào "cục thu to" trước
	sort.Slice(creditors, func(i, j int) bool { return creditors[i].Amount > creditors[j].Amount })
	sort.Slice(debtors, func(i, j int) bool { return debtors[i].Amount > debtors[j].Amount })

	var settlements []dto.DebtSettlement
	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		settleAmount := math.Min(debtors[i].Amount, creditors[j].Amount)
		settlements = append(settlements, dto.DebtSettlement{
			FromUserID: debtors[i].UserID,
			ToUserID:   creditors[j].UserID,
			Amount:     math.Round(settleAmount),
			Currency:   currency,
		})
		debtors[i].Amount -= settleAmount
		creditors[j].Amount -= settleAmount
		if debtors[i].Amount < 0.01 {
			i++
		}
		if creditors[j].Amount < 0.01 {
			j++
		}
	}
	return settlements
}

// ============================================================
// THUẬT TOÁN 2: NAIVE (CƠ BẢN) — NỢ AI TRẢ NẤY
// ============================================================

// NaiveAlgorithm không gộp hay bù trừ chéo. Mỗi khoản chi sinh ra
// giao dịch riêng từ từng người nợ → người trả. Số giao dịch = số splits.
// Dùng để benchmark: đây là baseline tệ nhất về số lượng giao dịch.
type NaiveAlgorithm struct{}

func (n *NaiveAlgorithm) Name() string { return "naive" }

func (n *NaiveAlgorithm) Calculate(expenses []models.Expense, splits []models.ExpenseSplit) []dto.DebtSettlement {
	currency := getCurrency(expenses)

	// Build map expense_id → payer_id để tra nhanh
	payerMap := make(map[uint]uint, len(expenses))
	for _, exp := range expenses {
		payerMap[exp.ID] = exp.PayerID
	}

	var settlements []dto.DebtSettlement
	for _, split := range splits {
		payerID, ok := payerMap[split.ExpenseID]
		if !ok {
			continue
		}
		// Bỏ qua nếu người nợ chính là người trả (tự nợ chính mình)
		if split.UserID == payerID {
			continue
		}
		if split.AmountOwed < 0.01 {
			continue
		}
		settlements = append(settlements, dto.DebtSettlement{
			FromUserID: split.UserID,
			ToUserID:   payerID,
			Amount:     math.Round(split.AmountOwed),
			Currency:   currency,
		})
	}
	return settlements
}

// ============================================================
// THUẬT TOÁN 3: BACKTRACKING (QUAY LUI) — TỐI ƯU TUYỆT ĐỐI
// ============================================================

// BacktrackingAlgorithm vét cạn mọi hoán vị sau khi tính Net Balance
// để tìm ra danh sách giao dịch có số lượng ÍT NHẤT tuyệt đối.
// Độ phức tạp: O(n!) — chỉ dùng cho nhóm nhỏ (≤ 10 người).
// Với nhóm lớn hơn, tự động fallback về Greedy để tránh timeout.
type BacktrackingAlgorithm struct{}

func (b *BacktrackingAlgorithm) Name() string { return "backtracking" }

const backtrackingMaxParticipants = 10 // Ngưỡng an toàn

func (b *BacktrackingAlgorithm) Calculate(expenses []models.Expense, splits []models.ExpenseSplit) []dto.DebtSettlement {
	balances := computeBalances(expenses, splits)
	currency := getCurrency(expenses)

	// Lọc ra những người có số dư khác 0 (bỏ qua người đã hòa)
	var nonZero []float64
	var userIDs []uint
	for userID, bal := range balances {
		if math.Abs(bal) > 0.01 {
			nonZero = append(nonZero, bal)
			userIDs = append(userIDs, userID)
		}
	}

	// Fallback về Greedy nếu quá nhiều người để tránh timeout
	if len(nonZero) > backtrackingMaxParticipants {
		fmt.Printf("⚠️ [BacktrackingAlgorithm] Nhóm có %d người vượt ngưỡng %d — fallback về Greedy để tránh timeout\n",
			len(nonZero), backtrackingMaxParticipants)
		greedy := &GreedyAlgorithm{}
		return greedy.Calculate(expenses, splits)
	}

	// Chạy backtracking để tìm số giao dịch tối thiểu
	bestTxns := make([]transaction, 0)
	currentTxns := make([]transaction, 0)
	balancesCopy := make([]float64, len(nonZero))
	copy(balancesCopy, nonZero)

	backtrack(balancesCopy, currentTxns, &bestTxns)

	// Map index → userID để build kết quả
	settlements := make([]dto.DebtSettlement, 0, len(bestTxns))
	for _, txn := range bestTxns {
		settlements = append(settlements, dto.DebtSettlement{
			FromUserID: userIDs[txn.fromIdx],
			ToUserID:   userIDs[txn.toIdx],
			Amount:     math.Round(txn.amount),
			Currency:   currency,
		})
	}
	return settlements
}

// transaction lưu 1 giao dịch trong quá trình backtracking (dùng index thay vì userID)
type transaction struct {
	fromIdx int
	toIdx   int
	amount  float64
}

// backtrack đệ quy vét cạn, cập nhật bestTxns khi tìm được lời giải tốt hơn.
func backtrack(balances []float64, current []transaction, best *[]transaction) {
	// Tìm con nợ đầu tiên (balance < 0)
	debtorIdx := -1
	for i, b := range balances {
		if b < -0.01 {
			debtorIdx = i
			break
		}
	}

	// Base case: không còn con nợ → đây là 1 lời giải hoàn chỉnh
	if debtorIdx == -1 {
		if len(*best) == 0 || len(current) < len(*best) {
			// Tìm được lời giải tốt hơn → cập nhật best
			newBest := make([]transaction, len(current))
			copy(newBest, current)
			*best = newBest
		}
		return
	}

	// Pruning: nếu số giao dịch hiện tại đã >= best thì cắt nhánh
	if len(*best) > 0 && len(current) >= len(*best) {
		return
	}

	// Thử trả nợ cho từng chủ nợ (balance > 0)
	for i, b := range balances {
		if b < 0.01 {
			continue
		}

		// Số tiền giao dịch = min(|nợ|, thu)
		debtAmt := -balances[debtorIdx]
		settleAmt := math.Min(debtAmt, b)

		// Thực hiện giao dịch
		balances[debtorIdx] += settleAmt
		balances[i] -= settleAmt
		current = append(current, transaction{fromIdx: debtorIdx, toIdx: i, amount: settleAmt})

		// Đệ quy tiếp
		backtrack(balances, current, best)

		// Hoàn tác (backtrack)
		current = current[:len(current)-1]
		balances[debtorIdx] -= settleAmt
		balances[i] += settleAmt
	}
}
