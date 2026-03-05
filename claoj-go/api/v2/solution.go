package v2

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/sanitization"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN SOLUTION MANAGEMENT
// ============================================================

// AdminSolutionList - GET /api/v2/admin/solutions
func AdminSolutionList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var solutions []struct {
		models.Solution
		ProblemCode string `gorm:"column:code"`
		ProblemName string `gorm:"column:name"`
	}

	if err := db.DB.Table("judge_solution").
		Joins("JOIN judge_problem ON judge_problem.id = judge_solution.problem_id").
		Select("judge_solution.*, judge_problem.code as problem_code, judge_problem.name as problem_name").
		Order("judge_solution.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&solutions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	total, _ := countRecords(models.Solution{})

	type Item struct {
		ID          uint      `json:"id"`
		ProblemID   uint      `json:"problem_id"`
		ProblemCode string    `json:"problem_code"`
		ProblemName string    `json:"problem_name"`
		Summary     string    `json:"summary"`
		IsPublic    bool      `json:"is_public"`
		IsOfficial  bool      `json:"is_official"`
		PublishOn   *time.Time `json:"publish_on"`
		Language    string    `json:"language"`
	}

	items := make([]Item, len(solutions))
	for i, s := range solutions {
		items[i] = Item{
			ID:          s.ID,
			ProblemID:   s.ProblemID,
			ProblemCode: s.ProblemCode,
			ProblemName: s.ProblemName,
			Summary:     s.Summary,
			IsPublic:    s.IsPublic,
			IsOfficial:  s.IsOfficial,
			PublishOn:   s.PublishOn,
			Language:    s.Language,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  items,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

// AdminSolutionDetail - GET /api/v2/admin/solution/:id
func AdminSolutionDetail(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid solution ID"))
		return
	}

	var solution models.Solution
	if err := db.DB.Preload("Problem").Preload("Authors").First(&solution, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("solution not found"))
		return
	}

	type Author struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}

	authors := make([]Author, len(solution.Authors))
	for i, a := range solution.Authors {
		authors[i] = Author{
			ID:       a.ID,
			Username: a.User.Username,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           solution.ID,
		"problem_id":   solution.ProblemID,
		"problem_code": solution.Problem.Code,
		"problem_name": solution.Problem.Name,
		"content":      solution.Content,
		"summary":      solution.Summary,
		"authors":      authors,
		"is_public":    solution.IsPublic,
		"is_official":  solution.IsOfficial,
		"publish_on":   solution.PublishOn,
		"valid_until":  solution.ValidUntil,
		"language":     solution.Language,
	})
}

