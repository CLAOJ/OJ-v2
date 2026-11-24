package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
)

// setupRegistrationDB builds on the admin user test DB (auth_user,
// judge_profile, registration_registrationprofile) with a fresh in-memory
// token store and an email-helper spy.
func setupRegistrationDB(t *testing.T) *[]struct{ To, Username, Link string } {
	t.Helper()
	setupAdminUserCreateDB(t)
	oldTokens := authHandlers.OneTimeTokens
	authHandlers.OneTimeTokens = tokenstore.NewMemoryOneTime()
	t.Cleanup(func() { authHandlers.OneTimeTokens = oldTokens })

	var calls []struct{ To, Username, Link string }
	old := sendVerificationEmail
	sendVerificationEmail = func(to, username, link string) error {
		calls = append(calls, struct{ To, Username, Link string }{to, username, link})
		return nil
	}
	t.Cleanup(func() { sendVerificationEmail = old })
	return &calls
}

// seedInactiveAccount creates an auth_user + judge_profile that is awaiting
// email verification (is_active=0 cannot be set on insert because the column
// defaults to 1).
func seedInactiveAccount(t *testing.T, username string, joined time.Time) (models.AuthUser, models.Profile) {
	t.Helper()
	u, p := seedAccount(t, username, false, false)
	require.NoError(t, db.DB.Model(&models.AuthUser{}).Where("id = ?", u.ID).
		Updates(map[string]any{"is_active": false, "date_joined": joined}).Error)
	u.IsActive = false
	return u, p
}

func registrationItem(t *testing.T, body map[string]any, username string) map[string]any {
	t.Helper()
	items, ok := body["data"].([]any)
	require.True(t, ok, "data must be a JSON array")
	for _, raw := range items {
		item := raw.(map[string]any)
		if item["username"] == username {
			return item
		}
	}
	t.Fatalf("user %q not in list", username)
	return nil
}

