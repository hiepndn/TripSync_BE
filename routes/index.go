package routes

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	AuthRoutes(r)
	GroupRoutes(r)
	ActivityRoutes(r)
	ExpenseRoutes(r)
	ChecklistRoutes(r)
	DocumentRoutes(r)
	AdminRoutes(r)
}
