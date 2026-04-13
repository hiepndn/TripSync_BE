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

	api := r.Group("/api")
	{
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/groups/:id/expenses", expenseController.CreateExpense)
			protected.GET("/groups/:id/expenses", expenseController.GetExpenseList)
			protected.GET("/groups/:id/debts", expenseController.GetOptimalDebts)
			protected.POST("/groups/:id/debts/settle", expenseController.SettleDebt)
			protected.GET("/groups/:id/expenses/summary", expenseController.GetExpenseSummary)
		}
	}

}
