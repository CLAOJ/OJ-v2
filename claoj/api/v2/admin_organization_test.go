package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// ============================================================
// Admin organization detail/create/update, the judge.organization_admin
// field gate and points recalculation. Reuses setupAdminUserCreateDB /
// seedAccount / authedRouter / doRequest.
// ============================================================

func seedOrganization(t *testing.T, name string) models.Organization {
	t.Helper()
	org := models.Organization{
		Name: name, Slug: name, ShortName: name,
		CreationDate: time.Date(2021, 5, 6, 7, 8, 9, 0, time.UTC),
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&org).Error)
	return org
}

func orgURL(org models.Organization, suffix string) string {
	return "/admin/organization/" + strconv.Itoa(int(org.ID)) + suffix
}

func grantOrgAdminPerm(t *testing.T, userID uint) {
	t.Helper()
	ct := models.DjangoContentType{AppLabel: "judge", Model: "organization"}
	require.NoError(t, db.DB.Create(&ct).Error)
	perm := models.AuthPermission{Name: "Can administrate organization", ContentTypeID: ct.ID, Codename: "organization_admin"}
	require.NoError(t, db.DB.Create(&perm).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: userID, PermissionID: perm.ID}).Error)
}

func dbOrgAdminIDs(t *testing.T, orgID uint) []uint {
	t.Helper()
	var ids []uint
	require.NoError(t, db.DB.Table("judge_organization_admins").
		Where("organization_id = ?", orgID).Order("profile_id").Pluck("profile_id", &ids).Error)
	return ids
}

