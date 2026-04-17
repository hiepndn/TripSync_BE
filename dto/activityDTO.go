package dto

import (
	"time"
	"tripsync-backend/models"
)

type ActivityResponse struct {
	models.Activity
	VoteCount         int     `json:"vote_count"`
	HasVoted          bool    `json:"has_voted"`
	AverageUserRating float64 `json:"average_user_rating"`
	MyRating          int     `json:"my_rating"`
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

type RateActivityReq struct {
	Rating int `json:"rating" binding:"required,min=1,max=5"`
}

type SuggestionResponse struct {
	models.Activity
	AverageUserRating float64 `json:"average_user_rating"`
}

type RatingContextItem struct {
	Name              string
	Type              string
	Location          string
	AverageUserRating float64
}

type RatingContext struct {
	HighlyRated []RatingContextItem
	PoorlyRated []RatingContextItem
}

// ExportActivityItem is the public export payload for a single activity.
// Internal fields (id, group_id, created_by, is_ai_generated, votes, ratings) are intentionally excluded.
type ExportActivityItem struct {
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Location      string    `json:"location"`
	Description   string    `json:"description"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	EstimatedCost float64   `json:"estimated_cost"`
	Currency      string    `json:"currency"`
	Lat           float64   `json:"lat"`
	Lng           float64   `json:"lng"`
}

// ImportActivitiesReq is the request body for the import endpoint.
type ImportActivitiesReq struct {
	SourceGroupID int `json:"source_group_id" binding:"required"`
}

// ImportFromJSONReq is the request body for the import-from-json endpoint.
type ImportFromJSONReq struct {
	Activities []ExportActivityItem `json:"activities" binding:"required"`
}
