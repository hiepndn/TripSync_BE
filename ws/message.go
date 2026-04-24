// ws/message.go — Định nghĩa các loại message gửi qua WebSocket
package ws

import "encoding/json"

// EventType là loại sự kiện
type EventType string

const (
	// EventAIDone: AI generation hoàn thành thành công
	EventAIDone EventType = "ai_done"
	// EventAIError: AI generation thất bại
	EventAIError EventType = "ai_error"
)

// WSMessage là cấu trúc JSON gửi từ BE → FE
type WSMessage struct {
	Event   EventType `json:"event"`
	GroupID uint      `json:"group_id"`
	Payload any       `json:"payload,omitempty"`
}

// Encode serialize message thành JSON bytes
func (m WSMessage) Encode() []byte {
	b, _ := json.Marshal(m)
	return b
}
