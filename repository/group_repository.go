package repository

import (
	"fmt"
	"time"
	"tripsync-backend/dto"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type GroupRepository interface {
	CreateGroupWithAdmin(group *models.Group, adminID uint) error
	GetGroupsByUserID(userID uint) ([]models.Group, error)
	GetGroupsByUserIDWithRole(userID uint) ([]dto.GroupWithRole, error)
	GetGroupByInviteCode(code string) (*models.Group, error)
	IsUserInGroup(groupID uint, userID uint) (bool, error)
	AddMember(member *models.GroupMember) error
	GetUserRoleInGroup(groupID uint, userID uint) (string, error)
	GetByID(id uint) (*models.Group, error)
	GetGroupMembers(groupID uint) ([]models.User, error)
	UpdateAIGeneratingStatus(groupID uint, status bool) error
	SetAIError(groupID uint, errMsg string) error
	UpdateGroup(groupID uint, req dto.UpdateGroupRequest) (*models.Group, error)
	RemoveMember(groupID uint, userID uint) error
	DeleteGroup(groupID uint) error
	UpdateVisibility(groupID uint, isPublic bool) error
	GetPublicGroups() ([]models.Group, error)
	GetPublicGroupDetail(groupID uint) (*models.Group, error)
	// Admin
	GetAllGroups(page, pageSize int, search string) ([]models.Group, int64, error)
	GetTotalGroups() (int64, error)
	AdminDeleteGroup(groupID uint) error
	GetGroupsByDay(days int) ([]map[string]interface{}, error)
	GetGrowthData(period int) ([]map[string]interface{}, error)
}

type groupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

// Dùng Transaction để đảm bảo tạo Group và gán quyền Admin diễn ra đồng thời
func (r *groupRepository) CreateGroupWithAdmin(group *models.Group, adminID uint) error {
	// Bắt đầu 1 transaction
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Lưu Group vào DB (GORM sẽ tự cấp ID mới cho group)
		if err := tx.Create(group).Error; err != nil {
			return err // Lỗi thì rollback
		}

		// 2. Tạo bản ghi GroupMember cho người tạo (Admin)
		adminMember := models.GroupMember{
			GroupID:  group.ID,
			UserID:   adminID,
			Role:     models.RoleAdmin, // Đừng quên import models nhé
			JoinedAt: time.Now(),
		}

		// 3. Lưu Admin vào bảng trung gian
		if err := tx.Create(&adminMember).Error; err != nil {
			return err // Lỗi thì rollback luôn cả cái Group vừa tạo
		}

		return nil // Thành công rực rỡ!
	})
}

func (r *groupRepository) GetGroupsByUserID(userID uint) ([]models.Group, error) {
	var groups []models.Group
	err := r.db.Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ?", userID).
		Find(&groups).Error
	return groups, err
}

func (r *groupRepository) GetGroupsByUserIDWithRole(userID uint) ([]dto.GroupWithRole, error) {
	// Lấy danh sách groups + role
	type rawRow struct {
		models.Group
		Role string `gorm:"column:role"`
	}
	var rawRows []rawRow
	err := r.db.Table("groups").
		Select("groups.*, group_members.role").
		Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ? AND groups.deleted_at IS NULL", userID).
		Scan(&rawRows).Error
	if err != nil {
		return nil, err
	}

	results := make([]dto.GroupWithRole, 0, len(rawRows))
	for _, row := range rawRows {
		// Lấy members thực của nhóm (tối đa 3 + tổng số)
		type memberRow struct {
			ID       uint   `gorm:"column:id"`
			FullName string `gorm:"column:full_name"`
			Avatar   string `gorm:"column:avatar"`
		}
		var members []memberRow
		r.db.Table("users").
			Select("users.id, users.full_name, users.avatar").
			Joins("JOIN group_members gm ON gm.user_id = users.id").
			Where("gm.group_id = ? AND users.deleted_at IS NULL", row.ID).
			Limit(3).
			Scan(&members)

		var totalCount int64
		r.db.Table("group_members").Where("group_id = ?", row.ID).Count(&totalCount)

		previews := make([]dto.MemberPreview, 0, len(members))
		for _, m := range members {
			previews = append(previews, dto.MemberPreview{
				ID:       m.ID,
				FullName: m.FullName,
				Avatar:   m.Avatar,
			})
		}

		results = append(results, dto.GroupWithRole{
			Group:          row.Group,
			Role:           row.Role,
			MemberCount:    int(totalCount),
			MemberPreviews: previews,
		})
	}
	return results, nil
}

