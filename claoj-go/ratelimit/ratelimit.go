// Package ratelimit implements rate limiting middleware for the API.
// Supports both IP-based and user-based rate limiting with configurable limits.
package ratelimit

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Limit defines the rate limit configuration
type Limit struct {
	// Requests per window
	Rate int

	// Window duration in seconds
	Window int
}

// Repository defines the storage interface for rate limiting
type Repository interface {
	IsAllowed(ctx context.Context, key string, limit Limit) (bool, error)
	GetRemaining(ctx context.Context, key string, limit Limit) (int, error)
	GetResetTime(ctx context.Context, key string, limit Limit) (time.Time, error)
}

// RedisRepository implements rate limiting using Redis
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis-based rate limiter
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// IsAllowed checks if a request is allowed under the rate limit using Redis
func (r *RedisRepository) IsAllowed(ctx context.Context, key string, limit Limit) (bool, error) {
	if r.client == nil {
		// Fallback to in-memory if Redis not available
		return false, nil
	}

	now := time.Now()

	// Use Redis INCR with EXPIRE for atomic rate limiting
	pipe := r.client.Pipeline()

	// Increment counter for this key
	intCmd := pipe.Incr(ctx, "ratelimit:"+key)

	// Set expiry if this is a new key
	pipe.ExpireAt(ctx, "ratelimit:"+key, now.Add(time.Duration(limit.Window)*time.Second))

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, err
	}

	count := intCmd.Val()
	return count <= int64(limit.Rate), nil
}

// GetRemaining returns the remaining requests in the current window
func (r *RedisRepository) GetRemaining(ctx context.Context, key string, limit Limit) (int, error) {
	if r.client == nil {
		return limit.Rate, nil
	}

	val, err := r.client.Get(ctx, "ratelimit:"+key).Int64()
	if err == redis.Nil {
		return limit.Rate, nil
	}
	if err != nil {
		return limit.Rate, nil
	}

	remaining := limit.Rate - int(val)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// GetResetTime returns when the rate limit resets
func (r *RedisRepository) GetResetTime(ctx context.Context, key string, limit Limit) (time.Time, error) {
	if r.client == nil {
		return time.Now().Add(time.Duration(limit.Window) * time.Second), nil
	}

	ttl, err := r.client.TTL(ctx, "ratelimit:"+key).Result()
	if err != nil {
		return time.Now().Add(time.Duration(limit.Window) * time.Second), nil
	}

	return time.Now().Add(ttl), nil
}

// InMemoryRepository implements rate limiting using in-memory storage (for development/testing)
type InMemoryRepository struct {
	mu     sync.Mutex
	data   map[string]*timeBucket
	window int
}

type timeBucket struct {
	count       int
	lastRequest time.Time
}

// NewInMemoryRepository creates a new in-memory rate limiter
func NewInMemoryRepository(window int) *InMemoryRepository {
	return &InMemoryRepository{
		data:   make(map[string]*timeBucket),
		window: window,
	}
}

// IsAllowed checks if a request is allowed under the rate limit
func (r *InMemoryRepository) IsAllowed(ctx context.Context, key string, limit Limit) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// Get or create bucket for this key
	bucket, exists := r.data[key]
	if !exists {
		bucket = &timeBucket{
			count:       0,
			lastRequest: now,
		}
		r.data[key] = bucket
	}

	// Check if window has expired
	windowStart := now.Add(-time.Duration(limit.Window) * time.Second)
	if bucket.lastRequest.Before(windowStart) {
		bucket.count = 0
		bucket.lastRequest = now
	}

	// Check rate limit
	if bucket.count >= limit.Rate {
		return false, nil
	}

	// Increment counter
	bucket.count++
	return true, nil
}

// GetRemaining returns the remaining requests in the current window
func (r *InMemoryRepository) GetRemaining(ctx context.Context, key string, limit Limit) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, exists := r.data[key]
	if !exists {
		return limit.Rate, nil
	}

	windowStart := time.Now().Add(-time.Duration(limit.Window) * time.Second)
	if bucket.lastRequest.Before(windowStart) {
		return limit.Rate, nil
	}

	remaining := limit.Rate - bucket.count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// GetResetTime returns when the rate limit resets
