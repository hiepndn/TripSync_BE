package config

import (
	"fmt"
	"log"
	"os"
	"tripsync-backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ConnectDB chỉ kết nối DB — KHÔNG chạy AutoMigrate.
// AutoMigrate được tách ra hàm RunMigrations() để chạy thủ công khi cần.
func ConnectDB() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=require TimeZone=Asia/Ho_Chi_Minh",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Lỗi kết nối DB: ", err)
	}

	fmt.Println("✅ Đã kết nối Database thành công!")
}

// RunMigrations chạy AutoMigrate cho tất cả models.
// KHÔNG gọi hàm này trong ConnectDB hay main server.
// Chỉ chạy thủ công khi cần cập nhật schema:
//
//	go run cmd/migrate/main.go
func RunMigrations() {
	if DB == nil {
		log.Fatal("❌ DB chưa được kết nối, gọi ConnectDB() trước")
	}

	fmt.Println("⏳ Đang migrate database...")

	err := DB.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.GroupMember{},
		&models.Expense{},
		&models.ExpenseSplit{},
		&models.Activity{},
		&models.ActivityVote{},
		&models.ActivityRating{},
		&models.ChecklistItem{},
		&models.Document{},
		&models.GroupFavorite{},
	)
	if err != nil {
		log.Fatal("❌ Lỗi Migrate: ", err)
	}

	fmt.Println("✅ Migrate thành công! Database đã cập nhật.")
}

func GetDB() *gorm.DB {
	return DB
}
