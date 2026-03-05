// Package lockout implements account lockout protection after failed login attempts
package lockout

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// MaxFailedAttempts before lockout
	MaxFailedAttempts = 10

	// LockoutDuration is how long the account stays locked
	LockoutDuration = 15 * time.Minute

	// BaseLockoutPrefix is the Redis key prefix
	BaseLockoutPrefix = "lockout:"
)

// Repository handles lockout state storage
type Repository struct {
	client *redis.Client
}

// NewRepository creates a new lockout repository
func NewRepository(client *redis.Client) *Repository {
	return &Repository{client: client}
}

// RecordFailedAttempt records a failed login attempt and returns current attempt count
func (r *Repository) RecordFailedAttempt(ctx context.Context, identifier string) (int64, error) {
	if r.client == nil {
		return 0, nil // No Redis, skip lockout tracking
	}

	key := BaseLockoutPrefix + identifier

	// Increment attempt counter
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// Set expiry on first attempt
	if count == 1 {
		r.client.Expire(ctx, key, LockoutDuration)
	}

	return count, nil
}

// IsLocked checks if an account is currently locked
func (r *Repository) IsLocked(ctx context.Context, identifier string) (bool, time.Duration, error) {
	if r.client == nil {
		return false, 0, nil
	}

	key := BaseLockoutPrefix + identifier

	// Get attempt count
	count, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	if count >= MaxFailedAttempts {
		// Get TTL to determine remaining lockout time
		ttl, err := r.client.TTL(ctx, key).Result()
		if err != nil {
			return true, 0, nil
		}
		return true, ttl, nil
	}

	return false, 0, nil
}

// GetRemainingAttempts returns the number of attempts remaining before lockout
func (r *Repository) GetRemainingAttempts(ctx context.Context, identifier string) (int64, error) {
	if r.client == nil {
		return MaxFailedAttempts, nil
	}

	key := BaseLockoutPrefix + identifier

	count, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return MaxFailedAttempts, nil
	}
	if err != nil {
		return 0, err
	}

	remaining := MaxFailedAttempts - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// Reset clears the lockout state for an account (e.g., after successful login)
func (r *Repository) Reset(ctx context.Context, identifier string) error {
	if r.client == nil {
		return nil
	}

	key := BaseLockoutPrefix + identifier
	return r.client.Del(ctx, key).Err()
}

// GetLockoutInfo returns detailed lockout information for an account
func (r *Repository) GetLockoutInfo(ctx context.Context, identifier string) (attempts int64, locked bool, ttl time.Duration, err error) {
	if r.client == nil {
		return 0, false, 0, nil
	}

	key := BaseLockoutPrefix + identifier

	// Get attempt count
	count, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, false, 0, nil
	}
	if err != nil {
		return 0, false, 0, err
	}

	locked = count >= MaxFailedAttempts
	if locked {
		ttl, _ = r.client.TTL(ctx, key).Result()
	}

	return count, locked, ttl, nil
}

// FormatLockoutMessage creates a user-friendly lockout message
func FormatLockoutMessage(remainingAttempts int64, lockoutTTL time.Duration) string {
	if remainingAttempts <= 0 {
		minutes := int(lockoutTTL.Minutes())
		if minutes < 1 {
			seconds := int(lockoutTTL.Seconds())
			return "Account temporarily locked. Please try again in " + strconv.Itoa(seconds) + " seconds."
		}
		return "Account temporarily locked. Please try again in " + strconv.Itoa(minutes) + " minutes."
	}

	if remainingAttempts <= 3 {
		return "Warning: " + strconv.Itoa(int(remainingAttempts)) + " failed attempts remaining before account lockout."
	}

	return ""
}
