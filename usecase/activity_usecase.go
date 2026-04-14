package usecase

import (
	"context"
	"errors"
	"time"
	"tripsync-backend/dto"
	"tripsync-backend/models"
	"tripsync-backend/repository"
)

type ActivityUseCase interface {
	GetGroupActivities(ctx context.Context, groupID int, userID int) ([]dto.ActivityResponse, error)
	CreateActivity(ctx context.Context, groupID int, userID int, req dto.CreateActivityReq) error
	ToggleActivityVote(ctx context.Context, activityID int, userID int) (bool, error)
	FinalizeActivity(ctx context.Context, groupID uint, activityID uint, userID uint) error
	UpdateActivity(userID, groupID, activityID int, req dto.UpdateActivityReq) error
	DeleteActivity(userID, groupID, activityID int) error
	RateActivity(ctx context.Context, groupID int, activityID int, userID int, rating int) error
	GetSuggestions(ctx context.Context, groupID int, activityType string, location string) ([]dto.SuggestionResponse, error)
}

type activityUseCaseImpl struct {
	repo      repository.ActivityRepository
	groupRepo repository.GroupRepository
}

func NewActivityUseCase(repo repository.ActivityRepository, groupRepo repository.GroupRepository) ActivityUseCase {
	return &activityUseCaseImpl{
		repo:      repo,
		groupRepo: groupRepo,
	}
}

func (u *activityUseCaseImpl) GetGroupActivities(ctx context.Context, groupID int, userID int) ([]dto.ActivityResponse, error) {
	// Kiểm tra xem userID có quyền xem group này không (gọi GroupRepo check role)
	// Tạm thời bỏ qua bước check role vì bạn đã có API getGroup làm việc đó

	activities, err := u.repo.GetActivitiesByGroup(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	return activities, nil
}

func (u *activityUseCaseImpl) CreateActivity(ctx context.Context, groupID int, userID int, req dto.CreateActivityReq) error {
	// 1. Lấy thông tin Group hiện tại (giả sử bạn có groupRepo)
	group, err := u.groupRepo.GetByID(uint(groupID))
	if err != nil {
		return err
	}

	// 2. Logic tính toán nới rộng ngày
	activityDate := req.StartTime.Truncate(24 * time.Hour)
	groupStartDate := group.StartDate.Truncate(24 * time.Hour)
	groupEndDate := group.EndDate.Truncate(24 * time.Hour)

	needsDateUpdate := false
	newStartDate := group.StartDate
	newEndDate := group.EndDate

	if activityDate.Before(groupStartDate) {
		newStartDate = activityDate
		needsDateUpdate = true
	}
	if activityDate.After(groupEndDate) {
		newEndDate = activityDate
		needsDateUpdate = true
	}
	uid := uint(userID)
	newActivity := &models.Activity{
		GroupID:       uint(groupID),
		Name:          req.Name,
		Type:          req.Type,
		Location:      req.Location,
		Description:   req.Description,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Status:        "PENDING", // Mặc định vào khu vực "Đang bỏ phiếu"
		CreatedBy:     &uid,
		Lat:           req.Lat,
		Lng:           req.Lng,
		PlaceID:       req.PlaceID,
		IsAIGenerated: false,
	}

	// 4. Gọi Repository thực thi Transaction
	return u.repo.CreateWithGroupDateUpdate(ctx, newActivity, needsDateUpdate, newStartDate, newEndDate)
}

func (u *activityUseCaseImpl) ToggleActivityVote(ctx context.Context, activityID int, userID int) (bool, error) {
	// Gọi repo thực thi logic toggle
	isAdded, err := u.repo.ToggleVote(ctx, activityID, userID)
	if err != nil {
		return false, err
	}
	return isAdded, nil
}

func (u *activityUseCaseImpl) FinalizeActivity(ctx context.Context, groupID uint, activityID uint, userID uint) error {
	// 1. Dùng chính hàm bạn vừa cung cấp để check role
	role, err := u.groupRepo.GetUserRoleInGroup(groupID, userID)
	if err != nil {
		return errors.New("bạn không phải là thành viên của nhóm này")
	}

	// 2. Chặn đứng nếu không phải ADMIN
	if role != "ADMIN" { // models.RoleAdmin nếu bạn dùng hằng số
		return errors.New("chỉ Admin mới có quyền chốt hoạt động vào lịch chính thức")
	}

	// 3. Nếu qua ải Admin, gọi Repo đổi status sang APPROVED
	return u.repo.UpdateStatus(ctx, activityID, "APPROVED")
}

func (uc *activityUseCaseImpl) UpdateActivity(userID, groupID, activityID int, req dto.UpdateActivityReq) error {
	// 1. Lấy activity ra check tồn tại
	activity, err := uc.repo.GetByID(activityID)
	if err != nil {
		return errors.New("hoạt động không tồn tại")
	}

	// 2. Phân quyền: Gọi đúng hàm GetUserRoleInGroup và ép kiểu sang uint
	role, _ := uc.groupRepo.GetUserRoleInGroup(uint(groupID), uint(userID))
	if role != "ADMIN" && (activity.CreatedBy == nil || *activity.CreatedBy != uint(userID)) {
		return errors.New("bạn không có quyền sửa hoạt động này")
	}

	// 3. Cập nhật các trường
	activity.Name = req.Name
	activity.Type = req.Type
	activity.Location = req.Location
	activity.Description = req.Description
	activity.StartTime = req.StartTime
	activity.EndTime = req.EndTime

	// 4. Lưu xuống DB
	return uc.repo.Update(activity)
}

func (uc *activityUseCaseImpl) DeleteActivity(userID, groupID, activityID int) error {
	activity, err := uc.repo.GetByID(activityID)
	if err != nil {
		return errors.New("hoạt động không tồn tại")
	}

	// Phân quyền: ADMIN hoặc chính chủ mới được xóa
	role, _ := uc.groupRepo.GetUserRoleInGroup(uint(groupID), uint(userID))
	if role != "ADMIN" && (activity.CreatedBy == nil || *activity.CreatedBy != uint(userID)) {
		return errors.New("bạn không có quyền xóa hoạt động này")
	}

	return uc.repo.Delete(activityID)
}

func (u *activityUseCaseImpl) RateActivity(ctx context.Context, groupID int, activityID int, userID int, rating int) error {
	_, err := u.groupRepo.GetUserRoleInGroup(uint(groupID), uint(userID))
	if err != nil {
		return errors.New("bạn không phải là thành viên của nhóm này")
	}
	return u.repo.UpsertRating(ctx, uint(activityID), uint(userID), rating)
}

func (u *activityUseCaseImpl) GetSuggestions(ctx context.Context, groupID int, activityType string, location string) ([]dto.SuggestionResponse, error) {
	validTypes := map[string]bool{"HOTEL": true, "ATTRACTION": true, "RESTAURANT": true, "CAMPING": true}
	if !validTypes[activityType] {
		return nil, errors.New("type must be one of HOTEL, ATTRACTION, RESTAURANT, CAMPING")
	}
	if location == "" {
		return nil, errors.New("location query parameter is required")
	}

	// Lấy route_destinations của group hiện tại để filter các group có chung điểm đến
	group, err := u.groupRepo.GetByID(uint(groupID))
	if err != nil {
		return nil, errors.New("không tìm thấy thông tin nhóm")
	}

	return u.repo.GetSuggestions(ctx, uint(groupID), activityType, location, group.RouteDestinations)
}
