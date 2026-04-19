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
	UnfinalizeActivity(ctx context.Context, groupID uint, activityID uint, userID uint) error
	UpdateActivity(userID, groupID, activityID int, req dto.UpdateActivityReq) error
	DeleteActivity(userID, groupID, activityID int) error
	DeleteAllActivities(ctx context.Context, groupID int, userID int) error
	RateActivity(ctx context.Context, groupID int, activityID int, userID int, rating int) error
	GetSuggestions(ctx context.Context, groupID int, activityType string, location string) ([]dto.SuggestionResponse, error)
	// Export / Import
	ExportActivities(ctx context.Context, groupID int) ([]dto.ExportActivityItem, error)
	ImportActivities(ctx context.Context, targetGroupID int, sourceGroupID int, userID int) (int, error)
	ImportFromJSON(ctx context.Context, targetGroupID int, userID int, items []dto.ExportActivityItem) (int, error)
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
		Status:        "PENDING",
		CreatedBy:     &uid,
		Lat:           req.Lat,
		Lng:           req.Lng,
		PlaceID:       req.PlaceID,
		EstimatedCost: req.EstimatedCost,
		Currency:      req.Currency,
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
	role, err := u.groupRepo.GetUserRoleInGroup(groupID, userID)
	if err != nil {
		return errors.New("bạn không phải là thành viên của nhóm này")
	}
	if role != "ADMIN" {
		return errors.New("chỉ Admin mới có quyền chốt hoạt động vào lịch chính thức")
	}
	return u.repo.UpdateStatus(ctx, activityID, "APPROVED")
}

