package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	database.AutoMigrate(&models.AuthUser{}, &models.Profile{})
	db.DB = database
	return database
}

func TestInMemoryRepository(t *testing.T) {
	repo := NewInMemoryRepository(60) // 60 second window
	ctx := context.Background()

	key := "test_key"

	// First request should pass
	allowed, err := repo.IsAllowed(ctx, key, Limit{Rate: 100, Window: 60})
	if err != nil {
		t.Fatalf("IsAllowed() error = %v", err)
	}
	if !allowed {
		t.Errorf("IsAllowed() = false, want true for first request")
	}

	// Exhaust the limit
	for i := 0; i < 99; i++ {
		repo.IsAllowed(ctx, key, Limit{Rate: 100, Window: 60})
	}

	// 101st request should be rate limited
	allowed, err = repo.IsAllowed(ctx, key, Limit{Rate: 100, Window: 60})
	if err != nil {
		t.Fatalf("IsAllowed() error = %v", err)
	}
	if allowed {
		t.Errorf("IsAllowed() = true, want false after limit exceeded")
	}
}

func TestInMemoryRepositoryExpiry(t *testing.T) {
	repo := NewInMemoryRepository(1) // 1 second window
	ctx := context.Background()

	key := "test_expiry_key"

	// Make a request
	allowed, err := repo.IsAllowed(ctx, key, Limit{Rate: 1, Window: 1})
	if err != nil {
		t.Fatalf("IsAllowed() error = %v", err)
	}
	if !allowed {
		t.Errorf("IsAllowed() = false, want true for first request")
	}

	// Second request should be rate limited
	allowed, err = repo.IsAllowed(ctx, key, Limit{Rate: 1, Window: 1})
	if err != nil {
		t.Fatalf("IsAllowed() error = %v", err)
	}
	if allowed {
		t.Errorf("IsAllowed() = true, want false - should be rate limited")
	}

	// Wait for window to expire
	time.Sleep(1500 * time.Millisecond)

	// Request should be allowed again
	allowed, err = repo.IsAllowed(ctx, key, Limit{Rate: 1, Window: 1})
	if err != nil {
		t.Fatalf("IsAllowed() error = %v", err)
	}
	if !allowed {
		t.Errorf("IsAllowed() = false, want true after window expired")
	}
}

func TestInMemoryRepositoryDifferentKeys(t *testing.T) {
	repo := NewInMemoryRepository(60)
	ctx := context.Background()

	// Different keys should have independent limits
	key1 := "user_1"
	key2 := "user_2"

	// Exhaust limit for key1
	for i := 0; i < 100; i++ {
		repo.IsAllowed(ctx, key1, Limit{Rate: 100, Window: 60})
	}

	// key2 should still be allowed
	allowed, err := repo.IsAllowed(ctx, key2, Limit{Rate: 100, Window: 60})
	if err != nil {
		t.Fatalf("IsAllowed() error = %v", err)
	}
	if !allowed {
		t.Errorf("IsAllowed() = false for key2, want true - keys should be independent")
	}
}

func TestInMemoryRepository_GetRemaining(t *testing.T) {
	repo := NewInMemoryRepository(60)
	ctx := context.Background()
	key := "test_remaining"

	limit := Limit{Rate: 10, Window: 60}

	// Initial remaining should equal rate
	remaining, err := repo.GetRemaining(ctx, key, limit)
	assert.NoError(t, err)
	assert.Equal(t, 10, remaining)

	// Make 3 requests
	for i := 0; i < 3; i++ {
		repo.IsAllowed(ctx, key, limit)
	}

	// Remaining should be 7
	remaining, err = repo.GetRemaining(ctx, key, limit)
	assert.NoError(t, err)
	assert.Equal(t, 7, remaining)
}

func TestInMemoryRepository_GetResetTime(t *testing.T) {
	repo := NewInMemoryRepository(60)
	ctx := context.Background()
	key := "test_reset"

	limit := Limit{Rate: 10, Window: 60}

	// Make a request to create the bucket
	repo.IsAllowed(ctx, key, limit)

	resetTime, err := repo.GetResetTime(ctx, key, limit)
	assert.NoError(t, err)

	// Reset time should be ~60 seconds from now
	expectedMin := time.Now().Add(59 * time.Second)
	expectedMax := time.Now().Add(61 * time.Second)
	assert.True(t, resetTime.After(expectedMin))
	assert.True(t, resetTime.Before(expectedMax))
}

func TestMiddleware_AllowedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := NewInMemoryRepository(60)

	router := gin.New()
	router.Use(Middleware(repo))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
}

func TestMiddleware_RateLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Use very low limit for testing
	repo := NewInMemoryRepository(60)

	router := gin.New()
	router.Use(Middleware(repo))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Exhaust the rate limit (default is 100 for /test)
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// 101st request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
}

func TestMiddleware_AdminEndpointLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := NewInMemoryRepository(60)

	router := gin.New()
	router.Use(Middleware(repo))
	router.GET("/api/v2/admin/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Admin endpoints have limit of 30 per minute
	for i := 0; i < 30; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/admin/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// 31st request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/api/v2/admin/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestMiddleware_LoginEndpointLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := NewInMemoryRepository(60)

	router := gin.New()
	router.Use(Middleware(repo))
	router.POST("/api/v2/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Login has strict limit of 10 per minute
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/auth/login", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// 11th request should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/api/v2/auth/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestGetClientKey_UserVsIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test IP-based key (no auth)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.1")
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	key := getClientKey(ctx, Limit{Rate: 100, Window: 60})
	assert.Equal(t, "ip:192.168.1.1", key)

	// Test user-based key (with auth)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.1")
	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Set("user_id", uint(123))

	key = getClientKey(ctx, Limit{Rate: 100, Window: 60})
	assert.Equal(t, "user:123:192.168.1.1", key)
}

func TestGetLimit_PathSpecific(t *testing.T) {
	tests := []struct {
		path       string
		wantRate   int
		wantWindow int
	}{
		{"/api/v2/admin/users", 30, 60},
		{"/api/v2/auth/login", 10, 60},
		{"/api/v2/auth/password/reset", 3, 3600},
		{"/api/v2/auth/totp/verify", 10, 60},
		{"/api/v2/auth/verify-email", 5, 60},
		{"/api/v2/auth/register", 10, 60},
		{"/api/v2/problem/TEST/submit", 5, 60},
		{"/api/v2/problems", 100, 60},
		{"/api/v2/users", 100, 60},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			limit := getLimit(tt.path)
			assert.Equal(t, tt.wantRate, limit.Rate)
			assert.Equal(t, tt.wantWindow, limit.Window)
		})
	}
}

func TestBurstMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := NewInMemoryRepository(60)
	burstSize := 10

	router := gin.New()
	router.Use(BurstMiddleware(repo, burstSize))
	router.GET("/burst", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// With burst, effective limit is 100 + 10 = 110
	for i := 0; i < 110; i++ {
		req := httptest.NewRequest(http.MethodGet, "/burst", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// 111th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/burst", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}
