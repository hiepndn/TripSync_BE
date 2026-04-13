package models

import (
	"time"
)

type GroupRole string

const (
	RoleAdmin  GroupRole = "ADMIN"
	RoleMember GroupRole = "MEMBER"
)

type Group struct {
	BaseModel
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	InviteCode  string    `gorm:"uniqueIndex;size:10" json:"invite_code"`
	IsPublic    bool      `gorm:"default:false" json:"is_public"`
	ShareToken  string    `gorm:"uniqueIndex" json:"share_token"`

	DepartureLocation string  `gorm:"column:departure_location" json:"departure_location"`
	RouteDestinations string  `gorm:"column:route_destinations" json:"route_destinations"`
	AccommodationPref string  `gorm:"column:accommodation_pref" json:"accommodation_pref"`
	ExpectedMembers   int     `gorm:"column:expected_members" json:"expected_members"`
	BudgetPerPerson   float64 `gorm:"column:budget_per_person;type:decimal(10,2)" json:"budget_per_person"`
	Currency          string  `gorm:"column:currency" json:"currency"`
	IsAIGenerating    bool    `gorm:"column:is_ai_generating;default:false" json:"is_ai_generating"`

	// Quan hệ
	Members    []User          `gorm:"many2many:group_members;" json:"members,omitempty"`
	Expenses   []Expense       `gorm:"foreignKey:GroupID" json:"expenses,omitempty"`
	Activities []Activity      `gorm:"foreignKey:GroupID" json:"activities,omitempty"`
	Checklists []ChecklistItem `gorm:"foreignKey:GroupID" json:"checklists,omitempty"`
	Documents  []Document      `gorm:"foreignKey:GroupID" json:"documents,omitempty"`
}

// Bảng trung gian tùy chỉnh để lưu quyền Admin/Member
type GroupMember struct {
	GroupID  uint      `gorm:"primaryKey" json:"group_id"`
	UserID   uint      `gorm:"primaryKey" json:"user_id"`
	Role     GroupRole `gorm:"default:'MEMBER'" json:"role"` // [cite: 67, 68]
	JoinedAt time.Time `json:"joined_at"`
}
