package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
	"tripsync-backend/dto"
	"tripsync-backend/models"
	"tripsync-backend/repository"
	"tripsync-backend/service"
)

type GroupUseCase interface {
	CreateGroup(req dto.CreateGroupRequest, userID uint) (*models.Group, error)
	GetUserGroups(userID uint) ([]models.Group, error)
	JoinGroupByCode(inviteCode string, userID uint) (*models.Group, error)
	GetGroupDetail(groupID uint, userID uint) (*models.Group, string, []models.User, error)
	RegenerateAIActivities(groupID uint, userID uint) error
	UpdateGroup(groupID uint, userID uint, req dto.UpdateGroupRequest) (*models.Group, error)
	RemoveMember(groupID uint, targetUserID uint, requestingUserID uint) error
	DeleteGroup(groupID uint, userID uint) error
}

type groupUseCase struct {
	groupRepo    repository.GroupRepository
	activityRepo repository.ActivityRepository
}

func NewGroupUseCase(groupRepo repository.GroupRepository, activityRepo repository.ActivityRepository) GroupUseCase {
	return &groupUseCase{
		groupRepo:    groupRepo,
		activityRepo: activityRepo,
	}
}

func (u *groupUseCase) CreateGroup(req dto.CreateGroupRequest, userID uint) (*models.Group, error) {
	if req.EndDate.Before(req.StartDate) {
		return nil, errors.New("ngày kết thúc không được trước ngày bắt đầu")
	}

	inviteCode := fmt.Sprintf("TRIP%s", time.Now().Format("150405"))
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

	err := u.groupRepo.CreateGroupWithAdmin(newGroup, userID)
	if err != nil {
		return nil, errors.New("lỗi khi tạo nhóm: " + err.Error())
	}

	// 🌟 GỌI HÀM CHẠY NGẦM ĐÃ ĐƯỢC TÁCH RIÊNG
	u.RunAIGenerationBackground(newGroup)

	return newGroup, nil
}

// 🌟 HÀM MỚI: XỬ LÝ REGENERATE AI (XÓA CŨ -> GỌI LẠI HÀM NGẦM)
func (u *groupUseCase) RegenerateAIActivities(groupID uint, userID uint) error {
	// 1. Phân quyền: Chỉ ADMIN mới được tạo lại
	role, err := u.groupRepo.GetUserRoleInGroup(groupID, userID)
	if err != nil || role != "ADMIN" {
		return errors.New("chỉ Admin (Owner) mới có quyền tạo lại lịch trình AI")
	}

	// 2. Lấy thông tin nhóm (Để xíu nữa quăng vào hàm AI)
	group, err := u.groupRepo.GetByID(groupID)
	if err != nil {
		return errors.New("không tìm thấy thông tin nhóm")
	}

	// 3. HARD DELETE: Xóa sạch các hoạt động AI cũ đang PENDING
	// (Đảm bảo ông đã khai báo hàm DeletePendingAIActivities trong activityRepo nhé)
	if err := u.activityRepo.DeletePendingAIActivities(groupID); err != nil {
		return errors.New("lỗi khi dọn dẹp lịch trình cũ")
	}

	// 4. Bật lại cờ Loading cho Frontend
	if err := u.groupRepo.UpdateAIGeneratingStatus(groupID, true); err != nil {
		return errors.New("lỗi khi cập nhật trạng thái AI")
	}

	// 5. GỌI LẠI HÀM CHẠY NGẦM
	u.RunAIGenerationBackground(group)

	return nil
}

