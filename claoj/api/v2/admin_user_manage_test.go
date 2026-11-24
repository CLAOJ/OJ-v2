package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// ============================================================
// Admin user management: detail, edit, groups/permissions replace,
// 2FA disable, list filters/sort. Uses setupAdminUserCreateDB/seedAccount
// and authedRouter/doRequest from the neighbouring test files.
// ============================================================

func profileURL(p models.Profile, suffix string) string {
	return "/admin/user/" + strconv.Itoa(int(p.ID)) + suffix
}

func detailBody(t *testing.T, w interface{ Bytes() []byte }) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Bytes(), &body))
	return body
}

// The detail payload must carry identity/profile/group/permission/security
// state for the admin UI but never a secret column (password hash, TOTP key,
// scratch codes, API token).
func TestAdminUserDetail_ShapeAndNoSecrets(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	totpKey, scratch := "enc-totp-secret", "enc-scratch-secret"
	apiToken, about, notes := "api-secret-token", "about me", "staff note"
	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", targetProfile.ID).Updates(map[string]any{
		"about": about, "notes": notes, "is_totp_enabled": true, "totp_key": totpKey,
		"scratch_codes": scratch, "api_token": apiToken, "timezone": "Asia/Ho_Chi_Minh",
		"mute": true, "user_script": "console.log(1)",
	}).Error)

	g := models.AuthGroup{Name: "moderators"}
	require.NoError(t, db.DB.Create(&g).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: target.ID, GroupID: g.ID}).Error)

	ct := models.DjangoContentType{AppLabel: "judge", Model: "problem"}
	require.NoError(t, db.DB.Create(&ct).Error)
	perm := models.AuthPermission{Name: "Can add problem", ContentTypeID: ct.ID, Codename: "add_problem"}
	require.NoError(t, db.DB.Create(&perm).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: target.ID, PermissionID: perm.ID}).Error)
	require.NoError(t, db.DB.Create(&models.WebAuthnCredential{
		UserID: targetProfile.ID, Name: "yubikey", CredID: "cred-1", PublicKey: "pk",
	}).Error)

	router := authedRouter(root.ID, http.MethodGet, "/admin/user/:id", AdminUserDetail)
	w := doRequest(router, http.MethodGet, profileURL(targetProfile, ""), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	body := detailBody(t, w.Body)
	require.Equal(t, "bob", body["username"])
	require.Equal(t, "bob@example.com", body["email"])
	require.Equal(t, "Asia/Ho_Chi_Minh", body["timezone"])
	require.Equal(t, true, body["mute"])
	require.Equal(t, about, body["about"])
	require.Equal(t, notes, body["notes"])
	require.Equal(t, "console.log(1)", body["user_script"])
	require.Equal(t, true, body["is_totp_enabled"])
	require.Equal(t, float64(1), body["webauthn_credential_count"])

	require.Equal(t, []any{float64(g.ID)}, body["group_ids"])
	groups, ok := body["groups"].([]any)
	require.True(t, ok, "groups must be a JSON array")
	require.Len(t, groups, 1)
	require.Equal(t, "moderators", groups[0].(map[string]any)["name"])

	perms, ok := body["permissions"].([]any)
	require.True(t, ok, "permissions must be a JSON array")
	require.Len(t, perms, 1)
	require.Equal(t, "judge.add_problem", perms[0].(map[string]any)["codename"])

	for _, secretKey := range []string{"password", "totp_key", "scratch_codes", "api_token"} {
		_, present := body[secretKey]
		require.False(t, present, "secret field %q must not be serialized", secretKey)
	}
	raw := w.Body.String()
	for _, secret := range []string{"old-hash", totpKey, scratch, apiToken} {
		require.NotContains(t, raw, secret, "secret value leaked into the admin JSON")
	}
}

