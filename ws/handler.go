// ws/handler.go — HTTP handler upgrade lên WebSocket
package ws

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// allowedOrigins đọc từ FRONTEND_URL env (giống CORS config trong main.go)
func isOriginAllowed(origin string) bool {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	for _, allowed := range strings.Split(frontendURL, ",") {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	return false
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Chỉ cho phép origin khớp với FRONTEND_URL — chặn request giả mạo
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return isOriginAllowed(origin)
	},
}

// HandleGroupWS là Gin handler cho route: GET /ws/groups/:id
// FE kết nối vào đây để nhận real-time event của nhóm
func HandleGroupWS(c *gin.Context) {
	groupIDStr := c.Param("id")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group ID không hợp lệ"})
		return
	}

	// Upgrade HTTP → WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &Client{
		GroupID: uint(groupID),
		Conn:    conn,
		Send:    make(chan []byte, 32), // Buffer 32 messages
	}

	GlobalHub.Register(client)

	// Chạy 2 goroutine: 1 đọc (detect disconnect), 1 ghi (nhận broadcast)
	go client.ReadPump(GlobalHub)
	go client.WritePump()
}
