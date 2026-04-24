package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"
	"tripsync-backend/dto"
	"tripsync-backend/models"
	"tripsync-backend/repository"
	"tripsync-backend/service"
	"tripsync-backend/ws"
)

type GroupUseCase interface {
	CreateGroup(ctx context.Context, req dto.CreateGroupRequest, userID uint) (*models.Group, error)
	GetUserGroups(ctx context.Context, userID uint) ([]models.Group, error)
	GetUserGroupsWithRole(ctx context.Context, userID uint) ([]dto.GroupWithRole, error)
	JoinGroupByCode(ctx context.Context, inviteCode string, userID uint) (*models.Group, error)
	GetGroupDetail(ctx context.Context, groupID uint, userID uint) (*models.Group, string, []models.User, error)
	RegenerateAIActivities(ctx context.Context, groupID uint, userID uint) error
	UpdateGroup(ctx context.Context, groupID uint, userID uint, req dto.UpdateGroupRequest) (*models.Group, error)
	RemoveMember(ctx context.Context, groupID uint, targetUserID uint, requestingUserID uint) error
	DeleteGroup(ctx context.Context, groupID uint, userID uint) error
	UpdateVisibility(ctx context.Context, groupID uint, userID uint, isPublic bool) error
	GetPublicGroups(ctx context.Context) ([]models.Group, error)
	GetPublicGroupDetail(ctx context.Context, groupID uint) (*dto.PublicGroupDetailResponse, error)
}

type groupUseCase struct {
	groupRepo    repository.GroupRepository
	activityRepo repository.ActivityRepository
	expenseRepo  repository.ExpenseRepository
}

func NewGroupUseCase(groupRepo repository.GroupRepository, activityRepo repository.ActivityRepository, expenseRepo repository.ExpenseRepository) GroupUseCase {
	return &groupUseCase{
		groupRepo:    groupRepo,
		activityRepo: activityRepo,
		expenseRepo:  expenseRepo,
	}
}

// generateInviteCode sinh mã mời 8 ký tự ngẫu nhiên (chữ hoa + số).
func generateInviteCode() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const length = 8
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

func (u *groupUseCase) CreateGroup(ctx context.Context, req dto.CreateGroupRequest, userID uint) (*models.Group, error) {
	if req.EndDate.Before(req.StartDate) {
		return nil, errors.New("ngày kết thúc không được trước ngày bắt đầu")
	}

	var inviteCode string
	for i := 0; i < 5; i++ {
		code, err := generateInviteCode()
		if err != nil {
			return nil, errors.New("lỗi khi sinh mã mời")
		}
		inviteCode = code
		if exists, _ := u.groupRepo.InviteCodeExists(ctx, inviteCode); !exists {
			break
		}
	}

	shareToken := fmt.Sprintf("share_%d", time.Now().UnixNano())

	newGroup := &models.Group{
		Name:              req.Name,
		Description:       req.Description,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		InviteCode:        inviteCode,
		ShareToken:        shareToken,
		IsPublic:          false,
		DepartureLocation: req.DepartureLocation,
		RouteDestinations: req.RouteDestinations,
		AccommodationPref: req.AccommodationPref,
		ExpectedMembers:   req.ExpectedMembers,
		BudgetPerPerson:   req.BudgetPerPerson,
		Currency:          req.Currency,
		IsAIGenerating:    true,
	}

	err := u.groupRepo.CreateGroupWithAdmin(ctx, newGroup, userID)
	if err != nil {
		return nil, errors.New("lỗi khi tạo nhóm: " + err.Error())
	}

	u.RunAIGenerationBackground(newGroup)

	return newGroup, nil
}

func (u *groupUseCase) RegenerateAIActivities(ctx context.Context, groupID uint, userID uint) error {
	role, err := u.groupRepo.GetUserRoleInGroup(ctx, groupID, userID)
	if err != nil || role != "ADMIN" {
		return errors.New("chỉ Admin (Owner) mới có quyền tạo lại lịch trình AI")
	}

	group, err := u.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return errors.New("không tìm thấy thông tin nhóm")
	}

	if err := u.activityRepo.DeletePendingAIActivities(groupID); err != nil {
		return errors.New("lỗi khi dọn dẹp lịch trình cũ")
	}

	if err := u.groupRepo.UpdateAIGeneratingStatus(ctx, groupID, true); err != nil {
		return errors.New("lỗi khi cập nhật trạng thái AI")
	}

	u.RunAIGenerationBackground(group)

	return nil
}