func TestAdminRegistrationList_FiltersSearchAndState(t *testing.T) {
	setupRegistrationDB(t)
	root, _ := seedAccount(t, "root", true, true)
	active, _ := seedAccount(t, "verified", false, false)
	pending, _ := seedInactiveAccount(t, "pending1", time.Now().Add(-2*time.Hour))
	pending2, _ := seedInactiveAccount(t, "pending2", time.Now().Add(-time.Hour))

	require.NoError(t, db.DB.Create(&models.RegistrationProfile{
		UserID: active.ID, ActivationKey: "active-key", Activated: true,
	}).Error)
	var activeReg models.RegistrationProfile
	require.NoError(t, db.DB.Where("user_id = ?", active.ID).First(&activeReg).Error)
	require.NoError(t, db.DB.Create(&models.RegistrationProfile{
		UserID: pending.ID, ActivationKey: "pending-key", Activated: false,
	}).Error)

	var pendingReg models.RegistrationProfile
	require.NoError(t, db.DB.Where("user_id = ?", pending.ID).First(&pendingReg).Error)

	// Only pending2 has a live v2 verification token.
	require.NoError(t, authHandlers.OneTimeTokens.Issue(tokenstore.KindEmailVerify, "token-pending2", pending2.ID, time.Hour))

	router := authedRouter(root.ID, http.MethodGet, "/admin/registrations", AdminRegistrationList)

	w := doRequest(router, http.MethodGet, "/admin/registrations?status=pending", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := detailBody(t, w.Body)
	require.Equal(t, float64(2), body["total"])

	p1 := registrationItem(t, body, "pending1")
	require.Equal(t, false, p1["is_active"])
	require.Equal(t, float64(pending.ID), p1["user_id"], "user_id is the auth_user id used by the actions")
	require.NotContains(t, p1, "id", "the v1 profile id is only exposed under registration")
	require.Equal(t, false, p1["has_pending_token"])
	reg := p1["registration"].(map[string]any)
	require.Equal(t, float64(pendingReg.ID), reg["id"])
	require.Equal(t, false, reg["activated"])
	require.Equal(t, true, reg["has_activation_key"], "the key itself is never serialized")

	p2 := registrationItem(t, body, "pending2")
	require.Equal(t, true, p2["has_pending_token"])
	require.Nil(t, p2["registration"], "no v1 row means registration is null")

	w = doRequest(router, http.MethodGet, "/admin/registrations?status=active", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body = detailBody(t, w.Body)
	require.Equal(t, float64(2), body["total"], "root and verified are active")
	require.Equal(t, float64(activeReg.ID), registrationItem(t, body, "verified")["registration"].(map[string]any)["id"])
	require.Equal(t, true, registrationItem(t, body, "verified")["registration"].(map[string]any)["activated"])

	w = doRequest(router, http.MethodGet, "/admin/registrations?search=pending1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, float64(1), detailBody(t, w.Body)["total"])

	// LIKE wildcards in the search term are escaped, not interpreted.
	w = doRequest(router, http.MethodGet, "/admin/registrations?search=%25", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, float64(0), detailBody(t, w.Body)["total"])

	w = doRequest(router, http.MethodGet, "/admin/registrations?status=nope", "")
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = doRequest(router, http.MethodGet, "/admin/registrations?order=up", "")
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Staff (not superuser) can read the list.
	staff, _ := seedAccount(t, "staff", true, false)
	router = authedRouter(staff.ID, http.MethodGet, "/admin/registrations", AdminRegistrationList)
	w = doRequest(router, http.MethodGet, "/admin/registrations", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminRegistrationActivate_IdempotentAndRevokesTokens(t *testing.T) {
	setupRegistrationDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, _ := seedInactiveAccount(t, "waiting", time.Now())
	require.NoError(t, db.DB.Create(&models.RegistrationProfile{
		UserID: target.ID, ActivationKey: "k", Activated: false,
	}).Error)
	require.NoError(t, authHandlers.OneTimeTokens.Issue(tokenstore.KindEmailVerify, "tok", target.ID, time.Hour))

	url := "/admin/registration/" + strconv.Itoa(int(target.ID)) + "/activate"
	router := authedRouter(root.ID, http.MethodPost, "/admin/registration/:userId/activate", AdminRegistrationActivate)

	w := doRequest(router, http.MethodPost, url, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var u models.AuthUser
	require.NoError(t, db.DB.First(&u, target.ID).Error)
	require.True(t, u.IsActive)

	var reg models.RegistrationProfile
	require.NoError(t, db.DB.Where("user_id = ?", target.ID).First(&reg).Error)
	require.True(t, reg.Activated)
	require.Equal(t, "", reg.ActivationKey, "v1 activation blanks the key so the link stops working")

	has, err := authHandlers.OneTimeTokens.HasOutstanding(tokenstore.KindEmailVerify, target.ID)
	require.NoError(t, err)
	require.False(t, has, "verification tokens are revoked on activation")

	w = doRequest(router, http.MethodPost, url, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, true, detailBody(t, w.Body)["already_active"])

	w = doRequest(router, http.MethodPost, "/admin/registration/999999/activate", "")
	require.Equal(t, http.StatusNotFound, w.Code)
}

// Most accounts come from v2 registration and have no v1 row at all; activate
// must still work (the v1 update simply matches no rows).
func TestAdminRegistrationActivate_WithoutV1Row(t *testing.T) {
	setupRegistrationDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, _ := seedInactiveAccount(t, "waiting", time.Now())

	router := authedRouter(root.ID, http.MethodPost, "/admin/registration/:userId/activate", AdminRegistrationActivate)
	w := doRequest(router, http.MethodPost, "/admin/registration/"+strconv.Itoa(int(target.ID))+"/activate", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var u models.AuthUser
	require.NoError(t, db.DB.First(&u, target.ID).Error)
	require.True(t, u.IsActive)
}

// If the v1 write fails the auth_user update must roll back: a half-applied
// activation would leave the account active with a live-looking v1 row.
func TestAdminRegistrationActivate_RollsBackOnV1Failure(t *testing.T) {
	setupRegistrationDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, _ := seedInactiveAccount(t, "waiting", time.Now())
	require.NoError(t, db.DB.Migrator().DropTable(&models.RegistrationProfile{}))

	router := authedRouter(root.ID, http.MethodPost, "/admin/registration/:userId/activate", AdminRegistrationActivate)
	w := doRequest(router, http.MethodPost, "/admin/registration/"+strconv.Itoa(int(target.ID))+"/activate", "")
	require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())

	var u models.AuthUser
	require.NoError(t, db.DB.First(&u, target.ID).Error)
	require.False(t, u.IsActive, "a failed v1 write must roll the activation back")
}

func TestAdminRegistrationMutations_SuperuserOnly(t *testing.T) {
	setupRegistrationDB(t)
	staff, _ := seedAccount(t, "staff", true, false)
	target, _ := seedInactiveAccount(t, "waiting", time.Now())

	activate := authedRouter(staff.ID, http.MethodPost, "/admin/registration/:userId/activate", AdminRegistrationActivate)
	resend := authedRouter(staff.ID, http.MethodPost, "/admin/registration/:userId/resend", AdminRegistrationResend)
	url := "/admin/registration/" + strconv.Itoa(int(target.ID))

	require.Equal(t, http.StatusForbidden, doRequest(activate, http.MethodPost, url+"/activate", "").Code)
	require.Equal(t, http.StatusForbidden, doRequest(resend, http.MethodPost, url+"/resend", "").Code)

	var u models.AuthUser
	require.NoError(t, db.DB.First(&u, target.ID).Error)
	require.False(t, u.IsActive, "a refused activate must not write")
}

func TestAdminRegistrationResend_GuardsAndTokenIssuance(t *testing.T) {
	calls := setupRegistrationDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, _ := seedInactiveAccount(t, "waiting", time.Now())

	router := authedRouter(root.ID, http.MethodPost, "/admin/registration/:userId/resend", AdminRegistrationResend)
	url := "/admin/registration/" + strconv.Itoa(int(target.ID)) + "/resend"

	w := doRequest(router, http.MethodPost, url, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	has, err := authHandlers.OneTimeTokens.HasOutstanding(tokenstore.KindEmailVerify, target.ID)
	require.NoError(t, err)
	require.True(t, has, "resend issues a fresh verification token")
	require.Len(t, *calls, 1)
	require.Equal(t, target.Email, (*calls)[0].To)
	require.Equal(t, target.Username, (*calls)[0].Username)
	require.Contains(t, (*calls)[0].Link, "/verify-email?token=")

	// An active account has nothing to verify.
	require.NoError(t, db.DB.Model(&models.AuthUser{}).Where("id = ?", target.ID).Update("is_active", true).Error)
	w = doRequest(router, http.MethodPost, url, "")
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Len(t, *calls, 1)

	w = doRequest(router, http.MethodPost, "/admin/registration/999999/resend", "")
	require.Equal(t, http.StatusNotFound, w.Code)
}

// Guards against the response accidentally serializing the v1 activation key:
// it is a live credential in v1.
func TestAdminRegistrationList_DoesNotLeakActivationKey(t *testing.T) {
	setupRegistrationDB(t)
	root, _ := seedAccount(t, "root", true, true)
	pending, _ := seedInactiveAccount(t, "pending", time.Now())
	require.NoError(t, db.DB.Create(&models.RegistrationProfile{
		UserID: pending.ID, ActivationKey: "super-secret-key", Activated: false,
	}).Error)

	router := authedRouter(root.ID, http.MethodGet, "/admin/registrations", AdminRegistrationList)
	w := doRequest(router, http.MethodGet, "/admin/registrations?status=pending", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "super-secret-key")

	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
}
