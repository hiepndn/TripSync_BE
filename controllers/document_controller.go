package controllers

import (
	"net/http"
	"strconv"
	"tripsync-backend/dto"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

type DocumentController struct {
	docUC usecase.DocumentUseCase
}

func NewDocumentController(uc usecase.DocumentUseCase) *DocumentController {
	return &DocumentController{docUC: uc}
}

// GET /groups/:id/documents
func (c *DocumentController) GetDocuments(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	docs, err := c.docUC.GetGroupDocuments(uint(groupID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Lấy danh sách tài liệu thành công",
		"data":    docs,
	})
}

// POST /groups/:id/documents
func (c *DocumentController) CreateDocument(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := uint(userIDVal.(float64))

	var req dto.CreateDocumentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	doc, err := c.docUC.CreateDocument(uint(groupID), userID, req)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Tải tài liệu lên thành công",
		"data":    doc,
	})
}

// DELETE /groups/:id/documents/:doc_id
func (c *DocumentController) DeleteDocument(ctx *gin.Context) {
	groupIDStr := ctx.Param("id")
	docIDStr := ctx.Param("doc_id")

	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
		return
	}

	docID, err := strconv.Atoi(docIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID tài liệu không hợp lệ"})
		return
	}

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Không tìm thấy thông tin user"})
		return
	}
	userID := uint(userIDVal.(float64))

	err = c.docUC.DeleteDocument(uint(docID), userID, uint(groupID))
	if err != nil {
		switch err.Error() {
		case "not found":
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Tài liệu không tồn tại"})
		case "forbidden":
			ctx.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền xóa tài liệu này"})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Xóa tài liệu thành công",
	})
}
