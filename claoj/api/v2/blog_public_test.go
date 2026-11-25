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
// Public blog authoring (v1 /post/new, /post/<id>/edit):
// the problem-count gate, the organization gate, edit ownership and
// title-derived slugs. Reuses setupAdminBlogPostDB / seedAccount /
// authedRouter / doRequest.
// ============================================================

func createBlogPost(t *testing.T, userID uint, body string) (int, map[string]any) {
	t.Helper()
	router := authedRouter(userID, http.MethodPost, "/blogs", PublicBlogPostCreate)
	w := doRequest(router, http.MethodPost, "/blogs", body)
	var parsed map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed
}

func patchBlogPost(t *testing.T, userID uint, id uint, body string) int {
	t.Helper()
	router := authedRouter(userID, http.MethodPatch, "/blog/:id", PublicBlogPostUpdate)
	w := doRequest(router, http.MethodPatch, "/blog/"+strconv.Itoa(int(id)), body)
	return w.Code
}

func seedOrgWithAdmin(t *testing.T, org *models.Organization, admin models.Profile) {
	t.Helper()
	require.NoError(t, db.DB.Model(org).Association("Admins").Append(&admin))
}

func TestPublicBlogPostCreate_ProblemCountGate(t *testing.T) {
	setupAdminBlogPostDB(t)
	lowUser, lowProfile := seedAccount(t, "low", false, false)
	require.NoError(t, db.DB.Model(&lowProfile).Update("problem_count", blogMinProblemCount-1).Error)
	okUser, okProfile := seedAccount(t, "ok", false, false)
	require.NoError(t, db.DB.Model(&okProfile).Update("problem_count", blogMinProblemCount).Error)
	root, _ := seedAccount(t, "root", true, true)

	code, _ := createBlogPost(t, lowUser.ID, `{"title":"Too Early","content":"body"}`)
	require.Equal(t, http.StatusForbidden, code, "below CLAOJ_BLOG_MIN_PROBLEM_COUNT is refused")

	code, body := createBlogPost(t, okUser.ID, `{"title":"Hello World!","content":"body","visible":true}`)
	require.Equal(t, http.StatusCreated, code, string(mustJSON(t, body)))
	created := body["blog_post"].(map[string]any)
	require.Equal(t, "helloworld", created["slug"], "v1 keeps only letters and digits")

	var post models.BlogPost
	require.NoError(t, db.DB.Preload("Authors.User").First(&post, uint(created["id"].(float64))).Error)
	require.Equal(t, "Hello World!", post.Title)
	require.True(t, post.Visible)
	require.Nil(t, post.OrganizationID)
	require.Len(t, post.Authors, 1)
	require.Equal(t, "ok", post.Authors[0].User.Username)

	// A superuser skips the problem-count gate.
	code, _ = createBlogPost(t, root.ID, `{"title":"Root Post","content":"body"}`)
	require.Equal(t, http.StatusCreated, code)

	// Reusing a title derives a distinct slug instead of failing on the
	// duplicate.
	code, body = createBlogPost(t, okUser.ID, `{"title":"Hello World!","content":"another"}`)
	require.Equal(t, http.StatusCreated, code)
	require.Equal(t, "helloworld-2", body["blog_post"].(map[string]any)["slug"])
}

func TestPublicBlogPostCreate_OrganizationGate(t *testing.T) {
	setupAdminBlogPostDB(t)
	authorUser, author := seedAccount(t, "author", false, false)
	require.NoError(t, db.DB.Model(&author).Update("problem_count", blogMinProblemCount).Error)
	// The org admin has solved nothing: v1's organization route skips the
	// problem-count gate.
	otherUser, other := seedAccount(t, "other", false, false)

	org := models.Organization{Name: "acme", Slug: "acme", ShortName: "ACME", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&org).Error)
	seedOrgWithAdmin(t, &org, other)

	orgBody := `{"title":"Org Post","content":"body","organization_id":` + strconv.Itoa(int(org.ID)) + `}`
	code, _ := createBlogPost(t, authorUser.ID, orgBody)
	require.Equal(t, http.StatusForbidden, code, "a non-admin cannot post as the organization")

	code, body := createBlogPost(t, otherUser.ID, orgBody)
	require.Equal(t, http.StatusCreated, code, string(mustJSON(t, body)))

	var post models.BlogPost
	require.NoError(t, db.DB.First(&post, uint(body["blog_post"].(map[string]any)["id"].(float64))).Error)
	require.NotNil(t, post.OrganizationID)
	require.Equal(t, org.ID, *post.OrganizationID)

	code, _ = createBlogPost(t, authorUser.ID,
		`{"title":"Missing Org","content":"body","organization_id":99999}`)
	require.Equal(t, http.StatusBadRequest, code)
}

func TestPublicBlogPostUpdate_Ownership(t *testing.T) {
	setupAdminBlogPostDB(t)
	authorUser, author := seedAccount(t, "author", false, false)
	editorUser, editor := seedAccount(t, "editor", false, false)
	orgAdminUser, orgAdmin := seedAccount(t, "orgadmin", false, false)
	root, _ := seedAccount(t, "root", true, true)

	org := models.Organization{Name: "acme", Slug: "acme", ShortName: "ACME", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&org).Error)
	seedOrgWithAdmin(t, &org, orgAdmin)

	mine := seedBlogPost(t, "mine", "mine", true, time.Now().Add(-time.Hour), &author, nil)
	orgPost := seedBlogPost(t, "orgpost", "orgpost", true, time.Now().Add(-time.Hour), &author, &org)
	personal := seedBlogPost(t, "personal", "personal", true, time.Now().Add(-time.Hour), &editor, nil)

	newPublish := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	body := `{"title":"Mine, edited","content":"new body","visible":false,"publish_on":"` +
		newPublish.Format(time.RFC3339) + `"}`

	require.Equal(t, http.StatusOK, patchBlogPost(t, authorUser.ID, mine.ID, body))
	var updated models.BlogPost
	require.NoError(t, db.DB.First(&updated, mine.ID).Error)
	require.Equal(t, "Mine, edited", updated.Title)
	require.Equal(t, "new body", updated.Content)
	require.False(t, updated.Visible)
	require.True(t, newPublish.Equal(updated.PublishOn), "publish_on: %v", updated.PublishOn)

	require.Equal(t, http.StatusForbidden, patchBlogPost(t, editorUser.ID, mine.ID, body),
		"another user cannot edit a personal post")
	require.Equal(t, http.StatusForbidden, patchBlogPost(t, orgAdminUser.ID, mine.ID, body),
		"an org admin cannot edit a personal post")

	require.Equal(t, http.StatusOK, patchBlogPost(t, orgAdminUser.ID, orgPost.ID, body),
		"the organization's admin may edit its posts")
	require.Equal(t, http.StatusForbidden, patchBlogPost(t, editorUser.ID, orgPost.ID, body))

	require.Equal(t, http.StatusOK, patchBlogPost(t, root.ID, personal.ID, `{"title":"staff edit"}`))
	// PATCH clears the organization only on an explicit null.
	require.Equal(t, http.StatusOK, patchBlogPost(t, authorUser.ID, orgPost.ID, `{"organization_id":null}`))
	var cleared models.BlogPost
	require.NoError(t, db.DB.First(&cleared, orgPost.ID).Error)
	require.Nil(t, cleared.OrganizationID)

	require.Equal(t, http.StatusNotFound, patchBlogPost(t, authorUser.ID, 99999, `{"title":"x"}`))
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}