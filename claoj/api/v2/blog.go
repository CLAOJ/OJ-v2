package v2

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/contribution"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/blogpost"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// blogMinProblemCount mirrors v1 settings.CLAOJ_BLOG_MIN_PROBLEM_COUNT: a
// personal post needs this many solved problems (superusers are exempt).
const blogMinProblemCount = 10

// BlogList – GET /api/v2/blogs
// Optional filters: ?organization=<id>, ?author=<username|profile id>.
func BlogList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	orgID := c.Query("organization")
	author := c.Query("author")

	var posts []models.BlogPost
	query := db.DB.Model(&models.BlogPost{}).Preload("Authors.User").
		Where("visible = ? AND publish_on <= ?", true, time.Now())

	if author != "" {
		// v1 UserBlogPage: one author's personal (non-organization) posts.
		profileID, ok := resolveAuthorProfileID(author)
		if !ok {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return
		}
		query = query.Joins("JOIN judge_blogpost_authors jba ON jba.blogpost_id = judge_blogpost.id").
			Where("jba.profile_id = ?", profileID).
			Where("organization_id IS NULL")
	}

	if orgID != "" {
		query = query.Where("organization_id = ?", orgID)
	} else if author == "" {
		query = query.Where("organization_id IS NULL AND global_post = ?", true)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	if err := query.Order("sticky DESC, publish_on DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Author struct {
		Username string `json:"username"`
	}
	type Item struct {
		ID        uint      `json:"id"`
		Title     string    `json:"title"`
		Slug      string    `json:"slug"`
		Authors   []Author  `json:"authors"`
		PublishOn time.Time `json:"publish_on"`
		Summary   string    `json:"summary"`
		Score     int       `json:"score"`
		Sticky    bool      `json:"sticky"`
	}

	items := make([]Item, len(posts))
	for i, p := range posts {
		authors := make([]Author, len(p.Authors))
		for j, a := range p.Authors {
			authors[j] = Author{a.User.Username}
		}
		items[i] = Item{
			p.ID, p.Title, p.Slug, authors, p.PublishOn, p.Summary, p.Score, p.Sticky,
		}
	}
	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// resolveAuthorProfileID maps a blog author filter — an auth_user username or
// a judge_profile.id — to the profile id. ok=false means no such profile.
func resolveAuthorProfileID(author string) (uint, bool) {
	var profile models.Profile
	if err := db.DB.Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("au.username = ?", author).First(&profile).Error; err == nil {
		return profile.ID, true
	}
	id, err := strconv.ParseUint(author, 10, 64)
	if err != nil {
		return 0, false
	}
	var count int64
	db.DB.Model(&models.Profile{}).Where("id = ?", id).Count(&count)
	return uint(id), count > 0
}

// BlogDetail – GET /api/v2/blog/:id
func BlogDetail(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	var post models.BlogPost
	if err := db.DB.Preload("Authors.User").
		Preload("Organization").
		First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("blog post not found"))
		return
	}

	// Check visibility manually for detail if not published or if unauthenticated
	if !post.Visible || post.PublishOn.After(time.Now()) {
		// Only authors or superadmins should see
		_, profile, ok := resolveUserProfile(c)
		if !ok || !canEditBlog(db.DB, &post, profile.ID) {
			c.JSON(http.StatusNotFound, apiError("blog post not found"))
			return
		}
	}

	// Serialize authors as clean DTOs instead of leaking full GORM models
	type AuthorDTO struct {
		Username string `json:"username"`
	}
	authorDTOs := make([]AuthorDTO, len(post.Authors))
	for i, a := range post.Authors {
		authorDTOs[i] = AuthorDTO{a.User.Username}
	}

	// can_edit drives the "edit post" link on the post page; it is the same
	// gate PublicBlogPostUpdate applies (author, org admin, or staff).
	canEdit := false
	if profileID, ok := auth.CurrentProfileID(c); ok && profileID > 0 {
		canEdit = canEditBlog(db.DB, &post, profileID) || auth.HasPerm(c, "judge.edit_all_post")
	}

	organizationID := post.OrganizationID
	organizationName := ""
	if post.Organization != nil {
		organizationName = post.Organization.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                post.ID,
		"title":             post.Title,
		"slug":              post.Slug,
		"content":           post.Content,
		"publish_on":        post.PublishOn,
		"summary":           post.Summary,
		"score":             post.Score,
		"sticky":            post.Sticky,
		"visible":           post.Visible,
		"organization_id":   organizationID,
		"organization_name": organizationName,
		"can_edit":          canEdit,
		"authors":           authorDTOs,
	})
}

