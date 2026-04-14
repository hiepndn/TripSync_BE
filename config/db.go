package config

import (
	"fmt"
	"log"
	"os"
	"tripsync-backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Biến DB này viết hoa chữ cái đầu để các package khác (như Repository) có thể gọi được
var DB *gorm.DB

func ConnectDB() {
	// Lấy thông tin từ biến môi trường (đã load ở main)
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Ho_Chi_Minh",
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

	fmt.Println("✅ Đã kết nối Database thành công (từ config)!")

	// --- THÊM ĐOẠN NÀY ĐỂ CHẠY MIGRATE ---
	fmt.Println("⏳ Đang migrate database...")

	// Liệt kê tất cả các Struct bạn vừa tạo vào đây
	err = DB.AutoMigrate(
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
		// Thêm Checklist, Documents nếu bạn tạo file models tương ứng
	)

	if err != nil {
		log.Fatal("❌ Lỗi Migrate: ", err)
	}

	fmt.Println("✅ Migrate thành công! Database đã cập nhật.")
}

// Hàm helper để lấy DB instance (nếu thích dùng kiểu Getter)
func GetDB() *gorm.DB {
	return DB
}
