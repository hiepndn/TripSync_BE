package routes

import (
	"tripsync-backend/config"
	"tripsync-backend/controllers"
	"tripsync-backend/middleware"
	"tripsync-backend/repository"

	"github.com/gin-gonic/gin"
)

func AdminRoutes(r *gin.Engine) {
	userRepo := repository.NewUserRepository(config.DB)
	groupRepo := repository.NewGroupRepository(config.DB)
	adminController := controllers.NewAdminController(userRepo, groupRepo)

	// Seed endpoint — không cần auth, bảo vệ bằng seed_key trong body
	r.POST("/api/admin/seed", adminController.SeedAdmin)

	admin := r.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.AdminMiddleware())
	{
		admin.GET("/stats", adminController.GetStats)
		admin.GET("/stats/chart", adminController.GetChartData)
		admin.GET("/stats/growth", adminController.GetGrowthChart)

		// Users
		admin.GET("/users", adminController.GetUsers)
		admin.PUT("/users/:id/role", adminController.UpdateUserRole)
		admin.DELETE("/users/:id", adminController.DeleteUser)

		// Groups
		admin.GET("/groups", adminController.GetGroups)
		admin.DELETE("/groups/:id", adminController.DeleteGroup)
	}
}