// ============================================================
// PUBLIC BLOG AUTHORING (v1 /post/new, /post/<id>/edit)
// ============================================================

// PublicBlogPostCreateRequest is the v1 BlogPostForm: title, publish_on,
// visible, content. The organization is chosen on the form when the author
// will post as one of their organizations.
type PublicBlogPostCreateRequest struct {
	Title          string     `json:"title" binding:"required"`
	Content        string     `json:"content" binding:"required"`
	PublishOn      *time.Time `json:"publish_on"`
	Visible        *bool      `json:"visible"`
	OrganizationID *uint      `json:"organization_id"`
}

// PublicBlogPostCreate – POST /api/v2/blogs
// v1 BlogPostCreate gate: a personal post needs blogMinProblemCount solved
// problems (or a superuser); posting as an organization needs its admin
// instead (v1's organization create view skips the problem-count check).
func PublicBlogPostCreate(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var req PublicBlogPostCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	if req.OrganizationID != nil {
		var count int64
		db.DB.Model(&models.Organization{}).Where("id = ?", *req.OrganizationID).Count(&count)
		if count == 0 {
			c.JSON(http.StatusBadRequest, apiUserError("organization not found"))
			return
		}
		if !isOrgAdmin(db.DB, *req.OrganizationID, profile.ID) {
			c.JSON(http.StatusForbidden, apiUserError("only the organization's administrators can post as it"))
			return
		}
	} else if profile.ProblemCount < blogMinProblemCount && !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiUserError("you need to solve at least "+
			strconv.Itoa(blogMinProblemCount)+" problems to create a blog post"))
		return
	}

	publishOn := time.Now()
	if req.PublishOn != nil {
		publishOn = *req.PublishOn
	}
	visible := false
	if req.Visible != nil {
		visible = *req.Visible
	}

	// v1 derives the initial slug from the title (letters and digits only) and
	// does not carry the form's publish_on; an empty derivation falls back to
	// "post".
	post, err := getBlogPostService().CreateBlogPost(blogpost.CreateBlogPostRequest{
		Title:          req.Title,
		Slug:           uniqueBlogSlug(req.Title),
		Content:        req.Content,
		PublishOn:      publishOn,
		Visible:        visible,
		AuthorIDs:      []uint{profile.ID},
		OrganizationID: req.OrganizationID,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "blog post created",
		"blog_post": gin.H{"id": post.ID, "slug": post.Slug},
	})
}

// PublicBlogPostUpdateRequest is the editable part of a post.
type PublicBlogPostUpdateRequest struct {
	Title     *string    `json:"title"`
	Content   *string    `json:"content"`
	PublishOn *time.Time `json:"publish_on"`
	Visible   *bool      `json:"visible"`
	// Raw so an explicit null (detach from the organization) differs from an
	// omitted key (leave it unchanged).
	OrganizationID json.RawMessage `json:"organization_id"`
}

// PublicBlogPostUpdate – PATCH /api/v2/blog/:id
// Authors edit their own posts; an organization's admins (and holders of
// judge.edit_all_post) may edit its posts.
func PublicBlogPostUpdate(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var post models.BlogPost
	if err := db.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("blog post not found"))
		return
	}
	staffEdit := auth.HasPerm(c, "judge.edit_all_post")
	if !staffEdit && !canEditBlog(db.DB, &post, profile.ID) {
		c.JSON(http.StatusForbidden, apiUserError("you cannot edit this blog post"))
		return
	}

	var req PublicBlogPostUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	organizationID, clearOrganization, validOrg := parseOrgID(req.OrganizationID)
	if !validOrg {
		c.JSON(http.StatusBadRequest, apiUserError("invalid organization_id"))
		return
	}
	// An echoed value is not a change: an author who is not the organization's
	// admin may resubmit the form without tripping the gate.
	changingOrg := clearOrganization || (organizationID != nil &&
		(post.OrganizationID == nil || *post.OrganizationID != *organizationID))
	if changingOrg && organizationID != nil && !staffEdit {
		var count int64
		db.DB.Model(&models.Organization{}).Where("id = ?", *organizationID).Count(&count)
		if count == 0 {
			c.JSON(http.StatusBadRequest, apiUserError("organization not found"))
			return
		}
		if !isOrgAdmin(db.DB, *organizationID, profile.ID) {
			c.JSON(http.StatusForbidden, apiUserError("only the organization's administrators can post as it"))
			return
		}
	}

	updated, err := getBlogPostService().UpdateBlogPost(blogpost.UpdateBlogPostRequest{
		BlogPostID:        post.ID,
		Title:             req.Title,
		Content:           req.Content,
		PublishOn:         req.PublishOn,
		Visible:           req.Visible,
		OrganizationID:    organizationID,
		ClearOrganization: clearOrganization,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "blog post updated",
		"blog_post": gin.H{"id": updated.ID, "slug": updated.Slug},
	})
}

