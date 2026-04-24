package repository

import (
	"context"
	"fmt"
	"time"
	"tripsync-backend/dto"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type GroupRepository interface {
	CreateGroupWithAdmin(ctx context.Context, group *models.Group, adminID uint) error
	GetGroupsByUserID(ctx context.Context, userID uint) ([]models.Group, error)
	GetGroupsByUserIDWithRole(ctx context.Context, userID uint) ([]dto.GroupWithRole, error)
	GetGroupByInviteCode(ctx context.Context, code string) (*models.Group, error)
	IsUserInGroup(ctx context.Context, groupID uint, userID uint) (bool, error)
	AddMember(ctx context.Context, member *models.GroupMember) error
	GetUserRoleInGroup(ctx context.Context, groupID uint, userID uint) (string, error)
	GetByID(ctx context.Context, id uint) (*models.Group, error)
	GetGroupMembers(ctx context.Context, groupID uint) ([]models.User, error)
	UpdateAIGeneratingStatus(ctx context.Context, groupID uint, status bool) error
	SetAIError(ctx context.Context, groupID uint, errMsg string) error
	UpdateGroup(ctx context.Context, groupID uint, req dto.UpdateGroupRequest) (*models.Group, error)
	RemoveMember(ctx context.Context, groupID uint, userID uint) error
	DeleteGroup(ctx context.Context, groupID uint) error
	UpdateVisibility(ctx context.Context, groupID uint, isPublic bool) error
	GetPublicGroups(ctx context.Context) ([]models.Group, error)
	GetPublicGroupDetail(ctx context.Context, groupID uint) (*models.Group, error)
	// InviteCodeExists kiểm tra mã mời đã tồn tại chưa (dùng khi sinh mã để tránh trùng)
	InviteCodeExists(ctx context.Context, inviteCode string) (bool, error)
	// Admin
	GetAllGroups(ctx context.Context, page, pageSize int, search string) ([]models.Group, int64, error)
	GetTotalGroups(ctx context.Context) (int64, error)
	AdminDeleteGroup(ctx context.Context, groupID uint) error
	GetGroupsByDay(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetGrowthData(ctx context.Context, period int) ([]map[string]interface{}, error)
}

type groupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

// Dùng Transaction để đảm bảo tạo Group và gán quyền Admin diễn ra đồng thời
func (r *groupRepository) CreateGroupWithAdmin(ctx context.Context, group *models.Group, adminID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}

		adminMember := models.GroupMember{
			GroupID:  group.ID,
			UserID:   adminID,
			Role:     models.RoleAdmin,
			JoinedAt: time.Now(),
		}

		if err := tx.Create(&adminMember).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *groupRepository) GetGroupsByUserID(ctx context.Context, userID uint) ([]models.Group, error) {
	var groups []models.Group
	err := r.db.WithContext(ctx).Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ?", userID).
		Find(&groups).Error
	return groups, err
}

func (r *groupRepository) GetGroupsByUserIDWithRole(ctx context.Context, userID uint) ([]dto.GroupWithRole, error) {
	type rawRow struct {
		models.Group
		Role string `gorm:"column:role"`
	}
	var rawRows []rawRow
	err := r.db.WithContext(ctx).Table("groups").
		Select("groups.*, group_members.role").
		Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ? AND groups.deleted_at IS NULL", userID).
		Scan(&rawRows).Error
	if err != nil {
		return nil, err
	}
	if len(rawRows) == 0 {
		return []dto.GroupWithRole{}, nil
	}

	groupIDs := make([]uint, 0, len(rawRows))
	for _, row := range rawRows {
		groupIDs = append(groupIDs, row.ID)
	}

	type memberAggRow struct {
		GroupID  uint   `gorm:"column:group_id"`
		UserID   uint   `gorm:"column:user_id"`
		FullName string `gorm:"column:full_name"`
		Avatar   string `gorm:"column:avatar"`
		RowNum   int    `gorm:"column:row_num"`
		Total    int    `gorm:"column:total"`
	}
	var memberRows []memberAggRow
	err = r.db.WithContext(ctx).Raw(`
		SELECT
			gm.group_id,
			u.id        AS user_id,
			u.full_name,
			u.avatar,
			ROW_NUMBER() OVER (PARTITION BY gm.group_id ORDER BY gm.joined_at ASC) AS row_num,
			COUNT(*)    OVER (PARTITION BY gm.group_id) AS total
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id AND u.deleted_at IS NULL
		WHERE gm.group_id IN ?
	`, groupIDs).Scan(&memberRows).Error
	if err != nil {
		return nil, err
	}

	type groupMeta struct {
		total    int
		previews []dto.MemberPreview
	}
	metaMap := make(map[uint]*groupMeta, len(groupIDs))
	for _, m := range memberRows {
		meta, ok := metaMap[m.GroupID]
		if !ok {
			meta = &groupMeta{}
			metaMap[m.GroupID] = meta
		}
		meta.total = m.Total
		if m.RowNum <= 3 {
			meta.previews = append(meta.previews, dto.MemberPreview{
				ID:       m.UserID,
				FullName: m.FullName,
				Avatar:   m.Avatar,
			})
		}
	}

	results := make([]dto.GroupWithRole, 0, len(rawRows))
	for _, row := range rawRows {
		meta := metaMap[row.ID]
		var count int
		var previews []dto.MemberPreview
		if meta != nil {
			count = meta.total
			previews = meta.previews
		}
		results = append(results, dto.GroupWithRole{
			Group:          row.Group,
			Role:           row.Role,
			MemberCount:    count,
			MemberPreviews: previews,
		})
	}
	return results, nil
}

func (r *groupRepository) GetGroupByInviteCode(ctx context.Context, code string) (*models.Group, error) {
	var group models.Group
	err := r.db.WithContext(ctx).Where("invite_code = ?", code).First(&group).Error
	return &group, err
}

func (r *groupRepository) IsUserInGroup(ctx context.Context, groupID uint, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Count(&count).Error
	return count > 0, err
}

func (r *groupRepository) AddMember(ctx context.Context, member *models.GroupMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *groupRepository) GetUserRoleInGroup(ctx context.Context, groupID uint, userID uint) (string, error) {
	var member models.GroupMember
	err := r.db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error
	if err != nil {
		return "", err
	}
	return string(member.Role), nil
}

func (r *groupRepository) GetByID(ctx context.Context, id uint) (*models.Group, error) {
	var group models.Group
	err := r.db.WithContext(ctx).First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) GetGroupMembers(ctx context.Context, groupID uint) ([]models.User, error) {
	var members []models.User
	err := r.db.WithContext(ctx).Table("users").
		Select("users.id, users.full_name, users.email, users.avatar, group_members.role").
		Joins("INNER JOIN group_members ON group_members.user_id = users.id").
		Where("group_members.group_id = ?", groupID).
		Scan(&members).Error

	if err != nil {
		return nil, err
	}
	return members, nil
}

func (r *groupRepository) UpdateAIGeneratingStatus(ctx context.Context, groupID uint, status bool) error {
	return r.db.WithContext(ctx).Model(&models.Group{}).
		Where("id = ?", groupID).
		Update("is_ai_generating", status).Error
}

func (r *groupRepository) SetAIError(ctx context.Context, groupID uint, errMsg string) error {
	return r.db.WithContext(ctx).Model(&models.Group{}).
		Where("id = ?", groupID).
		Updates(map[string]interface{}{
			"ai_error":         errMsg,
			"is_ai_generating": false,
		}).Error
}

func (r *groupRepository) UpdateGroup(ctx context.Context, groupID uint, req dto.UpdateGroupRequest) (*models.Group, error) {
	updates := map[string]interface{}{
		"name":               req.Name,
		"description":        req.Description,
		"start_date":         req.StartDate,
		"end_date":           req.EndDate,
		"departure_location": req.DepartureLocation,
		"route_destinations": req.RouteDestinations,
	}
	if err := r.db.WithContext(ctx).Model(&models.Group{}).Where("id = ?", groupID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.GetByID(ctx, groupID)
}

func (r *groupRepository) RemoveMember(ctx context.Context, groupID uint, userID uint) error {
	result := r.db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&models.GroupMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *groupRepository) DeleteGroup(ctx context.Context, groupID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Group{}, groupID).Error
	})
}

