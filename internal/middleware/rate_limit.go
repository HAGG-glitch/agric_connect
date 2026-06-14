package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
}

type visitor struct {
	count   int
	resetAt time.Time
}

func RateLimit(requestsPerMinute int) gin.HandlerFunc {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     requestsPerMinute,
	}

	go func() {
		for range time.Tick(5 * time.Minute) {
			rl.mu.Lock()
			now := time.Now()
			for ip, v := range rl.visitors {
				if now.After(v.resetAt) {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		rate := rl.rate
		// Stricter limits for diagnosis and transcription endpoints
		if c.FullPath() == "/api/v1/diagnoses" && c.Request.Method == "POST" {
			rate = 5
		} else if c.FullPath() == "/api/v1/ai/transcribe" {
			rate = 6
		}

		ip := c.ClientIP()
		rl.mu.Lock()
		v, ok := rl.visitors[ip]
		if !ok || time.Now().After(v.resetAt) {
			rl.visitors[ip] = &visitor{count: 1, resetAt: time.Now().Add(time.Minute)}
			rl.mu.Unlock()
			c.Next()
			return
		}
		v.count++
		if v.count > rate {
			rl.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please wait a moment.",
			})
			return
		}
		rl.mu.Unlock()
		c.Next()
	}
}
