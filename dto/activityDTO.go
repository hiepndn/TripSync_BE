package dto

import (
	"time"
	"tripsync-backend/models"
)

type ActivityResponse struct {
	models.Activity
	VoteCount int  `json:"vote_count"`
	HasVoted  bool `json:"has_voted"`
}
type CreateActivityReq struct {
	Name        string    `json:"name" binding:"required"`
	Type        string    `json:"type" binding:"required"`
	Location    string    `json:"location" binding:"required"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time" binding:"required"` // Chứa cả ngày và giờ
	EndTime     time.Time `json:"end_time" binding:"required"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	PlaceID     string    `json:"place_id"`
}
type UpdateActivityReq struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}
