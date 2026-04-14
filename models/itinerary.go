package models

import (
	"time"
)

type ActivityStatus string

const (
	StatusPending ActivityStatus = "PENDING"
	StatusApprove ActivityStatus = "APPROVE" // Đã chốt [cite: 30, 34]
)

type Activity struct {
	BaseModel
	GroupID     uint           `gorm:"not null" json:"group_id"`
	Name        string         `gorm:"not null" json:"name"`
	Type        string         `gorm:"not null" json:"type"` // THÊM CỘT NÀY: HOTEL / ATTRACTION / RESTAURANT / CAMPING
	Location    string         `json:"location"`
	Description string         `json:"description"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
	Status      ActivityStatus `gorm:"default:'PENDING'" json:"status"` // Sửa default thành PENDING
	Lat         float64        `json:"lat"`
	Lng         float64        `json:"lng"`
	PlaceID     string         `json:"place_id"` // Dùng string cho Google Place ID là rất chuẩn

	CreatedBy *uint `json:"created_by"` // SỬA THÀNH CON TRỎ ĐỂ CHO PHÉP NULL

	IsAIGenerated bool    `json:"is_ai_generated" gorm:"column:is_ai_generated;default:false"`
	EstimatedCost float64 `json:"estimatedCost" gorm:"column:estimated_cost;type:decimal(10,2)"`
	Currency      string  `json:"currency" gorm:"column:currency"`
	ImageURL      string  `json:"imageURL" gorm:"column:image_url"`
	Rating        float64 `json:"rating" gorm:"column:rating"`
	ExternalLink  string  `json:"externalLink" gorm:"column:external_link"`

	Votes []ActivityVote `gorm:"foreignKey:ActivityID;constraint:OnDelete:CASCADE;" json:"votes"`
}

type ActivityVote struct {
	ActivityID uint   `gorm:"primaryKey" json:"activity_id"`
	UserID     uint   `gorm:"primaryKey" json:"user_id"`
	VoteType   string `gorm:"default:'UP'" json:"vote_type"` // UP hoặc DOWN
}

type ActivityRating struct {
	ActivityID uint      `gorm:"primaryKey" json:"activity_id"`
	UserID     uint      `gorm:"primaryKey" json:"user_id"`
	Rating     int       `gorm:"not null" json:"rating"` // 1–5
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
