package v2

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// errUnknownContestIDs is returned when contest_ids names contests that do not exist.
var errUnknownContestIDs = errors.New("unknown contest ids")

// ============================================================
// CONTEST TAG ADMIN API
// ============================================================

// AdminContestTagList - GET /api/v2/admin/contest-tags
func AdminContestTagList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")

	var tags []models.ContestTag
	query := db.DB.Model(&models.ContestTag{})

	if search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	counts, err := contestCounts(tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		Color        string `json:"color"`
		Description  string `json:"description"`
		ContestCount int64  `json:"contest_count"`
	}

	items := make([]Item, len(tags))
	for i, tag := range tags {
		items[i] = Item{
			ID:           tag.ID,
			Name:         tag.Name,
			Color:        tag.Color,
			Description:  tag.Description,
			ContestCount: counts[tag.ID],
		}
	}

	c.JSON(http.StatusOK, apiList(items))
}

// AdminContestTagDetail - GET /api/v2/admin/contest-tag/:id
func AdminContestTagDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("tag not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	contestIDs, err := tagContestIDs(db.DB, tag.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	contestRefs, err := tagContestRefs(db.DB, tag.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            tag.ID,
		"name":          tag.Name,
		"color":         tag.Color,
		"description":   tag.Description,
		"contest_ids":   contestIDs,
		"contest_count": len(contestIDs),
		"contests":      contestRefs,
	})
}

// ContestTagCreateRequest
type ContestTagCreateRequest struct {
	Name        string  `json:"name" binding:"required"`
	Color       string  `json:"color" binding:"required"`
	Description string  `json:"description"`
	ContestIDs  *[]uint `json:"contest_ids"`
}

// AdminContestTagCreate - POST /api/v2/admin/contest-tags
func AdminContestTagCreate(c *gin.Context) {
	var req ContestTagCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Validate name format (alphanumeric, underscore, hyphen, max 20 chars)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(req.Name) {
		c.JSON(http.StatusBadRequest, apiError("tag name can only contain letters, numbers, underscore, and hyphen"))
		return
	}

	if len(req.Name) > 20 {
		c.JSON(http.StatusBadRequest, apiError("tag name must be 20 characters or less"))
		return
	}

	// Validate color format (hex color code)
	validColor := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	if !validColor.MatchString(req.Color) {
		c.JSON(http.StatusBadRequest, apiError("color must be a valid hex color code (e.g., #FF5733)"))
		return
	}

	tag := models.ContestTag{
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tag).Error; err != nil {
			return err
		}
		if req.ContestIDs == nil {
			return nil
		}
		ids, err := existingContestIDs(tx, *req.ContestIDs)
		if err != nil {
			return err
		}
		return replaceTagContests(tx, tag.ID, ids)
	})
	if err != nil {
		if errors.Is(err, errUnknownContestIDs) {
			c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"tag": gin.H{
			"id":          tag.ID,
			"name":        tag.Name,
			"color":       tag.Color,
			"description": tag.Description,
		},
	})
}

// ContestTagUpdateRequest
type ContestTagUpdateRequest struct {
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	Description string  `json:"description"`
	ContestIDs  *[]uint `json:"contest_ids"`
}

// AdminContestTagUpdate - PATCH /api/v2/admin/contest-tag/:id
func AdminContestTagUpdate(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("tag not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var req ContestTagUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if !validName.MatchString(req.Name) {
			c.JSON(http.StatusBadRequest, apiError("tag name can only contain letters, numbers, underscore, and hyphen"))
			return
		}
		if len(req.Name) > 20 {
			c.JSON(http.StatusBadRequest, apiError("tag name must be 20 characters or less"))
			return
		}
		updates["name"] = req.Name
	}

	if req.Color != "" {
		validColor := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
		if !validColor.MatchString(req.Color) {
			c.JSON(http.StatusBadRequest, apiError("color must be a valid hex color code (e.g., #FF5733)"))
			return
		}
		updates["color"] = req.Color
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&tag).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.ContestIDs == nil {
			return nil
		}
		ids, err := existingContestIDs(tx, *req.ContestIDs)
		if err != nil {
			return err
		}
		return replaceTagContests(tx, tag.ID, ids)
	})
	if err != nil {
		if errors.Is(err, errUnknownContestIDs) {
			c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tag": gin.H{
			"id":          tag.ID,
			"name":        tag.Name,
			"color":       tag.Color,
			"description": tag.Description,
		},
	})
}

