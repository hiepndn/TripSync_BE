package usecase

import (
	"context"
	"errors"
	"tripsync-backend/dto"
	"tripsync-backend/models"
	"tripsync-backend/repository"

	"gorm.io/gorm"
)

type DocumentUseCase interface {
	CreateDocument(ctx context.Context, groupID uint, userID uint, req dto.CreateDocumentRequest) (*models.Document, error)
	GetGroupDocuments(ctx context.Context, groupID uint) ([]models.Document, error)
	DeleteDocument(ctx context.Context, docID uint, userID uint, groupID uint) error
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

func (u *documentUseCase) CreateDocument(ctx context.Context, groupID uint, userID uint, req dto.CreateDocumentRequest) (*models.Document, error) {
	inGroup, err := u.groupRepo.IsUserInGroup(ctx, groupID, userID)
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

	if err := u.docRepo.Create(ctx, doc); err != nil {
		return nil, errors.New("lỗi khi lưu tài liệu: " + err.Error())
	}

	return doc, nil
}

func (u *documentUseCase) GetGroupDocuments(ctx context.Context, groupID uint) ([]models.Document, error) {
	docs, err := u.docRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return nil, errors.New("lỗi khi lấy danh sách tài liệu: " + err.Error())
	}
	return docs, nil
}

func (u *documentUseCase) DeleteDocument(ctx context.Context, docID uint, userID uint, groupID uint) error {
	doc, err := u.docRepo.GetByID(ctx, docID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("not found")
		}
		return errors.New("lỗi khi tìm tài liệu: " + err.Error())
	}

	if doc.UploadedByID != userID {
		role, err := u.groupRepo.GetUserRoleInGroup(ctx, groupID, userID)
		if err != nil || role != "ADMIN" {
			return errors.New("forbidden")
		}
	}

	return u.docRepo.Delete(ctx, docID)
}
