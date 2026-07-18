package tokenstore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Kinds of one-time tokens. These string values are part of the Redis key
// layout (ott:{kind}:...) and must stay stable.
const (
	KindPasswordReset = "pwreset"
	KindEmailVerify   = "emailverify"
)

// OneTimeStore issues and atomically consumes single-use tokens (password
// reset, email verification) outside the shared (Django-owned) MySQL
// schema. Tokens are opaque random strings that are never persisted
// verbatim: only their sha256 hash is used as a lookup key.
type OneTimeStore interface {
	// Issue records a new outstanding token of the given kind for userID,
	// expiring after ttl.
	Issue(kind string, token string, userID uint, ttl time.Duration) error
	// Consume atomically looks up and deletes the token (get-and-delete),
	// so a token can be redeemed at most once. ok=false covers both
	// "never issued" and "already used or expired".
	Consume(kind string, token string) (userID uint, ok bool, err error)
	// Invalidate revokes every outstanding token of this kind for userID
	// (e.g. because a new one was just issued, or the flow it protects
	// completed via a different token).
	Invalidate(kind string, userID uint) error
	// HasOutstanding reports whether userID has at least one live
	// (unconsumed, unexpired) token of this kind.
	HasOutstanding(kind string, userID uint) (bool, error)
}

// ---------------------------------------------------------------------
// In-memory implementation (tests, Redis-less dev)
// ---------------------------------------------------------------------

type memOneTimeEntry struct {
	userID    uint
	expiresAt time.Time
}

type memoryOneTime struct {
	mu     sync.Mutex
	values map[string]memOneTimeEntry // "kind:hash" -> entry
	byUser map[string]map[string]bool // "kind:userID" -> set of hash
}

// NewMemoryOneTime returns a process-local OneTimeStore backed by a map. It
// is used in tests and as a fallback when Redis is unavailable; tokens do
// not survive a process restart.
func NewMemoryOneTime() OneTimeStore {
	return &memoryOneTime{
		values: make(map[string]memOneTimeEntry),
		byUser: make(map[string]map[string]bool),
	}
}

func oneTimeValueKey(kind, hash string) string { return kind + ":" + hash }
func oneTimeUserKey(kind string, userID uint) string {
	return kind + ":" + strconv.FormatUint(uint64(userID), 10)
}

func (s *memoryOneTime) Issue(kind, token string, userID uint, ttl time.Duration) error {
	hash := hashToken(token)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.values[oneTimeValueKey(kind, hash)] = memOneTimeEntry{
		userID:    userID,
		expiresAt: time.Now().Add(ttl),
	}

	uk := oneTimeUserKey(kind, userID)
	if s.byUser[uk] == nil {
		s.byUser[uk] = make(map[string]bool)
	}
	s.byUser[uk][hash] = true

	return nil
}

func (s *memoryOneTime) Consume(kind, token string) (uint, bool, error) {
	hash := hashToken(token)
	key := oneTimeValueKey(kind, hash)

	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.values[key]
	if !ok {
		return 0, false, nil
	}
	// Always delete on read (get-and-delete), regardless of expiry.
	delete(s.values, key)
	if set, ok2 := s.byUser[oneTimeUserKey(kind, e.userID)]; ok2 {
		delete(set, hash)
	}

	if time.Now().After(e.expiresAt) {
		return 0, false, nil
	}
	return e.userID, true, nil
}

func (s *memoryOneTime) Invalidate(kind string, userID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	uk := oneTimeUserKey(kind, userID)
	for hash := range s.byUser[uk] {
		delete(s.values, oneTimeValueKey(kind, hash))
	}
	delete(s.byUser, uk)

	return nil
}

func (s *memoryOneTime) HasOutstanding(kind string, userID uint) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	uk := oneTimeUserKey(kind, userID)
	now := time.Now()
	for hash := range s.byUser[uk] {
		if e, ok := s.values[oneTimeValueKey(kind, hash)]; ok && now.Before(e.expiresAt) {
			return true, nil
		}
	}
	return false, nil
}

// ---------------------------------------------------------------------
// Redis implementation
// ---------------------------------------------------------------------

type redisOneTime struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisOneTime returns an OneTimeStore backed by Redis. Layout:
//   - ott:{kind}:{sha256hex(token)}     -> userID (string), TTL = the
//     token's lifetime
//   - ottuser:{kind}:{userID}           -> set of token hashes belonging
//     to that user, TTL extended (never shortened) to cover the
//     longest-lived member
//
// Consume performs an atomic GETDEL on the value key so a token can be
// redeemed at most once even under concurrent requests.
func NewRedisOneTime(client *redis.Client) OneTimeStore {
	return &redisOneTime{client: client, ctx: context.Background()}
}

func oneTimeRedisValueKey(kind, hash string) string { return "ott:" + kind + ":" + hash }
func oneTimeRedisUserKey(kind string, userID uint) string {
	return fmt.Sprintf("ottuser:%s:%d", kind, userID)
}

func (s *redisOneTime) Issue(kind, token string, userID uint, ttl time.Duration) error {
	hash := hashToken(token)

	if err := s.client.Set(s.ctx, oneTimeRedisValueKey(kind, hash), userID, ttl).Err(); err != nil {
		return err
	}

	uk := oneTimeRedisUserKey(kind, userID)
	if err := s.client.SAdd(s.ctx, uk, hash).Err(); err != nil {
		return err
	}
	return s.extendExpiry(uk, ttl)
}

// extendExpiry raises key's TTL to ttl if it currently has none or a
// shorter one, so an index set's TTL always covers its longest-lived
// member instead of being clobbered down to the most-recently-added one.
func (s *redisOneTime) extendExpiry(key string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	cur, err := s.client.TTL(s.ctx, key).Result()
	if err != nil {
		return err
	}
	if cur < 0 || ttl > cur {
		return s.client.Expire(s.ctx, key, ttl).Err()
	}
	return nil
}

func (s *redisOneTime) Consume(kind, token string) (uint, bool, error) {
	hash := hashToken(token)

	val, err := s.client.GetDel(s.ctx, oneTimeRedisValueKey(kind, hash)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}

	uid, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return uint(uid), true, nil
}

func (s *redisOneTime) Invalidate(kind string, userID uint) error {
	uk := oneTimeRedisUserKey(kind, userID)

	hashes, err := s.client.SMembers(s.ctx, uk).Result()
	if err != nil {
		return err
	}

	if len(hashes) > 0 {
		keys := make([]string, len(hashes))
		for i, h := range hashes {
			keys[i] = oneTimeRedisValueKey(kind, h)
		}
		if err := s.client.Del(s.ctx, keys...).Err(); err != nil {
			return err
		}
	}

	return s.client.Del(s.ctx, uk).Err()
}

func (s *redisOneTime) HasOutstanding(kind string, userID uint) (bool, error) {
	uk := oneTimeRedisUserKey(kind, userID)

	hashes, err := s.client.SMembers(s.ctx, uk).Result()
	if err != nil {
		return false, err
	}

	has := false
	var stale []string
	for _, h := range hashes {
		exists, err := s.client.Exists(s.ctx, oneTimeRedisValueKey(kind, h)).Result()
		if err != nil {
			return false, err
		}
		if exists > 0 {
			has = true
		} else {
			stale = append(stale, h)
		}
	}

	// Lazily prune hashes whose value key has already expired so the
	// index set doesn't grow unbounded.
	if len(stale) > 0 {
		s.client.SRem(s.ctx, uk, stale)
	}

	return has, nil
}
