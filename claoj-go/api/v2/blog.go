package v2

import (
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BlogList – GET /api/v2/blogs
func BlogList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	orgID := c.Query("organization")

	var posts []models.BlogPost
	query := db.DB.Preload("Authors.User").
		Where("visible = ? AND publish_on <= ?", true, time.Now())

	if orgID != "" {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("organization_id IS NULL AND global_post = ?", true)
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
	c.JSON(http.StatusOK, apiList(items))
}

// BlogDetail – GET /api/v2/blog/:id
func BlogDetail(c *gin.Context) {
	id := c.Param("id")
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

	c.JSON(http.StatusOK, gin.H{
		"id":         post.ID,
		"title":      post.Title,
		"slug":       post.Slug,
		"content":    post.Content,
		"publish_on": post.PublishOn,
		"summary":    post.Summary,
		"score":      post.Score,
		"sticky":     post.Sticky,
		"authors":    authorDTOs,
	})
}

// BlogVoteHandler – POST /api/v2/blog/:id/vote
func BlogVoteHandler(c *gin.Context) {
	id := c.Param("id")
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

		return tx.Model(&post).Update("score", post.Score).Error
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
