package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware kiểm tra user có role SUPERADMIN không.
// Phải dùng sau AuthMiddleware.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "SUPERADMIN" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền truy cập trang quản trị"})
			c.Abort()
			return
		}
		c.Next()
	}
}
