package v2

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// SitemapIndex - GET /sitemap.xml
// Returns the main sitemap index referencing individual sitemaps
func SitemapIndex(c *gin.Context) {
	baseURL := getBaseURL(c)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>` + baseURL + `/sitemap-static.xml</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
  </sitemap>
  <sitemap>
    <loc>` + baseURL + `/sitemap-problems.xml</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
  </sitemap>
  <sitemap>
    <loc>` + baseURL + `/sitemap-problem-groups.xml</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
  </sitemap>
  <sitemap>
    <loc>` + baseURL + `/sitemap-contests.xml</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
  </sitemap>
  <sitemap>
    <loc>` + baseURL + `/sitemap-users.xml</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
  </sitemap>
  <sitemap>
    <loc>` + baseURL + `/sitemap-blog-posts.xml</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
  </sitemap>
  <sitemap>
    <loc>` + baseURL + `/sitemap-organizations.xml</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
  </sitemap>
</sitemapindex>`

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// SitemapStatic - GET /sitemap-static.xml
// Returns static pages sitemap
func SitemapStatic(c *gin.Context) {
	baseURL := getBaseURL(c)
	now := time.Now().Format("2006-01-02")

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>` + baseURL + `/</loc>
    <lastmod>` + now + `</lastmod>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>` + baseURL + `/problems</loc>
    <lastmod>` + now + `</lastmod>
    <changefreq>daily</changefreq>
    <priority>0.9</priority>
  </url>
  <url>
    <loc>` + baseURL + `/contests</loc>
    <lastmod>` + now + `</lastmod>
    <changefreq>hourly</changefreq>
    <priority>0.9</priority>
  </url>
  <url>
    <loc>` + baseURL + `/users</loc>
    <lastmod>` + now + `</lastmod>
    <changefreq>daily</changefreq>
    <priority>0.8</priority>
  </url>
  <url>
    <loc>` + baseURL + `/blog</loc>
    <lastmod>` + now + `</lastmod>
    <changefreq>hourly</changefreq>
    <priority>0.8</priority>
  </url>
  <url>
    <loc>` + baseURL + `/organizations</loc>
    <lastmod>` + now + `</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.7</priority>
  </url>
  <url>
    <loc>` + baseURL + `/rankings</loc>
    <lastmod>` + now + `</lastmod>
    <changefreq>daily</changefreq>
    <priority>0.8</priority>
  </url>
  <url>
    <loc>` + baseURL + `/submissions</loc>
    <lastmod>` + now + `</lastmod>
    <changefreq>hourly</changefreq>
    <priority>0.7</priority>
  </url>
  <url>
    <loc>` + baseURL + `/about</loc>
    <lastmod>` + now + `</lastmod>
    <changefreq>monthly</changefreq>
    <priority>0.5</priority>
  </url>
</urlset>`

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// SitemapProblems - GET /sitemap-problems.xml
// Returns problems sitemap
func SitemapProblems(c *gin.Context) {
	baseURL := getBaseURL(c)

	var problems []models.Problem
	if err := db.DB.Where("is_public = ?", true).
		Select("code, is_manually_managed").
		Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`

	for _, p := range problems {
		xml += `
  <url>
    <loc>` + baseURL + `/problems/` + p.Code + `</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.8</priority>
  </url>`
	}

	xml += `
</urlset>`

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// SitemapProblemGroups - GET /sitemap-problem-groups.xml
// Returns problem groups sitemap
func SitemapProblemGroups(c *gin.Context) {
	baseURL := getBaseURL(c)

	var groups []models.ProblemGroup
	if err := db.DB.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`

	for _, g := range groups {
		xml += `
  <url>
    <loc>` + baseURL + `/problems/?group=` + g.Name + `</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
    <changefreq>monthly</changefreq>
    <priority>0.6</priority>
  </url>`
	}

	xml += `
</urlset>`

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// SitemapContests - GET /sitemap-contests.xml
// Returns contests sitemap
func SitemapContests(c *gin.Context) {
	baseURL := getBaseURL(c)

	var contests []models.Contest
	if err := db.DB.Where("is_visible = ? OR is_public = ?", true, true).
		Select("key").
		Find(&contests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`

	for _, ct := range contests {
		xml += `
  <url>
    <loc>` + baseURL + `/contests/` + ct.Key + `</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
    <changefreq>daily</changefreq>
    <priority>0.8</priority>
  </url>`
	}

	xml += `
</urlset>`

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// SitemapUsers - GET /sitemap-users.xml
// Returns users sitemap
func SitemapUsers(c *gin.Context) {
	baseURL := getBaseURL(c)
	pageStr := c.Query("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	pageSize := 1000
	offset := (page - 1) * pageSize

	var users []struct {
		UserID   uint   `gorm:"column:user_id"`
		Username string `gorm:"column:username"`
	}
	if err := db.DB.Table("judge_profile").
		Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
		Where("auth_user.is_active = ? AND judge_profile.is_unlisted = ?", true, false).
		Select("judge_profile.user_id, auth_user.username").
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	if len(users) == 0 {
		xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
</urlset>`
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusOK, xml)
		return
	}

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`

	for _, u := range users {
		xml += `
  <url>
    <loc>` + baseURL + `/user/` + escapeXML(u.Username) + `</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.7</priority>
  </url>`
	}

	xml += `
</urlset>`

	if len(users) == pageSize {
		// More pages available
		nextPage := page + 1
		xml = strings.Replace(xml, "</urlset>", `
  <url>
    <loc>`+baseURL+`/sitemap-users.xml?page=`+strconv.Itoa(nextPage)+`</loc>
  </url>
</urlset>`, 1)
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// SitemapBlogPosts - GET /sitemap-blog-posts.xml
// Returns blog posts sitemap
func SitemapBlogPosts(c *gin.Context) {
	baseURL := getBaseURL(c)

	var posts []models.BlogPost
	if err := db.DB.Where("visible = ? AND publish_on <= ?", true, time.Now()).
		Select("id").
		Order("publish_on DESC").
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`

	for _, p := range posts {
		xml += `
  <url>
    <loc>` + baseURL + `/blog/` + strconv.Itoa(int(p.ID)) + `</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
    <changefreq>monthly</changefreq>
    <priority>0.6</priority>
  </url>`
	}

	xml += `
</urlset>`

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// SitemapOrganizations - GET /sitemap-organizations.xml
// Returns organizations sitemap
func SitemapOrganizations(c *gin.Context) {
	baseURL := getBaseURL(c)

	var orgs []models.Organization
	if err := db.DB.Where("is_public = ?", true).
		Select("id, name").
		Find(&orgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`

	for _, o := range orgs {
		xml += `
  <url>
    <loc>` + baseURL + `/organization/` + strconv.Itoa(int(o.ID)) + `</loc>
    <lastmod>` + time.Now().Format("2006-01-02") + `</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.6</priority>
  </url>`
	}

	xml += `
</urlset>`

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// getBaseURL returns the base URL for the site
func getBaseURL(c *gin.Context) string {
	// Try to get from config, fallback to default
	baseURL := "https://beta.claoj.edu.vn"

	// Check if we have a site URL in context or config
	if siteURL := c.GetHeader("X-Site-URL"); siteURL != "" {
		baseURL = siteURL
	}

	return baseURL
}
