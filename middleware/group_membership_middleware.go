package middleware

import (
	"net/http"
	"strconv"
	"tripsync-backend/repository"

	"github.com/gin-gonic/gin"
)

// GroupMembershipMiddleware kiểm tra user có thuộc nhóm (:id) không.
// Phải dùng sau AuthMiddleware (cần user_id đã được set vào context).
func GroupMembershipMiddleware(groupRepo repository.GroupRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupIDStr := c.Param("id")
		groupID, err := strconv.Atoi(groupIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
			c.Abort()
			return
		}

		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa đăng nhập"})
			c.Abort()
			return
		}
		userID := uint(userIDVal.(float64))

		isMember, err := groupRepo.IsUserInGroup(c.Request.Context(), uint(groupID), userID)
		if err != nil || !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền truy cập nhóm này"})
			c.Abort()
			return
		}

		c.Next()
	}
}
