package routes

import (
	"tripsync-backend/config"
	"tripsync-backend/controllers"
	"tripsync-backend/middleware"
	"tripsync-backend/repository"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

func ChecklistRoutes(r *gin.Engine) {
	checkRepo := repository.NewChecklistRepository(config.DB)
	checkUC := usecase.NewChecklistUseCase(checkRepo)

	checkController := controllers.NewChecklistController(checkUC)
	checklistGroup := r.Group("/api/groups/:id/checklist")
	checklistGroup.Use(middleware.AuthMiddleware())
	{
		checklistGroup.POST("/", checkController.CreateItem)
		checklistGroup.GET("/", checkController.GetItems)
		checklistGroup.PATCH("/:itemId/toggle", checkController.ToggleComplete)
		checklistGroup.PATCH("/:itemId/assign", checkController.AssignMember)
		checklistGroup.DELETE("/:itemId", checkController.DeleteItem)
	}
}
