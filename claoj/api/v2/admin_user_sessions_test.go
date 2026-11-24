package v2

import (
	"context"
	"net/http"
	"testing"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// startAdminTestRedis points the permission cache at a fresh in-memory Redis
// so a test can observe BumpPermVersion's version increment. Every other test
// runs with cache.Client == nil (the no-Redis path).
func startAdminTestRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	oldClient, oldCtx := cache.Client, cache.Ctx
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache.Ctx = context.Background()
	t.Cleanup(func() { cache.Client, cache.Ctx = oldClient, oldCtx })
	return mr
}

// useRefreshStore installs an in-memory refresh-token store for the duration
// of the test and returns it.
func useRefreshStore(t *testing.T) tokenstore.Store {
	t.Helper()
	store := tokenstore.NewMemoryStore()
	old := authHandlers.RefreshStore
	authHandlers.RefreshStore = store
	t.Cleanup(func() { authHandlers.RefreshStore = old })
	return store
}

func seedRefreshToken(t *testing.T, store tokenstore.Store, authUserID uint, token string) {
	t.Helper()
	require.NoError(t, store.Save(token, tokenstore.Entry{
		UserID:    authUserID,
		FamilyID:  token + "-family",
		ExpiresAt: time.Now().Add(time.Hour),
	}))
}

func requireRevoked(t *testing.T, store tokenstore.Store, token string) {
	t.Helper()
	entry, found, err := store.Get(token)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, entry.Revoked, "expected refresh token %q to be revoked", token)
}

// Deactivating a user must end their sessions and drop the cached permission
// snapshot, otherwise an already-issued access/refresh pair keeps working.
func TestAdminUserUpdate_DeactivateRevokesSessionsAndBumpsPerms(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	store := useRefreshStore(t)
	seedRefreshToken(t, store, target.ID, "refresh-bob")
	mr := startAdminTestRedis(t)

	router := authedRouter(root.ID, http.MethodPatch, "/admin/user/:id", AdminUserUpdate)
	w := doRequest(router, http.MethodPatch, profileURL(targetProfile, ""), `{"is_active":false}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.AuthUser
	require.NoError(t, db.DB.First(&got, target.ID).Error)
	require.False(t, got.IsActive)
	requireRevoked(t, store, "refresh-bob")

	version, err := mr.Get("perm:version")
	require.NoError(t, err, "is_active changes must bump the permission cache version")
	require.Equal(t, "1", version)
}

// A save that echoes the current is_active back must not sign the user out.
func TestAdminUserUpdate_UnchangedIsActiveKeepsSessions(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	store := useRefreshStore(t)
	seedRefreshToken(t, store, target.ID, "refresh-bob")

	router := authedRouter(root.ID, http.MethodPatch, "/admin/user/:id", AdminUserUpdate)
	w := doRequest(router, http.MethodPatch, profileURL(targetProfile, ""), `{"is_active":true,"first_name":"Bob"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	entry, found, err := store.Get("refresh-bob")
	require.NoError(t, err)
	require.True(t, found)
	require.False(t, entry.Revoked, "an unchanged is_active must not revoke sessions")
}

func TestAdminUserDelete_RevokesSessions(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	store := useRefreshStore(t)
	seedRefreshToken(t, store, target.ID, "refresh-bob")

	router := authedRouter(root.ID, http.MethodDelete, "/admin/user/:id", AdminUserDelete)
	w := doRequest(router, http.MethodDelete, profileURL(targetProfile, ""), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.AuthUser
	require.NoError(t, db.DB.First(&got, target.ID).Error)
	require.False(t, got.IsActive)
	requireRevoked(t, store, "refresh-bob")
}

// Disabling 2FA must clear the last accepted TOTP timecode (otherwise it can
// gate a later re-enrolment) and must end sessions that predate the change.
func TestAdminUserSecurity_RevokesSessionsAndResetsTotpTimecode(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", targetProfile.ID).Updates(map[string]any{
		"is_totp_enabled": true, "totp_key": "enc-key", "scratch_codes": "enc-codes",
		"last_totp_timecode": 12345678,
	}).Error)

	store := useRefreshStore(t)
	seedRefreshToken(t, store, target.ID, "refresh-bob")

	router := authedRouter(root.ID, http.MethodPost, "/admin/user/:id/security", AdminUserSecurity)
	w := doRequest(router, http.MethodPost, profileURL(targetProfile, "/security"), `{"totp_enabled":false}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var p models.Profile
	require.NoError(t, db.DB.First(&p, targetProfile.ID).Error)
	require.False(t, p.IsTotpEnabled)
	require.Zero(t, p.LastTotpTimecode, "disabling TOTP must reset last_totp_timecode")
	requireRevoked(t, store, "refresh-bob")
}
