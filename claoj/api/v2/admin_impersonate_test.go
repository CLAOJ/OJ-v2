package v2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auditlog"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// useAuditStore installs an in-memory audit store for the duration of the test.
func useAuditStore(t *testing.T) auditlog.Store {
	t.Helper()
	prev := auditlog.Default
	store := auditlog.NewMemoryStore()
	auditlog.Default = store
	t.Cleanup(func() { auditlog.Default = prev })
	return store
}

// impersonatingRouter authenticates the request as userID but also carries the
// impersonation context the real middleware sets from the JWT claims.
func impersonatingRouter(userID, impersonatorID uint, familyID, method, path string, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path, func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("impersonator_id", impersonatorID)
		c.Set("family_id", familyID)
		handler(c)
	})
	return router
}

func cookieValue(t *testing.T, w *httptest.ResponseRecorder, name string) string {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name && c.Value != "" {
			return c.Value
		}
	}
	t.Fatalf("cookie %s not set", name)
	return ""
}

func startImpersonation(t *testing.T, actorID, targetProfileID uint) *httptest.ResponseRecorder {
	t.Helper()
	router := authedRouter(actorID, http.MethodPost, "/admin/impersonate/:userId", AdminImpersonateStart)
	return doRequest(router, http.MethodPost, "/admin/impersonate/"+strconv.Itoa(int(targetProfileID)), "")
}

