// middleware/rate_limit_middleware.go
//
// Rate Limiting chống DDoS hạng nhẹ dùng Token Bucket (golang.org/x/time/rate).
// Mỗi IP có bucket riêng — không ảnh hưởng lẫn nhau.
// Bucket cũ tự dọn sau 2 phút không dùng để tránh memory leak.
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// entry lưu limiter + thời điểm truy cập cuối để dọn dẹp
type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// ipStore quản lý limiters theo IP, thread-safe
type ipStore struct {
	mu      sync.Mutex
	entries map[string]*entry
	r       rate.Limit // tokens/giây
	b       int        // burst size
}

func newIPStore(r rate.Limit, b int) *ipStore {
	s := &ipStore{
		entries: make(map[string]*entry),
		r:       r,
		b:       b,
	}
	// Goroutine dọn dẹp bucket cũ mỗi 1 phút
	go func() {
		for {
			time.Sleep(time.Minute)
			s.mu.Lock()
			for ip, e := range s.entries {
				if time.Since(e.lastSeen) > 2*time.Minute {
					delete(s.entries, ip)
				}
			}
			s.mu.Unlock()
		}
	}()
	return s
}

func (s *ipStore) getLimiter(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[ip]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(s.r, s.b)}
		s.entries[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter
}

// ============================================================
// Quy tắc 1: Auth — 5 lần / 1 phút mỗi IP
// rate.Every(12s) = 5 tokens/phút, burst = 5
// ============================================================
var authStore = newIPStore(rate.Every(12*time.Second), 5)

// AuthRateLimit giới hạn login/register: 5 request/phút/IP
func AuthRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !authStore.getLimiter(ip).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Bạn đã thử quá nhiều lần. Vui lòng chờ 1 phút rồi thử lại.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ============================================================
// Quy tắc 2: AI Generate — 1 lần / 30 giây mỗi IP
// rate.Every(30s) = 1 token/30s, burst = 1
// ============================================================
var aiStore = newIPStore(rate.Every(30*time.Second), 1)

// AIRateLimit giới hạn generate AI: 1 request/30s/IP
func AIRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !aiStore.getLimiter(ip).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "AI đang xử lý, vui lòng chờ ít nhất 30 giây trước khi thử lại.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
