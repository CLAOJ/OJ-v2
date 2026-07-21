// Package tokenstore holds refresh-token session state outside of the
// shared (Django-owned) MySQL schema. Refresh tokens are opaque JWTs that
// are never persisted verbatim: only their sha256 hash is used as a lookup
// key, and the Entry payload carries everything needed for token-rotation
// reuse detection.
package tokenstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Entry describes a single refresh-token session.
type Entry struct {
	UserID   uint   `json:"user_id"`
	FamilyID string `json:"family_id"`
	Revoked  bool   `json:"revoked"`
	// RevokedAt is when Revoked was first set, and is what lets the refresh
	// handler tell a genuine token replay from two tabs of the same browser
	// refreshing simultaneously. Zero while the entry is live. Once set it is
	// never moved forward, so the grace window cannot renew itself.
	RevokedAt time.Time `json:"revoked_at,omitzero"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UserAgent string    `json:"user_agent,omitempty"`
	ClientIP  string    `json:"client_ip,omitempty"`
}

// Store manages refresh-token sessions. Implementations must keep a
// revoked entry retrievable (Get returns found=true, Revoked=true) until
// its natural expiry so that a reused/rotated-out token can still be
// detected and its whole family revoked — this is the token
// rotation-reuse-detection contract relied on by the auth handlers.
type Store interface {
	Save(token string, e Entry) error
	Get(token string) (*Entry, bool, error) // (entry, found, err)
	// Supersede retires a token because rotation replaced it with a successor.
	// It stamps RevokedAt so the handler can recognise a racing client that
	// still holds the old token; see the grace period in api/v2/auth.
	Supersede(token string) error
	// Revoke kills a token outright (logout). Unlike Supersede it leaves
	// RevokedAt zero, so presenting the token again is treated as reuse rather
	// than as a benign rotation collision.
	Revoke(token string) error
	RevokeFamily(familyID string) error
	RevokeAllForUser(userID uint) error
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------
// In-memory implementation (tests, Redis-less dev)
// ---------------------------------------------------------------------

type memoryStore struct {
	mu      sync.Mutex
	entries map[string]Entry           // token -> entry
	byFam   map[string]map[string]bool // familyID -> set of tokens
	byUser  map[uint]map[string]bool   // userID -> set of tokens
}

// NewMemoryStore returns a process-local Store backed by a map. It is used
// in tests and as a fallback when Redis is unavailable; sessions do not
// survive a process restart.
func NewMemoryStore() Store {
	return &memoryStore{
		entries: make(map[string]Entry),
		byFam:   make(map[string]map[string]bool),
		byUser:  make(map[uint]map[string]bool),
	}
}

func (s *memoryStore) Save(token string, e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[token] = e

	if s.byFam[e.FamilyID] == nil {
		s.byFam[e.FamilyID] = make(map[string]bool)
	}
	s.byFam[e.FamilyID][token] = true

	if s.byUser[e.UserID] == nil {
		s.byUser[e.UserID] = make(map[string]bool)
	}
	s.byUser[e.UserID][token] = true

	return nil
}

func (s *memoryStore) Get(token string) (*Entry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.entries[token]
	if !ok {
		return nil, false, nil
	}
	if time.Now().After(e.ExpiresAt) {
		return nil, false, nil
	}
	cp := e
	return &cp, true, nil
}

// markRevoked flags an entry. `superseded` records that rotation replaced the
// token (as opposed to it being killed outright), and is stamped only on the
// first transition so repeated revocations can't slide the grace window
// forward. Callers must hold s.mu.
func (s *memoryStore) markRevoked(token string, superseded bool) {
	e, ok := s.entries[token]
	if !ok || e.Revoked {
		return
	}
	e.Revoked = true
	if superseded {
		e.RevokedAt = time.Now()
	}
	s.entries[token] = e
}

func (s *memoryStore) Supersede(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.markRevoked(token, true)
	return nil
}

func (s *memoryStore) Revoke(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.markRevoked(token, false)
	return nil
}

func (s *memoryStore) RevokeFamily(familyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token := range s.byFam[familyID] {
		s.markRevoked(token, false)
	}
	return nil
}

func (s *memoryStore) RevokeAllForUser(userID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token := range s.byUser[userID] {
		s.markRevoked(token, false)
	}
	return nil
}

// ---------------------------------------------------------------------
// Redis implementation
// ---------------------------------------------------------------------

type redisStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStore returns a Store backed by Redis. Layout:
//   - rt:{sha256hex(token)}   -> JSON Entry, TTL = time until Entry.ExpiresAt
//   - rtfam:{familyID}        -> set of token hashes belonging to that family
//   - rtuser:{userID}         -> set of token hashes belonging to that user
//
// The index sets' TTL is extended (never shortened) as members are added,
// so it always covers the longest-lived member.
func NewRedisStore(client *redis.Client) Store {
	return &redisStore{client: client, ctx: context.Background()}
}

func valueKey(hash string) string   { return "rt:" + hash }
func famKey(familyID string) string { return "rtfam:" + familyID }
func userKey(userID uint) string    { return fmt.Sprintf("rtuser:%d", userID) }

func (s *redisStore) Save(token string, e Entry) error {
	hash := hashToken(token)
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	ttl := time.Until(e.ExpiresAt)
	if ttl < 0 {
		ttl = 0 // already-expired entries are stored without a TTL; Get() filters them by ExpiresAt
	}

	if err := s.client.Set(s.ctx, valueKey(hash), data, ttl).Err(); err != nil {
		return err
	}

	fk := famKey(e.FamilyID)
	if err := s.client.SAdd(s.ctx, fk, hash).Err(); err != nil {
		return err
	}
	if err := s.extendExpiry(fk, ttl); err != nil {
		return err
	}

	uk := userKey(e.UserID)
	if err := s.client.SAdd(s.ctx, uk, hash).Err(); err != nil {
		return err
	}
	if err := s.extendExpiry(uk, ttl); err != nil {
		return err
	}

	return nil
}

// extendExpiry raises key's TTL to ttl if it currently has none or a
// shorter one, so an index set's TTL always covers its longest-lived
// member instead of being clobbered down to the most-recently-added one.
func (s *redisStore) extendExpiry(key string, ttl time.Duration) error {
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

func (s *redisStore) Get(token string) (*Entry, bool, error) {
	hash := hashToken(token)
	data, err := s.client.Get(s.ctx, valueKey(hash)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, false, err
	}

	// Defensive check: an entry can linger without a Redis TTL if it was
	// saved already-expired (see Save). Treat it as not found and reap it.
	if time.Now().After(e.ExpiresAt) {
		s.client.Del(s.ctx, valueKey(hash))
		return nil, false, nil
	}

	return &e, true, nil
}

func (s *redisStore) Supersede(token string) error {
	return s.markRevokedHash(hashToken(token), true)
}

func (s *redisStore) Revoke(token string) error {
	return s.markRevokedHash(hashToken(token), false)
}

// markRevokedHash rewrites the value at rt:{hash} with Revoked=true while
// preserving whatever TTL currently remains on the key, so a subsequent
// Get still finds the (now revoked) entry until it naturally expires —
// this is what lets Refresh detect a reused/rotated-out token.
//
// `superseded` distinguishes rotation (the token was replaced by a successor,
// so a racing client presenting it deserves a retry) from an outright kill
// such as logout or a family revocation (where reuse must be treated as reuse).
func (s *redisStore) markRevokedHash(hash string, superseded bool) error {
	key := valueKey(hash)
	data, err := s.client.Get(s.ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil // nothing to revoke
	}
	if err != nil {
		return err
	}

	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return err
	}
	if e.Revoked {
		return nil
	}

	ttl, err := s.client.TTL(s.ctx, key).Result()
	if err != nil {
		return err
	}
	if ttl < 0 {
		ttl = 0
	}

	e.Revoked = true
	if superseded {
		e.RevokedAt = time.Now()
	}
	newData, err := json.Marshal(e)
	if err != nil {
		return err
	}

	return s.client.Set(s.ctx, key, newData, ttl).Err()
}

func (s *redisStore) RevokeFamily(familyID string) error {
	hashes, err := s.client.SMembers(s.ctx, famKey(familyID)).Result()
	if err != nil {
		return err
	}
	var firstErr error
	for _, h := range hashes {
		if err := s.markRevokedHash(h, false); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *redisStore) RevokeAllForUser(userID uint) error {
	hashes, err := s.client.SMembers(s.ctx, userKey(userID)).Result()
	if err != nil {
		return err
	}
	var firstErr error
	for _, h := range hashes {
		if err := s.markRevokedHash(h, false); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
