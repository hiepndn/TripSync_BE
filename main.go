package main

import (
	"log"
	"os"
	"strings"
	"tripsync-backend/config"
	"tripsync-backend/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load file .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Không tìm thấy file .env, check lại nhé")
	}

	// 2. Kết nối Database (Gọi hàm từ folder config)
	config.ConnectDB()

	// 3. Chạy Server
	r := gin.Default()

	// CORS: lấy danh sách origins từ env, hỗ trợ nhiều domain cách nhau bằng dấu phẩy
	allowedOrigins := os.Getenv("FRONTEND_URL")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000" // Fallback dev
	}
	origins := strings.Split(allowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	routes.SetupRoutes(r)

	port := os.Getenv("PORT")

	// Nếu không tìm thấy biến PORT (nghĩa là đang chạy ở máy tính Local)
	if port == "" {
		port = "8080" // Lấy tạm cổng 8080 để test
	}

	// Chạy server với cái cổng vừa lấy được
	log.Printf("🚀 Server đang chạy ở cổng %s", port)
	err := r.Run(":" + port)

	if err != nil {
		log.Fatal("❌ Lỗi sập server: ", err)
	}
}
