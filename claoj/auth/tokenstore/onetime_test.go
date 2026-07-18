package tokenstore

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// oneTimeFactories returns one OneTimeStore per backend so every behavioral
// test in this file runs against both the in-memory implementation and a
// miniredis-backed Redis implementation.
func oneTimeFactories(t *testing.T) map[string]OneTimeStore {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return map[string]OneTimeStore{
		"memory": NewMemoryOneTime(),
		"redis":  NewRedisOneTime(client),
	}
}

func TestOneTime_IssueConsume_ReturnsUserIDOnce(t *testing.T) {
	for name, store := range oneTimeFactories(t) {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, store.Issue(KindPasswordReset, "tok-a", 42, time.Hour))

			uid, ok, err := store.Consume(KindPasswordReset, "tok-a")
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, uint(42), uid)

			// Second consume of the same token must fail: single-use.
			uid2, ok2, err2 := store.Consume(KindPasswordReset, "tok-a")
			require.NoError(t, err2)
			require.False(t, ok2)
			require.Equal(t, uint(0), uid2)
		})
	}
}

func TestOneTime_Consume_Unknown_NotFound(t *testing.T) {
	for name, store := range oneTimeFactories(t) {
		t.Run(name, func(t *testing.T) {
			uid, ok, err := store.Consume(KindEmailVerify, "does-not-exist")
			require.NoError(t, err)
			require.False(t, ok)
			require.Equal(t, uint(0), uid)
		})
	}
}

func TestOneTime_KindsAreIsolated(t *testing.T) {
	for name, store := range oneTimeFactories(t) {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, store.Issue(KindPasswordReset, "shared-token", 1, time.Hour))

			// Consuming the same raw token string under a different kind
			// must not find it.
			uid, ok, err := store.Consume(KindEmailVerify, "shared-token")
			require.NoError(t, err)
			require.False(t, ok)
			require.Equal(t, uint(0), uid)

			// The original kind still has it.
			uid2, ok2, err2 := store.Consume(KindPasswordReset, "shared-token")
			require.NoError(t, err2)
			require.True(t, ok2)
			require.Equal(t, uint(1), uid2)
		})
	}
}

func TestOneTime_Invalidate_RevokesOutstandingTokens(t *testing.T) {
	for name, store := range oneTimeFactories(t) {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, store.Issue(KindEmailVerify, "tok-1", 7, time.Hour))
			require.NoError(t, store.Issue(KindEmailVerify, "tok-2", 7, time.Hour))
			require.NoError(t, store.Issue(KindEmailVerify, "tok-other-user", 8, time.Hour))

			require.NoError(t, store.Invalidate(KindEmailVerify, 7))

			_, ok1, err := store.Consume(KindEmailVerify, "tok-1")
			require.NoError(t, err)
			require.False(t, ok1, "invalidated token must not be consumable")

			_, ok2, err := store.Consume(KindEmailVerify, "tok-2")
			require.NoError(t, err)
			require.False(t, ok2, "invalidated token must not be consumable")

			uid3, ok3, err := store.Consume(KindEmailVerify, "tok-other-user")
			require.NoError(t, err)
			require.True(t, ok3, "a different user's token must be unaffected")
			require.Equal(t, uint(8), uid3)
		})
	}
}

func TestOneTime_HasOutstanding(t *testing.T) {
	for name, store := range oneTimeFactories(t) {
		t.Run(name, func(t *testing.T) {
			has, err := store.HasOutstanding(KindPasswordReset, 55)
			require.NoError(t, err)
			require.False(t, has, "no token issued yet")

			require.NoError(t, store.Issue(KindPasswordReset, "tok-outstanding", 55, time.Hour))

			has, err = store.HasOutstanding(KindPasswordReset, 55)
			require.NoError(t, err)
			require.True(t, has)

			_, ok, err := store.Consume(KindPasswordReset, "tok-outstanding")
			require.NoError(t, err)
			require.True(t, ok)

			has, err = store.HasOutstanding(KindPasswordReset, 55)
			require.NoError(t, err)
			require.False(t, has, "outstanding token was consumed")
		})
	}
}

func TestOneTime_HasOutstanding_AfterInvalidate(t *testing.T) {
	for name, store := range oneTimeFactories(t) {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, store.Issue(KindEmailVerify, "tok-x", 21, time.Hour))
			has, err := store.HasOutstanding(KindEmailVerify, 21)
			require.NoError(t, err)
			require.True(t, has)

			require.NoError(t, store.Invalidate(KindEmailVerify, 21))

			has, err = store.HasOutstanding(KindEmailVerify, 21)
			require.NoError(t, err)
			require.False(t, has)
		})
	}
}

func TestMemoryOneTime_Expired_NotConsumable(t *testing.T) {
	store := NewMemoryOneTime()
	require.NoError(t, store.Issue(KindPasswordReset, "short-lived", 3, 10*time.Millisecond))
	time.Sleep(30 * time.Millisecond)

	uid, ok, err := store.Consume(KindPasswordReset, "short-lived")
	require.NoError(t, err)
	require.False(t, ok, "expired token must not be consumable")
	require.Equal(t, uint(0), uid)
}

func TestMemoryOneTime_Expired_NotOutstanding(t *testing.T) {
	store := NewMemoryOneTime()
	require.NoError(t, store.Issue(KindPasswordReset, "short-lived-2", 3, 10*time.Millisecond))
	time.Sleep(30 * time.Millisecond)

	has, err := store.HasOutstanding(KindPasswordReset, 3)
	require.NoError(t, err)
	require.False(t, has, "expired token must not count as outstanding")
}

// TestRedisOneTime_Expired_NotConsumable uses miniredis's virtual clock
// (FastForward) rather than a real sleep, since miniredis TTLs only tick
// down when explicitly fast-forwarded.
func TestRedisOneTime_Expired_NotConsumable(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisOneTime(client)

	require.NoError(t, store.Issue(KindEmailVerify, "expiring-token", 9, time.Hour))
	mr.FastForward(2 * time.Hour)

	uid, ok, err := store.Consume(KindEmailVerify, "expiring-token")
	require.NoError(t, err)
	require.False(t, ok, "expired token must not be consumable")
	require.Equal(t, uint(0), uid)
}

func TestRedisOneTime_Expired_NotOutstanding(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisOneTime(client)

	require.NoError(t, store.Issue(KindEmailVerify, "expiring-token-2", 9, time.Hour))
	mr.FastForward(2 * time.Hour)

	has, err := store.HasOutstanding(KindEmailVerify, 9)
	require.NoError(t, err)
	require.False(t, has, "expired token must not count as outstanding")
}

// TestRedisOneTime_KeyLayout verifies the Redis-specific key layout: value
// at ott:{kind}:{sha256hex(token)}, membership tracked in the
// ottuser:{kind}:{userID} index set.
func TestRedisOneTime_KeyLayout(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisOneTime(client)

	require.NoError(t, store.Issue(KindPasswordReset, "layout-token", 99, time.Hour))

	sum := sha256.Sum256([]byte("layout-token"))
	hash := hex.EncodeToString(sum[:])

	require.True(t, mr.Exists("ott:pwreset:"+hash), "expected value key ott:{kind}:{sha256hex(token)}")

	userMember, err := mr.SIsMember("ottuser:pwreset:99", hash)
	require.NoError(t, err)
	require.True(t, userMember)
}
