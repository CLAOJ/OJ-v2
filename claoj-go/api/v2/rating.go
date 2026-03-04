package v2

import (
	"net/http"
	"strconv"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

// RatingLeaderboard - GET /api/v2/ratings/leaderboard
// Returns paginated list of users sorted by rating
func RatingLeaderboard(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "50")
	search := c.Query("search")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	// Build query
	query := db.DB.
		Table("judge_profile").
		Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("judge_profile.is_unlisted = ? AND judge_profile.rating IS NOT NULL", false)

	if search != "" {
		query = query.Where("au.username LIKE ?", "%"+search+"%")
	}

	// Calculate total count
	var total int64
	query.Count(&total)

	// Fetch profiles with ratings
	var profiles []models.Profile
	if err := query.
		Preload("User").
		Order("judge_profile.rating DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&profiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Calculate rating rank for each user
	type RatingEntry struct {
		Rank             int    `json:"rank"`
		Username         string `json:"username"`
		Rating           int    `json:"rating"`
		ContestsAttended int    `json:"contests_attended"`
		HighestRating    int    `json:"highest_rating"`
		AvatarURL        string `json:"avatar_url"`
	}

	entries := make([]RatingEntry, len(profiles))
	for i, p := range profiles {
		// Calculate global rank
		var rank int64
		db.DB.Model(&models.Profile{}).
			Where("rating > ?", p.Rating).
			Count(&rank)
		globalRank := int(rank) + 1

		// Count contests attended
		var contestCount int64
		db.DB.Model(&models.Rating{}).
			Where("user_id = ?", p.ID).
			Count(&contestCount)

		// Find highest rating
		var highestRating int
		db.DB.Model(&models.Rating{}).
			Where("user_id = ?", p.ID).
			Select("MAX(rating)").
			Scan(&highestRating)

		if highestRating == 0 {
			highestRating = *p.Rating
		}

		entries[i] = RatingEntry{
			Rank:             globalRank,
			Username:         p.User.Username,
			Rating:           *p.Rating,
			ContestsAttended: int(contestCount),
			HighestRating:    highestRating,
			AvatarURL:        getAvatarURL(&p),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"limit": limit,
		"data":  entries,
	})
}

// UserRating - GET /api/v2/user/:user/rating-detail
// Returns detailed rating information for a specific user
func UserRatingDetail(c *gin.Context) {
	username := c.Param("user")

	var profile models.Profile
	if err := db.DB.
		Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("au.username = ?", username).
		First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	// Get current rating
	var currentRating *int
	if profile.Rating != nil {
		currentRating = profile.Rating
	}

	// Count contests attended
	var contestCount int64
	db.DB.Model(&models.Rating{}).
		Where("user_id = ?", profile.ID).
		Count(&contestCount)

	// Find highest rating
	var highestRating int
	db.DB.Model(&models.Rating{}).
		Where("user_id = ?", profile.ID).
		Select("MAX(rating)").
		Scan(&highestRating)

	// Find lowest rating
	var lowestRating int
	db.DB.Model(&models.Rating{}).
		Where("user_id = ?", profile.ID).
		Select("MIN(rating)").
		Scan(&lowestRating)

	// Get recent rating changes
	type RatingChange struct {
		Date        string  `json:"date"`
		Contest     string  `json:"contest"`
		ContestKey  string  `json:"contest_key"`
		Rank        int     `json:"rank"`
		Rating      int     `json:"rating"`
		Performance float64 `json:"performance"`
	}

	var recentChanges []RatingChange
	db.DB.Table("judge_rating").
		Select("judge_rating.last_rated as date, jc.name as contest, jc.key as contest_key, judge_rating.rank, judge_rating.rating, judge_rating.performance").
		Joins("JOIN judge_contest jc ON jc.id = judge_rating.contest_id").
		Where("judge_rating.user_id = ?", profile.ID).
		Order("judge_rating.last_rated DESC").
		Limit(10).
		Scan(&recentChanges)

	// Calculate rating rank
	var ratingRank int64
	if profile.Rating != nil {
		db.DB.Model(&models.Profile{}).
			Where("rating > ?", *profile.Rating).
			Count(&ratingRank)
		ratingRank++
	}

	c.JSON(http.StatusOK, gin.H{
		"username":          profile.User.Username,
		"current_rating":    currentRating,
		"rating_rank":       ratingRank,
		"contests_attended": contestCount,
		"highest_rating":    highestRating,
		"lowest_rating":     lowestRating,
		"recent_changes":    recentChanges,
	})
}
