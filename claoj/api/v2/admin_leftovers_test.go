package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// setupAdminLeftoversDB adds the Phase 6 leftover tables to the shared admin
// test schema (judge_class comes in with Contest, the rest are new here).
func setupAdminLeftoversDB(t *testing.T) {
	t.Helper()
	setupAdminDetailDB(t)
	require.NoError(t, db.DB.AutoMigrate(
		&models.Class{}, &models.Comment{}, &models.CommentLock{},
		&models.ProblemPointsVote{}, &models.OrganizationRequest{},
		&models.Judge{}, &models.RuntimeVersion{},
	))
}

// grantLeftoverPerm seeds a Django permission row and attaches it to the user.
func grantLeftoverPerm(t *testing.T, userID uint, codename string) {
	t.Helper()
	ct := models.DjangoContentType{AppLabel: "judge", Model: "leftover"}
	require.NoError(t, db.DB.FirstOrCreate(&ct, models.DjangoContentType{AppLabel: "judge", Model: "leftover"}).Error)
	perm := models.AuthPermission{Name: codename, Codename: codename, ContentTypeID: ct.ID}
	require.NoError(t, db.DB.Create(&perm).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: userID, PermissionID: perm.ID}).Error)
}

func staffUser(t *testing.T, username string) models.AuthUser {
	t.Helper()
	u := models.AuthUser{Username: username, IsActive: true, IsStaff: true}
	require.NoError(t, db.DB.Create(&u).Error)
	return u
}

// leftoversRouter wires the real admin gate plus optional permission guards,
// mirroring api/router.go.
func leftoversRouter(userID uint, method, path string, guards []gin.HandlerFunc, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	r.Use(auth.AdminRequiredMiddleware())
	for _, g := range guards {
		r.Use(g)
	}
	r.Handle(method, path, handler)
	return r
}

