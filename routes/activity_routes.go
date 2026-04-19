package routes

import (
	"tripsync-backend/config"
	"tripsync-backend/controllers"
	"tripsync-backend/middleware"
	"tripsync-backend/repository"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

func ActivityRoutes(r *gin.Engine) {
	activityRepo := repository.NewActivityRepository(config.DB)
	groupRepo := repository.NewGroupRepository(config.DB)
	activityUC := usecase.NewActivityUseCase(activityRepo, groupRepo)
	activityController := controllers.NewActivityController(activityUC)

	// Khai báo Routes
	api := r.Group("/api")
	{
		// Public routes (no auth required)
		api.GET("/groups/:id/export", activityController.ExportActivities)

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// Routes yêu cầu phải là thành viên của nhóm
			groupRoutes := protected.Group("/groups/:id")
			groupRoutes.Use(middleware.GroupMembershipMiddleware(groupRepo))
			{
				groupRoutes.GET("/activity", activityController.GetActivities)
				groupRoutes.POST("/activity", activityController.CreateActivity)
				groupRoutes.POST("/activities/:activity_id/vote", activityController.ToggleVote)
				groupRoutes.PATCH("/activities/:activity_id/finalize", activityController.FinalizeActivity)
				groupRoutes.PUT("/activities/:activity_id", activityController.UpdateActivity)
				groupRoutes.DELETE("/activities/:activity_id", activityController.DeleteActivity)
				groupRoutes.DELETE("/activities", activityController.DeleteAllActivities)
				groupRoutes.POST("/activities/:activity_id/rate", activityController.RateActivity)
				groupRoutes.GET("/activities/suggestions", activityController.GetSuggestions)
				groupRoutes.POST("/import", activityController.ImportActivities)
				groupRoutes.POST("/import-json", activityController.ImportFromJSON)
			}
		}
	}
}
