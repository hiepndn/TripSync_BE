package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel thay thế cho gorm.Model, bổ sung chuẩn JSON snake_case
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