func (r *groupRepository) UpdateVisibility(ctx context.Context, groupID uint, isPublic bool) error {
	return r.db.WithContext(ctx).Model(&models.Group{}).
		Where("id = ?", groupID).
		Update("is_public", isPublic).Error
}

func (r *groupRepository) GetPublicGroups(ctx context.Context) ([]models.Group, error) {
	var groups []models.Group
	err := r.db.WithContext(ctx).Where("is_public = ?", true).Find(&groups).Error
	return groups, err
}

func (r *groupRepository) GetPublicGroupDetail(ctx context.Context, groupID uint) (*models.Group, error) {
	var group models.Group
	err := r.db.WithContext(ctx).Where("id = ? AND is_public = ?", groupID, true).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) GetAllGroups(ctx context.Context, page, pageSize int, search string) ([]models.Group, int64, error) {
	var groups []models.Group
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Group{})
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

func (r *groupRepository) GetTotalGroups(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Group{}).Count(&count).Error
	return count, err
}

func (r *groupRepository) AdminDeleteGroup(ctx context.Context, groupID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM activity_votes WHERE activity_id IN (SELECT id FROM activities WHERE group_id = ?)", groupID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM activity_ratings WHERE activity_id IN (SELECT id FROM activities WHERE group_id = ?)", groupID).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", groupID).Delete(&models.Activity{}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM expense_splits WHERE expense_id IN (SELECT id FROM expenses WHERE group_id = ?)", groupID).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", groupID).Delete(&models.Expense{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", groupID).Delete(&models.ChecklistItem{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", groupID).Delete(&models.Document{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupFavorite{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Group{}, groupID).Error
	})
}

func (r *groupRepository) GetGroupsByDay(ctx context.Context, days int) ([]map[string]interface{}, error) {
	type result struct {
		Day   string `gorm:"column:day"`
		Count int    `gorm:"column:count"`
	}
	var rows []result
	err := r.db.WithContext(ctx).Raw(`
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

func (r *groupRepository) GetGrowthData(ctx context.Context, period int) ([]map[string]interface{}, error) {
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

	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
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

func (r *groupRepository) InviteCodeExists(ctx context.Context, inviteCode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Group{}).
		Where("invite_code = ?", inviteCode).
		Count(&count).Error
	return count > 0, err
}