// RunAIGenerationBackground đóng gói toàn bộ logic goroutine AI để dùng chung
func (u *groupUseCase) RunAIGenerationBackground(g *models.Group) {
	go func(group *models.Group) {
		ctx := context.Background()

		defer func() {
			if r := recover(); r != nil {
				errMsg := fmt.Sprintf("Lỗi nội bộ hệ thống AI: %v", r)
				fmt.Printf("🔥 Panic trong goroutine AI nhóm %d: %v\n", group.ID, r)
				_ = u.groupRepo.SetAIError(ctx, group.ID, errMsg)
			}
		}()

		geminiSvc := service.NewGeminiService(u.activityRepo)
		agodaSvc := service.NewAgodaService()

		fmt.Println("⏳ Đang gọi Gemini 3 Flash Preview chạy nền...")
		aiActivities, err := geminiSvc.GenerateItinerary(ctx, group)
		if err != nil {
			fmt.Println("❌ Lỗi gọi Gemini:", err)
			_ = u.groupRepo.SetAIError(ctx, group.ID, "AI gặp sự cố khi tạo lịch trình. Vui lòng thử lại sau.")
			return
		}

		for _, aiAct := range aiActivities {
			validTypes := map[string]bool{"HOTEL": true, "ATTRACTION": true, "RESTAURANT": true, "CAMPING": true, "TRANSPORT": true}
			if !validTypes[aiAct.Type] {
				fmt.Printf("⚠️ Bỏ qua activity '%s' có type không hợp lệ: %s\n", aiAct.Name, aiAct.Type)
				continue
			}
			if aiAct.Type == "HOTEL" {
				tCheckIn, _ := time.Parse(time.RFC3339, aiAct.StartTime)
				tCheckOut, _ := time.Parse(time.RFC3339, aiAct.EndTime)

				if tCheckOut.Before(tCheckIn) || tCheckOut.Equal(tCheckIn) {
					tCheckOut = tCheckIn.AddDate(0, 0, 1)
				}

				checkInStr := tCheckIn.Format("2006-01-02")
				checkOutStr := tCheckOut.Format("2006-01-02")

				fmt.Printf("🔍 Tìm khách sạn tại %s (Từ %s đến %s - Ngân sách: %.0f %s)...\n",
					aiAct.Location, checkInStr, checkOutStr, aiAct.EstimatedCost, group.Currency)

				realHotels, err := agodaSvc.SearchHotels(ctx, aiAct.Location, checkInStr, checkOutStr, aiAct.EstimatedCost, group.Currency, group.ExpectedMembers, group.ID)

				if err == nil && len(realHotels) > 0 {
					for _, hotel := range realHotels {
						hotel.StartTime = tCheckIn
						hotel.EndTime = tCheckOut
						_ = u.activityRepo.Create(ctx, &hotel)
					}
					continue
				}
				fmt.Printf("⚠️ Agoda tạch ở %s, dùng tạm dự phòng của AI\n", aiAct.Location)
			}

			tStart, _ := time.Parse(time.RFC3339, aiAct.StartTime)
			tEnd, _ := time.Parse(time.RFC3339, aiAct.EndTime)

			normalActivity := &models.Activity{
				GroupID:       group.ID,
				Name:          aiAct.Name,
				Type:          aiAct.Type,
				Location:      aiAct.Location,
				Description:   aiAct.Description,
				StartTime:     tStart,
				EndTime:       tEnd,
				Status:        models.StatusPending,
				CreatedBy:     nil,
				Lat:           aiAct.Lat,
				Lng:           aiAct.Lng,
				IsAIGenerated: true,
				EstimatedCost: aiAct.EstimatedCost,
				Currency:      group.Currency,
			}
			_ = u.activityRepo.Create(ctx, normalActivity)
		}

		if err := u.groupRepo.UpdateAIGeneratingStatus(ctx, group.ID, false); err != nil {
			fmt.Printf("Lỗi khi tắt cờ AI cho nhóm %d: %v\n", group.ID, err)
		}
		fmt.Printf("✅ Đã tắt cờ IsAIGenerating cho nhóm %d\n", group.ID)
		fmt.Println("🎉 Đã lưu toàn bộ lịch trình AI và khách sạn vào Database!")

		// 🚀 Broadcast WebSocket: báo FE reload ngay, không cần polling
		ws.GlobalHub.Broadcast(group.ID, ws.WSMessage{
			Event:   ws.EventAIDone,
			GroupID: group.ID,
		}.Encode())
	}(g)
}

func (u *groupUseCase) GetUserGroups(ctx context.Context, userID uint) ([]models.Group, error) {
	return u.groupRepo.GetGroupsByUserID(ctx, userID)
}

func (u *groupUseCase) GetUserGroupsWithRole(ctx context.Context, userID uint) ([]dto.GroupWithRole, error) {
	return u.groupRepo.GetGroupsByUserIDWithRole(ctx, userID)
}