func (r *groupRepository) GetGroupByInviteCode(code string) (*models.Group, error) {
	var group models.Group
	err := r.db.Where("invite_code = ?", code).First(&group).Error
	return &group, err
}

func (r *groupRepository) IsUserInGroup(groupID uint, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Count(&count).Error
	return count > 0, err
}

func (r *groupRepository) AddMember(member *models.GroupMember) error {
	return r.db.Create(member).Error
}

func (r *groupRepository) GetUserRoleInGroup(groupID uint, userID uint) (string, error) {
	var member models.GroupMember
	// Truy vấn vào bảng group_members theo 2 khóa chính
	err := r.db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error
	if err != nil {
		return "", err // Nếu không thấy -> Không trong nhóm
	}
	return string(member.Role), nil // Trả về "ADMIN" hoặc "MEMBER"
}

func (r *groupRepository) GetByID(id uint) (*models.Group, error) {
	var group models.Group
	// Preload("Creator") nếu bác muốn lấy luôn thông tin người tạo nhóm lên
	err := r.db.First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) GetGroupMembers(groupID uint) ([]models.User, error) {
	var members []models.User
	// Dùng Raw Query Builder của GORM để JOIN 2 bảng
	err := r.db.Table("users").
		Select("users.id, users.full_name, users.email, users.avatar, group_members.role").
		Joins("INNER JOIN group_members ON group_members.user_id = users.id").
		Where("group_members.group_id = ?", groupID).
		Scan(&members).Error

	if err != nil {
		return nil, err
	}
	return members, nil
}

func (r *groupRepository) UpdateAIGeneratingStatus(groupID uint, status bool) error {
	return r.db.Model(&models.Group{}).
		Where("id = ?", groupID).
		Update("is_ai_generating", status).Error
}

// SetAIError lưu thông báo lỗi Gemini vào DB để FE có thể đọc qua polling
func (r *groupRepository) SetAIError(groupID uint, errMsg string) error {
	return r.db.Model(&models.Group{}).
		Where("id = ?", groupID).
		Updates(map[string]interface{}{
			"ai_error":         errMsg,
			"is_ai_generating": false,
		}).Error
}

func (r *groupRepository) UpdateGroup(groupID uint, req dto.UpdateGroupRequest) (*models.Group, error) {
	updates := map[string]interface{}{
		"name":               req.Name,
		"description":        req.Description,
		"start_date":         req.StartDate,
		"end_date":           req.EndDate,
		"departure_location": req.DepartureLocation,
		"route_destinations": req.RouteDestinations,
	}
	if err := r.db.Model(&models.Group{}).Where("id = ?", groupID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.GetByID(groupID)
}

func (r *groupRepository) RemoveMember(groupID uint, userID uint) error {
	result := r.db.Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&models.GroupMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *groupRepository) DeleteGroup(groupID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Group{}, groupID).Error
	})
}

func (r *groupRepository) UpdateVisibility(groupID uint, isPublic bool) error {
	return r.db.Model(&models.Group{}).
		Where("id = ?", groupID).
		Update("is_public", isPublic).Error
}

func (r *groupRepository) GetPublicGroups() ([]models.Group, error) {
	var groups []models.Group
	err := r.db.Where("is_public = ?", true).Find(&groups).Error
	return groups, err
}

