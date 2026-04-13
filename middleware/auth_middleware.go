package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn cần đăng nhập để thực hiện thao tác này"})
			c.Abort()
			return
		}

		secretKey := os.Getenv("JWT_SECRET")
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Lưu user_id vào context để các hàm sau sử dụng
			c.Set("user_id", claims["user_id"])
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc đã hết hạn"})
			c.Abort()
		}
	}
}
