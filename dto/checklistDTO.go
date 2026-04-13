package dto

type CreateChecklistItemReq struct {
	Title    string `json:"title" binding:"required"`
	Category string `json:"category" binding:"required"` // VD: "Đồ dùng chung"
}

type AssignMemberReq struct {
	AssigneeID *uint `json:"assignee_id"` // Dùng con trỏ để hỗ trợ truyền null (Hủy gán việc)
}
