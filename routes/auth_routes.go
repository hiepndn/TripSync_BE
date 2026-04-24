package routes

import (
	"tripsync-backend/config"
	"tripsync-backend/controllers"
	"tripsync-backend/middleware"
	"tripsync-backend/repository"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(r *gin.Engine) {
	// 1. Lắp ráp các layer lại với nhau
	userRepo := repository.NewUserRepository(config.DB)
	authUC := usecase.NewAuthUseCase(userRepo)

	// 2. Bơm UseCase vào cho Controller
	authController := controllers.NewAuthController(authUC)

	// 3. Đăng ký đường dẫn
	authGroup := r.Group("/api/auth")
	{
		// Rate limit: 5 lần/phút/IP cho login và register
		authGroup.POST("/login", middleware.AuthRateLimit(), authController.Login)
		authGroup.POST("/register", middleware.AuthRateLimit(), authController.Register)
		protected := authGroup.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/me", authController.GetMe)
			protected.PUT("/me", authController.UpdateProfile)
			protected.PUT("/me/password", authController.ChangePassword)
		}
	}

}