func (r *InMemoryRepository) GetResetTime(ctx context.Context, key string, limit Limit) (time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, exists := r.data[key]
	if !exists {
		return time.Now().Add(time.Duration(limit.Window) * time.Second), nil
	}

	return bucket.lastRequest.Add(time.Duration(limit.Window) * time.Second), nil
}

// Middleware creates a rate limiting middleware for Gin
func Middleware(r Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get limit from config or use default
		limit := getLimit(c.Request.URL.Path)

		// Get client identifier (IP or user ID)
		key := getClientKey(c, limit)

		// Check rate limit
		allowed, err := r.IsAllowed(c, key, limit)
		if err != nil {
			// Log error but don't block the request
			c.Set("rate_limit_error", err.Error())
		}

		if !allowed {
			c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Rate))
			c.Header("X-RateLimit-Remaining", "0")

			resetTime, _ := r.GetResetTime(c, key, limit)
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
			c.Header("X-RateLimit-Reset-After", strconv.Itoa(limit.Window))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		remaining, _ := r.GetRemaining(c, key, limit)
		resetTime, _ := r.GetResetTime(c, key, limit)

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
		c.Header("X-RateLimit-Reset-After", strconv.Itoa(limit.Window))

		c.Next()
	}
}

// getClientKey generates a unique key for rate limiting
func getClientKey(c *gin.Context, limit Limit) string {
	// Get IP address
	ips := c.Request.Header.Get("X-Forwarded-For")
	if ips == "" {
		ips = c.Request.Header.Get("X-Real-IP")
	}
	if ips == "" {
		ips = c.ClientIP()
	}

	// For authenticated users, include user ID in key
	if uid, exists := c.Get("user_id"); exists {
		return "user:" + strconv.Itoa(int(uid.(uint))) + ":" + ips
	}

	// For unauthenticated requests, use IP-based key
	return "ip:" + strings.ReplaceAll(ips, ":", "_")
}

// getLimit returns the rate limit for a given path
func getLimit(path string) Limit {
	// Default limits
	defaultLimit := Limit{
		Rate:   100,
		Window: 60, // 60 seconds
	}

	// Path-specific limits
	switch {
	case strings.HasPrefix(path, "/api/v2/admin/"):
		// Lower limit for admin endpoints
		return Limit{
			Rate:   30,
			Window: 60,
		}
	case path == "/api/v2/auth/login":
		// Strict limit for login (prevent brute force)
		return Limit{
			Rate:   10,
			Window: 60,
		}
	case path == "/api/v2/auth/password/reset":
		// Very strict limit for password reset (3 per hour)
		return Limit{
			Rate:   3,
			Window: 3600,
		}
	case strings.HasPrefix(path, "/api/v2/auth/totp/"):
		// Strict limit for TOTP operations (prevent brute force)
		return Limit{
			Rate:   10,
			Window: 60,
		}
	case path == "/api/v2/auth/verify-email" || path == "/api/v2/auth/resend-verification":
		// Limit email verification endpoints
		return Limit{
			Rate:   5,
			Window: 60,
		}
	case path == "/api/v2/auth/register":
		// Limit registration to prevent spam
		return Limit{
			Rate:   10,
			Window: 60,
		}
	case strings.HasPrefix(path, "/api/v2/auth/"):
		// Other auth endpoints
		return Limit{
			Rate:   30,
			Window: 60,
		}
	case strings.HasPrefix(path, "/api/v2/problem/") && strings.HasSuffix(path, "/submit"):
		// Medium limit for submissions (heavy operation)
		return Limit{
			Rate:   5,
			Window: 60,
		}
	default:
		return defaultLimit
	}
}

// BurstMiddleware adds burstable rate limiting
func BurstMiddleware(r Repository, burstSize int) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := getLimit(c.Request.URL.Path)

		key := getClientKey(c, limit)

		// Use token bucket algorithm for burst
		allowed, _ := r.IsAllowed(c, key, Limit{
			Rate:   limit.Rate + burstSize,
			Window: limit.Window,
		})

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(limit.Window))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		remaining, _ := r.GetRemaining(c, key, limit)
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		c.Next()
	}
}

// ParseRateLimitHeader parses X-RateLimit-Reset header
func ParseRateLimitHeader(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	unix, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(unix, 0), true
}
