package middleware

import (
	"net/http"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimitConfig struct {
	GlobalPerMinute     int
	APIPerMinute        int
	WeatherPerMinute    int
	DiagnosisPerMinute  int
	TranscribePerMinute int
}

func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalPerMinute:     60,
		APIPerMinute:        20,
		WeatherPerMinute:    30,
		DiagnosisPerMinute:  5,
		TranscribePerMinute: 6,
	}
}

type visitor struct {
	count   int
	resetAt time.Time
}

type visitTracker struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
	window   time.Duration
}

func newVisitTracker(rate int, window time.Duration) *visitTracker {
	vt := &visitTracker{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
	go vt.cleanup()
	return vt
}

func (vt *visitTracker) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		vt.mu.Lock()
		now := time.Now()
		for key, v := range vt.visitors {
			if now.After(v.resetAt) {
				delete(vt.visitors, key)
			}
		}
		vt.mu.Unlock()
	}
}

func (vt *visitTracker) Allow(key string) (bool, int) {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	now := time.Now()
	v, ok := vt.visitors[key]
	if !ok || now.After(v.resetAt) {
		vt.visitors[key] = &visitor{count: 1, resetAt: now.Add(vt.window)}
		return true, vt.rate - 1
	}

	v.count++
	if v.count > vt.rate {
		retryAfter := int(time.Until(v.resetAt).Seconds())
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}
	return true, vt.rate - v.count
}

type multiRateLimiter struct {
	global     *visitTracker
	api        *visitTracker
	weather    *visitTracker
	diagnosis  *visitTracker
	transcribe *visitTracker
}

func newMultiRateLimiter(cfg RateLimitConfig) *multiRateLimiter {
	return &multiRateLimiter{
		global:     newVisitTracker(cfg.GlobalPerMinute, time.Minute),
		api:        newVisitTracker(cfg.APIPerMinute, time.Minute),
		weather:    newVisitTracker(cfg.WeatherPerMinute, time.Minute),
		diagnosis:  newVisitTracker(cfg.DiagnosisPerMinute, time.Minute),
		transcribe: newVisitTracker(cfg.TranscribePerMinute, time.Minute),
	}
}

func (m *multiRateLimiter) pickTracker(c *gin.Context) *visitTracker {
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}

	if path == "/health" {
		return nil
	}

	if strings.HasPrefix(path, "/static") || path == "/favicon.ico" {
		return nil
	}

	if path == "/api/v1/weather" {
		return m.weather
	}

	if path == "/api/v1/ai/transcribe" {
		return m.transcribe
	}

	if path == "/api/v1/diagnoses" && c.Request.Method == "POST" {
		return m.diagnosis
	}

	return m.api
}

func (m *multiRateLimiter) resolveKey(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if exists {
		if id, ok := userID.(string); ok && id != "" {
			return "user:" + id
		}
	}

	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		if parts := strings.Split(forwarded, ","); len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return "ip:" + ip
			}
		}
	}

	return "ip:" + c.ClientIP()
}

func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	limiter := newMultiRateLimiter(cfg)

	return func(c *gin.Context) {
		tracker := limiter.pickTracker(c)
		if tracker == nil {
			c.Next()
			return
		}

		key := limiter.resolveKey(c)

		allowed, retryAfter := tracker.Allow(key)
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			accept := c.GetHeader("Accept")
			if strings.Contains(accept, "text/html") {
				c.HTML(http.StatusTooManyRequests, "error.html", gin.H{
					"Title":              "Too Many Requests",
					"Year":               time.Now().Year(),
					"ErrorMessage":       fmt.Sprintf("Too many requests. Please wait %d seconds and try again.", retryAfter),
					"ErrorCode":          http.StatusTooManyRequests,
				})
				c.Abort()
			} else {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":              "Too many requests. Please wait a moment.",
					"retry_after_seconds": retryAfter,
				})
			}
			return
		}

		c.Next()
	}
}
