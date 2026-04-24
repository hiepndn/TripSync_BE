package usecase

import (
	"context"
	"errors"
	"tripsync-backend/dto"
	"tripsync-backend/models"
	"tripsync-backend/repository"
)

type ChecklistUseCase interface {
	CreateItem(ctx context.Context, groupID uint, req dto.CreateChecklistItemReq) (*models.ChecklistItem, error)
	GetItemsByGroup(ctx context.Context, groupID uint) ([]models.ChecklistItem, error)
	ToggleComplete(ctx context.Context, itemID uint, groupID uint, userID uint) error
	AssignMember(ctx context.Context, itemID uint, groupID uint, req dto.AssignMemberReq) error
	DeleteItem(ctx context.Context, itemID uint, groupID uint) error
}

type checklistUseCase struct {
	repo repository.ChecklistRepository
}

func NewChecklistUseCase(repo repository.ChecklistRepository) ChecklistUseCase {
	return &checklistUseCase{repo: repo}
}

func (u *checklistUseCase) CreateItem(ctx context.Context, groupID uint, req dto.CreateChecklistItemReq) (*models.ChecklistItem, error) {
	item := &models.ChecklistItem{
		GroupID:     groupID,
		Title:       req.Title,
		Category:    req.Category,
		IsCompleted: false,
	}

	if err := u.repo.CreateItem(ctx, item); err != nil {
		return nil, errors.New("không thể tạo việc cần làm: " + err.Error())
	}
	return item, nil
}

func (u *checklistUseCase) GetItemsByGroup(ctx context.Context, groupID uint) ([]models.ChecklistItem, error) {
	return u.repo.GetItemsByGroup(ctx, groupID)
}

// ToggleComplete đổi trạng thái Xong <-> Chưa xong
func (u *checklistUseCase) ToggleComplete(ctx context.Context, itemID uint, groupID uint, userID uint) error {
	item, err := u.repo.GetItemByID(ctx, itemID, groupID)
	if err != nil {
		return errors.New("không tìm thấy công việc")
	}

	item.IsCompleted = !item.IsCompleted
	if item.IsCompleted {
		item.CompletedByID = &userID
	} else {
		item.CompletedByID = nil
	}

	return u.repo.UpdateItem(ctx, item)
}

// AssignMember giao việc cho thành viên
func (u *checklistUseCase) AssignMember(ctx context.Context, itemID uint, groupID uint, req dto.AssignMemberReq) error {
	item, err := u.repo.GetItemByID(ctx, itemID, groupID)
	if err != nil {
		return errors.New("không tìm thấy công việc")
	}

	item.AssigneeID = req.AssigneeID
	return u.repo.UpdateItem(ctx, item)
}

func (u *checklistUseCase) DeleteItem(ctx context.Context, itemID uint, groupID uint) error {
	return u.repo.DeleteItem(ctx, itemID, groupID)
}