func (u *activityUseCaseImpl) UnfinalizeActivity(ctx context.Context, groupID uint, activityID uint, userID uint) error {
	role, err := u.groupRepo.GetUserRoleInGroup(groupID, userID)
	if err != nil {
		return errors.New("bạn không phải là thành viên của nhóm này")
	}
	if role != "ADMIN" {
		return errors.New("chỉ Admin mới có quyền hủy chốt hoạt động")
	}
	return u.repo.UpdateStatus(ctx, activityID, "PENDING")
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
	activity.EstimatedCost = req.EstimatedCost
	if req.Currency != "" {
		activity.Currency = req.Currency
	}

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

func (uc *activityUseCaseImpl) DeleteAllActivities(ctx context.Context, groupID int, userID int) error {
	role, err := uc.groupRepo.GetUserRoleInGroup(uint(groupID), uint(userID))
	if err != nil || role != "ADMIN" {
		return errors.New("chỉ Admin mới có quyền xóa toàn bộ lịch trình")
	}
	return uc.repo.DeleteAllActivities(ctx, uint(groupID))
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

// ExportActivities returns the public export payload for a group's activities.
// No authentication is required — the caller is responsible for ensuring the group exists.
func (u *activityUseCaseImpl) ExportActivities(ctx context.Context, groupID int) ([]dto.ExportActivityItem, error) {
	// Verify group exists
	_, err := u.groupRepo.GetByID(uint(groupID))
	if err != nil {
		return nil, errors.New("group_not_found")
	}

	activities, err := u.repo.GetRawActivitiesByGroup(ctx, uint(groupID))
	if err != nil {
		return nil, err
	}

	items := make([]dto.ExportActivityItem, 0, len(activities))
	for _, a := range activities {
		items = append(items, dto.ExportActivityItem{
			Name:          a.Name,
			Type:          a.Type,
			Location:      a.Location,
			Description:   a.Description,
			StartTime:     a.StartTime,
			EndTime:       a.EndTime,
			EstimatedCost: a.EstimatedCost,
			Currency:      a.Currency,
			Lat:           a.Lat,
			Lng:           a.Lng,
		})
	}
	return items, nil
}

// ImportActivities copies all activities from sourceGroupID into targetGroupID.
// The caller (userID) must be ADMIN of the target group and a member of the source group.
// All imported activities have their times remapped to the target group's start_date.
func (u *activityUseCaseImpl) ImportActivities(ctx context.Context, targetGroupID int, sourceGroupID int, userID int) (int, error) {
	// Guard: self-import
	if targetGroupID == sourceGroupID {
		return 0, errors.New("self_import")
	}

	// Check user is ADMIN of target group
	targetRole, err := u.groupRepo.GetUserRoleInGroup(uint(targetGroupID), uint(userID))
	if err != nil || targetRole != "ADMIN" {
		return 0, errors.New("not_admin_of_target")
	}

	// Check user is a member of source group
	isMember, err := u.groupRepo.IsUserInGroup(uint(sourceGroupID), uint(userID))
	if err != nil || !isMember {
		return 0, errors.New("not_member_of_source")
	}

	// Fetch target group to get start_date for time remapping
	targetGroup, err := u.groupRepo.GetByID(uint(targetGroupID))
	if err != nil {
		return 0, errors.New("target_group_not_found")
	}

	// Fetch source activities
	sourceActivities, err := u.repo.GetRawActivitiesByGroup(ctx, uint(sourceGroupID))
	if err != nil {
		return 0, err
	}

	if len(sourceActivities) == 0 {
		return 0, nil
	}

	// Find the earliest start_time in source activities to use as base for remapping
	minTime := sourceActivities[0].StartTime
	for _, src := range sourceActivities {
		if src.StartTime.Before(minTime) {
			minTime = src.StartTime
		}
	}
	// Convert to UTC then normalize to start of day
	minTimeUTC := minTime.UTC()
	minDay := time.Date(minTimeUTC.Year(), minTimeUTC.Month(), minTimeUTC.Day(), 0, 0, 0, 0, time.UTC)
	targetBase := time.Date(targetGroup.StartDate.Year(), targetGroup.StartDate.Month(), targetGroup.StartDate.Day(), 0, 0, 0, 0, time.UTC)

	// Build new activities for target group with remapped times
	newActivities := make([]models.Activity, 0, len(sourceActivities))
	for _, src := range sourceActivities {
		offsetStart := src.StartTime.UTC().Sub(minDay)
		offsetEnd := src.EndTime.UTC().Sub(minDay)
		newActivities = append(newActivities, models.Activity{
			GroupID:       uint(targetGroupID),
			Name:          src.Name,
			Type:          src.Type,
			Location:      src.Location,
			Description:   src.Description,
			StartTime:     targetBase.Add(offsetStart),
			EndTime:       targetBase.Add(offsetEnd),
			EstimatedCost: src.EstimatedCost,
			Currency:      src.Currency,
			Lat:           src.Lat,
			Lng:           src.Lng,
			Status:        models.StatusPending,
			IsAIGenerated: false,
		})
	}

	if err := u.repo.BulkCreateActivities(ctx, newActivities); err != nil {
		return 0, err
	}

	return len(newActivities), nil
}

// ImportFromJSON inserts activities from a JSON payload (ExportActivityItem slice) into the target group.
// The caller must be ADMIN of the target group. Times are remapped to target group's start_date.
func (u *activityUseCaseImpl) ImportFromJSON(ctx context.Context, targetGroupID int, userID int, items []dto.ExportActivityItem) (int, error) {
	// Check user is ADMIN of target group
	targetRole, err := u.groupRepo.GetUserRoleInGroup(uint(targetGroupID), uint(userID))
	if err != nil || targetRole != "ADMIN" {
		return 0, errors.New("not_admin_of_target")
	}

	// Fetch target group to get start_date for time remapping
	targetGroup, err := u.groupRepo.GetByID(uint(targetGroupID))
	if err != nil {
		return 0, errors.New("target_group_not_found")
	}

	if len(items) == 0 {
		return 0, nil
	}

	// Find the earliest start_time in JSON items to use as base for remapping
	minTime := items[0].StartTime
	for _, item := range items {
		if item.StartTime.Before(minTime) {
			minTime = item.StartTime
		}
	}
	// Convert to UTC then normalize to start of day — ensures consistent offset calculation
	minTimeUTC := minTime.UTC()
	minDay := time.Date(minTimeUTC.Year(), minTimeUTC.Month(), minTimeUTC.Day(), 0, 0, 0, 0, time.UTC)
	targetBase := time.Date(targetGroup.StartDate.Year(), targetGroup.StartDate.Month(), targetGroup.StartDate.Day(), 0, 0, 0, 0, time.UTC)

	newActivities := make([]models.Activity, 0, len(items))
	for _, item := range items {
		offsetStart := item.StartTime.UTC().Sub(minDay)
		offsetEnd := item.EndTime.UTC().Sub(minDay)
		newActivities = append(newActivities, models.Activity{
			GroupID:       uint(targetGroupID),
			Name:          item.Name,
			Type:          item.Type,
			Location:      item.Location,
			Description:   item.Description,
			StartTime:     targetBase.Add(offsetStart),
			EndTime:       targetBase.Add(offsetEnd),
			EstimatedCost: item.EstimatedCost,
			Currency:      item.Currency,
			Lat:           item.Lat,
			Lng:           item.Lng,
			Status:        models.StatusPending,
			IsAIGenerated: false,
		})
	}

	if err := u.repo.BulkCreateActivities(ctx, newActivities); err != nil {
		return 0, err
	}

	return len(newActivities), nil
}
