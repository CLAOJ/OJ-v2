package auditlog

import (
	"context"
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

func TestStore_Write_List_NewestFirst(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			base := time.Now().Add(-time.Hour)
			e1 := Entry{UserID: 1, Username: "alice", Action: "CREATE", Resource: "Problem", Status: "success", CreatedAt: base}
			e2 := Entry{UserID: 1, Username: "alice", Action: "UPDATE", Resource: "Problem", Status: "success", CreatedAt: base.Add(time.Minute)}
			e3 := Entry{UserID: 2, Username: "bob", Action: "DELETE", Resource: "Contest", Status: "success", CreatedAt: base.Add(2 * time.Minute)}

			require.NoError(t, store.Write(e1))
			require.NoError(t, store.Write(e2))
			require.NoError(t, store.Write(e3))

			entries, total, err := store.List(Filters{}, 1, 20)
			require.NoError(t, err)
			require.EqualValues(t, 3, total)
			require.Len(t, entries, 3)

			// newest first
			require.Equal(t, "DELETE", entries[0].Action)
			require.Equal(t, "UPDATE", entries[1].Action)
			require.Equal(t, "CREATE", entries[2].Action)

			for _, e := range entries {
				require.NotEmpty(t, e.ID, "Write must assign an ID")
			}
		})
	}
}

func TestStore_List_FilterByAction(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, store.Write(Entry{Action: "CREATE", Resource: "Problem"}))
			require.NoError(t, store.Write(Entry{Action: "UPDATE", Resource: "Problem"}))
			require.NoError(t, store.Write(Entry{Action: "DELETE", Resource: "Contest"}))

			entries, total, err := store.List(Filters{Action: "UPDATE"}, 1, 20)
			require.NoError(t, err)
			require.EqualValues(t, 1, total)
			require.Len(t, entries, 1)
			require.Equal(t, "UPDATE", entries[0].Action)
		})
	}
}

func TestStore_List_FilterByResourceStatusUserID(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, store.Write(Entry{UserID: 1, Action: "CREATE", Resource: "Problem", Status: "success"}))
			require.NoError(t, store.Write(Entry{UserID: 2, Action: "CREATE", Resource: "Problem", Status: "failure"}))
			require.NoError(t, store.Write(Entry{UserID: 1, Action: "CREATE", Resource: "Contest", Status: "success"}))

			entries, total, err := store.List(Filters{Resource: "Problem", Status: "success", UserID: 1}, 1, 20)
			require.NoError(t, err)
			require.EqualValues(t, 1, total)
			require.Len(t, entries, 1)
			require.Equal(t, "Problem", entries[0].Resource)
			require.Equal(t, uint(1), entries[0].UserID)
		})
	}
}

func TestStore_List_FilterByDateRange(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			require.NoError(t, store.Write(Entry{Action: "A", CreatedAt: base}))
			require.NoError(t, store.Write(Entry{Action: "B", CreatedAt: base.AddDate(0, 0, 5)}))
			require.NoError(t, store.Write(Entry{Action: "C", CreatedAt: base.AddDate(0, 0, 10)}))

			start := base.AddDate(0, 0, 2)
			end := base.AddDate(0, 0, 8)
			entries, total, err := store.List(Filters{Start: &start, End: &end}, 1, 20)
			require.NoError(t, err)
			require.EqualValues(t, 1, total)
			require.Len(t, entries, 1)
			require.Equal(t, "B", entries[0].Action)
		})
	}
}

func TestStore_List_Pagination(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < 5; i++ {
				require.NoError(t, store.Write(Entry{Action: "CREATE"}))
			}

			page1, total, err := store.List(Filters{}, 1, 2)
			require.NoError(t, err)
			require.EqualValues(t, 5, total)
			require.Len(t, page1, 2)

			page2, total, err := store.List(Filters{}, 2, 2)
			require.NoError(t, err)
			require.EqualValues(t, 5, total)
			require.Len(t, page2, 2)

			page3, total, err := store.List(Filters{}, 3, 2)
			require.NoError(t, err)
			require.EqualValues(t, 5, total)
			require.Len(t, page3, 1)

			// no overlap between pages
			require.NotEqual(t, page1[0].ID, page2[0].ID)
		})
	}
}

func TestStore_Get_ByID(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, store.Write(Entry{Action: "CREATE", Resource: "Problem", Username: "alice"}))

			entries, _, err := store.List(Filters{}, 1, 20)
			require.NoError(t, err)
			require.Len(t, entries, 1)

			got, err := store.Get(entries[0].ID)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, "CREATE", got.Action)
			require.Equal(t, "Problem", got.Resource)
			require.Equal(t, "alice", got.Username)
		})
	}
}

func TestStore_Get_Unknown_ReturnsNil(t *testing.T) {
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			got, err := store.Get("does-not-exist")
			require.NoError(t, err)
			require.Nil(t, got)
		})
	}
}

func TestStore_Get_UnknownButValidStreamID_ReturnsNil(t *testing.T) {
	// Redis stream IDs look like "<ms>-<seq>"; a well-formed but unused ID
	// must still resolve to "not found", not an error.
	for name, store := range storeFactories(t) {
		t.Run(name, func(t *testing.T) {
			got, err := store.Get("1-1")
			require.NoError(t, err)
			require.Nil(t, got)
		})
	}
}

// TestRedisStore_StreamLayout verifies the Redis-specific layout described
// in the design: entries are written to the "audit:log" stream via XADD.
func TestRedisStore_StreamLayout(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisStore(client)

	require.NoError(t, store.Write(Entry{Action: "CREATE", Resource: "Problem"}))

	require.True(t, mr.Exists("audit:log"), "expected stream key audit:log")

	length, err := client.XLen(context.Background(), "audit:log").Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, length)
}
