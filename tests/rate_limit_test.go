package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agriconnect-ai/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRateLimit_HealthExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := middleware.DefaultRateLimitConfig()
	cfg.WeatherPerMinute = 1

	r := gin.New()
	r.Use(middleware.RateLimit(cfg))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/api/v1/weather", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.JSON(http.StatusOK, gin.H{"data": "weather"})
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("health request %d: expected 200, got %d", i, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/weather", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusTooManyRequests {
		t.Fatal("weather should not be rate-limited after health checks")
	}
}

func TestRateLimit_StaticExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := middleware.DefaultRateLimitConfig()
	r := gin.New()
	r.Use(middleware.RateLimit(cfg))
	r.GET("/static/css/app.css", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/static/css/app.css", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatal("static files should not be rate-limited")
		}
	}

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/favicon.ico", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatal("favicon should not be rate-limited")
		}
	}
}

func TestRateLimit_WeatherSeparateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := middleware.DefaultRateLimitConfig()
	cfg.WeatherPerMinute = 2
	cfg.APIPerMinute = 100

	r := gin.New()
	r.Use(middleware.RateLimit(cfg))
	r.GET("/api/v1/weather", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.JSON(http.StatusOK, gin.H{"data": "weather"})
	})
	r.GET("/api/v1/conversations", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.JSON(http.StatusOK, gin.H{"data": "convs"})
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/v1/weather", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("weather request %d: unexpected 429", i)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/weather", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatal("expected 429 for third weather request")
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429 response")
	}
}

func TestRateLimit_DifferentUsersSeparateBuckets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := middleware.DefaultRateLimitConfig()
	cfg.WeatherPerMinute = 1

	r := gin.New()
	r.Use(func(c *gin.Context) {
		uid := c.Query("uid")
		if uid != "" {
			c.Set("user_id", uid)
		}
		c.Next()
	})
	r.Use(middleware.RateLimit(cfg))
	r.GET("/api/v1/weather", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "weather"})
	})

	userA := uuid.New().String()
	userB := uuid.New().String()

	reqA := httptest.NewRequest("GET", "/api/v1/weather?uid="+userA, nil)
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	if wA.Code != http.StatusOK {
		t.Fatalf("user A first: expected 200, got %d", wA.Code)
	}

	reqB := httptest.NewRequest("GET", "/api/v1/weather?uid="+userB, nil)
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	if wB.Code != http.StatusOK {
		t.Fatalf("user B first: expected 200, got %d", wB.Code)
	}

	reqA2 := httptest.NewRequest("GET", "/api/v1/weather?uid="+userA, nil)
	wA2 := httptest.NewRecorder()
	r.ServeHTTP(wA2, reqA2)
	if wA2.Code != http.StatusTooManyRequests {
		t.Fatal("user A second: expected 429")
	}
}

func TestRateLimit_RetryAfterHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := middleware.DefaultRateLimitConfig()
	cfg.WeatherPerMinute = 1

	r := gin.New()
	r.Use(middleware.RateLimit(cfg))
	r.GET("/api/v1/weather", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.JSON(http.StatusOK, gin.H{"data": "weather"})
	})

	req := httptest.NewRequest("GET", "/api/v1/weather", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	req2 := httptest.NewRequest("GET", "/api/v1/weather", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Fatal("expected 429 for second request with limit=1")
	}

	retryAfter := w2.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestRateLimit_AnonymousUsesIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := middleware.DefaultRateLimitConfig()
	cfg.APIPerMinute = 5

	r := gin.New()
	r.Use(middleware.RateLimit(cfg))
	r.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d: unexpected 429 before limit reached", i)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatal("expected 429 after exceeding limit")
	}
}
