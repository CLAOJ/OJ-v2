package auth

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPermsDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{},
		&models.AuthGroupPermission{}, &models.AuthUserPermission{},
		&models.Profile{}, &models.Problem{}, &models.Contest{}, &models.Solution{},
		&models.Organization{},
	))
	db.DB = database
}

// seedPerm creates judge.<codename> and returns its ID.
func seedPerm(t *testing.T, codename string) uint {
	t.Helper()
	var ct models.DjangoContentType
	require.NoError(t, db.DB.FirstOrCreate(&ct, models.DjangoContentType{AppLabel: "judge", Model: "problem"}).Error)
	perm := models.AuthPermission{Name: codename, Codename: codename, ContentTypeID: ct.ID}
	require.NoError(t, db.DB.Create(&perm).Error)
	return perm.ID
}

func TestResolve_GroupAndDirectPerms(t *testing.T) {
	setupPermsDB(t)
	user := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)

	directID := seedPerm(t, "see_private_problem")
	groupPermID := seedPerm(t, "rejudge_submission")
	group := models.AuthGroup{Name: "Judges"}
	require.NoError(t, db.DB.Create(&group).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: user.ID, PermissionID: directID}).Error)
	require.NoError(t, db.DB.Create(&models.AuthGroupPermission{GroupID: group.ID, PermissionID: groupPermID}).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: user.ID, GroupID: group.ID}).Error)

	access, err := LoadUserAccess(user.ID)
	require.NoError(t, err)
	require.True(t, access.Perms.Has("judge.see_private_problem"))
	require.True(t, access.Perms.Has("judge.rejudge_submission"))
	require.False(t, access.Perms.Has("judge.edit_all_problem"))
}

func TestResolve_InactiveDeniesAll_SuperuserAllowsAll(t *testing.T) {
	setupPermsDB(t)
	inactive := models.AuthUser{Username: "banned", IsActive: false}
	super := models.AuthUser{Username: "root", IsActive: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&inactive).Error)
	require.NoError(t, db.DB.Create(&super).Error)

	a1, err := LoadUserAccess(inactive.ID)
	require.NoError(t, err)
	require.False(t, a1.HasPerm("judge.see_private_problem"))

	a2, err := LoadUserAccess(super.ID)
	require.NoError(t, err)
	require.True(t, a2.HasPerm("judge.anything_at_all"))
}

func TestAnonymousAccessDeniesAll(t *testing.T) {
	a := AnonymousAccess()
	require.False(t, a.HasPerm("judge.see_private_problem"))
	require.False(t, a.IsStaff)
}

// startMiniredis points cache.Client/cache.Ctx at a fresh in-memory Redis
// for the duration of the test, restoring cache.Client to nil (the
// no-Redis path used by every other test in this file) on cleanup.
func startMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache.Ctx = context.Background()
	t.Cleanup(func() { cache.Client = nil })
	return mr
}

func TestLoadUserAccess_RedisBackedCaching(t *testing.T) {
	setupPermsDB(t)
	mr := startMiniredis(t)

	user := models.AuthUser{Username: "redis-user", IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)
	permID := seedPerm(t, "see_private_problem")
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: user.ID, PermissionID: permID}).Error)

	access, err := LoadUserAccess(user.ID)
	require.NoError(t, err)
	require.True(t, access.HasPerm("judge.see_private_problem"))

	// The resolved access must have been written to Redis under the key
	// permCacheKey computes, so a second call is served from the cache.
	key := permCacheKey(user.ID)
	require.True(t, mr.Exists(key), "expected %q to be cached in redis", key)

	access2, err := LoadUserAccess(user.ID)
	require.NoError(t, err)
	require.True(t, access2.HasPerm("judge.see_private_problem"))
}

// TestBumpPermVersion_InvalidatesRedisCache is the regression test for the
// cold-start cache-version collision: permCacheKey's fallback version (used
// when permVersionKey is absent from Redis) must differ from "1", because
// Redis's INCR on an absent key returns 1. With a "1" fallback, the very
// first BumpPermVersion() after a fresh Redis would produce the *same*
// cache key as the pre-bump fallback, so this test would still observe the
// stale (pre-grant) permission set after the bump. It only passes because
// permCacheKey's fallback is "0".
func TestBumpPermVersion_InvalidatesRedisCache(t *testing.T) {
	setupPermsDB(t)
	startMiniredis(t)

	user := models.AuthUser{Username: "bump-user", IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)
	firstPermID := seedPerm(t, "see_private_problem")
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: user.ID, PermissionID: firstPermID}).Error)

	access, err := LoadUserAccess(user.ID)
	require.NoError(t, err)
	require.True(t, access.HasPerm("judge.see_private_problem"))
	require.False(t, access.HasPerm("judge.rejudge_submission"))

	// Grant a NEW permission directly in the DB, bypassing the cache.
	newPermID := seedPerm(t, "rejudge_submission")
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: user.ID, PermissionID: newPermID}).Error)

	BumpPermVersion()

	updated, err := LoadUserAccess(user.ID)
	require.NoError(t, err)
	require.True(t, updated.HasPerm("judge.rejudge_submission"),
		"BumpPermVersion did not invalidate the cache: newly granted permission is missing")
}

func TestGetAccess_NoUserID_ReturnsAnonymousAccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	access := GetAccess(c)
	require.NotNil(t, access)
	require.False(t, access.HasPerm("judge.see_private_problem"))
	require.False(t, access.IsStaff)
}

func TestGetAccess_WrongUserIDType_ReturnsAnonymousAccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "not-a-uint")

	require.NotPanics(t, func() {
		access := GetAccess(c)
		require.False(t, access.HasPerm("judge.see_private_problem"))
	})
}

func TestGetAccess_MemoizesOnContext(t *testing.T) {
	setupPermsDB(t)
	user := models.AuthUser{Username: "memo-user", IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", user.ID)

	first := GetAccess(c)
	second := GetAccess(c)
	require.Same(t, first, second, "GetAccess should return the memoized pointer on the second call")

	memoized, ok := c.Get(accessCtxKey)
	require.True(t, ok, "expected the resolved access to be memoized on the gin context")
	require.Same(t, first, memoized)
}

func TestHasPerm_GinContext(t *testing.T) {
	setupPermsDB(t)
	user := models.AuthUser{Username: "hasperm-user", IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)
	permID := seedPerm(t, "see_private_problem")
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: user.ID, PermissionID: permID}).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", user.ID)

	require.True(t, HasPerm(c, "judge.see_private_problem"))
	require.False(t, HasPerm(c, "judge.rejudge_submission"))
}
