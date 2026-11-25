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
// Public organization creation (v1 /organizations/create): the
// judge.add_organization gate, the admin-of limit and the caller becoming an
// administrator. Reuses setupAdminUserCreateDB / seedAccount /
// authedRouter / doRequest.
// ============================================================

func grantAddOrganization(t *testing.T, userID uint) {
	t.Helper()
	ct := models.DjangoContentType{AppLabel: "judge", Model: "organization"}
	require.NoError(t, db.DB.Create(&ct).Error)
	perm := models.AuthPermission{Name: "Can add organization", ContentTypeID: ct.ID, Codename: "add_organization"}
	require.NoError(t, db.DB.Create(&perm).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: userID, PermissionID: perm.ID}).Error)
}

func createOrganization(t *testing.T, userID uint, body string) (int, map[string]any) {
	t.Helper()
	router := authedRouter(userID, http.MethodPost, "/organizations", PublicOrganizationCreate)
	w := doRequest(router, http.MethodPost, "/organizations", body)
	var parsed map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed
}

func seedAdminOf(t *testing.T, profile models.Profile, name string) {
	t.Helper()
	org := models.Organization{Name: name, Slug: name, ShortName: name, CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&org).Error)
	require.NoError(t, db.DB.Model(&org).Association("Admins").Append(&profile))
}

func TestPublicOrganizationCreate_PermissionGate(t *testing.T) {
	setupAdminUserCreateDB(t)
	user, _ := seedAccount(t, "plain", false, false)

	code, _ := createOrganization(t, user.ID, `{"name":"Acme","slug":"acme"}`)
	require.Equal(t, http.StatusForbidden, code, "creating without judge.add_organization is refused")

	grantAddOrganization(t, user.ID)
	code, body := createOrganization(t, user.ID, `{"name":"Acme","slug":"acme"}`)
	require.Equal(t, http.StatusCreated, code, string(mustJSON(t, body)))

	var org models.Organization
	require.NoError(t, db.DB.Where("slug = ?", "acme").First(&org).Error)

	var profile models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", user.ID).First(&profile).Error)
	require.Equal(t, []uint{profile.ID}, dbOrgAdminIDs(t, org.ID), "the caller becomes an admin")
	require.True(t, org.IsUnlisted, "v1 creates unlisted organizations")
	require.Equal(t, "acme", org.ShortName, "v1 defaults the short name to the slug")
}

func TestPublicOrganizationCreate_AdminOfLimit(t *testing.T) {
	setupAdminUserCreateDB(t)
	user, profile := seedAccount(t, "orgfan", false, false)
	grantAddOrganization(t, user.ID)
	for i := 0; i < organizationAdminLimit; i++ {
		seedAdminOf(t, profile, "org"+strconv.Itoa(i))
	}

	code, _ := createOrganization(t, user.ID, `{"name":"One More","slug":"one-more"}`)
	require.Equal(t, http.StatusForbidden, code, "the v1 admin-of limit applies")

	// A superuser is exempt (v1's spam_organization escape only resolves for
	// superusers).
	root, _ := seedAccount(t, "root", true, true)
	for i := 0; i < organizationAdminLimit+1; i++ {
		seedAdminOf(t, profile, "extra"+strconv.Itoa(i))
	}
	code, _ = createOrganization(t, root.ID, `{"name":"Root Org","slug":"root-org"}`)
	require.Equal(t, http.StatusCreated, code)
}

func TestPublicOrganizationCreate_ExtraAdminsAndLongShortName(t *testing.T) {
	setupAdminUserCreateDB(t)
	user, _ := seedAccount(t, "creator", false, false)
	_, mate := seedAccount(t, "mate", false, false)
	grantAddOrganization(t, user.ID)

	code, body := createOrganization(t, user.ID, `{
		"name":"A Very Long Organization Name","slug":"a-very-long-organization-slug-name",
		"about":"about text","admins":[`+strconv.Itoa(int(mate.ID))+`]}`)
	require.Equal(t, http.StatusCreated, code, string(mustJSON(t, body)))

	var org models.Organization
	require.NoError(t, db.DB.Where("slug = ?", "a-very-long-organization-slug-name").First(&org).Error)
	require.Len(t, org.ShortName, 20, "v1 truncates the short name to 20 characters")
	require.Len(t, dbOrgAdminIDs(t, org.ID), 2, "the caller and the chosen co-admin")

	code, _ = createOrganization(t, user.ID, `{"name":"Bad","slug":"bad","admins":[99999]}`)
	require.Equal(t, http.StatusBadRequest, code)
}