// AdminContestTagDelete - DELETE /api/v2/admin/contest-tag/:id
func AdminContestTagDelete(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	if err := db.DB.Delete(&models.ContestTag{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "tag deleted",
	})
}

// AdminContestAddTag - POST /api/v2/admin/contest/:key/tags/:tagId
func AdminContestAddTag(c *gin.Context) {
	contestKey := c.Param("key")
	tagIdStr := c.Param("tagId")

	var contest models.Contest
	if err := db.DB.Where("`key` = ?", contestKey).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	var tagId uint
	if err := parseUint(tagIdStr, &tagId); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, tagId).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("tag not found"))
		return
	}

	if err := db.DB.Model(&contest).Association("Tags").Append(&tag); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "tag added to contest",
	})
}

// AdminContestRemoveTag - DELETE /api/v2/admin/contest/:key/tags/:tagId
func AdminContestRemoveTag(c *gin.Context) {
	contestKey := c.Param("key")
	tagIdStr := c.Param("tagId")

	var contest models.Contest
	if err := db.DB.Where("`key` = ?", contestKey).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	var tagId uint
	if err := parseUint(tagIdStr, &tagId); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, tagId).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("tag not found"))
		return
	}

	if err := db.DB.Model(&contest).Association("Tags").Delete(&tag); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "tag removed from contest",
	})
}

// contestCounts maps tag id -> number of included contests.
func contestCounts(tags []models.ContestTag) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(tags))
	if len(tags) == 0 {
		return counts, nil
	}
	ids := make([]uint, len(tags))
	for i, tag := range tags {
		ids[i] = tag.ID
	}
	var rows []struct {
		ContestTagID uint  `gorm:"column:contesttag_id"`
		Count        int64 `gorm:"column:count"`
	}
	if err := db.DB.Table("judge_contest_tags").
		Select("contesttag_id, COUNT(*) AS count").
		Where("contesttag_id IN ?", ids).
		Group("contesttag_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		counts[r.ContestTagID] = r.Count
	}
	return counts, nil
}

func tagContestIDs(tx *gorm.DB, tagID uint) ([]uint, error) {
	ids := []uint{}
	if err := tx.Table("judge_contest_tags").
		Where("contesttag_id = ?", tagID).
		Order("contest_id ASC").
		Pluck("contest_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

type contestRef struct {
	ID   uint   `gorm:"column:id"`
	Key  string `gorm:"column:key"`
	Name string `gorm:"column:name"`
}

func tagContestRefs(tx *gorm.DB, tagID uint) ([]gin.H, error) {
	var refs []contestRef
	if err := tx.Table("judge_contest").
		Select("judge_contest.id, judge_contest.`key`, judge_contest.name").
		Joins("JOIN judge_contest_tags ON judge_contest_tags.contest_id = judge_contest.id").
		Where("judge_contest_tags.contesttag_id = ?", tagID).
		Order("judge_contest.`key` ASC").
		Scan(&refs).Error; err != nil {
		return nil, err
	}
	out := make([]gin.H, len(refs))
	for i, r := range refs {
		out[i] = gin.H{"id": r.ID, "key": r.Key, "name": r.Name}
	}
	return out, nil
}

// existingContestIDs validates that every contest id exists and returns the
// de-duplicated list.
func existingContestIDs(tx *gorm.DB, ids []uint) ([]uint, error) {
	seen := make(map[uint]bool, len(ids))
	unique := make([]uint, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return unique, nil
	}

	var found []uint
	if err := tx.Model(&models.Contest{}).Where("id IN ?", unique).Pluck("id", &found).Error; err != nil {
		return nil, err
	}
	have := make(map[uint]bool, len(found))
	for _, id := range found {
		have[id] = true
	}
	var missing []uint
	for _, id := range unique {
		if !have[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
		return nil, fmt.Errorf("%w: %v", errUnknownContestIDs, missing)
	}
	return unique, nil
}

// replaceTagContests rewrites judge_contest_tags for one tag only, leaving
// every other tag's rows alone.
func replaceTagContests(tx *gorm.DB, tagID uint, ids []uint) error {
	if err := tx.Exec("DELETE FROM judge_contest_tags WHERE contesttag_id = ?", tagID).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := tx.Exec("INSERT INTO judge_contest_tags (contest_id, contesttag_id) VALUES (?, ?)", id, tagID).Error; err != nil {
			return err
		}
	}
	return nil
}