func TestAdminImpersonate_StartIssuesTargetTokensWithActorClaim(t *testing.T) {
	setupAdminUserCreateDB(t)
	useRefreshStore(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	w := startImpersonation(t, root.ID, targetProfile.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	access, err := auth.VerifyToken(cookieValue(t, w, "access_token"), "access")
	require.NoError(t, err)
	require.Equal(t, target.ID, access.UserID, "session must belong to the target")
	require.Equal(t, root.ID, access.ImpersonatorID)

	refresh, err := auth.VerifyToken(cookieValue(t, w, "refresh_token"), "refresh")
	require.NoError(t, err)
	require.Equal(t, root.ID, refresh.ImpersonatorID, "refresh token must carry the claim too")
	// Impersonated sessions get a short refresh TTL: the exit window is
	// exactly this long, so it must not be the usual 7 days.
	require.WithinDuration(t, time.Now().Add(auth.ImpersonationRefreshTTL),
		refresh.ExpiresAt.Time, time.Minute)

	entry, found, err := authHandlers.RefreshStore.Get(cookieValue(t, w, "refresh_token"))
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, target.ID, entry.UserID)
	require.WithinDuration(t, time.Now().Add(auth.ImpersonationRefreshTTL),
		entry.ExpiresAt, time.Minute)

	// The actor's own session must not be revoked by starting one.
	require.NotEqual(t, root.ID, entry.UserID)
}

func TestAdminImpersonate_ExitRestoresActorAndRevokesFamily(t *testing.T) {
	setupAdminUserCreateDB(t)
	store := useRefreshStore(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	w := startImpersonation(t, root.ID, targetProfile.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	impersonatedRefresh := cookieValue(t, w, "refresh_token")
	claims, err := auth.VerifyToken(impersonatedRefresh, "refresh")
	require.NoError(t, err)

	exitRouter := impersonatingRouter(target.ID, root.ID, claims.FamilyID, http.MethodPost, "/admin/impersonate/exit", AdminImpersonateExit)
	we := doRequest(exitRouter, http.MethodPost, "/admin/impersonate/exit", "")
	require.Equal(t, http.StatusOK, we.Code, we.Body.String())

	restored, err := auth.VerifyToken(cookieValue(t, we, "access_token"), "access")
	require.NoError(t, err)
	require.Equal(t, root.ID, restored.UserID)
	require.Zero(t, restored.ImpersonatorID, "restored session must not be impersonated")
	require.True(t, restored.IsAdmin)

	entry, found, err := store.Get(impersonatedRefresh)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, entry.Revoked, "the impersonated family must be revoked on exit")
}

// A deactivated actor must not get their session back on exit — and, more
// importantly, the impersonated family must already be revoked when the 403
// goes out, so the target session cannot outlive the refusal.
func TestAdminImpersonate_ExitAfterActorDeactivated_RevokesAndForbids(t *testing.T) {
	setupAdminUserCreateDB(t)
	store := useRefreshStore(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	w := startImpersonation(t, root.ID, targetProfile.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	impersonatedRefresh := cookieValue(t, w, "refresh_token")
	claims, err := auth.VerifyToken(impersonatedRefresh, "refresh")
	require.NoError(t, err)

	require.NoError(t, db.DB.Model(&models.AuthUser{}).Where("id = ?", root.ID).Update("is_active", false).Error)

	exitRouter := impersonatingRouter(target.ID, root.ID, claims.FamilyID, http.MethodPost, "/admin/impersonate/exit", AdminImpersonateExit)
	we := doRequest(exitRouter, http.MethodPost, "/admin/impersonate/exit", "")
	require.Equal(t, http.StatusForbidden, we.Code, we.Body.String())

	entry, found, err := store.Get(impersonatedRefresh)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, entry.Revoked, "family must be revoked even when the exit is refused")
}

func TestAdminImpersonate_ExitAfterActorDemoted_RevokesAndForbids(t *testing.T) {
	setupAdminUserCreateDB(t)
	store := useRefreshStore(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	w := startImpersonation(t, root.ID, targetProfile.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	impersonatedRefresh := cookieValue(t, w, "refresh_token")
	claims, err := auth.VerifyToken(impersonatedRefresh, "refresh")
	require.NoError(t, err)

	require.NoError(t, db.DB.Model(&models.AuthUser{}).Where("id = ?", root.ID).Update("is_superuser", false).Error)

	exitRouter := impersonatingRouter(target.ID, root.ID, claims.FamilyID, http.MethodPost, "/admin/impersonate/exit", AdminImpersonateExit)
	we := doRequest(exitRouter, http.MethodPost, "/admin/impersonate/exit", "")
	require.Equal(t, http.StatusForbidden, we.Code, we.Body.String())

	entry, found, err := store.Get(impersonatedRefresh)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, entry.Revoked, "family must be revoked even when the exit is refused")
}

func TestAdminImpersonate_StartRejectsInactiveActor(t *testing.T) {
	setupAdminUserCreateDB(t)
	useRefreshStore(t)
	root, _ := seedAccount(t, "root", true, true)
	_, targetProfile := seedAccount(t, "bob", false, false)

	require.NoError(t, db.DB.Model(&models.AuthUser{}).Where("id = ?", root.ID).Update("is_active", false).Error)

	w := startImpersonation(t, root.ID, targetProfile.ID)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
}

func TestAdminImpersonate_ExitWithoutImpersonation_Rejects(t *testing.T) {
	setupAdminUserCreateDB(t)
	useRefreshStore(t)
	target, _ := seedAccount(t, "bob", false, false)

	exitRouter := impersonatingRouter(target.ID, 0, "", http.MethodPost, "/admin/impersonate/exit", AdminImpersonateExit)
	we := doRequest(exitRouter, http.MethodPost, "/admin/impersonate/exit", "")
	require.Equal(t, http.StatusBadRequest, we.Code, we.Body.String())
}

func TestAdminImpersonate_Guards(t *testing.T) {
	setupAdminUserCreateDB(t)
	useRefreshStore(t)
	root, _ := seedAccount(t, "root", true, true)
	staff, _ := seedAccount(t, "staff", true, false)
	otherSuper, _ := seedAccount(t, "otherroot", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	w := startImpersonation(t, staff.ID, targetProfile.ID)
	require.Equal(t, http.StatusForbidden, w.Code, "non-superuser caller")

	w = startImpersonation(t, root.ID, targetProfile.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	cases := []struct {
		name   string
		actor  uint
		params string
		want   int
	}{
		{"self", root.ID, strconv.Itoa(int(profileIDOf(t, root.ID))), http.StatusBadRequest},
		{"superuser target", root.ID, strconv.Itoa(int(profileIDOf(t, otherSuper.ID))), http.StatusBadRequest},
		{"inactive target", root.ID, strconv.Itoa(int(targetProfile.ID)), http.StatusBadRequest},
	}
	require.NoError(t, db.DB.Model(&models.AuthUser{}).Where("id = ?", target.ID).Update("is_active", false).Error)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := authedRouter(tc.actor, http.MethodPost, "/admin/impersonate/:userId", AdminImpersonateStart)
			w := doRequest(router, http.MethodPost, "/admin/impersonate/"+tc.params, "")
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}

	// Nested impersonation is rejected before any token is issued.
	nested := impersonatingRouter(target.ID, root.ID, "fam", http.MethodPost, "/admin/impersonate/:userId", AdminImpersonateStart)
	w = doRequest(nested, http.MethodPost, "/admin/impersonate/"+strconv.Itoa(int(targetProfile.ID)), "")
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func profileIDOf(t *testing.T, authUserID uint) uint {
	t.Helper()
	var p models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", authUserID).First(&p).Error)
	return p.ID
}

// A refresh must keep the impersonation claim, otherwise the session silently
// turns into a normal login of the target after 15 minutes.
func TestRefresh_PreservesImpersonatorClaim(t *testing.T) {
	setupAdminUserCreateDB(t)
	useRefreshStore(t)
	root, _ := seedAccount(t, "root", true, true)
	_, targetProfile := seedAccount(t, "bob", false, false)

	w := startImpersonation(t, root.ID, targetProfile.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	refreshToken := cookieValue(t, w, "refresh_token")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/refresh", authHandlers.Refresh)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	wr := httptest.NewRecorder()
	router.ServeHTTP(wr, req)
	require.Equal(t, http.StatusOK, wr.Code, wr.Body.String())

	claims, err := auth.VerifyToken(cookieValue(t, wr, "access_token"), "access")
	require.NoError(t, err)
	require.Equal(t, root.ID, claims.ImpersonatorID)
}

func TestAdminImpersonate_WritesAuditRows(t *testing.T) {
	setupAdminUserCreateDB(t)
	useRefreshStore(t)
	store := useAuditStore(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	w := startImpersonation(t, root.ID, targetProfile.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	claims, err := auth.VerifyToken(cookieValue(t, w, "refresh_token"), "refresh")
	require.NoError(t, err)

	exitRouter := impersonatingRouter(target.ID, root.ID, claims.FamilyID, http.MethodPost, "/admin/impersonate/exit", AdminImpersonateExit)
	we := doRequest(exitRouter, http.MethodPost, "/admin/impersonate/exit", "")
	require.Equal(t, http.StatusOK, we.Code, we.Body.String())

	entries, _, err := store.List(auditlog.Filters{}, 1, 50)
	require.NoError(t, err)
	byAction := map[string]auditlog.Entry{}
	for _, e := range entries {
		byAction[e.Action] = e
	}
	start, ok := byAction["impersonate_start"]
	require.True(t, ok, "missing impersonate_start audit row")
	require.Equal(t, root.ID, start.UserID)
	require.Equal(t, "root", start.Username)
	require.Contains(t, start.Details, target.Username)

	exit, ok := byAction["impersonate_exit"]
	require.True(t, ok, "missing impersonate_exit audit row")
	require.Equal(t, root.ID, exit.UserID)
}

func TestCurrentUser_ReportsImpersonation(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, _ := seedAccount(t, "bob", true, false)

	router := impersonatingRouter(target.ID, root.ID, "fam", http.MethodGet, "/user/me", CurrentUser)
	_ = root
	w := doRequest(router, http.MethodGet, "/user/me", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "bob", body["username"])
	require.True(t, body["is_impersonating"].(bool))
	require.Equal(t, "root", body["impersonator_username"])
}
