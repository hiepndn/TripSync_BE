package models

import "time"

// GroupFavorite lưu trữ nhóm yêu thích của user
type GroupFavorite struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	GroupID   uint      `gorm:"primaryKey" json:"group_id"`
	CreatedAt time.Time `json:"created_at"`

	// Preload relations
	Group Group `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}
