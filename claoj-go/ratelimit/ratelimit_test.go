package ratelimit

import (
	"context"
	"testing"
	"time"
)

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
