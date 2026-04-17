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
			protected.POST("/groups", groupController.CreateGroup)
			protected.GET("/groups", groupController.GetGroups)
			protected.POST("/groups/join", groupController.JoinGroup)
			protected.GET("/groups/:id", groupController.GetDetail)
			protected.POST("/groups/:id/regenerate-ai", groupController.RegenerateAI)
			protected.PUT("/groups/:id", groupController.UpdateGroup)
			protected.PUT("/groups/:id/visibility", groupController.UpdateVisibility)
			protected.DELETE("/groups/:id/members/:user_id", groupController.KickMember)
			protected.DELETE("/groups/:id", groupController.DeleteGroup)

			// Favorites
			protected.POST("/groups/:id/favorite", favoriteController.ToggleFavorite)
			protected.GET("/favorites", favoriteController.GetFavorites)
		}
	}
}
