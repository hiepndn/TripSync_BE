package dto

type CreateDocumentRequest struct {
	FileName string `json:"file_name" binding:"required"`
	FileURL  string `json:"file_url"  binding:"required"`
	FileType string `json:"file_type" binding:"required"`
	FileSize int64  `json:"file_size" binding:"required"`
	Category string `json:"category"  binding:"required"`
}
