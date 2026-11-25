package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// ============================================================
// Organization self-service: org-scoped content creation (v1
// *CreateOrganization views), join-with-reason requests and the request
// state tabs. Reuses setupAdminUserCreateDB / seedAccount / authedRouter /
// doRequest.
// ============================================================

type orgSelfServiceFixture struct {
	org, otherOrg models.Organization

	adminUser    models.AuthUser
	admin        models.Profile // org admin, holds the three create perms
	permUser     models.AuthUser
	permOnly     models.Profile // create perms, not an org admin
	strangerUser models.AuthUser
	stranger     models.Profile
	staffUser    models.AuthUser // staff without any create perm
	staff        models.Profile
	superUser    models.AuthUser
	superProfile models.Profile
	group        models.ProblemGroup
}

var orgCreatePerms = []string{
	"judge.create_organization_problem",
	"judge.create_private_contest",
	"judge.edit_organization_post",
}

// grantOrgCreatePerms gives the user the three organization create permissions
// directly (judge.* codenames under one content type).
func grantOrgCreatePerms(t *testing.T, userID uint) {
	t.Helper()
	ct := models.DjangoContentType{AppLabel: "judge", Model: "organization"}
	require.NoError(t, db.DB.Create(&ct).Error)
	for _, codename := range orgCreatePerms {
		perm := models.AuthPermission{Name: codename, ContentTypeID: ct.ID, Codename: codename[6:]}
		require.NoError(t, db.DB.Create(&perm).Error)
		require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: userID, PermissionID: perm.ID}).Error)
	}
}

func setupOrgSelfServiceDB(t *testing.T) orgSelfServiceFixture {
	t.Helper()
	setupAdminUserCreateDB(t)
	require.NoError(t, db.DB.AutoMigrate(
		&models.ProblemGroup{}, &models.Problem{}, &models.ProblemType{},
		&models.BlogPost{}, &models.Contest{}, &models.ContestTag{},
		&models.OrganizationRequest{},
	))

	org := models.Organization{Name: "acme", Slug: "acme", ShortName: "ACME", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&org).Error)
	otherOrg := models.Organization{Name: "other", Slug: "other", ShortName: "OTHER", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&otherOrg).Error)

	adminUser, admin := seedAccount(t, "orgadmin", false, false)
	permUser, permOnly := seedAccount(t, "permuser", false, false)
	strangerUser, stranger := seedAccount(t, "stranger", false, false)
	staffUser, staff := seedAccount(t, "staff", true, false)
	superUser, superProfile := seedAccount(t, "root", true, true)

	require.NoError(t, db.DB.Model(&org).Association("Admins").Append(&admin))
	require.NoError(t, db.DB.Model(&org).Association("Members").Append(&admin))
	grantOrgCreatePerms(t, adminUser.ID)
	grantOrgCreatePerms(t, permUser.ID)

	group := models.ProblemGroup{Name: "g", FullName: "Group"}
	require.NoError(t, db.DB.Create(&group).Error)

	return orgSelfServiceFixture{
		org: org, otherOrg: otherOrg,
		adminUser: adminUser, admin: admin,
		permUser: permUser, permOnly: permOnly,
		strangerUser: strangerUser, stranger: stranger,
		staffUser: staffUser, staff: staff,
		superUser: superUser, superProfile: superProfile,
		group: group,
	}
}

func orgCreateRequest(t *testing.T, userID uint, method, path, kind string, handler gin.HandlerFunc, body string) (int, []byte) {
	t.Helper()
	router := authedRouter(userID, method, "/organization/:id/"+kind, handler)
	w := doRequest(router, method, path, body)
	return w.Code, w.Body.Bytes()
}

func orgProblemBody(fx orgSelfServiceFixture, code string, orgIDs []uint) string {
	ids, _ := json.Marshal(orgIDs)
	return fmt.Sprintf(
		`{"code":%q,"name":"Name","description":"body","points":100,"time_limit":1,"memory_limit":262144,"group_id":%d,"organization_ids":%s,"is_organization_private":false}`,
		code, fx.group.ID, ids)
}