func TestAdminPointsVoteList_MappingSearchFilter(t *testing.T) {
	setupAdminLeftoversDB(t)
	alice := createContestTestUser(t, "alice")
	bob := createContestTestUser(t, "bob")
	p1 := models.Problem{Code: "aplusb", Name: "A+B"}
	p2 := models.Problem{Code: "ccc", Name: "CCC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p1).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p2).Error)
	votes := []models.ProblemPointsVote{
		{Points: 10, VoterID: alice.ID, ProblemID: p1.ID, VoteTime: time.Now()},
		{Points: 20, VoterID: bob.ID, ProblemID: p2.ID, VoteTime: time.Now()},
	}
	for i := range votes {
		require.NoError(t, db.DB.Omit("Voter", "Problem").Create(&votes[i]).Error)
	}

	var body struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total"`
	}
	call := func(query string) {
		w := doRequest(authedRouter(1, http.MethodGet, "/admin/points-votes", AdminPointsVoteList),
			http.MethodGet, "/admin/points-votes"+query, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	}

	call("")
	require.EqualValues(t, 2, body.Total)

	call("?search=ali")
	require.EqualValues(t, 1, body.Total)
	require.Equal(t, "alice", body.Data[0]["voter"])
	require.Equal(t, "aplusb", body.Data[0]["problem"])
	require.Equal(t, "A+B", body.Data[0]["problem_name"])
	require.EqualValues(t, 10, body.Data[0]["points"])
	require.NotEmpty(t, body.Data[0]["vote_time"])

	call("?search=ccc")
	require.EqualValues(t, 1, body.Total)
	require.Equal(t, "bob", body.Data[0]["voter"])

	call("?problem=ccc")
	require.EqualValues(t, 1, body.Total)
	require.Equal(t, "ccc", body.Data[0]["problem"])
}

func TestAdminLeftoverLists_RequireStaff(t *testing.T) {
	setupAdminLeftoversDB(t)
	plain := models.AuthUser{Username: "plain", IsActive: true}
	require.NoError(t, db.DB.Create(&plain).Error)

	lists := []struct {
		path    string
		handler gin.HandlerFunc
	}{
		{"/admin/points-votes", AdminPointsVoteList},
		{"/admin/organization-requests", AdminOrganizationRequestList},
		{"/admin/comment-locks", AdminCommentLockList},
	}
	for _, list := range lists {
		w := doRequest(leftoversRouter(plain.ID, http.MethodGet, list.path, nil, list.handler),
			http.MethodGet, list.path, "")
		require.Equal(t, http.StatusForbidden, w.Code, list.path+": "+w.Body.String())
	}
}

func TestAdminOrganizationRequestList_Filters(t *testing.T) {
	setupAdminLeftoversDB(t)
	member := createContestTestUser(t, "member")
	org := createContestTestOrg(t, "org-one")
	other := createContestTestOrg(t, "org-two")
	now := time.Now()
	requests := []models.OrganizationRequest{
		{UserID: member.ID, OrganizationID: org.ID, Time: now, State: "P", Reason: "let me in"},
		{UserID: member.ID, OrganizationID: other.ID, Time: now, State: "A", Reason: "ok"},
	}
	for i := range requests {
		require.NoError(t, db.DB.Omit("User", "Organization").Create(&requests[i]).Error)
	}

	var body struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total"`
	}
	call := func(query string) {
		w := doRequest(authedRouter(1, http.MethodGet, "/admin/organization-requests", AdminOrganizationRequestList),
			http.MethodGet, "/admin/organization-requests"+query, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	}

	call("")
	require.EqualValues(t, 2, body.Total)

	call("?state=A")
	require.EqualValues(t, 1, body.Total)
	require.Equal(t, "org-two", body.Data[0]["organization"])
	require.Equal(t, "A", body.Data[0]["state"])
	require.Equal(t, "member", body.Data[0]["username"])
	require.Equal(t, "ok", body.Data[0]["reason"])

	call("?search=org-one")
	require.EqualValues(t, 1, body.Total)
	require.Equal(t, "org-one", body.Data[0]["organization"])

	// v1 stores P/A/R; anything else is rejected rather than ignored.
	w := doRequest(authedRouter(1, http.MethodGet, "/admin/organization-requests", AdminOrganizationRequestList),
		http.MethodGet, "/admin/organization-requests?state=X", "")
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestAdminClassCRUD_MapsRemainingColumns(t *testing.T) {
	setupAdminLeftoversDB(t)
	org := createContestTestOrg(t, "class-org")
	teacher := createContestTestUser(t, "teacher")
	student := createContestTestUser(t, "student")
	code := "ABC123"

	createBody := `{"name":"Class A","organization_id":` + itoa(uint(org.ID)) +
		`,"description":"desc","access_code":"` + code + `","is_active":true,"admin_ids":[` +
		itoa(uint(teacher.ID)) + `],"member_ids":[` + itoa(uint(student.ID)) + `]}`
	w := doRequest(authedRouter(1, http.MethodPost, "/admin/classes", AdminClassCreate),
		http.MethodPost, "/admin/classes", createBody)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var created struct {
		Class map[string]any `json:"class"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	classID := uint(created.Class["id"].(float64))
	require.Equal(t, "class-a", created.Class["slug"])
	require.Equal(t, "desc", created.Class["description"])
	require.Equal(t, code, created.Class["access_code"])
	require.Len(t, created.Class["admins"], 1)
	require.Equal(t, "teacher", created.Class["admins"].([]any)[0].(map[string]any)["username"])
	require.Len(t, created.Class["members"], 1)

	w = doRequest(authedRouter(1, http.MethodGet, "/admin/class/:id", AdminClassDetail),
		http.MethodGet, "/admin/class/"+itoa(classID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	updateBody := `{"name":"Class B","description":"desc2","access_code":"ZZZ","member_ids":[]}`
	w = doRequest(authedRouter(1, http.MethodPatch, "/admin/class/:id", AdminClassUpdate),
		http.MethodPatch, "/admin/class/"+itoa(classID), updateBody)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var updated struct {
		Class map[string]any `json:"class"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	require.Equal(t, "Class B", updated.Class["name"])
	require.Equal(t, "desc2", updated.Class["description"])
	require.Equal(t, "ZZZ", updated.Class["access_code"])
	require.Empty(t, updated.Class["members"])
	require.Len(t, updated.Class["admins"], 1)

	w = doRequest(authedRouter(1, http.MethodDelete, "/admin/class/:id", AdminClassDelete),
		http.MethodDelete, "/admin/class/"+itoa(classID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var count int64
	db.DB.Model(&models.Class{}).Count(&count)
	require.Zero(t, count)
}

func TestAdminClassCRUD_Gates(t *testing.T) {
	setupAdminLeftoversDB(t)
	staff := staffUser(t, "staffer")
	org := createContestTestOrg(t, "gate-org")
	body := `{"name":"Gated","organization_id":` + itoa(uint(org.ID)) + `}`

	// Staff without judge.add_class cannot create (v1 ClassAdmin.has_add_permission).
	r := leftoversRouter(staff.ID, http.MethodPost, "/admin/classes", []gin.HandlerFunc{auth.RequirePerm("judge.add_class")}, AdminClassCreate)
	w := doRequest(r, http.MethodPost, "/admin/classes", body)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	grantLeftoverPerm(t, staff.ID, "add_class")
	r = leftoversRouter(staff.ID, http.MethodPost, "/admin/classes", []gin.HandlerFunc{auth.RequirePerm("judge.add_class")}, AdminClassCreate)
	w = doRequest(r, http.MethodPost, "/admin/classes", body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
}

func TestAdminCommentLockCRUD(t *testing.T) {
	setupAdminLeftoversDB(t)

	w := doRequest(authedRouter(1, http.MethodPost, "/admin/comment-locks", AdminCommentLockCreate),
		http.MethodPost, "/admin/comment-locks", `{"page":"p/aplusb"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// v1's unique page: a second lock on the same page is rejected.
	w = doRequest(authedRouter(1, http.MethodPost, "/admin/comment-locks", AdminCommentLockCreate),
		http.MethodPost, "/admin/comment-locks", `{"page":"p/aplusb"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	w = doRequest(authedRouter(1, http.MethodGet, "/admin/comment-locks", AdminCommentLockList),
		http.MethodGet, "/admin/comment-locks", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var list struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.EqualValues(t, 1, list.Total)
	lockID := uint(list.Data[0]["id"].(float64))

	w = doRequest(authedRouter(1, http.MethodPatch, "/admin/comment-lock/:id", AdminCommentLockUpdate),
		http.MethodPatch, "/admin/comment-lock/"+itoa(lockID), `{"page":"p/ccc"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doRequest(authedRouter(1, http.MethodDelete, "/admin/comment-lock/:id", AdminCommentLockDelete),
		http.MethodDelete, "/admin/comment-lock/"+itoa(lockID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doRequest(authedRouter(1, http.MethodDelete, "/admin/comment-lock/:id", AdminCommentLockDelete),
		http.MethodDelete, "/admin/comment-lock/"+itoa(lockID), "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

func TestCommentCreate_RespectsCommentLockAndOverride(t *testing.T) {
	setupAdminLeftoversDB(t)
	author := createContestTestUser(t, "poster")
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.CommentLock{Page: "p/aplusb"}).Error)

	body := `{"page":"p/aplusb","body":"hello"}`
	w := doRequest(authedRouter(author.UserID, http.MethodPost, "/comments", CommentCreate),
		http.MethodPost, "/comments", body)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	// A different page is still writable.
	w = doRequest(authedRouter(author.UserID, http.MethodPost, "/comments", CommentCreate),
		http.MethodPost, "/comments", `{"page":"p/ccc","body":"hello"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// judge.override_comment_lock bypasses the lock (v1 comments.py:62).
	grantLeftoverPerm(t, author.UserID, "override_comment_lock")
	w = doRequest(authedRouter(author.UserID, http.MethodPost, "/comments", CommentCreate),
		http.MethodPost, "/comments", body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
}