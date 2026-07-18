package tokenstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// storeFactories returns one Store per backend so every behavioral test in
// this file runs against both the in-memory implementation and a
// miniredis-backed Redis implementation. This keeps the two backends
// honest against the same contract.
func storeFactories(t *testing.T) map[string]Store {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return map[string]Store{
		"memory": NewMemoryStore(),
		"redis":  NewRedisStore(client),
	}
}

func TestStore_SaveGetRoundtrip(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			entry := Entry{
				UserID:    42,
				FamilyID:  "fam-1",
				ExpiresAt: time.Now().Add(time.Hour),
				CreatedAt: time.Now(),
				UserAgent: "TestAgent/1.0",
				ClientIP:  "127.0.0.1",
			}
			require.NoError(t, store.Save("token-a", entry))

			got, found, err := store.Get("token-a")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, entry.UserID, got.UserID)
			require.Equal(t, entry.FamilyID, got.FamilyID)
			require.Equal(t, entry.UserAgent, got.UserAgent)
			require.Equal(t, entry.ClientIP, got.ClientIP)
			require.False(t, got.Revoked)
			require.WithinDuration(t, entry.ExpiresAt, got.ExpiresAt, time.Second)
		})
	}
}

func TestStore_GetUnknown_NotFound(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			got, found, err := store.Get("does-not-exist")
			require.NoError(t, err)
			require.False(t, found)
			require.Nil(t, got)
		})
	}
}

func TestStore_Revoke_FoundWithRevokedTrue(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			entry := Entry{
				UserID:    1,
				FamilyID:  "fam-revoke",
				ExpiresAt: time.Now().Add(time.Hour),
			}
			require.NoError(t, store.Save("token-b", entry))
			require.NoError(t, store.Revoke("token-b"))

			got, found, err := store.Get("token-b")
			require.NoError(t, err)
			require.True(t, found, "revoked entries must remain retrievable until TTL for reuse detection")
			require.True(t, got.Revoked)
		})
	}
}

func TestStore_RevokeFamily_RevokesAllMembers(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			fam := "fam-shared"
			e1 := Entry{UserID: 1, FamilyID: fam, ExpiresAt: time.Now().Add(time.Hour)}
			e2 := Entry{UserID: 1, FamilyID: fam, ExpiresAt: time.Now().Add(time.Hour)}
			otherFam := Entry{UserID: 1, FamilyID: "fam-other", ExpiresAt: time.Now().Add(time.Hour)}
			require.NoError(t, store.Save("token-c1", e1))
			require.NoError(t, store.Save("token-c2", e2))
			require.NoError(t, store.Save("token-other", otherFam))

			require.NoError(t, store.RevokeFamily(fam))

			got1, found1, err := store.Get("token-c1")
			require.NoError(t, err)
			require.True(t, found1)
			require.True(t, got1.Revoked, "first family member should be revoked")

			got2, found2, err := store.Get("token-c2")
			require.NoError(t, err)
			require.True(t, found2)
			require.True(t, got2.Revoked, "second family member should be revoked")

			gotOther, foundOther, err := store.Get("token-other")
			require.NoError(t, err)
			require.True(t, foundOther)
			require.False(t, gotOther.Revoked, "tokens in a different family must not be revoked")
		})
	}
}

func TestStore_RevokeAllForUser(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			e1 := Entry{UserID: 7, FamilyID: "fam-u1", ExpiresAt: time.Now().Add(time.Hour)}
			e2 := Entry{UserID: 7, FamilyID: "fam-u2", ExpiresAt: time.Now().Add(time.Hour)}
			otherUser := Entry{UserID: 8, FamilyID: "fam-u3", ExpiresAt: time.Now().Add(time.Hour)}
			require.NoError(t, store.Save("token-d1", e1))
			require.NoError(t, store.Save("token-d2", e2))
			require.NoError(t, store.Save("token-d3", otherUser))

			require.NoError(t, store.RevokeAllForUser(7))

			got1, _, err := store.Get("token-d1")
			require.NoError(t, err)
			require.True(t, got1.Revoked)

			got2, _, err := store.Get("token-d2")
			require.NoError(t, err)
			require.True(t, got2.Revoked)

			got3, _, err := store.Get("token-d3")
			require.NoError(t, err)
			require.False(t, got3.Revoked, "a different user's session must not be revoked")
		})
	}
}

func TestStore_ExpiredEntry_NotFound(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			entry := Entry{
				UserID:    9,
				FamilyID:  "fam-expired",
				ExpiresAt: time.Now().Add(-time.Hour), // already expired
			}
			require.NoError(t, store.Save("token-expired", entry))

			got, found, err := store.Get("token-expired")
			require.NoError(t, err)
			require.False(t, found, "an entry past its ExpiresAt must not be returned")
			require.Nil(t, got)
		})
	}
}

// TestRedisStore_KeyLayout verifies the Redis-specific key layout described
// in the design: value at rt:{sha256hex(token)}, membership tracked in
// rtfam:{familyID} and rtuser:{userID} index sets.
func TestRedisStore_KeyLayout(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisStore(client)

	entry := Entry{
		UserID:    99,
		FamilyID:  "fam-layout",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, store.Save("layout-token", entry))

	sum := sha256.Sum256([]byte("layout-token"))
	hash := hex.EncodeToString(sum[:])

	require.True(t, mr.Exists("rt:"+hash), "expected value key rt:{sha256hex(token)}")

	famMember, err := mr.SIsMember(fmt.Sprintf("rtfam:%s", entry.FamilyID), hash)
	require.NoError(t, err)
	require.True(t, famMember)

	userMember, err := mr.SIsMember(fmt.Sprintf("rtuser:%d", entry.UserID), hash)
	require.NoError(t, err)
	require.True(t, userMember)
}
