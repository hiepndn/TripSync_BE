package routes

import (
	"tripsync-backend/ws"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// WebSocket endpoint — không cần auth middleware vì chỉ nhận event
	r.GET("/ws/groups/:id", ws.HandleGroupWS)

	AuthRoutes(r)
	GroupRoutes(r)
	ActivityRoutes(r)
	ExpenseRoutes(r)
	ChecklistRoutes(r)
	DocumentRoutes(r)
	AdminRoutes(r)
}
