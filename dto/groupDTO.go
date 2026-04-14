package dto

import "time"

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
