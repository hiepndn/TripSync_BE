// ws/handler.go — HTTP handler upgrade lên WebSocket
package ws

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Cho phép mọi origin trong dev — production nên check origin cụ thể
	CheckOrigin: func(r *http.Request) bool { return true },
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