// =========================================================================
// 🌟 HELPER METHOD: ĐÓNG GÓI TOÀN BỘ LOGIC GOROUTINE VÀO ĐÂY ĐỂ DÙNG CHUNG
// =========================================================================
func (u *groupUseCase) RunAIGenerationBackground(g *models.Group) {
	go func(group *models.Group) {
		ctx := context.Background()

		// DEFER: Luôn TẮT CỜ khi kết thúc
		defer func() {
			err := u.groupRepo.UpdateAIGeneratingStatus(group.ID, false)
			if err != nil {
				fmt.Printf("Lỗi khi tắt cờ AI cho nhóm %d: %v\n", group.ID, err)
			} else {
				fmt.Printf("✅ Đã tắt cờ IsAIGenerating cho nhóm %d\n", group.ID)
			}
		}()

		geminiSvc := service.NewGeminiService(u.activityRepo)
		agodaSvc := service.NewAgodaService()

		fmt.Println("⏳ Đang gọi Gemini 3 Flash Preview chạy nền...")
		aiActivities, err := geminiSvc.GenerateItinerary(ctx, group)
		if err != nil {
			fmt.Println("❌ Lỗi gọi Gemini:", err)
			return
		}

		for _, aiAct := range aiActivities {
			// Bỏ qua các activity có type không hợp lệ
			validTypes := map[string]bool{"HOTEL": true, "ATTRACTION": true, "RESTAURANT": true, "CAMPING": true, "TRANSPORT": true}
			if !validTypes[aiAct.Type] {
				fmt.Printf("⚠️ Bỏ qua activity '%s' có type không hợp lệ: %s\n", aiAct.Name, aiAct.Type)
				continue
			}
			// XỬ LÝ GỌI AGODA NẾU LÀ KHÁCH SẠN
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

			// NẾU LÀ ĐI CHƠI/ĂN UỐNG HOẶC FALLBACK TỪ KHÁCH SẠN
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

		fmt.Println("🎉 Đã lưu toàn bộ lịch trình AI và khách sạn vào Database!")
	}(g)
}

func (u *groupUseCase) GetUserGroups(userID uint) ([]models.Group, error) {
	return u.groupRepo.GetGroupsByUserID(userID)
}

func (u *groupUseCase) JoinGroupByCode(inviteCode string, userID uint) (*models.Group, error) {
	// 1. Tìm nhóm theo Invite Code
	group, err := u.groupRepo.GetGroupByInviteCode(inviteCode)
	if err != nil {
		return nil, errors.New("Mã mời không hợp lệ hoặc nhóm không tồn tại")
	}

	// 2. Kiểm tra xem người dùng đã ở trong nhóm chưa
	inGroup, err := u.groupRepo.IsUserInGroup(group.ID, userID)
	if err != nil {
		return nil, errors.New("Lỗi hệ thống khi kiểm tra thành viên")
	}
	if inGroup {
		return nil, errors.New("Bạn đã là thành viên của nhóm này rồi!")
	}

	// 3. Thêm user vào bảng group_members với quyền MEMBER
	newMember := models.GroupMember{
		GroupID:  group.ID,
		UserID:   userID,
		Role:     "MEMBER", // Có thể thay bằng models.RoleMember nếu bạn đã định nghĩa hằng số
		JoinedAt: time.Now(),
	}

	if err := u.groupRepo.AddMember(&newMember); err != nil {
		return nil, errors.New("Không thể tham gia nhóm lúc này, vui lòng thử lại")
	}

	return group, nil
}

func (u *groupUseCase) GetGroupDetail(groupID uint, userID uint) (*models.Group, string, []models.User, error) {
	// 1. Kiểm tra quyền (như cũ)
	role, err := u.groupRepo.GetUserRoleInGroup(groupID, userID)
	if err != nil {
		return nil, "", nil, errors.New("bạn không có quyền truy cập hoặc nhóm không tồn tại")
	}

	// 2. Lấy chi tiết nhóm (như cũ)
	group, err := u.groupRepo.GetByID(groupID)
	if err != nil {
		return nil, "", nil, errors.New("không tìm thấy thông tin nhóm")
	}

	// 3. Lấy full danh sách thành viên + Role (MỚI THÊM)
	members, err := u.groupRepo.GetGroupMembers(groupID)
	if err != nil {
		return nil, "", nil, errors.New("lỗi khi lấy danh sách thành viên")
	}

	return group, role, members, nil
}

func (u *groupUseCase) UpdateGroup(groupID uint, userID uint, req dto.UpdateGroupRequest) (*models.Group, error) {
	role, err := u.groupRepo.GetUserRoleInGroup(groupID, userID)
	if err != nil || role != "ADMIN" {
		return nil, errors.New("chỉ Admin mới có quyền chỉnh sửa thông tin nhóm")
	}
	if !req.EndDate.IsZero() && !req.StartDate.IsZero() && req.EndDate.Before(req.StartDate) {
		return nil, errors.New("ngày kết thúc không được trước ngày bắt đầu")
	}
	return u.groupRepo.UpdateGroup(groupID, req)
}

func (u *groupUseCase) RemoveMember(groupID uint, targetUserID uint, requestingUserID uint) error {
	if targetUserID == requestingUserID {
		return errors.New("không thể tự xóa chính mình khỏi nhóm")
	}
	role, err := u.groupRepo.GetUserRoleInGroup(groupID, requestingUserID)
	if err != nil || role != "ADMIN" {
		return errors.New("chỉ Admin mới có quyền xóa thành viên")
	}
	return u.groupRepo.RemoveMember(groupID, targetUserID)
}

func (u *groupUseCase) DeleteGroup(groupID uint, userID uint) error {
	role, err := u.groupRepo.GetUserRoleInGroup(groupID, userID)
	if err != nil || role != "ADMIN" {
		return errors.New("chỉ Admin mới có quyền xóa nhóm")
	}
	return u.groupRepo.DeleteGroup(groupID)
}