const orgContestBody = `{"key":%q,"name":"Org Contest","description":"body","start_time":"2026-01-01T00:00:00Z","end_time":"2026-01-02T00:00:00Z","format_name":"default","organization_ids":[%d],"is_organization_private":false}`

const orgBlogBody = `{"title":"Org Post","slug":%q,"content":"body","summary":"summary","publish_on":"2026-01-01T00:00:00Z","global_post":true,"organization_id":%d}`

// Each org-scoped create requires its create permission AND organization
// administration (v1 CustomAdminOrganizationMixin); a superuser passes via the
// edit-all bypass.
func TestOrganizationCreate_Gates(t *testing.T) {
	fx := setupOrgSelfServiceDB(t)
	orgPath := "/organization/" + strconv.Itoa(int(fx.org.ID))

	problem := orgProblemBody(fx, "orgp", nil)
	contest := fmt.Sprintf(orgContestBody, "orgc", fx.otherOrg.ID)
	blog := fmt.Sprintf(orgBlogBody, "orgpost", fx.otherOrg.ID)

	for _, tc := range []struct {
		name    string
		handler gin.HandlerFunc
		kind    string
		body    string
	}{
		{"problem", OrganizationProblemCreate, "problems", problem},
		{"contest", OrganizationContestCreate, "contests", contest},
		{"blog", OrganizationBlogPostCreate, "blog-posts", blog},
	} {
		path := orgPath + "/" + tc.kind
		// Member with the create permission but not an org admin.
		code, _ := orgCreateRequest(t, fx.permUser.ID, http.MethodPost, path, tc.kind, tc.handler, tc.body)
		require.Equal(t, http.StatusForbidden, code, "%s: a non-admin member must be refused", tc.name)

		// Staff without the permission.
		code, _ = orgCreateRequest(t, fx.staffUser.ID, http.MethodPost, path, tc.kind, tc.handler, tc.body)
		require.Equal(t, http.StatusForbidden, code, "%s: staff without the create permission must be refused", tc.name)

		// Stranger.
		code, _ = orgCreateRequest(t, fx.strangerUser.ID, http.MethodPost, path, tc.kind, tc.handler, tc.body)
		require.Equal(t, http.StatusForbidden, code, "%s: a stranger must be refused", tc.name)

		// Superuser holds every permission and the edit-all bypass.
		code, resp := orgCreateRequest(t, fx.superUser.ID, http.MethodPost, path, tc.kind, tc.handler, tc.body)
		require.NotEqual(t, http.StatusForbidden, code, "%s: superuser must pass the gate: %s", tc.name, resp)
	}
}

