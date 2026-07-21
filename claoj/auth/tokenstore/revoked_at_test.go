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
func TestSupersedeRecordsRevokedAt(t *testing.T) {
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

	require.NoError(t, s.Supersede("tok"))

	e, found, err = s.Get("tok")
	require.NoError(t, err)
	require.True(t, found, "a retired entry stays retrievable for reuse detection")
	require.True(t, e.Revoked)
	assert.False(t, e.RevokedAt.IsZero(), "Supersede must stamp RevokedAt")
	assert.False(t, e.RevokedAt.Before(before), "RevokedAt must not predate the save")
	assert.False(t, e.RevokedAt.After(time.Now()), "RevokedAt must not be in the future")
}

func TestSupersedeIsIdempotentAndKeepsTheFirstTimestamp(t *testing.T) {
	s := NewMemoryStore()
	require.NoError(t, s.Save("tok", Entry{
		UserID: 7, FamilyID: "fam-1",
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}))

	require.NoError(t, s.Supersede("tok"))
	first, _, err := s.Get("tok")
	require.NoError(t, err)

	require.NoError(t, s.Supersede("tok"))
	second, _, err := s.Get("tok")
	require.NoError(t, err)

	assert.Equal(t, first.RevokedAt, second.RevokedAt,
		"re-revoking must not slide the timestamp forward, or the grace window would renew itself")
}

func TestRevokeFamilyMarksEveryMemberWithoutAGraceWindow(t *testing.T) {
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
		assert.True(t, e.RevokedAt.IsZero(),
			"%s: a family revocation is a security action, not a rotation -- it must NOT open a grace window", tok)
	}
}

// Logout kills a token outright. Presenting it again is reuse, not a racing
// tab, so it must not be granted the rotation grace window either.
func TestRevokeLeavesNoGraceWindow(t *testing.T) {
	s := NewMemoryStore()
	require.NoError(t, s.Save("tok", Entry{
		UserID: 7, FamilyID: "fam-1",
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}))

	require.NoError(t, s.Revoke("tok"))

	e, found, err := s.Get("tok")
	require.NoError(t, err)
	require.True(t, found)
	assert.True(t, e.Revoked)
	assert.True(t, e.RevokedAt.IsZero(), "Revoke must not stamp RevokedAt")
}

// A token retired by rotation and then caught up in a family revocation must
// keep its original rotation timestamp rather than having it cleared or moved.
func TestRevokeFamilyDoesNotDisturbAnAlreadySupersededEntry(t *testing.T) {
	s := NewMemoryStore()
	exp := time.Now().Add(time.Hour)
	require.NoError(t, s.Save("old", Entry{UserID: 1, FamilyID: "fam", ExpiresAt: exp, CreatedAt: time.Now()}))
	require.NoError(t, s.Supersede("old"))
	before, _, err := s.Get("old")
	require.NoError(t, err)

	require.NoError(t, s.RevokeFamily("fam"))

	after, _, err := s.Get("old")
	require.NoError(t, err)
	assert.Equal(t, before.RevokedAt, after.RevokedAt)
}
