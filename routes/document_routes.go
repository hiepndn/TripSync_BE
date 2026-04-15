package routes

import (
	"tripsync-backend/config"
	"tripsync-backend/controllers"
	"tripsync-backend/middleware"
	"tripsync-backend/repository"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

func DocumentRoutes(r *gin.Engine) {
	docRepo := repository.NewDocumentRepository(config.DB)
	groupRepo := repository.NewGroupRepository(config.DB)
	docUC := usecase.NewDocumentUseCase(docRepo, groupRepo)
	docController := controllers.NewDocumentController(docUC)

	api := r.Group("/api")
	{
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/groups/:id/documents", docController.GetDocuments)
			protected.POST("/groups/:id/documents", docController.CreateDocument)
			protected.DELETE("/groups/:id/documents/:doc_id", docController.DeleteDocument)
		}
	}
}
