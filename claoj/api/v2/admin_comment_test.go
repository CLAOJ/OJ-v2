package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/auditlog"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/middleware"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ============================================================
// Admin comments: metadata patch, page_type filter, revisions,
// bulk hide/unhide.
// ============================================================

func setupAdminCommentDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.Comment{}, &models.CommentRevision{},
		&models.AuthUser{}, &models.Profile{},
		// resolveFromDB joins these, so a superuser check needs them to exist.
		&models.AuthGroup{}, &models.DjangoContentType{}, &models.AuthPermission{},
		&models.AuthUserGroup{}, &models.AuthGroupPermission{}, &models.AuthUserPermission{},
	))
	db.DB = database
}

func seedComment(t *testing.T, authorID uint, page, body string, hidden bool) models.Comment {
	t.Helper()
	cm := models.Comment{
		AuthorID: authorID, Time: time.Now().Add(-time.Hour), Page: page,
		Body: body, Hidden: hidden, Revisions: 1,
		Lft: 1, Rght: 2, TreeID: 1, Level: 0,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&cm).Error)
	return cm
}

type commentListBody struct {
	Data  []map[string]any `json:"data"`
	Total int64            `json:"total"`
}

func listAdminComments(t *testing.T, query string) commentListBody {
	t.Helper()
	router := authedRouter(1, http.MethodGet, "/admin/comments", AdminCommentList)
	w := doRequest(router, http.MethodGet, "/admin/comments"+query, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body commentListBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestAdminCommentList_PageTypeFilter(t *testing.T) {
	setupAdminCommentDB(t)
	_, root := seedAccount(t, "root", true, true)

	// v2 slash prefixes and the legacy v1 colon codes must land in the same type.
	seedComment(t, root.ID, "p/aplusb", "on a problem", false)
	seedComment(t, root.ID, "p:legacy", "on a legacy problem", false)
	seedComment(t, root.ID, "e/aplusb", "on an editorial", false)
	seedComment(t, root.ID, "s:legacy", "on a legacy solution", false)
	seedComment(t, root.ID, "c/round1", "on a contest", false)
	seedComment(t, root.ID, "c:round1", "on a legacy contest", false)
	seedComment(t, root.ID, "blog/12", "on a blog post", false)
	seedComment(t, root.ID, "b:34", "on a legacy blog post", false)

	pages := func(query string) []string {
		t.Helper()
		var out []string
		for _, row := range listAdminComments(t, query).Data {
			out = append(out, row["page"].(string))
		}
		return out
	}

	t.Run("problem", func(t *testing.T) {
		require.ElementsMatch(t, []string{"p/aplusb", "p:legacy"}, pages("?page_type=problem"))
	})
	t.Run("editorial", func(t *testing.T) {
		require.ElementsMatch(t, []string{"e/aplusb", "s:legacy"}, pages("?page_type=editorial"))
	})
	t.Run("contest", func(t *testing.T) {
		require.ElementsMatch(t, []string{"c/round1", "c:round1"}, pages("?page_type=contest"))
	})
	t.Run("blog", func(t *testing.T) {
		require.ElementsMatch(t, []string{"blog/12", "b:34"}, pages("?page_type=blog"))
	})
	t.Run("unknown value is ignored", func(t *testing.T) {
		require.Equal(t, int64(8), listAdminComments(t, "?page_type=maybe").Total)
	})
	t.Run("combines with hidden filter", func(t *testing.T) {
		seedComment(t, root.ID, "p/hidden", "hidden problem comment", true)
		body := listAdminComments(t, "?page_type=problem&hidden=false")
		require.Equal(t, int64(2), body.Total)
		body = listAdminComments(t, "?page_type=problem&hidden=true")
		require.Equal(t, int64(1), body.Total)
	})
}

func TestAdminCommentList_Revisions(t *testing.T) {
	setupAdminCommentDB(t)
	seedAccount(t, "root", true, true)
	// Burn an auth_user id so the commenter's profile id and auth id diverge:
	// the list must return judge_profile.id, not the auth user id.
	require.NoError(t, db.DB.Create(&models.AuthUser{
		Username: "gap", Email: "gap@example.com", Password: "x", IsActive: true,
	}).Error)
	adaUser, ada := seedAccount(t, "ada", false, false)
	require.NotEqual(t, adaUser.ID, ada.ID)
	seedComment(t, ada.ID, "p/aplusb", "hello", false)

	body := listAdminComments(t, "")
	require.Equal(t, float64(1), body.Data[0]["revisions"])
	require.Equal(t, float64(ada.ID), body.Data[0]["author_id"], "author_id is a profile id")
	require.Equal(t, "ada", body.Data[0]["username"])
}

func TestAdminCommentUpdate_Metadata(t *testing.T) {
	setupAdminCommentDB(t)
	_, root := seedAccount(t, "root", true, true)
	_, ada := seedAccount(t, "ada", false, false)
	cm := seedComment(t, root.ID, "p/aplusb", "old body", false)
	oldTime := cm.Time

	router := authedRouter(1, http.MethodPatch, "/admin/comment/:id", AdminCommentUpdate)
	w := doRequest(router, http.MethodPatch, fmt.Sprintf("/admin/comment/%d", cm.ID),
		fmt.Sprintf(`{"author_id":%d,"page":"e/aplusb","body":"new body","hidden":true,"reason":"cleanup"}`, ada.ID))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Comment
	require.NoError(t, db.DB.First(&got, cm.ID).Error)
	require.Equal(t, ada.ID, got.AuthorID)
	require.Equal(t, "e/aplusb", got.Page)
	require.Equal(t, "new body", got.Body)
	require.True(t, got.Hidden)
	require.Equal(t, 2, got.Revisions, "a body edit increments the revision counter")
	require.WithinDuration(t, oldTime, got.Time, time.Second, "time stays read-only")
	require.Equal(t, 0, got.Score, "score stays read-only")

	var revisions []models.CommentRevision
	require.NoError(t, db.DB.Where("comment_id = ?", cm.ID).Find(&revisions).Error)
	require.Len(t, revisions, 1)
	require.Equal(t, "old body", revisions[0].Body)
	require.Equal(t, root.ID, revisions[0].EditorID)

	body := listAdminComments(t, "")
	require.Equal(t, float64(2), body.Data[0]["revisions"])
	require.Equal(t, float64(ada.ID), body.Data[0]["author_id"])
}

func TestAdminCommentUpdate_Validation(t *testing.T) {
	setupAdminCommentDB(t)
	_, root := seedAccount(t, "root", true, true)
	cm := seedComment(t, root.ID, "p/aplusb", "body", false)

	router := authedRouter(1, http.MethodPatch, "/admin/comment/:id", AdminCommentUpdate)
	patch := func(body string) int {
		t.Helper()
		w := doRequest(router, http.MethodPatch, fmt.Sprintf("/admin/comment/%d", cm.ID), body)
		return w.Code
	}

	require.Equal(t, http.StatusBadRequest, patch(`{"author_id":0}`))
	require.Equal(t, http.StatusBadRequest, patch(`{"author_id":99999}`), "unknown profile")
	require.Equal(t, http.StatusBadRequest, patch(`{"page":"  "}`), "empty page")
	require.Equal(t, http.StatusBadRequest, patch(fmt.Sprintf(`{"page":"%s"}`, strings.Repeat("x", 31))), "page over the column width")
	require.Equal(t, http.StatusBadRequest, patch(`{"page":"noseparator"}`), "page without a type separator")
	require.Equal(t, http.StatusBadRequest, patch(`{"page":"p/not allowed"}`), "page with characters the parser drops")
	require.Equal(t, http.StatusBadRequest, patch(`{"parent_id":2}`), "reparenting is not supported")
	require.Equal(t, http.StatusBadRequest, patch(`{"parent_id":null}`), "even an explicit null parent is rejected")
	require.Equal(t, http.StatusOK, patch(`{"page":"p:legacy_code"}`), "legacy colon codes stay valid")
	require.Equal(t, http.StatusOK, patch(`{"page":"c/round1"}`), "v2 slash codes stay valid")

	var got models.Comment
	require.NoError(t, db.DB.First(&got, cm.ID).Error)
	require.Equal(t, "c/round1", got.Page, "only the last accepted page edit stuck")
	require.Equal(t, root.ID, got.AuthorID)
	require.Equal(t, 1, got.Revisions, "page-only patches don't bump revisions")
}

// A hidden-only admin patch is not a content edit, so the revision counter and
// the revision table must be untouched.
func TestAdminCommentUpdate_HiddenOnlyKeepsRevisions(t *testing.T) {
	setupAdminCommentDB(t)
	_, root := seedAccount(t, "root", true, true)
	cm := seedComment(t, root.ID, "p/aplusb", "body", false)

	router := authedRouter(1, http.MethodPatch, "/admin/comment/:id", AdminCommentUpdate)
	w := doRequest(router, http.MethodPatch, fmt.Sprintf("/admin/comment/%d", cm.ID), `{"hidden":true}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Comment
	require.NoError(t, db.DB.First(&got, cm.ID).Error)
	require.True(t, got.Hidden)
	require.Equal(t, 1, got.Revisions)

	var revisions int64
	require.NoError(t, db.DB.Model(&models.CommentRevision{}).Where("comment_id = ?", cm.ID).Count(&revisions).Error)
	require.Zero(t, revisions)
}

// The public self-edit path must bump revisions too: v1 does
// (judge/views/comment.py:159) and the admin list surfaces that counter.
func TestCommentUpdate_BumpsRevisions(t *testing.T) {
	setupAdminCommentDB(t)
	_, root := seedAccount(t, "root", false, false)
	cm := seedComment(t, root.ID, "p/aplusb", "old body", false)

	router := authedRouter(root.UserID, http.MethodPatch, "/comment/:id", CommentUpdate)
	w := doRequest(router, http.MethodPatch, fmt.Sprintf("/comment/%d", cm.ID), `{"body":"new body"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Comment
	require.NoError(t, db.DB.First(&got, cm.ID).Error)
	require.Equal(t, "new body", got.Body)
	require.Equal(t, 2, got.Revisions, "author edit increments the counter")
}

// The admin audit middleware is what records the bulk action; mounting it here
// proves the endpoint's request (path + ids) lands in the trail.
func TestAdminCommentBulkHide_RowsAndAudit(t *testing.T) {
	setupAdminCommentDB(t)
	_, root := seedAccount(t, "root", true, true)
	a := seedComment(t, root.ID, "p/a", "a", false)
	b := seedComment(t, root.ID, "p/b", "b", false)
	c := seedComment(t, root.ID, "p/c", "c", false)

	prev := auditlog.Default
	store := auditlog.NewMemoryStore()
	auditlog.Default = store
	t.Cleanup(func() { auditlog.Default = prev })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) { ctx.Set("user_id", root.ID) })
	router.Use(middleware.Audit())
	router.POST("/admin/comments/bulk-hide", AdminCommentBulkHide)

	post := func(body string) *httptest.ResponseRecorder {
		return doRequest(router, http.MethodPost, "/admin/comments/bulk-hide", body)
	}

	w := post(fmt.Sprintf(`{"ids":[%d,%d],"hidden":true}`, a.ID, b.ID))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"updated":2,"hidden":true}`, w.Body.String())

	var rows []models.Comment
	require.NoError(t, db.DB.Order("id").Find(&rows).Error)
	require.True(t, rows[0].Hidden)
	require.True(t, rows[1].Hidden)
	require.False(t, rows[2].Hidden, "unselected rows are untouched")

	w = post(fmt.Sprintf(`{"ids":[%d],"hidden":false}`, a.ID))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&rows[0], a.ID).Error)
	require.False(t, rows[0].Hidden)

	// Validation.
	require.Equal(t, http.StatusBadRequest, post(`{"ids":[],"hidden":true}`).Code, "empty ids")
	require.Equal(t, http.StatusBadRequest, post(`{"ids":[0],"hidden":true}`).Code, "id 0")
	require.Equal(t, http.StatusBadRequest, post(fmt.Sprintf(`{"ids":[%d]}`, c.ID)).Code, "missing hidden")
	require.Equal(t, http.StatusBadRequest, post(`{"ids":"nope","hidden":true}`).Code, "non-numeric ids")

	oversize := make([]string, 501)
	for i := range oversize {
		oversize[i] = fmt.Sprintf("%d", c.ID)
	}
	require.Equal(t, http.StatusBadRequest, post(`{"ids":[`+strings.Join(oversize, ",")+`],"hidden":true}`).Code, "over 500 ids")

	// Audit trail: the POST is recorded as a CREATE with the path and ids.
	deadline := time.Now().Add(2 * time.Second)
	var entry *auditlog.Entry
	for time.Now().Before(deadline) && entry == nil {
		entries, _, err := store.List(auditlog.Filters{Action: "CREATE"}, 1, 50)
		require.NoError(t, err)
		for i := range entries {
			if entries[i].Status != "success" || !strings.Contains(entries[i].Details, "/admin/comments/bulk-hide") {
				continue
			}
			entry = &entries[i]
			break
		}
		if entry == nil {
			time.Sleep(10 * time.Millisecond)
		}
	}
	require.NotNil(t, entry, "bulk hide was not audited")
	require.Equal(t, root.ID, entry.UserID)
	require.Equal(t, "success", entry.Status)
	require.Contains(t, entry.Details, fmt.Sprintf("%d", a.ID))

	// Unknown ids are counted, not failed: MySQL reports changed rows.
	w = post(`{"ids":[99999],"hidden":true}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"updated":0,"hidden":true}`, w.Body.String())
}
