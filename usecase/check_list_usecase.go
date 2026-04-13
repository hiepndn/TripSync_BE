package usecase

import (
	"errors"
	"tripsync-backend/dto"
	"tripsync-backend/models"
	"tripsync-backend/repository"
)

type ChecklistUseCase interface {
	CreateItem(groupID uint, req dto.CreateChecklistItemReq) (*models.ChecklistItem, error)
	GetItemsByGroup(groupID uint) ([]models.ChecklistItem, error)
	ToggleComplete(itemID uint, groupID uint, userID uint) error
	AssignMember(itemID uint, groupID uint, req dto.AssignMemberReq) error
	DeleteItem(itemID uint, groupID uint) error
}

type checklistUseCase struct {
	repo repository.ChecklistRepository
}

func NewChecklistUseCase(repo repository.ChecklistRepository) ChecklistUseCase {
	return &checklistUseCase{repo: repo}
}

func (u *checklistUseCase) CreateItem(groupID uint, req dto.CreateChecklistItemReq) (*models.ChecklistItem, error) {
	item := &models.ChecklistItem{
		GroupID:     groupID,
		Title:       req.Title,
		Category:    req.Category,
		IsCompleted: false,
	}

	if err := u.repo.CreateItem(item); err != nil {
		return nil, errors.New("không thể tạo việc cần làm: " + err.Error())
	}
	return item, nil
}

func (u *checklistUseCase) GetItemsByGroup(groupID uint) ([]models.ChecklistItem, error) {
	return u.repo.GetItemsByGroup(groupID)
}

// Logic: Đổi trạng thái Xong <-> Chưa xong
func (u *checklistUseCase) ToggleComplete(itemID uint, groupID uint, userID uint) error {
	item, err := u.repo.GetItemByID(itemID, groupID)
	if err != nil {
		return errors.New("không tìm thấy công việc")
	}

	item.IsCompleted = !item.IsCompleted
	if item.IsCompleted {
		// Nếu đánh dấu xong, lưu lại ID người bấm
		item.CompletedByID = &userID
	} else {
		// Bỏ đánh dấu thì clear ID
		item.CompletedByID = nil
	}

	return u.repo.UpdateItem(item)
}

// Logic: Giao việc cho thành viên
func (u *checklistUseCase) AssignMember(itemID uint, groupID uint, req dto.AssignMemberReq) error {
	item, err := u.repo.GetItemByID(itemID, groupID)
	if err != nil {
		return errors.New("không tìm thấy công việc")
	}

	item.AssigneeID = req.AssigneeID
	return u.repo.UpdateItem(item)
}

func (u *checklistUseCase) DeleteItem(itemID uint, groupID uint) error {
	return u.repo.DeleteItem(itemID, groupID)
}
