// Package blogpost provides blog post management services.
package blogpost

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
	"gorm.io/gorm"
)

// BlogPostService provides blog post management operations.
type BlogPostService struct{}

// NewBlogPostService creates a new BlogPostService instance.
func NewBlogPostService() *BlogPostService {
	return &BlogPostService{}
}

// ListBlogPosts retrieves a paginated list of blog posts.
func (s *BlogPostService) ListBlogPosts(req ListBlogPostsRequest) (*ListBlogPostsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var posts []models.BlogPost
	query := db.DB.Model(&models.BlogPost{}).
		Preload("Authors.User").
		Preload("Organization")

	if req.VisibleOnly {
		query = query.Where("visible = ?", true)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Order("publish_on DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&posts).Error; err != nil {
		return nil, err
	}

	result := make([]BlogPost, len(posts))
	for i, p := range posts {
		result[i] = blogPostToModel(p)
	}

	return &ListBlogPostsResponse{
		BlogPosts:  result,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

// GetBlogPost retrieves a blog post by ID with full details.
func (s *BlogPostService) GetBlogPost(req GetBlogPostRequest) (*BlogPostDetail, error) {
	if req.BlogPostID == 0 {
		return nil, ErrInvalidBlogPostID
	}

	var post models.BlogPost
	if err := db.DB.
		Preload("Authors.User").
		Preload("Organization").
		First(&post, req.BlogPostID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrBlogPostNotFound
		}
		return nil, err
	}

	detail := &BlogPostDetail{
		BlogPost: blogPostToModel(post),
	}

	authorNames := make([]string, len(post.Authors))
	for i, a := range post.Authors {
		authorNames[i] = a.User.Username
	}
	detail.AuthorNames = authorNames

	if post.Organization != nil {
		detail.OrganizationName = &post.Organization.Name
	}

	return detail, nil
}

// CreateBlogPost creates a new blog post.
func (s *BlogPostService) CreateBlogPost(req CreateBlogPostRequest) (*BlogPost, error) {
	if req.Title == "" {
		return nil, ErrEmptyTitle
	}
	if req.Slug == "" {
		return nil, ErrEmptySlug
	}
	if req.Content == "" {
		return nil, ErrEmptyContent
	}

	// Check if slug already exists
	var existing models.BlogPost
	if err := db.DB.Where("slug = ?", req.Slug).First(&existing).Error; err == nil {
		return nil, ErrSlugExists
	}

	post := models.BlogPost{
		Title:          req.Title,
		Slug:           req.Slug,
		Content:        sanitization.SanitizeBlogContent(req.Content),
		Summary:        sanitization.SanitizeBlogSummary(req.Summary),
		PublishOn:      req.PublishOn,
		Visible:        req.Visible,
		Sticky:         req.Sticky,
		GlobalPost:     req.GlobalPost,
		OgImage:        req.OgImage,
		OrganizationID: req.OrganizationID,
	}

	if err := db.DB.Create(&post).Error; err != nil {
		return nil, err
	}

	// Associate authors if provided
	if len(req.AuthorIDs) > 0 {
		var authors []models.Profile
		if err := db.DB.Where("id IN ?", req.AuthorIDs).Find(&authors).Error; err == nil {
			db.DB.Model(&post).Association("Authors").Append(&authors)
		}
	}

	// Reload post with relations
	db.DB.Preload("Authors").Preload("Organization").First(&post, post.ID)

	result := blogPostToModel(post)
	return &result, nil
}

// UpdateBlogPost updates an existing blog post.
func (s *BlogPostService) UpdateBlogPost(req UpdateBlogPostRequest) (*BlogPost, error) {
	if req.BlogPostID == 0 {
		return nil, ErrInvalidBlogPostID
	}

	var post models.BlogPost
	if err := db.DB.First(&post, req.BlogPostID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrBlogPostNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Slug != nil {
		// Check if new slug is taken by another post
		var existing models.BlogPost
		if err := db.DB.Where("slug = ? AND id != ?", *req.Slug, req.BlogPostID).First(&existing).Error; err == nil {
			return nil, ErrSlugExists
		}
		updates["slug"] = *req.Slug
	}
	if req.Content != nil {
		updates["content"] = sanitization.SanitizeBlogContent(*req.Content)
	}
	if req.Summary != nil {
		updates["summary"] = sanitization.SanitizeBlogSummary(*req.Summary)
	}
	if req.PublishOn != nil {
		updates["publish_on"] = *req.PublishOn
	}
	if req.Visible != nil {
		updates["visible"] = *req.Visible
	}
	if req.Sticky != nil {
		updates["sticky"] = *req.Sticky
	}
	if req.GlobalPost != nil {
		updates["global_post"] = *req.GlobalPost
	}
	if req.OgImage != nil {
		updates["og_image"] = *req.OgImage
	}
	if req.OrganizationID != nil {
		updates["organization_id"] = *req.OrganizationID
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&post).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Update authors if provided
	if req.AuthorIDs != nil {
		db.DB.Model(&post).Association("Authors").Clear()
		if len(req.AuthorIDs) > 0 {
			var authors []models.Profile
			if err := db.DB.Where("id IN ?", req.AuthorIDs).Find(&authors).Error; err == nil {
				db.DB.Model(&post).Association("Authors").Append(&authors)
			}
		}
	}

	// Reload post with relations
	db.DB.Preload("Authors").Preload("Organization").First(&post, post.ID)

	result := blogPostToModel(post)
	return &result, nil
}

// DeleteBlogPost performs a soft delete by setting visible to false.
func (s *BlogPostService) DeleteBlogPost(req DeleteBlogPostRequest) error {
	if req.BlogPostID == 0 {
		return ErrInvalidBlogPostID
	}

	var post models.BlogPost
	if err := db.DB.First(&post, req.BlogPostID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrBlogPostNotFound
		}
		return err
	}

	// Soft delete: set visible to false
	return db.DB.Model(&post).Update("visible", false).Error
}

// Helper functions

func blogPostToModel(p models.BlogPost) BlogPost {
	authorIDs := make([]uint, len(p.Authors))
	for i, a := range p.Authors {
		authorIDs[i] = a.ID
	}

	return BlogPost{
		ID:             p.ID,
		Title:          p.Title,
		Slug:           p.Slug,
		AuthorIDs:      authorIDs,
		PublishOn:      p.PublishOn,
		Content:        p.Content,
		Summary:        p.Summary,
		Visible:        p.Visible,
		Sticky:         p.Sticky,
		Score:          p.Score,
		GlobalPost:     p.GlobalPost,
		OgImage:        p.OgImage,
		OrganizationID: p.OrganizationID,
	}
}
