package usecase

import (
	"errors"
	"tripsync-backend/dto"
	"tripsync-backend/models"
	"tripsync-backend/repository"

	"gorm.io/gorm"
)

type DocumentUseCase interface {
	CreateDocument(groupID uint, userID uint, req dto.CreateDocumentRequest) (*models.Document, error)
	GetGroupDocuments(groupID uint) ([]models.Document, error)
	DeleteDocument(docID uint, userID uint, groupID uint) error
}

type documentUseCase struct {
	docRepo   repository.DocumentRepository
	groupRepo repository.GroupRepository
}

func NewDocumentUseCase(docRepo repository.DocumentRepository, groupRepo repository.GroupRepository) DocumentUseCase {
	return &documentUseCase{
		docRepo:   docRepo,
		groupRepo: groupRepo,
	}
}

func (u *documentUseCase) CreateDocument(groupID uint, userID uint, req dto.CreateDocumentRequest) (*models.Document, error) {
	// Kiểm tra user có trong nhóm không
	inGroup, err := u.groupRepo.IsUserInGroup(groupID, userID)
	if err != nil || !inGroup {
		return nil, errors.New("bạn không phải thành viên của nhóm này")
	}

	doc := &models.Document{
		GroupID:      groupID,
		FileName:     req.FileName,
		FileURL:      req.FileURL,
		FileType:     req.FileType,
		FileSize:     req.FileSize,
		Category:     req.Category,
		UploadedByID: userID,
	}

	if err := u.docRepo.Create(doc); err != nil {
		return nil, errors.New("lỗi khi lưu tài liệu: " + err.Error())
	}

	return doc, nil
}

func (u *documentUseCase) GetGroupDocuments(groupID uint) ([]models.Document, error) {
	docs, err := u.docRepo.GetByGroupID(groupID)
	if err != nil {
		return nil, errors.New("lỗi khi lấy danh sách tài liệu: " + err.Error())
	}
	return docs, nil
}

func (u *documentUseCase) DeleteDocument(docID uint, userID uint, groupID uint) error {
	// Lấy thông tin tài liệu
	doc, err := u.docRepo.GetByID(docID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("not found")
		}
		return errors.New("lỗi khi tìm tài liệu: " + err.Error())
	}

	// Kiểm tra quyền: phải là Uploader hoặc ADMIN
	if doc.UploadedByID != userID {
		role, err := u.groupRepo.GetUserRoleInGroup(groupID, userID)
		if err != nil || role != "ADMIN" {
			return errors.New("forbidden")
		}
	}

	return u.docRepo.Delete(docID)
}
