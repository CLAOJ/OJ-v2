package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/blogpost"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN BLOG POST MANAGEMENT API
// ============================================================

// AdminBlogPostList - GET /api/v2/admin/blog-posts
// List all blog posts
func AdminBlogPostList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var visible *bool
	switch c.Query("visible") {
	case "true":
		v := true
		visible = &v
	case "false":
		v := false
		visible = &v
	}

	resp, err := getBlogPostService().ListBlogPosts(blogpost.ListBlogPostsRequest{
		Page:        page,
		PageSize:    pageSize,
		VisibleOnly: false,
		Search:      c.Query("search"),
		Visible:     visible,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type BlogPostItem struct {
		ID               uint      `json:"id"`
		Title            string    `json:"title"`
		Slug             string    `json:"slug"`
		AuthorNames      []string  `json:"author_names"`
		PublishOn        time.Time `json:"publish_on"`
		Visible          bool      `json:"visible"`
		Sticky           bool      `json:"sticky"`
		GlobalPost       bool      `json:"global_post"`
		OrganizationID   *uint     `json:"organization_id"`
		OrganizationName *string   `json:"organization_name"`
		Score            int       `json:"score"`
	}

	items := make([]BlogPostItem, len(resp.BlogPosts))
	for i, post := range resp.BlogPosts {
		items[i] = BlogPostItem{
			ID:               post.ID,
			Title:            post.Title,
			Slug:             post.Slug,
			AuthorNames:      post.AuthorNames,
			PublishOn:        post.PublishOn,
			Visible:          post.Visible,
			Sticky:           post.Sticky,
			GlobalPost:       post.GlobalPost,
			OrganizationID:   post.OrganizationID,
			OrganizationName: post.OrganizationName,
			Score:            post.Score,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, resp.Total))
}

// AdminBlogPostDetail - GET /api/v2/admin/blog-post/:id
// Get blog post detail
func AdminBlogPostDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid blog post id"))
		return
	}

	detail, err := getBlogPostService().GetBlogPost(blogpost.GetBlogPostRequest{
		BlogPostID: uint(id),
	})
	if err != nil {
		if err == blogpost.ErrBlogPostNotFound {
			c.JSON(http.StatusNotFound, apiError("blog post not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                detail.BlogPost.ID,
		"title":             detail.BlogPost.Title,
		"slug":              detail.BlogPost.Slug,
		"content":           detail.BlogPost.Content,
		"summary":           detail.BlogPost.Summary,
		"author_ids":        detail.BlogPost.AuthorIDs,
		"author_names":      detail.AuthorNames,
		"publish_on":        detail.BlogPost.PublishOn,
		"visible":           detail.BlogPost.Visible,
		"sticky":            detail.BlogPost.Sticky,
		"global_post":       detail.BlogPost.GlobalPost,
		"og_image":          detail.BlogPost.OgImage,
		"organization_id":   detail.BlogPost.OrganizationID,
		"organization_name": detail.OrganizationName,
		"score":             detail.BlogPost.Score,
	})
}

// AdminBlogPostCreateRequest - POST /api/v2/admin/blog-posts
type AdminBlogPostCreateRequest struct {
	Title          string    `json:"title" binding:"required"`
	Slug           string    `json:"slug" binding:"required"`
	Content        string    `json:"content" binding:"required"`
	Summary        string    `json:"summary" binding:"required"`
	AuthorIDs      []uint    `json:"author_ids"`
	PublishOn      time.Time `json:"publish_on" binding:"required"`
	Visible        bool      `json:"visible"`
	Sticky         bool      `json:"sticky"`
	GlobalPost     bool      `json:"global_post"`
	OgImage        string    `json:"og_image"`
	OrganizationID *uint     `json:"organization_id"`
}

// AdminBlogPostCreate - POST /api/v2/admin/blog-posts
// Create a new blog post
func AdminBlogPostCreate(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !auth.HasPerm(c, "judge.edit_all_post") {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminBlogPostCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	createBlogPostFromInput(c, req, profile.ID, nil)
}

// createBlogPostFromInput validates and inserts a blog post. A non-nil org
// forces the organization onto the post (v1 BlogPostCreateOrganization) and
// keeps the post off the global feed.
func createBlogPostFromInput(c *gin.Context, req AdminBlogPostCreateRequest, callerProfileID uint, org *models.Organization) {
	authorIDs := req.AuthorIDs
	if len(authorIDs) == 0 {
		authorIDs = []uint{callerProfileID}
	}
	if org != nil {
		req.OrganizationID = &org.ID
		req.GlobalPost = false
		authorIDs = withProfileID(authorIDs, callerProfileID)
	}

	post, err := getBlogPostService().CreateBlogPost(blogpost.CreateBlogPostRequest{
		Title:          req.Title,
		Slug:           req.Slug,
		Content:        req.Content,
		Summary:        req.Summary,
		AuthorIDs:      authorIDs,
		PublishOn:      req.PublishOn,
		Visible:        req.Visible,
		Sticky:         req.Sticky,
		GlobalPost:     req.GlobalPost,
		OgImage:        req.OgImage,
		OrganizationID: req.OrganizationID,
	})
	if err != nil {
		if err == blogpost.ErrSlugExists {
			c.JSON(http.StatusBadRequest, apiUserError("slug already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "blog post created",
		"blog_post": gin.H{"id": post.ID, "slug": post.Slug},
	})
}

// AdminBlogPostUpdateRequest - PATCH /api/v2/admin/blog-post/:id
type AdminBlogPostUpdateRequest struct {
	Title      *string    `json:"title"`
	Slug       *string    `json:"slug"`
	Content    *string    `json:"content"`
	Summary    *string    `json:"summary"`
	AuthorIDs  []uint     `json:"author_ids"`
	PublishOn  *time.Time `json:"publish_on"`
	Visible    *bool      `json:"visible"`
	Sticky     *bool      `json:"sticky"`
	GlobalPost *bool      `json:"global_post"`
	OgImage    *string    `json:"og_image"`
	// Kept raw so an explicit null (detach from organization) differs from an
	// omitted key (leave it unchanged). A plain *uint collapses both to nil.
	OrganizationID json.RawMessage `json:"organization_id"`
}

// parseOrgID maps the raw organization_id of a PATCH body: absent leaves the
// post's organization alone, null clears it, a non-zero number sets it.
// ok is false for anything else (a string, 0, an object).
func parseOrgID(raw json.RawMessage) (id *uint, clear, ok bool) {
	if len(raw) == 0 {
		return nil, false, true
	}
	if string(raw) == "null" {
		return nil, true, true
	}
	if err := json.Unmarshal(raw, &id); err != nil || id == nil || *id == 0 {
		return nil, false, false
	}
	return id, false, true
}

// AdminBlogPostUpdate - PATCH /api/v2/admin/blog-post/:id
// Update a blog post
func AdminBlogPostUpdate(c *gin.Context) {
	_, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !auth.HasPerm(c, "judge.edit_all_post") {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid blog post id"))
		return
	}

	var req AdminBlogPostUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	organizationID, clearOrganization, validOrg := parseOrgID(req.OrganizationID)
	if !validOrg {
		c.JSON(http.StatusBadRequest, apiError("invalid organization_id"))
		return
	}

	post, err := getBlogPostService().UpdateBlogPost(blogpost.UpdateBlogPostRequest{
		BlogPostID:        uint(id),
		Title:             req.Title,
		Slug:              req.Slug,
		Content:           req.Content,
		Summary:           req.Summary,
		AuthorIDs:         req.AuthorIDs,
		PublishOn:         req.PublishOn,
		Visible:           req.Visible,
		Sticky:            req.Sticky,
		GlobalPost:        req.GlobalPost,
		OgImage:           req.OgImage,
		OrganizationID:    organizationID,
		ClearOrganization: clearOrganization,
	})
	if err != nil {
		if err == blogpost.ErrBlogPostNotFound {
			c.JSON(http.StatusNotFound, apiError("blog post not found"))
			return
		}
		if err == blogpost.ErrSlugExists {
			c.JSON(http.StatusBadRequest, apiError("slug already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "blog post updated",
		"blog_post": gin.H{"id": post.ID, "slug": post.Slug},
	})
}

// AdminBlogPostDelete - DELETE /api/v2/admin/blog-post/:id
// Delete a blog post
func AdminBlogPostDelete(c *gin.Context) {
	_, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !auth.HasPerm(c, "judge.edit_all_post") {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid blog post id"))
		return
	}

	if err := getBlogPostService().DeleteBlogPost(blogpost.DeleteBlogPostRequest{
		BlogPostID: uint(id),
	}); err != nil {
		if err == blogpost.ErrBlogPostNotFound {
			c.JSON(http.StatusNotFound, apiError("blog post not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "blog post deleted",
	})
}
