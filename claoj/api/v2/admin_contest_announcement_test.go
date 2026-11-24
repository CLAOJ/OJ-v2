package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// announcementRouter mounts an announcements route behind the real admin guard
// (auth.RequireContestEdit), so permission and :key handling are exercised.
func announcementRouter(userID uint, method, path string, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path,
		func(c *gin.Context) { c.Set("user_id", userID) },
		auth.RequireContestEdit(),
		handler,
	)
	return router
}

func TestContestAnnouncementAdminRoundTrip(t *testing.T) {
	root, plain, _ := setupSmallGapsDB(t)
	contest := models.Contest{Key: "c1", Name: "C1"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)
	other := models.Contest{Key: "c2", Name: "C2"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&other).Error)

	const base = "/admin/contest/:key/announcements"
	listRouter := announcementRouter(root.ID, http.MethodGet, base, AdminContestAnnouncementList)
	createRouter := announcementRouter(root.ID, http.MethodPost, base, AdminContestAnnouncementCreate)
	const byID = base + "/:id"
	updateRouter := announcementRouter(root.ID, http.MethodPatch, byID, AdminContestAnnouncementUpdate)
	deleteRouter := announcementRouter(root.ID, http.MethodDelete, byID, AdminContestAnnouncementDelete)
	list := func() []contestAnnouncementItem {
		w := doRequest(listRouter, http.MethodGet, "/admin/contest/c1/announcements", "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body struct {
			Data []contestAnnouncementItem `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body.Data
	}
	rowCount := func() int64 {
		var n int64
		require.NoError(t, db.DB.Model(&models.ContestAnnouncement{}).Count(&n).Error)
		return n
	}

	t.Run("non-editor is forbidden", func(t *testing.T) {
		router := announcementRouter(plain.ID, http.MethodPost, base, AdminContestAnnouncementCreate)
		w := doRequest(router, http.MethodPost, "/admin/contest/c1/announcements", `{"title":"A","description":"B"}`)
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		require.Equal(t, int64(0), rowCount())
	})

	t.Run("blank title is rejected", func(t *testing.T) {
		w := doRequest(createRouter, http.MethodPost, "/admin/contest/c1/announcements", `{"title":"  ","description":"B"}`)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	t.Run("title over 100 characters is rejected", func(t *testing.T) {
		long := ""
		for i := 0; i < 101; i++ {
			long += "a"
		}
		w := doRequest(createRouter, http.MethodPost, "/admin/contest/c1/announcements",
			`{"title":"`+long+`","description":"B"}`)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		require.Equal(t, int64(0), rowCount())
	})

	t.Run("blank description is rejected", func(t *testing.T) {
		w := doRequest(createRouter, http.MethodPost, "/admin/contest/c1/announcements", `{"title":"A","description":" "}`)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	var created contestAnnouncementItem
	t.Run("create", func(t *testing.T) {
		w := doRequest(createRouter, http.MethodPost, "/admin/contest/c1/announcements",
			`{"title":"Vòng 1 bắt đầu","description":"Chúc các bạn may mắn!"}`)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
		require.NotZero(t, created.ID)
		require.Equal(t, contest.ID, created.ContestID)
		require.Equal(t, "Vòng 1 bắt đầu", created.Title)
		require.False(t, created.Date.IsZero())
	})

	t.Run("list", func(t *testing.T) {
		items := list()
		require.Len(t, items, 1)
		require.Equal(t, created.ID, items[0].ID)
		require.Equal(t, "Chúc các bạn may mắn!", items[0].Description)
	})

	t.Run("update keeps the original timestamp", func(t *testing.T) {
		target := "/admin/contest/c1/announcements/" + strconv.Itoa(int(created.ID))
		w := doRequest(updateRouter, http.MethodPatch, target, `{"title":"Vòng 1 đã bắt đầu","description":"Nội dung mới"}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var got models.ContestAnnouncement
		require.NoError(t, db.DB.First(&got, created.ID).Error)
		require.Equal(t, "Vòng 1 đã bắt đầu", got.Title)
		require.Equal(t, "Nội dung mới", got.Description)
		require.Equal(t, created.Date.Unix(), got.Date.Unix())
	})

	t.Run("update rejects a blank title", func(t *testing.T) {
		target := "/admin/contest/c1/announcements/" + strconv.Itoa(int(created.ID))
		w := doRequest(updateRouter, http.MethodPatch, target, `{"title":"","description":"x"}`)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	t.Run("rows of another contest are not reachable", func(t *testing.T) {
		w := doRequest(createRouter, http.MethodPost, "/admin/contest/c2/announcements",
			`{"title":"C2 news","description":"Only c2"}`)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		var c2 contestAnnouncementItem
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &c2))
		require.Equal(t, other.ID, c2.ContestID)

		target := "/admin/contest/c1/announcements/" + strconv.Itoa(int(c2.ID))
		require.Equal(t, http.StatusNotFound, doRequest(updateRouter, http.MethodPatch, target, `{"title":"x","description":"y"}`).Code)
		require.Equal(t, http.StatusNotFound, doRequest(deleteRouter, http.MethodDelete, target, "").Code)

		var got models.ContestAnnouncement
		require.NoError(t, db.DB.First(&got, c2.ID).Error)
		require.Equal(t, "C2 news", got.Title)
	})

	t.Run("delete", func(t *testing.T) {
		target := "/admin/contest/c1/announcements/" + strconv.Itoa(int(created.ID))
		w := doRequest(deleteRouter, http.MethodDelete, target, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Equal(t, int64(1), rowCount())

		require.Equal(t, http.StatusNotFound, doRequest(deleteRouter, http.MethodDelete, target, "").Code)
		require.Equal(t, http.StatusBadRequest, doRequest(deleteRouter, http.MethodDelete, "/admin/contest/c1/announcements/abc", "").Code)
	})

	t.Run("unknown contest is not found", func(t *testing.T) {
		require.Equal(t, http.StatusNotFound, doRequest(listRouter, http.MethodGet, "/admin/contest/nope/announcements", "").Code)
	})
}

func TestContestAnnouncementDescriptionIsSanitized(t *testing.T) {
	root, _, _ := setupSmallGapsDB(t)
	contest := models.Contest{Key: "san", Name: "San"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)

	const base = "/admin/contest/:key/announcements"
	createRouter := announcementRouter(root.ID, http.MethodPost, base, AdminContestAnnouncementCreate)
	updateRouter := announcementRouter(root.ID, http.MethodPatch, base+"/:id", AdminContestAnnouncementUpdate)

	w := doRequest(createRouter, http.MethodPost, "/admin/contest/san/announcements",
		`{"title":"Tên","description":"<script>alert(1)</script><p>an toàn</p>"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var got models.ContestAnnouncement
	require.NoError(t, db.DB.Where("contest_id = ?", contest.ID).First(&got).Error)
	require.Equal(t, "<p>an toàn</p>", got.Description)

	w = doRequest(updateRouter, http.MethodPatch, "/admin/contest/san/announcements/"+strconv.Itoa(int(got.ID)),
		`{"title":"Tên","description":"<script>alert(2)</script>mới"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&got, got.ID).Error)
	require.Equal(t, "mới", got.Description)
}
