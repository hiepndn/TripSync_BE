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
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/groups/:id/activity", activityController.GetActivities)
			protected.POST("/groups/:id/activity", activityController.CreateActivity)
			protected.POST("/groups/:id/activities/:activity_id/vote", activityController.ToggleVote)
			protected.PATCH("/groups/:id/activities/:activity_id/finalize", activityController.FinalizeActivity)
			protected.PUT("/groups/:id/activities/:activity_id", activityController.UpdateActivity)
			protected.DELETE("/groups/:id/activities/:activity_id", activityController.DeleteActivity)
			protected.POST("/groups/:id/activities/:activity_id/rate", activityController.RateActivity)
			protected.GET("/groups/:id/activities/suggestions", activityController.GetSuggestions)
		}
	}
}
