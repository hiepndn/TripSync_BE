package models

type ChecklistItem struct {
	BaseModel
	GroupID  uint   `gorm:"not null" json:"group_id"`
	Title    string `gorm:"not null" json:"title"` // Tên công việc (VD: Chuẩn bị lều trại)
	Category string `json:"category"`              // Danh mục (VD: Đồ dùng chung, Ăn uống)

	// Người được giao việc (Có thể NULL nếu chưa ai nhận)
	AssigneeID *uint `json:"assignee_id"`

	IsCompleted bool `gorm:"default:false" json:"is_completed"` // Đã xong chưa?

	// Người đánh dấu hoàn thành (Có thể khác người được giao)
	CompletedByID *uint `json:"completed_by_id"`

	Assignee    *User `gorm:"foreignKey:AssigneeID" json:"assignee"`
	CompletedBy *User `gorm:"foreignKey:CompletedByID" json:"completed_by"`
}
