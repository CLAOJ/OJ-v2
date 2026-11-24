package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ============================================================
// Admin blog post list: populated relations + filters + filtered total.
// ============================================================

func setupAdminBlogPostDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.BlogPost{},
	))
	db.DB = database
}

type blogListBody struct {
	Data  []map[string]any `json:"data"`
	Total int64            `json:"total"`
}

func seedBlogPost(t *testing.T, title, slug string, visible bool, publishOn time.Time, author *models.Profile, org *models.Organization) models.BlogPost {
	t.Helper()
	post := models.BlogPost{
		Title: title, Slug: slug, Content: "body", Summary: "sum",
		Visible: visible, PublishOn: publishOn, OrganizationID: orgID(org),
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&post).Error)
	if author != nil {
		require.NoError(t, db.DB.Model(&post).Association("Authors").Append(author))
	}
	return post
}

func orgID(org *models.Organization) *uint {
	if org == nil {
		return nil
	}
	return &org.ID
}

func listBlogPosts(t *testing.T, query string) blogListBody {
	t.Helper()
	router := authedRouter(1, http.MethodGet, "/admin/blog-posts", AdminBlogPostList)
	w := doRequest(router, http.MethodGet, "/admin/blog-posts"+query, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body blogListBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestAdminBlogPostList_PopulatesAuthorsAndOrganization(t *testing.T) {
	setupAdminBlogPostDB(t)
	_, ada := seedAccount(t, "ada", false, false)
	org := models.Organization{Name: "Acme", Slug: "acme", ShortName: "AC", CreationDate: time.Now()}
	require.NoError(t, db.DB.Create(&org).Error)

	seedBlogPost(t, "Hello", "hello", true, time.Now().Add(-time.Hour), &ada, &org)
	seedBlogPost(t, "Loose", "loose", true, time.Now().Add(-2*time.Hour), nil, nil)

	body := listBlogPosts(t, "")
	require.Equal(t, int64(2), body.Total)
	require.Len(t, body.Data, 2)

	first := body.Data[0] // publish_on DESC
	require.Equal(t, "Hello", first["title"])
	require.Equal(t, []any{"ada"}, first["author_names"])
	require.Equal(t, float64(org.ID), first["organization_id"])
	require.Equal(t, "Acme", first["organization_name"])
	require.NotContains(t, first, "organization", "list no longer uses the string-only organization field")

	second := body.Data[1]
	require.Equal(t, []any{}, second["author_names"])
	require.Nil(t, second["organization_id"])
	require.Nil(t, second["organization_name"])
}

func TestAdminBlogPostList_FiltersAndTotal(t *testing.T) {
	setupAdminBlogPostDB(t)
	_, ada := seedAccount(t, "ada", false, false)
	org := models.Organization{Name: "Acme", Slug: "acme", ShortName: "AC", CreationDate: time.Now()}
	require.NoError(t, db.DB.Create(&org).Error)

	now := time.Now()
	seedBlogPost(t, "Alpha visible", "alpha", true, now, &ada, &org)
	seedBlogPost(t, "Beta hidden", "beta", false, now.Add(-time.Hour), nil, nil)
	seedBlogPost(t, "Gamma visible", "gamma-news", true, now.Add(-2*time.Hour), nil, nil)

	t.Run("visible=true", func(t *testing.T) {
		body := listBlogPosts(t, "?visible=true")
		require.Equal(t, int64(2), body.Total)
		require.Len(t, body.Data, 2)
		for _, row := range body.Data {
			require.Equal(t, true, row["visible"])
		}
	})

	t.Run("visible=false", func(t *testing.T) {
		body := listBlogPosts(t, "?visible=false")
		require.Equal(t, int64(1), body.Total)
		require.Equal(t, "Beta hidden", body.Data[0]["title"])
	})

	t.Run("search matches title or slug and filters total", func(t *testing.T) {
		body := listBlogPosts(t, "?search=alpha")
		require.Equal(t, int64(1), body.Total)
		require.Equal(t, "alpha", body.Data[0]["slug"])

		body = listBlogPosts(t, "?search=news")
		require.Equal(t, int64(1), body.Total, "slug LIKE must be matched")
		require.Equal(t, "Gamma visible", body.Data[0]["title"])
	})

	t.Run("search and visible combine", func(t *testing.T) {
		body := listBlogPosts(t, "?search=a&visible=true")
		require.Equal(t, int64(2), body.Total, "alpha + gamma both visible")
		body = listBlogPosts(t, "?search=a&visible=false")
		require.Equal(t, int64(1), body.Total)
	})

	t.Run("unknown visible value is ignored", func(t *testing.T) {
		body := listBlogPosts(t, "?visible=maybe")
		require.Equal(t, int64(3), body.Total)
	})
}
