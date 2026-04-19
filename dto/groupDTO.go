package dto

import (
	"time"
	"tripsync-backend/models"
)

type UpdateGroupRequest struct {
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	StartDate         time.Time `json:"start_date"`
	EndDate           time.Time `json:"end_date"`
	DepartureLocation string    `json:"departure_location"`
	RouteDestinations string    `json:"route_destinations"`
}

type CreateGroupRequest struct {
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`

	// --- 5 TRƯỜNG MỚI CHO AI PLANNING ---
	DepartureLocation string  `json:"departure_location" binding:"required"` // VD: "Hà Nội"
	RouteDestinations string  `json:"route_destinations" binding:"required"` // VD: "Hà Nội, Huế, Đà Nẵng"
	AccommodationPref string  `json:"accommodation_pref" binding:"required"` // HOTEL / CAMPING / MIXED
	ExpectedMembers   int     `json:"expected_members" binding:"required,min=1"`
	BudgetPerPerson   float64 `json:"budget_per_person" binding:"required,min=0"`
	Currency          string  `json:"currency" binding:"required"` // VD: "VND"
}

// UpdateVisibilityRequest là body cho PUT /groups/:id/visibility
type UpdateVisibilityRequest struct {
	IsPublic bool `json:"is_public"`
}

// MemberPreview là thông tin tóm tắt của 1 thành viên để hiển thị avatar
type MemberPreview struct {
	ID       uint   `json:"id"`
	FullName string `json:"full_name"`
	Avatar   string `json:"avatar"`
}

type GroupWithRole struct {
	models.Group
	Role           string          `json:"role"` // ADMIN or MEMBER
	MemberCount    int             `json:"member_count"`
	MemberPreviews []MemberPreview `json:"member_previews"` // tối đa 3 người đầu
}

// PublicExpenseSummary tổng hợp chi tiêu cho trang public
type PublicExpenseSummary struct {
	TotalAmount  float64 `json:"total_amount"`
	ExpenseCount int     `json:"expense_count"`
}

// PublicGroupDetailResponse là response cho GET /groups/public/:id
type PublicGroupDetailResponse struct {
	GroupInfo      *models.Group        `json:"group_info"`
	Activities     []models.Activity    `json:"activities"`
	ExpenseSummary PublicExpenseSummary `json:"expense_summary"`
}