func (r *groupRepository) GetPublicGroupDetail(groupID uint) (*models.Group, error) {
	var group models.Group
	err := r.db.Where("id = ? AND is_public = ?", groupID, true).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) GetAllGroups(page, pageSize int, search string) ([]models.Group, int64, error) {
	var groups []models.Group
	var total int64

	query := r.db.Model(&models.Group{})
	if search != "" {
		query = query.Where("name ILIKE ? OR route_destinations ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)
	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&groups).Error

	return groups, total, err
}

func (r *groupRepository) GetTotalGroups() (int64, error) {
	var count int64
	err := r.db.Model(&models.Group{}).Count(&count).Error
	return count, err
}

func (r *groupRepository) AdminDeleteGroup(groupID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Junction tables không có DeletedAt → hard delete

		// 1. Activity votes & ratings
		if err := tx.Exec("DELETE FROM activity_votes WHERE activity_id IN (SELECT id FROM activities WHERE group_id = ?)", groupID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM activity_ratings WHERE activity_id IN (SELECT id FROM activities WHERE group_id = ?)", groupID).Error; err != nil {
			return err
		}

		// 2. Soft delete activities (có BaseModel → có DeletedAt)
		if err := tx.Where("group_id = ?", groupID).Delete(&models.Activity{}).Error; err != nil {
			return err
		}

		// 3. Expense splits
		if err := tx.Exec("DELETE FROM expense_splits WHERE expense_id IN (SELECT id FROM expenses WHERE group_id = ?)", groupID).Error; err != nil {
			return err
		}

		// 4. Soft delete expenses
		if err := tx.Where("group_id = ?", groupID).Delete(&models.Expense{}).Error; err != nil {
			return err
		}

		// 5. Soft delete checklists
		if err := tx.Where("group_id = ?", groupID).Delete(&models.ChecklistItem{}).Error; err != nil {
			return err
		}

		// 6. Soft delete documents
		if err := tx.Where("group_id = ?", groupID).Delete(&models.Document{}).Error; err != nil {
			return err
		}

		// 7. Group favorites (không có DeletedAt → hard delete)
		if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupFavorite{}).Error; err != nil {
			return err
		}

		// 8. Group members (không có DeletedAt → hard delete)
		if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMember{}).Error; err != nil {
			return err
		}

		// 9. Soft delete group
		return tx.Delete(&models.Group{}, groupID).Error
	})
}

func (r *groupRepository) GetGroupsByDay(days int) ([]map[string]interface{}, error) {
	type result struct {
		Day   string `gorm:"column:day"`
		Count int    `gorm:"column:count"`
	}
	var rows []result
	err := r.db.Raw(`
		SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'DD/MM') AS day, COUNT(*) AS count
		FROM groups
		WHERE created_at >= NOW() - INTERVAL '? days' AND deleted_at IS NULL
		GROUP BY day
		ORDER BY MIN(created_at)
	`, days).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(rows))
	for i, r := range rows {
		out[i] = map[string]interface{}{"day": r.Day, "count": r.Count}
	}
	return out, nil
}

func (r *groupRepository) GetGrowthData(period int) ([]map[string]interface{}, error) {
	type row struct {
		Label string  `gorm:"column:label"`
		Total float64 `gorm:"column:total"`
	}
	var rows []row

	var query string
	switch {
	case period == 1:
		query = `
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'HH24:00') AS label,
			       COUNT(*) AS total
			FROM groups
			WHERE DATE(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh') = CURRENT_DATE
			  AND deleted_at IS NULL
			GROUP BY label ORDER BY label`
	case period <= 90:
		query = fmt.Sprintf(`
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'DD/MM') AS label,
			       COUNT(*) AS total
			FROM groups
			WHERE created_at >= NOW() - INTERVAL '%d days' AND deleted_at IS NULL
			GROUP BY label ORDER BY MIN(created_at)`, period)
	case period == 0:
		query = `
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'MM/YYYY') AS label,
			       COUNT(*) AS total
			FROM groups
			WHERE deleted_at IS NULL
			GROUP BY label ORDER BY MIN(created_at)`
	default:
		query = fmt.Sprintf(`
			SELECT TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'MM/YYYY') AS label,
			       COUNT(*) AS total
			FROM groups
			WHERE created_at >= NOW() - INTERVAL '%d days' AND deleted_at IS NULL
			GROUP BY label ORDER BY MIN(created_at)`, period)
	}

	if err := r.db.Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]map[string]interface{}, len(rows))
	var cumulative float64
	for i, row := range rows {
		cumulative += row.Total
		out[i] = map[string]interface{}{
			"label": row.Label,
			"count": row.Total,
			"total": cumulative,
		}
	}
	return out, nil
}