func TestAdminUserUpdate_IdentityAndProfile(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	router := authedRouter(root.ID, http.MethodPatch, "/admin/user/:id", AdminUserUpdate)
	w := doRequest(router, http.MethodPatch, profileURL(targetProfile, ""),
		`{"username":"bobby","email":"bobby@example.com","first_name":"Bob","last_name":"Tester",
		  "timezone":"Europe/Paris","language_id":1,"ace_theme":"monokai","site_theme":"dark",
		  "mute":true,"notes":"n","user_script":"// hi","username_display_override":"Bobby T",
		  "display_rank":"admin","ban_reason":"spam","is_unlisted":true}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var u models.AuthUser
	require.NoError(t, db.DB.First(&u, target.ID).Error)
	require.Equal(t, "bobby", u.Username)
	require.Equal(t, "bobby@example.com", u.Email)
	require.Equal(t, "Bob", u.FirstName)
	require.Equal(t, "Tester", u.LastName)

	var p models.Profile
	require.NoError(t, db.DB.First(&p, targetProfile.ID).Error)
	require.Equal(t, "Europe/Paris", p.Timezone)
	require.Equal(t, uint(1), p.LanguageID)
	require.Equal(t, "monokai", p.AceTheme)
	require.Equal(t, "dark", p.SiteTheme)
	require.True(t, p.Mute)
	require.Equal(t, "n", *p.Notes)
	require.Equal(t, "// hi", p.UserScript)
	require.Equal(t, "Bobby T", p.UsernameDisplayOverride)
	require.Equal(t, "admin", p.DisplayRank)
	require.Equal(t, "spam", *p.BanReason)
	require.True(t, p.IsUnlisted)
}

func TestAdminUserUpdate_Validation(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	seedAccount(t, "taken", false, false)
	_, targetProfile := seedAccount(t, "bob", false, false)

	router := authedRouter(root.ID, http.MethodPatch, "/admin/user/:id", AdminUserUpdate)
	cases := []struct {
		name string
		body string
		want int
	}{
		{"bad username", `{"username":"bad name!"}`, http.StatusBadRequest},
		{"too long username", `{"username":"` + strings.Repeat("a", 31) + `"}`, http.StatusBadRequest},
		{"bad email", `{"email":"nope"}`, http.StatusBadRequest},
		{"unknown language", `{"language_id":999}`, http.StatusBadRequest},
		{"duplicate username", `{"username":"taken","timezone":"Europe/Paris"}`, http.StatusConflict},
		{"duplicate email", `{"email":"taken@example.com"}`, http.StatusConflict},
		{"display-name email", `{"email":"Bob <bob@example.com>"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(router, http.MethodPatch, profileURL(targetProfile, ""), tc.body)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}

	// A rejected username must not have applied the rest of the same patch.
	var p models.Profile
	require.NoError(t, db.DB.First(&p, targetProfile.ID).Error)
	require.Equal(t, "UTC", p.Timezone, "a rejected patch must not apply any field")
	require.Nil(t, p.Notes)

	require.Equal(t, http.StatusBadRequest,
		doRequest(router, http.MethodPatch, "/admin/user/abc", `{"email":"x@example.com"}`).Code)
	require.Equal(t, http.StatusNotFound,
		doRequest(router, http.MethodPatch, "/admin/user/9999", `{"email":"x@example.com"}`).Code)
}

func TestAdminUserUpdate_SelfGuards(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, rootProfile := seedAccount(t, "root", true, true)
	_, otherProfile := seedAccount(t, "bob", false, false)

	router := authedRouter(root.ID, http.MethodPatch, "/admin/user/:id", AdminUserUpdate)
	self := profileURL(rootProfile, "")

	for _, body := range []string{
		`{"is_staff":false}`,
		`{"is_superuser":false}`,
		`{"is_active":false}`,
	} {
		w := doRequest(router, http.MethodPatch, self, body)
		require.Equal(t, http.StatusBadRequest, w.Code, body+" -> "+w.Body.String())
	}

	var got models.AuthUser
	require.NoError(t, db.DB.First(&got, root.ID).Error)
	require.True(t, got.IsStaff)
	require.True(t, got.IsSuperuser)
	require.True(t, got.IsActive)

	// Deactivating somebody else is still allowed.
	w := doRequest(router, http.MethodPatch, profileURL(otherProfile, ""), `{"is_active":false}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminUserUpdate_StaffFlagsNeedSuperuser(t *testing.T) {
	setupAdminUserCreateDB(t)
	staff, _ := seedAccount(t, "mod", true, false)
	target, targetProfile := seedAccount(t, "bob", false, false)

	router := authedRouter(staff.ID, http.MethodPatch, "/admin/user/:id", AdminUserUpdate)
	w := doRequest(router, http.MethodPatch, profileURL(targetProfile, ""), `{"is_staff":true}`)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	var got models.AuthUser
	require.NoError(t, db.DB.First(&got, target.ID).Error)
	require.False(t, got.IsStaff)
}

func TestAdminUserUpdate_RejectedPatchDoesNotCommitFlags(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	seedAccount(t, "taken", false, false)
	target, targetProfile := seedAccount(t, "bob", false, false)

	router := authedRouter(root.ID, http.MethodPatch, "/admin/user/:id", AdminUserUpdate)
	url := profileURL(targetProfile, "")

	// A taken username is only discovered in the service; the role flags in
	// the same patch must not be committed by a 409.
	w := doRequest(router, http.MethodPatch, url, `{"is_staff":true,"username":"taken"}`)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	var got models.AuthUser
	require.NoError(t, db.DB.First(&got, target.ID).Error)
	require.False(t, got.IsStaff, "a rejected patch must not set is_staff")

	// Same for validation that happens before the service (bad language_id).
	w = doRequest(router, http.MethodPatch, url, `{"is_superuser":true,"language_id":999}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&got, target.ID).Error)
	require.False(t, got.IsSuperuser, "a rejected patch must not set is_superuser")

	// A valid patch still writes both the auth_user and profile rows.
	w = doRequest(router, http.MethodPatch, url, `{"is_staff":true,"first_name":"Bob"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&got, target.ID).Error)
	require.True(t, got.IsStaff)
	require.Equal(t, "Bob", got.FirstName)
}

func TestAdminUserUpdate_StaffEditingPrivilegedTargetForbidden(t *testing.T) {
	setupAdminUserCreateDB(t)
	staff, _ := seedAccount(t, "mod", true, false)
	root, rootProfile := seedAccount(t, "root", true, true)
	_, plainProfile := seedAccount(t, "bob", false, false)

	staffRouter := authedRouter(staff.ID, http.MethodPatch, "/admin/user/:id", AdminUserUpdate)

	// Staff may not edit a superuser at all, not even their name.
	w := doRequest(staffRouter, http.MethodPatch, profileURL(rootProfile, ""), `{"first_name":"Hacked"}`)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	var got models.AuthUser
	require.NoError(t, db.DB.First(&got, root.ID).Error)
	require.Empty(t, got.FirstName)

	// Staff may still edit a regular user's identity/profile.
	w = doRequest(staffRouter, http.MethodPatch, profileURL(plainProfile, ""),
		`{"first_name":"Bobby","username":"bob2","timezone":"Europe/Paris"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// But not their account status.
	w = doRequest(staffRouter, http.MethodPatch, profileURL(plainProfile, ""), `{"is_active":false}`)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	// A superuser can edit a superuser.
	rootRouter := authedRouter(root.ID, http.MethodPatch, "/admin/user/:id", AdminUserUpdate)
	w = doRequest(rootRouter, http.MethodPatch, profileURL(rootProfile, ""), `{"first_name":"Rooty"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&got, root.ID).Error)
	require.Equal(t, "Rooty", got.FirstName)
}

func TestAdminUserReplaceGroups(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	groups := make([]models.AuthGroup, 3)
	for i := range groups {
		groups[i] = models.AuthGroup{Name: "g" + strconv.Itoa(i+1)}
		require.NoError(t, db.DB.Create(&groups[i]).Error)
	}
	require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: target.ID, GroupID: groups[0].ID}).Error)

	router := authedRouter(root.ID, http.MethodPut, "/admin/user/:id/groups", AdminUserReplaceGroups)
	url := profileURL(targetProfile, "/groups")

	w := doRequest(router, http.MethodPut, url,
		`{"group_ids":[`+strconv.Itoa(int(groups[1].ID))+`,`+strconv.Itoa(int(groups[2].ID))+`]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	memberships := func() []uint {
		var ids []uint
		require.NoError(t, db.DB.Model(&models.AuthUserGroup{}).Where("user_id = ?", target.ID).
			Order("group_id").Pluck("group_id", &ids).Error)
		return ids
	}
	require.Equal(t, []uint{groups[1].ID, groups[2].ID}, memberships())

	// Duplicate ids collapse to one join row (Django's through table has a
	// unique (user_id, group_id) constraint, so a repeat would fail the insert).
	w = doRequest(router, http.MethodPut, url,
		`{"group_ids":[`+strconv.Itoa(int(groups[1].ID))+`,`+strconv.Itoa(int(groups[1].ID))+`]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []uint{groups[1].ID}, memberships())

	// Unknown id is a 400 and leaves membership untouched.
	w = doRequest(router, http.MethodPut, url, `{"group_ids":[999]}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Equal(t, []uint{groups[1].ID}, memberships())

	// Missing field is a 400; an explicit empty list clears.
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPut, url, `{}`).Code)
	w = doRequest(router, http.MethodPut, url, `{"group_ids":[]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Empty(t, memberships())

	// Group/permission mutation stays superuser-only.
	staff, _ := seedAccount(t, "mod", true, false)
	staffRouter := authedRouter(staff.ID, http.MethodPut, "/admin/user/:id/groups", AdminUserReplaceGroups)
	require.Equal(t, http.StatusForbidden, doRequest(staffRouter, http.MethodPut, url, `{"group_ids":[]}`).Code)
}

func TestAdminUserReplacePermissions(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	ct := models.DjangoContentType{AppLabel: "judge", Model: "submission"}
	require.NoError(t, db.DB.Create(&ct).Error)
	perms := make([]models.AuthPermission, 2)
	for i := range perms {
		perms[i] = models.AuthPermission{
			Name: "perm " + strconv.Itoa(i), ContentTypeID: ct.ID,
			Codename: "perm_" + strconv.Itoa(i),
		}
		require.NoError(t, db.DB.Create(&perms[i]).Error)
	}

	router := authedRouter(root.ID, http.MethodPut, "/admin/user/:id/permissions", AdminUserReplacePermissions)
	url := profileURL(targetProfile, "/permissions")

	w := doRequest(router, http.MethodPut, url,
		`{"permission_ids":[`+strconv.Itoa(int(perms[0].ID))+`,`+strconv.Itoa(int(perms[1].ID))+`]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var ids []uint
	require.NoError(t, db.DB.Model(&models.AuthUserPermission{}).Where("user_id = ?", target.ID).
		Order("permission_id").Pluck("permission_id", &ids).Error)
	require.Equal(t, []uint{perms[0].ID, perms[1].ID}, ids)

	w = doRequest(router, http.MethodPut, url, `{"permission_ids":[999]}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Len(t, ids, 2, "a rejected replace must not change anything")

	w = doRequest(router, http.MethodPut, url, `{"permission_ids":[]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var count int64
	require.NoError(t, db.DB.Model(&models.AuthUserPermission{}).Where("user_id = ?", target.ID).Count(&count).Error)
	require.Zero(t, count)
}

func TestAdminUserSecurity_DisableOnly(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	_, targetProfile := seedAccount(t, "bob", false, false)

	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", targetProfile.ID).Updates(map[string]any{
		"is_totp_enabled": true, "totp_key": "enc-key", "scratch_codes": "enc-codes",
		"is_webauthn_enabled": true,
	}).Error)
	require.NoError(t, db.DB.Create(&models.WebAuthnCredential{
		UserID: targetProfile.ID, Name: "key", CredID: "cred-1", PublicKey: "pk",
	}).Error)

	router := authedRouter(root.ID, http.MethodPost, "/admin/user/:id/security", AdminUserSecurity)
	url := profileURL(targetProfile, "/security")

	// Enabling through the admin API is refused (must use the user's setup).
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, url, `{"totp_enabled":true}`).Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, url, `{"webauthn_enabled":true}`).Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, url, `{}`).Code)

	// Nothing was cleared by the refused requests.
	var still models.Profile
	require.NoError(t, db.DB.First(&still, targetProfile.ID).Error)
	require.NotNil(t, still.TotpKey)
	require.True(t, still.IsWebauthnEnabled)

	w := doRequest(router, http.MethodPost, url, `{"totp_enabled":false,"webauthn_enabled":false}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var p models.Profile
	require.NoError(t, db.DB.First(&p, targetProfile.ID).Error)
	require.False(t, p.IsTotpEnabled)
	require.Nil(t, p.TotpKey, "the encrypted TOTP key must be wiped")
	require.Nil(t, p.ScratchCodes, "the encrypted scratch codes must be wiped")
	require.False(t, p.IsWebauthnEnabled)
	var creds int64
	require.NoError(t, db.DB.Model(&models.WebAuthnCredential{}).Where("user_id = ?", targetProfile.ID).Count(&creds).Error)
	require.Zero(t, creds)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, false, body["is_totp_enabled"])
	require.Equal(t, false, body["is_webauthn_enabled"])
	require.Equal(t, float64(0), body["webauthn_credential_count"])

	staff, _ := seedAccount(t, "mod", true, false)
	staffRouter := authedRouter(staff.ID, http.MethodPost, "/admin/user/:id/security", AdminUserSecurity)
	require.Equal(t, http.StatusForbidden, doRequest(staffRouter, http.MethodPost, url, `{"totp_enabled":false}`).Code)
}

func TestAdminUserList_FiltersSortAndGroups(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)

	type seeded struct {
		user    models.AuthUser
		profile models.Profile
	}
	seed := func(name string, staff, superuser, active bool, points float64, joined time.Time) seeded {
		u := models.AuthUser{
			Username: name, Email: name + "@example.com", Password: "hash", IsActive: active,
			IsStaff: staff, IsSuperuser: superuser, DateJoined: joined, LastLogin: &joined,
		}
		require.NoError(t, db.DB.Create(&u).Error)
		// is_active has a DB default of 1, so GORM drops an explicit false on
		// insert; write it back to seed the inactive case.
		require.NoError(t, db.DB.Model(&models.AuthUser{}).Where("id = ?", u.ID).Update("is_active", active).Error)
		p := models.Profile{UserID: u.ID, Timezone: "UTC", Points: points}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
		return seeded{u, p}
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	seed("zed", false, false, true, 10, base)
	amy := seed("amy", true, false, true, 50, base.Add(time.Hour))
	seed("old", false, false, false, 0, base.Add(-time.Hour))

	g := models.AuthGroup{Name: "vip"}
	require.NoError(t, db.DB.Create(&g).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: amy.user.ID, GroupID: g.ID}).Error)

	router := authedRouter(root.ID, http.MethodGet, "/admin/users", AdminUserList)
	list := func(query string) (int, map[string]any) {
		w := doRequest(router, http.MethodGet, "/admin/users"+query, "")
		if w.Code != http.StatusOK {
			return w.Code, nil
		}
		return w.Code, detailBody(t, w.Body)
	}
	usernames := func(body map[string]any) []string {
		var names []string
		for _, row := range body["data"].([]any) {
			names = append(names, row.(map[string]any)["username"].(string))
		}
		return names
	}

	code, body := list("?staff=true")
	require.Equal(t, http.StatusOK, code)
	require.ElementsMatch(t, []string{"amy", "root"}, usernames(body))
	require.Equal(t, float64(2), body["total"])

	code, body = list("?active=false")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{"old"}, usernames(body))

	code, body = list("?superuser=true")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{"root"}, usernames(body))

	code, body = list("?group_id=" + strconv.Itoa(int(g.ID)))
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{"amy"}, usernames(body))

	code, body = list("?sort=username&order=asc")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{"amy", "old", "root", "zed"}, usernames(body))

	code, body = list("?sort=points&order=desc")
	require.Equal(t, http.StatusOK, code)
	names := usernames(body)
	require.Equal(t, []string{"amy", "zed"}, names[:2])
	require.ElementsMatch(t, []string{"root", "old"}, names[2:], "zero-point users tie")

	// Group names ride along with the page (one query, no N+1).
	code, body = list("?search=amy")
	require.Equal(t, http.StatusOK, code)
	row := body["data"].([]any)[0].(map[string]any)
	require.Equal(t, []any{"vip"}, []any{
		row["groups"].([]any)[0].(map[string]any)["name"],
	})
	require.Equal(t, float64(amy.profile.ID), row["id"], "list rows carry the judge_profile id")

	for _, bad := range []string{"?sort=email", "?order=up", "?staff=maybe", "?group_id=abc"} {
		require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodGet, "/admin/users"+bad, "").Code, bad)
	}
}
