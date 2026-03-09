package v2

import (
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ContestJoinRequest handles the request body for joining a contest
type ContestJoinRequest struct {
	Virtual bool `json:"virtual"` // If true, start a virtual participation
}

// ContestJoin handles a user joining a contest or starting a virtual participation.
// POST /api/v2/contest/:key/join
func ContestJoin(c *gin.Context) {
	key := c.Param("key")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID missing"})
		return
	}

	var ct models.Contest
	if err := db.DB.Where("key = ? AND is_visible = ?", key, true).First(&ct).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "contest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	now := time.Now()

	// Parse request body
	var req ContestJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// For backward compatibility, treat missing body as regular join
		req.Virtual = false
	}

	// Virtual participation can only be started after the contest has ended
	if req.Virtual {
		// Check if contest has ended
		if now.Before(ct.EndTime) {
			c.JSON(http.StatusForbidden, gin.H{"error": "virtual participation can only be started after the contest ends"})
			return
		}

		// Fetch user profile
		var profile models.Profile
		if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "profile not found"})
			return
		}

		// Check if user already has a virtual participation for this contest
		var existingPart models.ContestParticipation
		err := db.DB.Where("contest_id = ? AND user_id = ? AND virtual > 0", ct.ID, profile.ID).First(&existingPart).Error
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "already have a virtual participation for this contest"})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		// Create virtual participation
		// Virtual participation starts at the current time, but simulates the contest duration
		part := models.ContestParticipation{
			ContestID:      ct.ID,
			UserID:         profile.ID,
			RealStart:      now, // Actual time when virtual participation started
			Score:          0,
			Cumtime:        0,
			Virtual:        1, // 1 = virtual participation (positive number indicates virtual)
			FormatData:     nil,
			IsDisqualified: false,
			Tiebreaker:     0,
		}

		if err := db.DB.Create(&part).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start virtual participation"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "successfully started virtual participation",
			"virtual":      true,
			"start_time":   now,
			"end_time":     now.Add(ct.EndTime.Sub(ct.StartTime)), // Virtual end time based on original duration
			"contest_name": ct.Name,
		})
		return
	}

	// Regular participation - contest must be active
	if now.Before(ct.StartTime) || now.After(ct.EndTime) {
		c.JSON(http.StatusForbidden, gin.H{"error": "contest is not active"})
		return
	}

	// Fetch user profile
	var profile models.Profile
	if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "profile not found"})
		return
	}

	// Check for existing participation (live or spectate)
	var part models.ContestParticipation
	err := db.DB.Where("contest_id = ? AND user_id = ?", ct.ID, profile.ID).First(&part).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already joined"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Create participation
	part = models.ContestParticipation{
		ContestID:      ct.ID,
		UserID:         profile.ID,
		Score:          0,
		Cumtime:        0,
		Virtual:        0,
		FormatData:     nil,
		IsDisqualified: false,
		Tiebreaker:     0,
	}

	if err := db.DB.Create(&part).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully joined contest"})
}