// uniqueBlogSlug derives v1's title-based slug ("letters and digits only")
// and appends -2, -3, … when that slug is already taken.
func uniqueBlogSlug(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	base := b.String()
	if base == "" {
		base = "post"
	}
	slug := base
	for i := 2; ; i++ {
		var count int64
		db.DB.Model(&models.BlogPost{}).Where("slug = ?", slug).Count(&count)
		if count == 0 {
			return slug
		}
		slug = base + "-" + strconv.Itoa(i)
	}
}

// BlogVoteHandler – POST /api/v2/blog/:id/vote
func BlogVoteHandler(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if profile.Mute {
		c.JSON(http.StatusForbidden, apiError("you are muted and cannot vote"))
		return
	}

	if profile.ProblemCount == 0 {
		c.JSON(http.StatusBadRequest, apiError("you must solve at least one problem to vote"))
		return
	}

	var reqBody struct {
		Delta int `json:"delta" binding:"required"` // 1 or -1
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	if reqBody.Delta != 1 && reqBody.Delta != -1 {
		c.JSON(http.StatusBadRequest, apiError("invalid delta"))
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var post models.BlogPost
		if err := tx.First(&post, id).Error; err != nil {
			return errors.New("blog post not found")
		}

		// Cannot vote own post
		var authorExists bool
		tx.Table("judge_blogpost_authors").
			Where("blogpost_id = ? AND profile_id = ?", post.ID, profile.ID).
			Limit(1).Select("1").Scan(&authorExists)
		if authorExists {
			return errors.New("you cannot vote for your own blog post")
		}

		var vote models.BlogVote
		err := tx.Where("blog_id = ? AND voter_id = ?", post.ID, profile.ID).First(&vote).Error
		if err == nil {
			// Vote exists
			if vote.Score == reqBody.Delta {
				// Retract vote
				if err := tx.Delete(&vote).Error; err != nil {
					return err
				}
				post.Score -= reqBody.Delta
			} else {
				// Change vote
				oldDelta := vote.Score
				vote.Score = reqBody.Delta
				if err := tx.Save(&vote).Error; err != nil {
					return err
				}
				post.Score = post.Score - oldDelta + reqBody.Delta
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// New vote
			vote = models.BlogVote{
				BlogID:  post.ID,
				VoterID: profile.ID,
				Score:   reqBody.Delta,
			}
			if err := tx.Create(&vote).Error; err != nil {
				return err
			}
			post.Score += reqBody.Delta
		} else {
			return err
		}

		err = tx.Model(&post).Update("score", post.Score).Error
		if err == nil {
			// Update every author's contribution points (authorship is the
			// judge_blogpost_authors m2m; judge_blogpost has no author_id).
			var authorProfileIDs []uint
			tx.Table("judge_blogpost_authors").
				Where("blogpost_id = ?", post.ID).
				Pluck("profile_id", &authorProfileIDs)
			for _, pid := range authorProfileIDs {
				contribution.UpdateProfileContributionPoints(pid)
			}
		}
		return err
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vote recorded"})
}

func canEditBlog(tx *gorm.DB, post *models.BlogPost, profileID uint) bool {
	// Check if author
	var exists bool
	tx.Table("judge_blogpost_authors").
		Where("blogpost_id = ? AND profile_id = ?", post.ID, profileID).
		Limit(1).Select("1").Scan(&exists)
	if exists {
		return true
	}
	// Check if org admin
	if post.OrganizationID != nil {
		return isOrgAdmin(tx, *post.OrganizationID, profileID)
	}
	return false
}

// escapeXML escapes special XML characters using Go's built-in function
func escapeXML(s string) string {
	// Build escaped string manually since xml.EscapeText needs a writer
	result := ""
	for _, r := range s {
		switch r {
		case '&':
			result += "&amp;"
		case '<':
			result += "&lt;"
		case '>':
			result += "&gt;"
		case '"':
			result += "&quot;"
		case '\'':
			result += "&apos;"
		default:
			result += string(r)
		}
	}
	return result
}
