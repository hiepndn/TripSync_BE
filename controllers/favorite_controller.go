package controllers

import (
	"net/http"
	"strconv"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

type FavoriteController struct {
	uc usecase.FavoriteUseCase
}

func NewFavoriteController(uc usecase.FavoriteUseCase) *FavoriteController {
	return &FavoriteController{uc: uc}
}

// POST /api/groups/:id/favorite — toggle yêu thích
func (fc *FavoriteController) ToggleFavorite(c *gin.Context) {
	groupIDStr := c.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group_id không hợp lệ"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa đăng nhập"})
		return
	}
	userID := uint(userIDVal.(float64))

	isFavorited, err := fc.uc.ToggleFavorite(userID, uint(groupID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	msg := "Đã thêm vào yêu thích"
	if !isFavorited {
		msg = "Đã xóa khỏi yêu thích"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      msg,
		"is_favorited": isFavorited,
	})
}

// GET /api/favorites — lấy danh sách nhóm yêu thích
func (fc *FavoriteController) GetFavorites(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa đăng nhập"})
		return
	}
	userID := uint(userIDVal.(float64))

	groups, err := fc.uc.GetFavorites(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách yêu thích"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    groups,
	})
}
