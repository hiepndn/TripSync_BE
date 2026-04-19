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
			groupRoutes := protected.Group("/groups/:id")
			groupRoutes.Use(middleware.GroupMembershipMiddleware(groupRepo))
			{
				groupRoutes.GET("/documents", docController.GetDocuments)
				groupRoutes.POST("/documents", docController.CreateDocument)
				groupRoutes.DELETE("/documents/:doc_id", docController.DeleteDocument)
			}
		}
	}
}
