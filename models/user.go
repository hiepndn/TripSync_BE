package models

type User struct {
	BaseModel
	Email    string `gorm:"uniqueIndex;not null" json:"email"`
	Password string `gorm:"not null" json:"-"`
	FullName string `json:"full_name"`
	Avatar   string `json:"avatar"`
	Role     string `gorm:"default:'USER'" json:"role"`

	// Quan hệ ngược (để truy vấn User đang ở nhóm nào)
	Groups []Group `gorm:"many2many:group_members;" json:"groups,omitempty"`
}
