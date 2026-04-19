package routes

import (
	"tripsync-backend/config"
	"tripsync-backend/controllers"
	"tripsync-backend/middleware"
	"tripsync-backend/repository"
	"tripsync-backend/usecase"

	"github.com/gin-gonic/gin"
)

func ExpenseRoutes(r *gin.Engine) {
	expenseRepo := repository.NewExpenseRepository(config.DB)
	expenseUC := usecase.NewExpenseUseCase(expenseRepo)
	expenseController := controllers.NewExpenseController(expenseUC)
	groupRepo := repository.NewGroupRepository(config.DB)

	api := r.Group("/api")
	{
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			groupRoutes := protected.Group("/groups/:id")
			groupRoutes.Use(middleware.GroupMembershipMiddleware(groupRepo))
			{
				groupRoutes.POST("/expenses", expenseController.CreateExpense)
				groupRoutes.GET("/expenses", expenseController.GetExpenseList)
				groupRoutes.GET("/debts", expenseController.GetOptimalDebts)
				groupRoutes.POST("/debts/settle", expenseController.SettleDebt)
				groupRoutes.GET("/expenses/summary", expenseController.GetExpenseSummary)
			}
		}
	}
}
