package repository

import (
	"time"
	"tripsync-backend/models"

	"gorm.io/gorm"
)

type GroupRepository interface {
	CreateGroupWithAdmin(group *models.Group, adminID uint) error
	GetGroupsByUserID(userID uint) ([]models.Group, error)
	GetGroupByInviteCode(code string) (*models.Group, error)
	IsUserInGroup(groupID uint, userID uint) (bool, error)
	AddMember(member *models.GroupMember) error
	GetUserRoleInGroup(groupID uint, userID uint) (string, error)
	GetByID(id uint) (*models.Group, error)
	GetGroupMembers(groupID uint) ([]models.User, error)
	UpdateAIGeneratingStatus(groupID uint, status bool) error
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
	// Dùng JOIN để lấy các nhóm mà user này là thành viên
	err := r.db.Joins("JOIN group_members ON group_members.group_id = groups.id").
		Where("group_members.user_id = ?", userID).
		Find(&groups).Error
	return groups, err
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
