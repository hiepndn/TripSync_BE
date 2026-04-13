package main

import (
	"log"
	"tripsync-backend/config" // Import folder config
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
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	routes.SetupRoutes(r)

	r.Run(":8080")
}