// Org admins create with the organization forced onto the object, the private
// flags set and the caller added as an author.
func TestOrganizationCreate_ForcesOrganization(t *testing.T) {
	fx := setupOrgSelfServiceDB(t)
	orgPath := "/organization/" + strconv.Itoa(int(fx.org.ID))

	// Problem: the body tries to attach the other organization.
	code, resp := orgCreateRequest(t, fx.adminUser.ID, http.MethodPost, orgPath+"/problems", "problems",
		OrganizationProblemCreate, orgProblemBody(fx, "orgp", []uint{fx.otherOrg.ID}))
	require.Equal(t, http.StatusOK, code, string(resp))
	var p models.Problem
	require.NoError(t, db.DB.Where("code = ?", "orgp").First(&p).Error)
	require.True(t, p.IsOrganizationPrivate)
	var orgIDs []uint
	require.NoError(t, db.DB.Table("judge_problem_organizations").Where("problem_id = ?", p.ID).
		Pluck("organization_id", &orgIDs).Error)
	require.Equal(t, []uint{fx.org.ID}, orgIDs)
	var authorIDs []uint
	require.NoError(t, db.DB.Table("judge_problem_authors").Where("problem_id = ?", p.ID).
		Pluck("profile_id", &authorIDs).Error)
	require.Contains(t, authorIDs, fx.admin.ID)

	// Contest.
	code, resp = orgCreateRequest(t, fx.adminUser.ID, http.MethodPost, orgPath+"/contests", "contests",
		OrganizationContestCreate, fmt.Sprintf(orgContestBody, "orgc", fx.otherOrg.ID))
	require.Equal(t, http.StatusOK, code, string(resp))
	var ct models.Contest
	require.NoError(t, db.DB.Where(&models.Contest{Key: "orgc"}).First(&ct).Error)
	require.True(t, ct.IsOrganizationPrivate)
	orgIDs = nil
	require.NoError(t, db.DB.Table("judge_contest_organizations").Where("contest_id = ?", ct.ID).
		Pluck("organization_id", &orgIDs).Error)
	require.Equal(t, []uint{fx.org.ID}, orgIDs)
	authorIDs = nil
	require.NoError(t, db.DB.Table("judge_contest_authors").Where("contest_id = ?", ct.ID).
		Pluck("profile_id", &authorIDs).Error)
	require.Contains(t, authorIDs, fx.admin.ID)

	// Blog post: global_post is forced off and the organization forced on.
	code, resp = orgCreateRequest(t, fx.adminUser.ID, http.MethodPost, orgPath+"/blog-posts", "blog-posts",
		OrganizationBlogPostCreate, fmt.Sprintf(orgBlogBody, "orgpost", fx.otherOrg.ID))
	require.Equal(t, http.StatusCreated, code, string(resp))
	var post models.BlogPost
	require.NoError(t, db.DB.Where("slug = ?", "orgpost").First(&post).Error)
	require.NotNil(t, post.OrganizationID)
	require.Equal(t, fx.org.ID, *post.OrganizationID)
	require.False(t, post.GlobalPost)
	authorIDs = nil
	require.NoError(t, db.DB.Table("judge_blogpost_authors").Where("blogpost_id = ?", post.ID).
		Pluck("profile_id", &authorIDs).Error)
	require.Contains(t, authorIDs, fx.admin.ID)
}

// requestList calls the request list handler as userID.
func requestList(t *testing.T, userID, orgID uint, query string) (int, []map[string]any) {
	t.Helper()
	router := authedRouter(userID, http.MethodGet, "/organization/:id/requests", OrganizationRequestList)
	w := doRequest(router, http.MethodGet, "/organization/"+strconv.Itoa(int(orgID))+"/requests"+query, "")
	if w.Code != http.StatusOK {
		return w.Code, nil
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body.Data
}

// The list honours ?state=P/A/R/log/all instead of always returning pending.
func TestOrganizationRequestList_StateFilter(t *testing.T) {
	fx := setupOrgSelfServiceDB(t)

	seed := func(state string, minutes int) models.OrganizationRequest {
		r := models.OrganizationRequest{
			UserID: fx.stranger.ID, OrganizationID: fx.org.ID,
			Time: time.Now().Add(time.Duration(minutes) * time.Minute), State: state, Reason: "reason " + state,
		}
		require.NoError(t, db.DB.Create(&r).Error)
		return r
	}
	seed("P", -3)
	seed("A", -2)
	seed("R", -1)
	// Another organization's request never leaks in.
	require.NoError(t, db.DB.Create(&models.OrganizationRequest{
		UserID: fx.stranger.ID, OrganizationID: fx.otherOrg.ID, Time: time.Now(), State: "P", Reason: "other",
	}).Error)

	states := func(query string) []string {
		code, rows := requestList(t, fx.adminUser.ID, fx.org.ID, query)
		require.Equal(t, http.StatusOK, code)
		out := make([]string, len(rows))
		for i, r := range rows {
			out[i] = r["state"].(string)
		}
		return out
	}
	require.Equal(t, []string{"P"}, states(""), "default is pending")
	require.Equal(t, []string{"P"}, states("?state=P"))
	require.Equal(t, []string{"A"}, states("?state=A"))
	require.Equal(t, []string{"R"}, states("?state=R"))
	require.Equal(t, []string{"R", "A"}, states("?state=log"), "log is newest first")
	require.Len(t, states("?state=all"), 3)

	code, rows := requestList(t, fx.adminUser.ID, fx.org.ID, "?state=A")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "stranger", rows[0]["username"])
	require.Equal(t, fx.stranger.ID, uint(rows[0]["user_id"].(float64)))
	require.Equal(t, "reason A", rows[0]["reason"])

	code, _ = requestList(t, fx.adminUser.ID, fx.org.ID, "?state=bogus")
	require.Equal(t, http.StatusBadRequest, code)

	code, _ = requestList(t, fx.strangerUser.ID, fx.org.ID, "")
	require.Equal(t, http.StatusForbidden, code, "only org admins may read the request list")
}