func (u *groupUseCase) JoinGroupByCode(ctx context.Context, inviteCode string, userID uint) (*models.Group, error) {
	group, err := u.groupRepo.GetGroupByInviteCode(ctx, inviteCode)
	if err != nil {
		return nil, errors.New("Mã mời không hợp lệ hoặc nhóm không tồn tại")
	}

	inGroup, err := u.groupRepo.IsUserInGroup(ctx, group.ID, userID)
	if err != nil {
		return nil, errors.New("Lỗi hệ thống khi kiểm tra thành viên")
	}
	if inGroup {
		return nil, errors.New("Bạn đã là thành viên của nhóm này rồi!")
	}

	newMember := models.GroupMember{
		GroupID:  group.ID,
		UserID:   userID,
		Role:     "MEMBER",
		JoinedAt: time.Now(),
	}

	if err := u.groupRepo.AddMember(ctx, &newMember); err != nil {
		return nil, errors.New("Không thể tham gia nhóm lúc này, vui lòng thử lại")
	}

	return group, nil
}

func (u *groupUseCase) GetGroupDetail(ctx context.Context, groupID uint, userID uint) (*models.Group, string, []models.User, error) {
	role, err := u.groupRepo.GetUserRoleInGroup(ctx, groupID, userID)
	if err != nil {
		return nil, "", nil, errors.New("bạn không có quyền truy cập hoặc nhóm không tồn tại")
	}

	group, err := u.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, "", nil, errors.New("không tìm thấy thông tin nhóm")
	}

	members, err := u.groupRepo.GetGroupMembers(ctx, groupID)
	if err != nil {
		return nil, "", nil, errors.New("lỗi khi lấy danh sách thành viên")
	}

	return group, role, members, nil
}

func (u *groupUseCase) UpdateGroup(ctx context.Context, groupID uint, userID uint, req dto.UpdateGroupRequest) (*models.Group, error) {
	role, err := u.groupRepo.GetUserRoleInGroup(ctx, groupID, userID)
	if err != nil || role != "ADMIN" {
		return nil, errors.New("chỉ Admin mới có quyền chỉnh sửa thông tin nhóm")
	}
	if !req.EndDate.IsZero() && !req.StartDate.IsZero() && req.EndDate.Before(req.StartDate) {
		return nil, errors.New("ngày kết thúc không được trước ngày bắt đầu")
	}
	return u.groupRepo.UpdateGroup(ctx, groupID, req)
}

func (u *groupUseCase) RemoveMember(ctx context.Context, groupID uint, targetUserID uint, requestingUserID uint) error {
	if targetUserID == requestingUserID {
		return errors.New("không thể tự xóa chính mình khỏi nhóm")
	}
	role, err := u.groupRepo.GetUserRoleInGroup(ctx, groupID, requestingUserID)
	if err != nil || role != "ADMIN" {
		return errors.New("chỉ Admin mới có quyền xóa thành viên")
	}
	return u.groupRepo.RemoveMember(ctx, groupID, targetUserID)
}

func (u *groupUseCase) DeleteGroup(ctx context.Context, groupID uint, userID uint) error {
	role, err := u.groupRepo.GetUserRoleInGroup(ctx, groupID, userID)
	if err != nil || role != "ADMIN" {
		return errors.New("chỉ Admin mới có quyền xóa nhóm")
	}
	return u.groupRepo.DeleteGroup(ctx, groupID)
}

func (u *groupUseCase) UpdateVisibility(ctx context.Context, groupID uint, userID uint, isPublic bool) error {
	_, err := u.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return errors.New("không tìm thấy nhóm")
	}

	role, err := u.groupRepo.GetUserRoleInGroup(ctx, groupID, userID)
	if err != nil || role != "ADMIN" {
		return errors.New("chỉ Admin mới có quyền thay đổi chế độ công khai")
	}

	return u.groupRepo.UpdateVisibility(ctx, groupID, isPublic)
}

func (u *groupUseCase) GetPublicGroups(ctx context.Context) ([]models.Group, error) {
	return u.groupRepo.GetPublicGroups(ctx)
}

func (u *groupUseCase) GetPublicGroupDetail(ctx context.Context, groupID uint) (*dto.PublicGroupDetailResponse, error) {
	group, err := u.groupRepo.GetPublicGroupDetail(ctx, groupID)
	if err != nil {
		_, existErr := u.groupRepo.GetByID(ctx, groupID)
		if existErr != nil {
			return nil, errors.New("not_found")
		}
		return nil, errors.New("forbidden")
	}

	activities, err := u.activityRepo.GetActivitiesByGroup(ctx, int(groupID), 0)
	if err != nil {
		activities = nil
	}

	var activityModels []models.Activity
	for _, a := range activities {
		activityModels = append(activityModels, a.Activity)
	}

	expenses, _, err := u.expenseRepo.GetAllByGroup(ctx, groupID)
	var totalAmount float64
	expenseCount := 0
	if err == nil {
		for _, exp := range expenses {
			totalAmount += exp.Amount
			expenseCount++
		}
	}

	return &dto.PublicGroupDetailResponse{
		GroupInfo:  group,
		Activities: activityModels,
		ExpenseSummary: dto.PublicExpenseSummary{
			TotalAmount:  totalAmount,
			ExpenseCount: expenseCount,
		},
	}, nil
}
