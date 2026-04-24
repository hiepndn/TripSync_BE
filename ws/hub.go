// ws/hub.go
//
// WebSocket Hub quản lý tất cả kết nối theo groupID.
// Khi AI generation xong, Goroutine gọi Hub.Broadcast(groupID, message)
// để "bắn pháo hiệu" tới tất cả client đang xem nhóm đó.
package ws

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

// Client đại diện cho 1 kết nối WebSocket của 1 user đang xem 1 nhóm
type Client struct {
	GroupID uint
	Conn    *websocket.Conn
	Send    chan []byte // Channel để gửi message không blocking
}

// Hub quản lý toàn bộ clients, nhóm theo groupID
type Hub struct {
	mu      sync.RWMutex
	clients map[uint]map[*Client]struct{} // groupID → set of clients
}

// Global hub instance — dùng chung toàn app
var GlobalHub = &Hub{
	clients: make(map[uint]map[*Client]struct{}),
}

// Register đăng ký 1 client mới vào hub
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.GroupID] == nil {
		h.clients[c.GroupID] = make(map[*Client]struct{})
	}
	h.clients[c.GroupID][c] = struct{}{}
	fmt.Printf("🔌 [WS] Client kết nối nhóm %d — tổng: %d\n", c.GroupID, len(h.clients[c.GroupID]))
}

// Unregister xóa client khỏi hub và đóng channel
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if group, ok := h.clients[c.GroupID]; ok {
		delete(group, c)
		close(c.Send)
		if len(group) == 0 {
			delete(h.clients, c.GroupID)
		}
	}
	fmt.Printf("🔌 [WS] Client ngắt kết nối nhóm %d\n", c.GroupID)
}

// Broadcast gửi message tới tất cả client đang xem groupID.
// Gọi từ Goroutine AI sau khi generation xong.
func (h *Hub) Broadcast(groupID uint, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	group, ok := h.clients[groupID]
	if !ok {
		return
	}
	for client := range group {
		select {
		case client.Send <- message:
		default:
			// Channel đầy (client chậm) → bỏ qua, không block
		}
	}
	fmt.Printf("📡 [WS] Broadcast nhóm %d → %d clients: %s\n", groupID, len(group), string(message))
}

// WritePump đọc từ Send channel và ghi ra WebSocket connection.
// Chạy trong goroutine riêng cho mỗi client.
func (c *Client) WritePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

// ReadPump đọc message từ client (chủ yếu để detect disconnect).
// Chạy trong goroutine riêng cho mỗi client.
func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.Unregister(c)
		c.Conn.Close()
	}()
	for {
		// Chỉ cần đọc để detect close frame — không xử lý message từ client
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			break
		}
	}
}
