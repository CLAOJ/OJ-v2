// Package auditlog stores admin audit-trail entries outside the shared
// (Django-owned) MySQL schema. Entries live in a Redis Stream in
// production ("audit:log", capped to the most recent ~100000 entries) or
// in an in-memory slice for tests and Redis-less dev, mirroring the
// pattern established by auth/tokenstore for other v2-only state.
package auditlog

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Entry is a single audit-trail record.
type Entry struct {
	ID         string    `json:"id"` // stream ID, e.g. "1721300000000-0" (Redis) or a synthetic monotonic ID (memory)
	UserID     uint      `json:"user_id"`
	Username   string    `json:"username"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Details    string    `json:"details"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// Filters narrows List results. A zero value for a field means "no filter"
// on that field.
type Filters struct {
	Action   string
	Resource string
	Status   string
	UserID   uint
	Start    *time.Time
	End      *time.Time
}

// Store persists and queries audit-log entries.
type Store interface {
	Write(e Entry) error
	List(f Filters, page, pageSize int) ([]Entry, int64, error) // newest first
	Get(id string) (*Entry, error)
}

// Default is the process-wide audit store. It is nil until main.go wires
// it up (auditlog.NewRedisStore/NewMemoryStore), following the same
// settable-global pattern Tasks 8/9 used (see
// api/v2/auth.RefreshStore / OneTimeTokens) so both the audit middleware
// (package middleware) and the admin audit-log handlers (package v2) can
// reach the same store without an import cycle.
var Default Store

// matches reports whether e satisfies every non-zero field in f.
func matches(e Entry, f Filters) bool {
	if f.Action != "" && e.Action != f.Action {
		return false
	}
	if f.Resource != "" && e.Resource != f.Resource {
		return false
	}
	if f.Status != "" && e.Status != f.Status {
		return false
	}
	if f.UserID != 0 && e.UserID != f.UserID {
		return false
	}
	if f.Start != nil && e.CreatedAt.Before(*f.Start) {
		return false
	}
	if f.End != nil && e.CreatedAt.After(*f.End) {
		return false
	}
	return true
}

// paginate slices a newest-first, already-filtered slice into the
// requested page. page is 1-indexed; out-of-range pages return an empty
// (non-nil) slice.
func paginate(entries []Entry, page, pageSize int) []Entry {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(entries) {
		return []Entry{}
	}
	end := start + pageSize
	if end > len(entries) {
		end = len(entries)
	}
	return entries[start:end]
}

// ---------------------------------------------------------------------
// In-memory implementation (tests, Redis-less dev)
// ---------------------------------------------------------------------

type memoryStore struct {
	mu      sync.Mutex
	entries []Entry // newest first
	seq     int64
}

// NewMemoryStore returns a process-local Store backed by a slice. It is
// used in tests and as a fallback when Redis is unavailable; entries do
// not survive a process restart.
func NewMemoryStore() Store {
	return &memoryStore{}
}

func (s *memoryStore) Write(e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	e.ID = fmt.Sprintf("%d-0", s.seq) // synthetic monotonic ID, shaped like a Redis stream ID
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}

	// Prepend so the slice stays newest-first (mirrors XREVRANGE order).
	s.entries = append([]Entry{e}, s.entries...)
	return nil
}

func (s *memoryStore) List(f Filters, page, pageSize int) ([]Entry, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		if matches(e, f) {
			filtered = append(filtered, e)
		}
	}

	total := int64(len(filtered))
	return paginate(filtered, page, pageSize), total, nil
}

func (s *memoryStore) Get(id string) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.entries {
		if e.ID == id {
			cp := e
			return &cp, nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------
// Redis implementation
// ---------------------------------------------------------------------

// streamKey is the Redis Stream all audit entries are XADD-ed to.
const streamKey = "audit:log"

// maxStreamLen bounds the stream via MAXLEN ~ (approximate trimming, cheap
// for Redis to enforce) so it can't grow unbounded.
const maxStreamLen = 100000

type redisStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStore returns a Store backed by a Redis Stream. Layout:
//   - audit:log -> a single stream, one XADD per audit entry, trimmed to
//     MAXLEN ~100000 (approximate) so it never grows unbounded.
//
// List scans the stream newest-first via XREVRANGE and applies Filters
// in-code (a stream has no secondary indexes); Get looks up a single
// entry by its exact stream ID via XRANGE id id.
func NewRedisStore(client *redis.Client) Store {
	return &redisStore{client: client, ctx: context.Background()}
}

func entryToValues(e Entry) map[string]interface{} {
	return map[string]interface{}{
		"user_id":     strconv.FormatUint(uint64(e.UserID), 10),
		"username":    e.Username,
		"action":      e.Action,
		"resource":    e.Resource,
		"resource_id": e.ResourceID,
		"ip_address":  e.IPAddress,
		"user_agent":  e.UserAgent,
		"details":     e.Details,
		"status":      e.Status,
		"created_at":  e.CreatedAt.Format(time.RFC3339Nano),
	}
}

func valuesToEntry(id string, values map[string]interface{}) Entry {
	e := Entry{ID: id}
	if v, ok := values["user_id"].(string); ok {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			e.UserID = uint(n)
		}
	}
	if v, ok := values["username"].(string); ok {
		e.Username = v
	}
	if v, ok := values["action"].(string); ok {
		e.Action = v
	}
	if v, ok := values["resource"].(string); ok {
		e.Resource = v
	}
	if v, ok := values["resource_id"].(string); ok {
		e.ResourceID = v
	}
	if v, ok := values["ip_address"].(string); ok {
		e.IPAddress = v
	}
	if v, ok := values["user_agent"].(string); ok {
		e.UserAgent = v
	}
	if v, ok := values["details"].(string); ok {
		e.Details = v
	}
	if v, ok := values["status"].(string); ok {
		e.Status = v
	}
	if v, ok := values["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			e.CreatedAt = t
		}
	}
	return e
}

func (s *redisStore) Write(e Entry) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	return s.client.XAdd(s.ctx, &redis.XAddArgs{
		Stream: streamKey,
		MaxLen: maxStreamLen,
		Approx: true,
		Values: entryToValues(e),
	}).Err()
}

func (s *redisStore) List(f Filters, page, pageSize int) ([]Entry, int64, error) {
	msgs, err := s.client.XRevRange(s.ctx, streamKey, "+", "-").Result()
	if err != nil {
		return nil, 0, err
	}

	filtered := make([]Entry, 0, len(msgs))
	for _, m := range msgs {
		e := valuesToEntry(m.ID, m.Values)
		if matches(e, f) {
			filtered = append(filtered, e)
		}
	}

	total := int64(len(filtered))
	return paginate(filtered, page, pageSize), total, nil
}

// streamIDPattern matches a well-formed Redis stream ID ("<ms>-<seq>").
// Get validates against it before querying Redis so an arbitrary/malformed
// caller-supplied ID (e.g. a bad :id URL param) resolves to "not found"
// instead of a Redis "invalid stream ID" error.
var streamIDPattern = regexp.MustCompile(`^\d+-\d+$`)

func (s *redisStore) Get(id string) (*Entry, error) {
	if !streamIDPattern.MatchString(id) {
		return nil, nil
	}

	msgs, err := s.client.XRange(s.ctx, streamKey, id, id).Result()
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, nil
	}
	e := valuesToEntry(msgs[0].ID, msgs[0].Values)
	return &e, nil
}
