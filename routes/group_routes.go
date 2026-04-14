package routes

import (
	"tripsync-backend/config"
	"tripsync-backend/controllers"
	"tripsync-backend/middleware"
	"tripsync-backend/repository"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

func GroupRoutes(r *gin.Engine) {
	groupRepo := repository.NewGroupRepository(config.DB)
	activityRepo := repository.NewActivityRepository(config.DB)
	groupUC := usecase.NewGroupUseCase(groupRepo, activityRepo)
	groupController := controllers.NewGroupController(groupUC)

	// Khai báo Routes
	api := r.Group("/api")
	{
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/groups", groupController.CreateGroup)
			protected.GET("/groups", groupController.GetGroups)
			protected.POST("/groups/join", groupController.JoinGroup)
			protected.GET("/groups/:id", groupController.GetDetail)
			protected.POST("/groups/:id/regenerate-ai", groupController.RegenerateAI)
			protected.PUT("/groups/:id", groupController.UpdateGroup)
			protected.DELETE("/groups/:id/members/:user_id", groupController.KickMember)
			protected.DELETE("/groups/:id", groupController.DeleteGroup)
		}
	}
}