// AdminSolutionCreate - POST /api/v2/admin/solutions
func AdminSolutionCreate(c *gin.Context) {
	var input struct {
		ProblemID   uint     `json:"problem_id" binding:"required"`
		Content     string   `json:"content" binding:"required"`
		Summary     string   `json:"summary"`
		AuthorIDs   []uint   `json:"author_ids"`
		IsPublic    bool     `json:"is_public"`
		IsOfficial  bool     `json:"is_official"`
		PublishOn   *string  `json:"publish_on"`
		ValidUntil  *string  `json:"valid_until"`
		Language    string   `json:"language"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Check if problem exists
	var problem models.Problem
	if err := db.DB.First(&problem, input.ProblemID).Error; err != nil {
		c.JSON(http.StatusBadRequest, apiError("problem not found"))
		return
	}

	// Check if solution already exists for this problem
	var existing models.Solution
	if err := db.DB.Where("problem_id = ?", input.ProblemID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, apiError("solution already exists for this problem, use update instead"))
		return
	}

	// Parse publish_on if provided
	var publishOn *time.Time
	if input.PublishOn != nil {
		t, err := time.Parse(time.RFC3339, *input.PublishOn)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiError("invalid publish_on format, use RFC3339"))
			return
		}
		publishOn = &t
	}

	// Parse valid_until if provided
	var validUntil *time.Time
	if input.ValidUntil != nil {
		t, err := time.Parse(time.RFC3339, *input.ValidUntil)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiError("invalid valid_until format, use RFC3339"))
			return
		}
		validUntil = &t
	}

	solution := models.Solution{
		ProblemID:  input.ProblemID,
		Content:    sanitization.SanitizeProblemContent(input.Content),
		Summary:    sanitization.SanitizeComment(input.Summary),
		IsPublic:   input.IsPublic,
		IsOfficial: input.IsOfficial,
		PublishOn:  publishOn,
		ValidUntil: validUntil,
		Language:   input.Language,
	}

	if err := db.DB.Create(&solution).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Handle authors
	if len(input.AuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", input.AuthorIDs).Find(&authors)
		db.DB.Model(&solution).Association("Authors").Append(&authors)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"solution": gin.H{
			"id": solution.ID,
		},
	})
}

// AdminSolutionUpdate - PATCH /api/v2/admin/solution/:id
func AdminSolutionUpdate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid solution ID"))
		return
	}

	var solution models.Solution
	if err := db.DB.First(&solution, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("solution not found"))
		return
	}

	var input struct {
		Content     *string  `json:"content"`
		Summary     *string  `json:"summary"`
		AuthorIDs   []uint   `json:"author_ids"`
		IsPublic    *bool    `json:"is_public"`
		IsOfficial  *bool    `json:"is_official"`
		PublishOn   *string  `json:"publish_on"`
		ValidUntil  *string  `json:"valid_until"`
		Language    *string  `json:"language"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Update fields if provided
	updates := make(map[string]interface{})
	if input.Content != nil {
		updates["content"] = sanitization.SanitizeProblemContent(*input.Content)
	}
	if input.Summary != nil {
		updates["summary"] = sanitization.SanitizeComment(*input.Summary)
	}
	if input.IsPublic != nil {
		updates["is_public"] = *input.IsPublic
	}
	if input.IsOfficial != nil {
		updates["is_official"] = *input.IsOfficial
	}
	if input.Language != nil {
		updates["language"] = *input.Language
	}

	// Parse publish_on if provided
	if input.PublishOn != nil {
		if *input.PublishOn == "" {
			updates["publish_on"] = (*time.Time)(nil)
		} else {
			t, err := time.Parse(time.RFC3339, *input.PublishOn)
			if err != nil {
				c.JSON(http.StatusBadRequest, apiError("invalid publish_on format, use RFC3339"))
				return
			}
			updates["publish_on"] = t
		}
	}

	// Parse valid_until if provided
	if input.ValidUntil != nil {
		if *input.ValidUntil == "" {
			updates["valid_until"] = (*time.Time)(nil)
		} else {
			t, err := time.Parse(time.RFC3339, *input.ValidUntil)
			if err != nil {
				c.JSON(http.StatusBadRequest, apiError("invalid valid_until format, use RFC3339"))
				return
			}
			updates["valid_until"] = t
		}
	}

	if len(updates) > 0 {
		db.DB.Model(&solution).Updates(updates)
	}

	// Update authors if provided
	if input.AuthorIDs != nil {
		var authors []models.Profile
		db.DB.Where("id IN ?", input.AuthorIDs).Find(&authors)
		db.DB.Model(&solution).Association("Authors").Replace(&authors)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// AdminSolutionDelete - DELETE /api/v2/admin/solution/:id
func AdminSolutionDelete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid solution ID"))
		return
	}

	var solution models.Solution
	if err := db.DB.First(&solution, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("solution not found"))
		return
	}

	db.DB.Delete(&solution)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// ============================================================
// PUBLIC SOLUTION API
// ============================================================

// ProblemSolution - GET /api/v2/problem/:code/solution
func ProblemSolution(c *gin.Context) {
	code := c.Param("code")

	// Get problem
	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("problem not found"))
		return
	}

	// Get solution
	var solution models.Solution
	if err := db.DB.Preload("Authors").Where("problem_id = ?", problem.ID).First(&solution).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("solution not found"))
		return
	}

	// Check if solution is public
	if !solution.IsPublic {
		// Check if user has permission (admin or author)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusForbidden, apiError("solution is not public"))
			return
		}

		// Check if user is admin
		var profile models.Profile
		if err := db.DB.Preload("Roles").Where("user_id = ?", userID).First(&profile).Error; err != nil {
			c.JSON(http.StatusForbidden, apiError("solution is not public"))
			return
		}

		isAdmin := false
		for _, role := range profile.Roles {
			if role.Name == "admin" {
				isAdmin = true
				break
			}
		}

		// Check if user is an author
		isAuthor := false
		for _, author := range solution.Authors {
			if author.ID == profile.ID {
				isAuthor = true
				break
			}
		}

		if !isAdmin && !isAuthor {
			c.JSON(http.StatusForbidden, apiError("solution is not public"))
			return
		}
	}

	// Check publish_on date
	if solution.PublishOn != nil && solution.PublishOn.After(time.Now()) {
		c.JSON(http.StatusNotFound, apiError("solution not yet published"))
		return
	}

	// Check valid_until date
	if solution.ValidUntil != nil && solution.ValidUntil.Before(time.Now()) {
		c.JSON(http.StatusNotFound, apiError("solution has expired"))
		return
	}

	type Author struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}

	authors := make([]Author, len(solution.Authors))
	for i, a := range solution.Authors {
		authors[i] = Author{
			ID:       a.ID,
			Username: a.User.Username,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          solution.ID,
		"problem_id":  solution.ProblemID,
		"content":     solution.Content,
		"summary":     solution.Summary,
		"authors":     authors,
		"is_official": solution.IsOfficial,
		"publish_on":  solution.PublishOn,
		"language":    solution.Language,
	})
}

// ProblemSolutionExists - GET /api/v2/problem/:code/solution/exists
func ProblemSolutionExists(c *gin.Context) {
	code := c.Param("code")

	// Get problem
	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"exists": false})
		return
	}

	// Check if solution exists and is public
	var count int64
	db.DB.Model(&models.Solution{}).
		Where("problem_id = ? AND is_public = ?", problem.ID, true).
		Count(&count)

	// Check publish_on for unpublished solutions
	if count > 0 {
		var solution models.Solution
		db.DB.Where("problem_id = ? AND is_public = ?", problem.ID, true).First(&solution)
		if solution.PublishOn != nil && solution.PublishOn.After(time.Now()) {
			count = 0
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"exists": count > 0,
	})
}
