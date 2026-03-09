// Package comment provides comment management services.
package comment

import (
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
	"gorm.io/gorm"
)

// CommentService provides comment management operations.
type CommentService struct{}

// NewCommentService creates a new CommentService instance.
func NewCommentService() *CommentService {
	return &CommentService{}
}

// GetComment retrieves a comment by ID.
func (s *CommentService) GetComment(req GetCommentRequest) (*Comment, error) {
	if req.CommentID == 0 {
		return nil, ErrInvalidCommentID
	}

	var comment models.Comment
	if err := db.DB.First(&comment, req.CommentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	result := commentToModel(comment)
	return &result, nil
}

// UpdateComment updates a comment with optional revision tracking.
// If the body is changed, a revision record is created before updating.
func (s *CommentService) UpdateComment(req UpdateCommentRequest) (*Comment, error) {
	if req.CommentID == 0 {
		return nil, ErrInvalidCommentID
	}

	var comment models.Comment
	if err := db.DB.First(&comment, req.CommentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	// Create revision if body is being changed
	if req.Body != nil && *req.Body != comment.Body {
		revision := models.CommentRevision{
			CommentID: comment.ID,
			EditorID:  req.EditorID,
			Time:      time.Now(),
			Body:      comment.Body, // Save old body
			Reason:    req.Reason,
		}
		if err := db.DB.Create(&revision).Error; err != nil {
			return nil, ErrRevisionCreateFail
		}
		// Sanitize and update body
		comment.Body = sanitization.SanitizeComment(*req.Body)
	}

	if req.Hidden != nil {
		comment.Hidden = *req.Hidden
	}

	if err := db.DB.Save(&comment).Error; err != nil {
		return nil, err
	}

	result := commentToModel(comment)
	return &result, nil
}

// DeleteComment performs a hard delete of a comment.
func (s *CommentService) DeleteComment(req DeleteCommentRequest) error {
	if req.CommentID == 0 {
		return ErrInvalidCommentID
	}

	var comment models.Comment
	if err := db.DB.First(&comment, req.CommentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrCommentNotFound
		}
		return err
	}

	return db.DB.Delete(&comment).Error
}

// ListComments retrieves a paginated list of comments.
func (s *CommentService) ListComments(req ListCommentsRequest) (*ListCommentsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var comments []models.Comment
	query := db.DB.Model(&models.Comment{}).Preload("Author.User")

	if req.PageFilter != "" {
		query = query.Where("page = ?", req.PageFilter)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Order("time DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&comments).Error; err != nil {
		return nil, err
	}

	result := make([]Comment, len(comments))
	for i, c := range comments {
		result[i] = commentToModel(c)
	}

	return &ListCommentsResponse{
		Comments: result,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetCommentRevisions retrieves all revisions for a comment.
func (s *CommentService) GetCommentRevisions(commentID uint) ([]CommentRevision, error) {
	var revisions []models.CommentRevision
	if err := db.DB.Where("comment_id = ?", commentID).
		Preload("Editor.User").
		Order("time DESC").
		Find(&revisions).Error; err != nil {
		return nil, err
	}

	result := make([]CommentRevision, len(revisions))
	for i, r := range revisions {
		result[i] = revisionToModel(r)
	}
	return result, nil
}

// Helper functions

func commentToModel(c models.Comment) Comment {
	return Comment{
		ID:       c.ID,
		AuthorID: c.AuthorID,
		Time:     c.Time,
		Page:     c.Page,
		Score:    c.Score,
		Body:     c.Body,
		Hidden:   c.Hidden,
		ParentID: c.ParentID,
	}
}

func revisionToModel(r models.CommentRevision) CommentRevision {
	return CommentRevision{
		ID:        r.ID,
		CommentID: r.CommentID,
		EditorID:  r.EditorID,
		Time:      r.Time,
		Body:      r.Body,
		Reason:    r.Reason,
	}
}
