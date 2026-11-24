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

// translationRouter mounts a translations route behind the real admin guard
// (auth.RequireProblemEdit), so permission and :code handling are exercised.
func translationRouter(userID uint, method, path string, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path,
		func(c *gin.Context) { c.Set("user_id", userID) },
		auth.RequireProblemEdit(),
		handler,
	)
	return router
}

func TestProblemTranslationAdminRoundTrip(t *testing.T) {
	root, plain, _ := setupSmallGapsDB(t)
	p := models.Problem{Code: "p1", Name: "P1"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)

	const base = "/admin/problem/:code/translations"
	listRouter := translationRouter(root.ID, http.MethodGet, base, ProblemTranslationList)
	createRouter := translationRouter(root.ID, http.MethodPost, base, ProblemTranslationCreate)
	const byID = base + "/:id"
	updateRouter := translationRouter(root.ID, http.MethodPatch, byID, ProblemTranslationUpdate)
	deleteRouter := translationRouter(root.ID, http.MethodDelete, byID, ProblemTranslationDelete)
	list := func() []problemTranslationItem {
		w := doRequest(listRouter, http.MethodGet, "/admin/problem/p1/translations", "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body struct {
			Data []problemTranslationItem `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body.Data
	}
	rowCount := func() int64 {
		var n int64
		require.NoError(t, db.DB.Model(&models.ProblemTranslation{}).Count(&n).Error)
		return n
	}

	t.Run("non-editor is forbidden", func(t *testing.T) {
		router := translationRouter(plain.ID, http.MethodPost, base, ProblemTranslationCreate)
		w := doRequest(router, http.MethodPost, "/admin/problem/p1/translations", `{"language":"vi","name":"A","description":"B"}`)
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		require.Equal(t, int64(0), rowCount())
	})

	t.Run("invalid language is rejected", func(t *testing.T) {
		w := doRequest(createRouter, http.MethodPost, "/admin/problem/p1/translations", `{"language":"fr","name":"A","description":"B"}`)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		require.Equal(t, int64(0), rowCount())
	})

	t.Run("blank name is rejected", func(t *testing.T) {
		w := doRequest(createRouter, http.MethodPost, "/admin/problem/p1/translations", `{"language":"vi","name":"  ","description":"B"}`)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	var created problemTranslationItem
	t.Run("create", func(t *testing.T) {
		w := doRequest(createRouter, http.MethodPost, "/admin/problem/p1/translations",
			`{"language":"vi","name":"Tổng hai số","description":"Cho hai số nguyên."}`)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
		require.NotZero(t, created.ID)
		require.Equal(t, p.ID, created.ProblemID)
		require.Equal(t, "vi", created.Language)
		require.Equal(t, "Tổng hai số", created.Name)
	})

	t.Run("duplicate language is rejected", func(t *testing.T) {
		w := doRequest(createRouter, http.MethodPost, "/admin/problem/p1/translations",
			`{"language":"vi","name":"Khác","description":"Khác"}`)
		require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
		require.Equal(t, int64(1), rowCount())
	})

	t.Run("list", func(t *testing.T) {
		items := list()
		require.Len(t, items, 1)
		require.Equal(t, created.ID, items[0].ID)
		require.Equal(t, "Tổng hai số", items[0].Name)
		require.Equal(t, "Cho hai số nguyên.", items[0].Description)
	})

	t.Run("update", func(t *testing.T) {
		target := "/admin/problem/p1/translations/" + strconv.Itoa(int(created.ID))
		w := doRequest(updateRouter, http.MethodPatch, target, `{"language":"vi","name":"Cộng hai số","description":"Đề mới"}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var got models.ProblemTranslation
		require.NoError(t, db.DB.First(&got, created.ID).Error)
		require.Equal(t, "Cộng hai số", got.Name)
		require.Equal(t, "Đề mới", got.Description)
	})

	t.Run("update onto a taken language is rejected", func(t *testing.T) {
		w := doRequest(createRouter, http.MethodPost, "/admin/problem/p1/translations",
			`{"language":"en","name":"Sum","description":"Add them."}`)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

		var en models.ProblemTranslation
		require.NoError(t, db.DB.Where("problem_id = ? AND language = ?", p.ID, "en").First(&en).Error)
		target := "/admin/problem/p1/translations/" + strconv.Itoa(int(en.ID))
		w = doRequest(updateRouter, http.MethodPatch, target, `{"language":"vi","name":"Sum","description":"Add them."}`)
		require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	})

	t.Run("delete", func(t *testing.T) {
		target := "/admin/problem/p1/translations/" + strconv.Itoa(int(created.ID))
		w := doRequest(deleteRouter, http.MethodDelete, target, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Equal(t, int64(1), rowCount())

		require.Equal(t, http.StatusNotFound, doRequest(deleteRouter, http.MethodDelete, target, "").Code)
		require.Equal(t, http.StatusBadRequest, doRequest(deleteRouter, http.MethodDelete, "/admin/problem/p1/translations/abc", "").Code)
	})

	t.Run("unknown problem is not found", func(t *testing.T) {
		require.Equal(t, http.StatusNotFound, doRequest(listRouter, http.MethodGet, "/admin/problem/nope/translations", "").Code)
	})
}

func TestProblemTranslationDescriptionIsSanitized(t *testing.T) {
	root, _, _ := setupSmallGapsDB(t)
	p := models.Problem{Code: "san", Name: "San"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)

	const base = "/admin/problem/:code/translations"
	createRouter := translationRouter(root.ID, http.MethodPost, base, ProblemTranslationCreate)
	updateRouter := translationRouter(root.ID, http.MethodPatch, base+"/:id", ProblemTranslationUpdate)

	w := doRequest(createRouter, http.MethodPost, "/admin/problem/san/translations",
		`{"language":"vi","name":"Tên","description":"<script>alert(1)</script><p>an toàn</p>"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var got models.ProblemTranslation
	require.NoError(t, db.DB.Where("problem_id = ? AND language = ?", p.ID, "vi").First(&got).Error)
	require.Equal(t, "<p>an toàn</p>", got.Description)

	w = doRequest(updateRouter, http.MethodPatch, "/admin/problem/san/translations/"+strconv.Itoa(int(got.ID)),
		`{"language":"vi","name":"Tên","description":"<script>alert(2)</script>mới"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&got, got.ID).Error)
	require.Equal(t, "mới", got.Description)
}