// cmd/migrate/main.go
//
// Script chạy AutoMigrate thủ công — TÁCH BIỆT hoàn toàn với server.
//
// Cách dùng:
//
//	# Dev
//	go run cmd/migrate/main.go
//
//	# Production (build trước rồi chạy)
//	go build -o migrate cmd/migrate/main.go
//	./migrate
//
// Lưu ý: Cần có file .env hoặc set biến môi trường trước khi chạy.
package main

import (
	"fmt"
	"log"
	"tripsync-backend/config"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env (bỏ qua lỗi nếu chạy trên server dùng env thật)
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Không tìm thấy .env, dùng biến môi trường hệ thống")
	}

	fmt.Println("🔌 Kết nối database...")
	config.ConnectDB()

	fmt.Println("🚀 Bắt đầu migrate...")
	config.RunMigrations()

	fmt.Println("✅ Migrate hoàn tất. Server có thể khởi động an toàn.")
}