func TestAdminOrganizationDetail_Shape(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	_, adminProfile := seedAccount(t, "ada", false, false)

	org := seedOrganization(t, "acme")
	slots, accessCode := 7, "secret7"
	require.NoError(t, db.DB.Model(&org).Updates(map[string]any{
		"about": "<p>hi</p>", "is_open": true, "slots": slots, "access_code": accessCode,
		"logo_override_image": "/logo/acme.png", "member_count": 3, "performance_points": 12.5,
	}).Error)
	require.NoError(t, db.DB.Model(&org).Association("Admins").Append(&adminProfile))

	router := authedRouter(root.ID, http.MethodGet, "/admin/organization/:id", AdminOrganizationDetail)
	w := doRequest(router, http.MethodGet, orgURL(org, ""), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	body := detailBody(t, w.Body)
	require.Equal(t, float64(org.ID), body["id"])
	require.Equal(t, "acme", body["name"])
	require.Equal(t, "<p>hi</p>", body["about"])
	require.Equal(t, true, body["is_open"])
	require.Equal(t, float64(slots), body["slots"])
	require.Equal(t, accessCode, body["access_code"])
	require.Equal(t, "/logo/acme.png", body["logo_override_image"])
	require.Equal(t, float64(3), body["member_count"])
	require.Equal(t, 12.5, body["performance_points"])

	var stored models.Organization
	require.NoError(t, db.DB.First(&stored, org.ID).Error)
	require.Equal(t, stored.CreationDate.Format(time.RFC3339), body["creation_date"])

	require.Equal(t, []any{map[string]any{"id": float64(adminProfile.ID), "username": "ada"}}, body["admins"])
	require.NotContains(t, body, "organization", "detail must be flat")
}

func TestAdminOrganizationDetail_NotFoundAndBadID(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	router := authedRouter(root.ID, http.MethodGet, "/admin/organization/:id", AdminOrganizationDetail)

	w := doRequest(router, http.MethodGet, "/admin/organization/999", "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	w = doRequest(router, http.MethodGet, "/admin/organization/nope", "")
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestAdminOrganizationCreateUpdate_RoundTrip(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	_, ada := seedAccount(t, "ada", false, false)
	_, bob := seedAccount(t, "bob", false, false)

	router := authedRouter(root.ID, http.MethodPost, "/admin/organizations", AdminOrganizationCreate)
	w := doRequest(router, http.MethodPost, "/admin/organizations", `{
		"name":"Round Trip","slug":"round-trip","short_name":"RT","about":"about",
		"is_open":true,"is_unlisted":true,"slots":5,"access_code":"code123",
		"logo_override_image":"/logo/rt.png","admins":[`+strconv.Itoa(int(ada.ID))+`]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	created := detailBody(t, w.Body)
	require.Equal(t, "Round Trip", created["name"])
	require.NotContains(t, created, "organization", "create returns the flat detail body")
	require.Len(t, created["admins"], 1)

	var org models.Organization
	require.NoError(t, db.DB.Where("slug = ?", "round-trip").First(&org).Error)
	require.Equal(t, "/logo/rt.png", org.LogoOverrideImage)
	require.Equal(t, []uint{ada.ID}, dbOrgAdminIDs(t, org.ID))

	update := authedRouter(root.ID, http.MethodPatch, "/admin/organization/:id", AdminOrganizationUpdate)
	w = doRequest(update, http.MethodPatch, orgURL(org, ""), `{
		"logo_override_image":"/logo/rt2.png","admins":[`+strconv.Itoa(int(bob.ID))+`],
		"is_open":false,"slots":9}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	updated := detailBody(t, w.Body)
	require.Equal(t, "/logo/rt2.png", updated["logo_override_image"])
	require.NotContains(t, updated, "organization", "update returns the flat detail body")

	require.NoError(t, db.DB.First(&org, org.ID).Error)
	require.Equal(t, "/logo/rt2.png", org.LogoOverrideImage)
	require.False(t, org.IsOpen)
	require.Equal(t, 9, *org.Slots)
	require.Equal(t, []uint{bob.ID}, dbOrgAdminIDs(t, org.ID), "admins are replaced, not merged")

	// An explicit empty list clears the admins.
	w = doRequest(update, http.MethodPatch, orgURL(org, ""), `{"admins":[]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Empty(t, dbOrgAdminIDs(t, org.ID))
}

// A patch that does not mention admins must not touch the m2m table; the
// empty-slice case is covered by the round-trip test above.
func TestAdminOrganizationUpdate_OmittingAdminsKeepsMembership(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	_, ada := seedAccount(t, "ada", false, false)
	_, bob := seedAccount(t, "bob", false, false)
	org := seedOrganization(t, "keep")
	require.NoError(t, db.DB.Model(&org).Association("Admins").Append(&ada, &bob))

	router := authedRouter(root.ID, http.MethodPatch, "/admin/organization/:id", AdminOrganizationUpdate)
	w := doRequest(router, http.MethodPatch, orgURL(org, ""), `{"name":"Keep2"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, []uint{ada.ID, bob.ID}, dbOrgAdminIDs(t, org.ID),
		"a patch without admins must leave membership intact")
	require.Len(t, detailBody(t, w.Body)["admins"], 2)
}

// The is_unlisted column has a DB default of 1, so a plain GORM Create would
// store true even when the request says false. The create form always sends
// the value, so false must persist as false.
func TestAdminOrganizationCreate_UnlistedPersists(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)

	router := authedRouter(root.ID, http.MethodPost, "/admin/organizations", AdminOrganizationCreate)
	create := func(body string) map[string]any {
		w := doRequest(router, http.MethodPost, "/admin/organizations", body)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		return detailBody(t, w.Body)
	}

	listed := create(`{"name":"Listed","slug":"listed","short_name":"L","is_unlisted":false}`)
	require.Equal(t, false, listed["is_unlisted"])
	var listedRow models.Organization
	require.NoError(t, db.DB.Where("slug = ?", "listed").First(&listedRow).Error)
	require.False(t, listedRow.IsUnlisted)

	hidden := create(`{"name":"Hidden","slug":"hidden","short_name":"H","is_unlisted":true}`)
	require.Equal(t, true, hidden["is_unlisted"])
	var hiddenRow models.Organization
	require.NoError(t, db.DB.Where("slug = ?", "hidden").First(&hiddenRow).Error)
	require.True(t, hiddenRow.IsUnlisted)
}

// v1 keeps admins/is_open/slots readonly on create without
// judge.organization_admin; the create handler must 403 before inserting.
func TestAdminOrganizationCreate_RestrictedFieldsGate(t *testing.T) {
	setupAdminUserCreateDB(t)
	_, plainUser := seedAccount(t, "staffer", true, false)
	_, ada := seedAccount(t, "ada", false, false)

	router := authedRouter(plainUser.ID, http.MethodPost, "/admin/organizations", AdminOrganizationCreate)
	bodies := []string{
		`{"name":"Open","slug":"open","short_name":"O","is_open":true}`,
		`{"name":"Slots","slug":"slots","short_name":"S","slots":5}`,
		`{"name":"Admins","slug":"admins","short_name":"A","admins":[` + strconv.Itoa(int(ada.ID)) + `]}`,
	}
	for _, body := range bodies {
		w := doRequest(router, http.MethodPost, "/admin/organizations", body)
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	}

	var count int64
	require.NoError(t, db.DB.Model(&models.Organization{}).Count(&count).Error)
	require.Zero(t, count, "a rejected create must not insert the organization")

	w := doRequest(router, http.MethodPost, "/admin/organizations", `{"name":"Plain","slug":"plain","short_name":"P"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminOrganizationUpdate_ClearSlots(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	org := seedOrganization(t, "slotted")
	require.NoError(t, db.DB.Model(&org).Update("slots", 5).Error)

	router := authedRouter(root.ID, http.MethodPatch, "/admin/organization/:id", AdminOrganizationUpdate)
	w := doRequest(router, http.MethodPatch, orgURL(org, ""), `{"clear_slots":true}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var stored models.Organization
	require.NoError(t, db.DB.First(&stored, org.ID).Error)
	require.Nil(t, stored.Slots)
}

func TestAdminOrganizationUpdate_CreationDateIgnored(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	org := seedOrganization(t, "dated")

	router := authedRouter(root.ID, http.MethodPatch, "/admin/organization/:id", AdminOrganizationUpdate)
	w := doRequest(router, http.MethodPatch, orgURL(org, ""), `{"name":"Dated2","creation_date":"2000-01-01T00:00:00Z"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var stored models.Organization
	require.NoError(t, db.DB.First(&stored, org.ID).Error)
	require.Equal(t, "Dated2", stored.Name)
	require.Equal(t, org.CreationDate, stored.CreationDate, "creation_date is auto_now_add in Django")
}

// v1 keeps admins/is_open/slots readonly without judge.organization_admin:
// echoing them back passes, changing any of them is 403.
func TestAdminOrganizationUpdate_RestrictedFieldsGate(t *testing.T) {
	setupAdminUserCreateDB(t)
	_, plainUser := seedAccount(t, "orgadmin", true, false)
	_, adminProfile := seedAccount(t, "ada", false, false)
	org := seedOrganization(t, "gated")
	require.NoError(t, db.DB.Model(&org).Update("slots", 5).Error)
	require.NoError(t, db.DB.Model(&org).Association("Admins").Append(&adminProfile))

	withoutPerm := authedRouter(plainUser.ID, http.MethodPatch, "/admin/organization/:id", AdminOrganizationUpdate)
	adminJSON := strconv.Itoa(int(adminProfile.ID))

	cases := []struct {
		name string
		body string
		want int
	}{
		{"echoed values pass", `{"name":"Gated","is_open":false,"admins":[` + adminJSON + `]}`, http.StatusOK},
		{"open status change", `{"is_open":true}`, http.StatusForbidden},
		{"slots change", `{"slots":4}`, http.StatusForbidden},
		{"slots clear", `{"clear_slots":true}`, http.StatusForbidden},
		{"admins change", `{"admins":[]}`, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(withoutPerm, http.MethodPatch, orgURL(org, ""), tc.body)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}

	grantOrgAdminPerm(t, plainUser.ID)
	withPerm := authedRouter(plainUser.ID, http.MethodPatch, "/admin/organization/:id", AdminOrganizationUpdate)
	w := doRequest(withPerm, http.MethodPatch, orgURL(org, ""), `{"is_open":true,"slots":4,"admins":[]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var stored models.Organization
	require.NoError(t, db.DB.First(&stored, org.ID).Error)
	require.True(t, stored.IsOpen)
	require.Equal(t, 4, *stored.Slots)
	require.Empty(t, dbOrgAdminIDs(t, org.ID))

	w = doRequest(withPerm, http.MethodPatch, orgURL(org, ""), `{"clear_slots":true}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&stored, org.ID).Error)
	require.Nil(t, stored.Slots)
}

// GET /user/me exposes can_organization_admin so the FE can disable the
// admin/slots/open controls; superusers pass through HasPerm, and the flag
// must follow a granted judge.organization_admin permission.
func TestCurrentUser_CanOrganizationAdmin(t *testing.T) {
	setupAdminUserCreateDB(t)
	superuser, _ := seedAccount(t, "root", true, true)
	staff, _ := seedAccount(t, "staffer", true, false)

	flag := func(userID uint) any {
		router := authedRouter(userID, http.MethodGet, "/user/me", CurrentUser)
		w := doRequest(router, http.MethodGet, "/user/me", "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body["can_organization_admin"]
	}

	require.Equal(t, true, flag(superuser.ID), "superusers hold every permission")
	require.Equal(t, false, flag(staff.ID))

	grantOrgAdminPerm(t, staff.ID)
	require.Equal(t, true, flag(staff.ID))
}

// GET /user/me exposes the user's granted Django permission codenames so
// permission-gated controls (e.g. submission lock/archive, the suggest link)
// can be enabled per user - for any authenticated user, not only staff.
// Superusers report is_superuser; their permissions list stays limited to real
// grants instead of the whole catalogue.
func TestCurrentUser_Permissions(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	granted, _ := seedAccount(t, "granted", false, false)
	other, _ := seedAccount(t, "other", false, false)
	grantOrgAdminPerm(t, granted.ID)

	perms := func(userID uint) []any {
		router := authedRouter(userID, http.MethodGet, "/user/me", CurrentUser)
		w := doRequest(router, http.MethodGet, "/user/me", "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		list, ok := body["permissions"].([]any)
		require.True(t, ok, "permissions must be a JSON array")
		return list
	}

	require.Equal(t, []any{"judge.organization_admin"}, perms(granted.ID),
		"a non-staff perm holder reports their own grant")
	require.Empty(t, perms(other.ID), "a user without the grant must not report it")
	require.Empty(t, perms(root.ID), "superusers get their real grants, not the catalogue")
}

func TestAdminOrganizationUpdate_UnknownAdminIDs(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	org := seedOrganization(t, "badids")

	router := authedRouter(root.ID, http.MethodPatch, "/admin/organization/:id", AdminOrganizationUpdate)
	w := doRequest(router, http.MethodPatch, orgURL(org, ""), `{"admins":[777]}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "777")

	w = doRequest(router, http.MethodPatch, "/admin/organization/999", `{"name":"x"}`)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

// CalculateOrganizationPoints sums the org's members' performance points, so a
// member with 100 PP must move the org from 0 to 100.
func TestAdminOrganizationRecalculate(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	_, memberProfile := seedAccount(t, "member", false, false)
	org := seedOrganization(t, "scored")

	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", memberProfile.ID).
		Update("performance_points", 100).Error)
	require.NoError(t, db.DB.Model(&org).Association("Members").Append(&memberProfile))

	router := authedRouter(root.ID, http.MethodPost, "/admin/organization/:id/recalculate", AdminOrganizationRecalculate)
	w := doRequest(router, http.MethodPost, orgURL(org, "/recalculate"), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 100.0, detailBody(t, w.Body)["performance_points"])

	var stored models.Organization
	require.NoError(t, db.DB.First(&stored, org.ID).Error)
	require.Equal(t, 100.0, stored.PerformancePoints)

	w = doRequest(router, http.MethodPost, "/admin/organization/999/recalculate", "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}
