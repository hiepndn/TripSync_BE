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
	expenseRepo := repository.NewExpenseRepository(config.DB)
	groupUC := usecase.NewGroupUseCase(groupRepo, activityRepo, expenseRepo)
	groupController := controllers.NewGroupController(groupUC)

	favoriteRepo := repository.NewFavoriteRepository(config.DB)
	favoriteUC := usecase.NewFavoriteUseCase(favoriteRepo)
	favoriteController := controllers.NewFavoriteController(favoriteUC)

	// Khai báo Routes
	api := r.Group("/api")
	{
		// Public routes (không cần auth)
		api.GET("/groups/public", groupController.GetPublicGroups)
		api.GET("/groups/public/:id", groupController.GetPublicGroupDetail)

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// Routes không cần check membership
			protected.POST("/groups", groupController.CreateGroup)
			protected.GET("/groups", groupController.GetGroups)
			protected.POST("/groups/join", groupController.JoinGroup)
			protected.GET("/favorites", favoriteController.GetFavorites)

			// Routes yêu cầu phải là thành viên của nhóm
			groupRoutes := protected.Group("/groups/:id")
			groupRoutes.Use(middleware.GroupMembershipMiddleware(groupRepo))
			{
				groupRoutes.GET("", groupController.GetDetail)
				groupRoutes.POST("/regenerate-ai", groupController.RegenerateAI)
				groupRoutes.PUT("", groupController.UpdateGroup)
				groupRoutes.PUT("/visibility", groupController.UpdateVisibility)
				groupRoutes.DELETE("/members/:user_id", groupController.KickMember)
				groupRoutes.DELETE("", groupController.DeleteGroup)
				groupRoutes.POST("/favorite", favoriteController.ToggleFavorite)
			}
		}
	}
}
