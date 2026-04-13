package models

type Document struct {
	BaseModel
	GroupID  uint   `gorm:"not null" json:"group_id"`
	FileName string `gorm:"not null" json:"file_name"` // Tên hiển thị (VD: Vé máy bay VNA.pdf)
	FileURL  string `gorm:"not null" json:"file_url"`  // Link lưu trữ thật (S3, Cloudinary...)
	FileType string `json:"file_type"`                 // PDF, JPG, XLSX...
	Category string `json:"category"`                  // Danh mục (Vé máy bay, Khách sạn)
	FileSize int64  `json:"file_size"`                 // Dung lượng (tính bằng bytes)

	UploadedByID uint  `gorm:"not null" json:"uploaded_by_id"`
	UploadedBy   User  `gorm:"foreignKey:UploadedByID" json:"uploaded_by"`
	ActivityID   *uint `gorm:"column:activity_id"`
}
