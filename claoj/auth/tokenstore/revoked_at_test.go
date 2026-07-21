package tokenstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Refresh-token rotation needs to tell two situations apart:
//
//   - a genuine replay, where a token rotated out long ago resurfaces (an
//     attacker holding a stolen copy) — the whole family must die; and
//   - two tabs of the same browser refreshing within the same instant, where
//     the loser presents a token the winner rotated out microseconds ago.
//
// Both look identical through Revoked alone, so Revoke must record *when* the
// revocation happened and the handler can then apply a short grace window.
func TestRevokeRecordsRevokedAt(t *testing.T) {
	s := NewMemoryStore()

	before := time.Now()
	require.NoError(t, s.Save("tok", Entry{
		UserID:    7,
		FamilyID:  "fam-1",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}))

	e, found, err := s.Get("tok")
	require.NoError(t, err)
	require.True(t, found)
	assert.True(t, e.RevokedAt.IsZero(), "a live entry must not carry a revocation timestamp")

	require.NoError(t, s.Revoke("tok"))

	e, found, err = s.Get("tok")
	require.NoError(t, err)
	require.True(t, found, "a revoked entry stays retrievable for reuse detection")
	require.True(t, e.Revoked)
	assert.False(t, e.RevokedAt.IsZero(), "Revoke must stamp RevokedAt")
	assert.False(t, e.RevokedAt.Before(before), "RevokedAt must not predate the save")
	assert.False(t, e.RevokedAt.After(time.Now()), "RevokedAt must not be in the future")
}

func TestRevokeIsIdempotentAndKeepsTheFirstTimestamp(t *testing.T) {
	s := NewMemoryStore()
	require.NoError(t, s.Save("tok", Entry{
		UserID: 7, FamilyID: "fam-1",
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}))

	require.NoError(t, s.Revoke("tok"))
	first, _, err := s.Get("tok")
	require.NoError(t, err)

	require.NoError(t, s.Revoke("tok"))
	second, _, err := s.Get("tok")
	require.NoError(t, err)

	assert.Equal(t, first.RevokedAt, second.RevokedAt,
		"re-revoking must not slide the timestamp forward, or the grace window would renew itself")
}

func TestRevokeFamilyStampsEveryMember(t *testing.T) {
	s := NewMemoryStore()
	exp := time.Now().Add(time.Hour)
	require.NoError(t, s.Save("a", Entry{UserID: 1, FamilyID: "fam", ExpiresAt: exp, CreatedAt: time.Now()}))
	require.NoError(t, s.Save("b", Entry{UserID: 1, FamilyID: "fam", ExpiresAt: exp, CreatedAt: time.Now()}))

	require.NoError(t, s.RevokeFamily("fam"))

	for _, tok := range []string{"a", "b"} {
		e, found, err := s.Get(tok)
		require.NoError(t, err)
		require.True(t, found)
		assert.True(t, e.Revoked, "%s should be revoked", tok)
		assert.False(t, e.RevokedAt.IsZero(), "%s should carry a revocation timestamp", tok)
	}
}
