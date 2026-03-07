package v2

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj-go/service/blogpost"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN BLOG POST MANAGEMENT API
// ============================================================

// AdminBlogPostList - GET /api/v2/admin/blog-posts
// List all blog posts
func AdminBlogPostList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getBlogPostService().ListBlogPosts(blogpost.ListBlogPostsRequest{
		Page:        page,
		PageSize:    pageSize,
		VisibleOnly: false,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type BlogPostItem struct {
		ID           uint      `json:"id"`
		Title        string    `json:"title"`
		Slug         string    `json:"slug"`
		AuthorNames  []string  `json:"author_names"`
		PublishOn    time.Time `json:"publish_on"`
		Visible      bool      `json:"visible"`
		Sticky       bool      `json:"sticky"`
		GlobalPost   bool      `json:"global_post"`
		Organization *string   `json:"organization,omitempty"`
		Score        int       `json:"score"`
	}

	items := make([]BlogPostItem, len(resp.BlogPosts))
	for i, post := range resp.BlogPosts {
		items[i] = BlogPostItem{
			ID:           post.ID,
			Title:        post.Title,
			Slug:         post.Slug,
			PublishOn:    post.PublishOn,
			Visible:      post.Visible,
			Sticky:       post.Sticky,
			GlobalPost:   post.GlobalPost,
			Score:        post.Score,
			Organization: nil,
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
	user, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser && !user.IsStaff {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminBlogPostCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	authorIDs := req.AuthorIDs
	if len(authorIDs) == 0 {
		authorIDs = []uint{profile.ID}
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
			c.JSON(http.StatusBadRequest, apiError("slug already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "blog post created",
		"blog_post": gin.H{"id": post.ID, "slug": post.Slug},
	})
}

// AdminBlogPostUpdateRequest - PATCH /api/v2/admin/blog-post/:id
type AdminBlogPostUpdateRequest struct {
	Title          *string    `json:"title"`
	Slug           *string    `json:"slug"`
	Content        *string    `json:"content"`
	Summary        *string    `json:"summary"`
	AuthorIDs      []uint     `json:"author_ids"`
	PublishOn      *time.Time `json:"publish_on"`
	Visible        *bool      `json:"visible"`
	Sticky         *bool      `json:"sticky"`
	GlobalPost     *bool      `json:"global_post"`
	OgImage        *string    `json:"og_image"`
	OrganizationID *uint      `json:"organization_id"`
}

// AdminBlogPostUpdate - PATCH /api/v2/admin/blog-post/:id
// Update a blog post
func AdminBlogPostUpdate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser && !user.IsStaff {
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

	post, err := getBlogPostService().UpdateBlogPost(blogpost.UpdateBlogPostRequest{
		BlogPostID:     uint(id),
		Title:          req.Title,
		Slug:           req.Slug,
		Content:        req.Content,
		Summary:        req.Summary,
		AuthorIDs:      req.AuthorIDs,
		PublishOn:      req.PublishOn,
		Visible:        req.Visible,
		Sticky:         req.Sticky,
		GlobalPost:     req.GlobalPost,
		OgImage:        req.OgImage,
		OrganizationID: req.OrganizationID,
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
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser && !user.IsStaff {
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