// A closed organization accepts one pending request per user, and stores the
// reason typed into the form.
func TestOrganizationRequestCreate_Reason(t *testing.T) {
	fx := setupOrgSelfServiceDB(t)
	path := "/organization/" + strconv.Itoa(int(fx.org.ID)) + "/request"
	newRouter := func() *gin.Engine {
		return authedRouter(fx.strangerUser.ID, http.MethodPost, "/organization/:id/request", RequestJoinOrganization)
	}

	w := doRequest(newRouter(), http.MethodPost, path, `{"reason":"I maintain the mirrors"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var req models.OrganizationRequest
	require.NoError(t, db.DB.Where("user_id = ? AND organization_id = ?", fx.stranger.ID, fx.org.ID).First(&req).Error)
	require.Equal(t, "P", req.State)
	require.Equal(t, "I maintain the mirrors", req.Reason)

	w = doRequest(newRouter(), http.MethodPost, path, `{"reason":"again"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, "a second pending request is refused")

	// An open organization is joined directly, not requested.
	require.NoError(t, db.DB.Model(&fx.otherOrg).Update("is_open", true).Error)
	openRouter := authedRouter(fx.strangerUser.ID, http.MethodPost, "/organization/:id/request", RequestJoinOrganization)
	w = doRequest(openRouter, http.MethodPost, "/organization/"+strconv.Itoa(int(fx.otherOrg.ID))+"/request",
		`{"reason":"hi"}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// Approving a request requires organization administration and grants the
// membership; a second handle of the same request is refused.
func TestOrganizationRequestHandle_Permissions(t *testing.T) {
	fx := setupOrgSelfServiceDB(t)
	req := models.OrganizationRequest{
		UserID: fx.stranger.ID, OrganizationID: fx.org.ID, Time: time.Now(), State: "P", Reason: "r",
	}
	require.NoError(t, db.DB.Create(&req).Error)
	handlePath := "/organization/request/" + strconv.Itoa(int(req.ID)) + "/handle"
	handle := func(userID uint) int {
		router := authedRouter(userID, http.MethodPost, "/organization/request/:rid/handle", HandleOrganizationRequest)
		return doRequest(router, http.MethodPost, handlePath, `{"action":"approve"}`).Code
	}

	require.Equal(t, http.StatusBadRequest, handle(fx.strangerUser.ID), "non-admins must not handle requests")
	var count int64
	require.NoError(t, db.DB.Table("judge_profile_organizations").
		Where("profile_id = ? AND organization_id = ?", fx.stranger.ID, fx.org.ID).Count(&count).Error)
	require.Zero(t, count)

	require.Equal(t, http.StatusOK, handle(fx.adminUser.ID))
	require.NoError(t, db.DB.First(&req, req.ID).Error)
	require.Equal(t, "A", req.State)
	require.NoError(t, db.DB.Table("judge_profile_organizations").
		Where("profile_id = ? AND organization_id = ?", fx.stranger.ID, fx.org.ID).Count(&count).Error)
	require.EqualValues(t, 1, count)

	require.Equal(t, http.StatusBadRequest, handle(fx.adminUser.ID), "an already handled request cannot be re-handled")
}
