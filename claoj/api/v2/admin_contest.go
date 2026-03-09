package v2

import (
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/service/contest"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN CONTEST MANAGEMENT API
// ============================================================

// AdminContestList - GET /api/v2/admin/contests
func AdminContestList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getContestService().ListContests(contest.ListContestsRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID                    uint      `json:"id"`
		Key                   string    `json:"key"`
		Name                  string    `json:"name"`
		StartTime             time.Time `json:"start_time"`
		EndTime               time.Time `json:"end_time"`
		IsVisible             bool      `json:"is_visible"`
		IsRated               bool      `json:"is_rated"`
		UserCount             int       `json:"user_count"`
		FormatName            string    `json:"format_name"`
		IsOrganizationPrivate bool      `json:"is_organization_private"`
	}
	items := make([]Item, len(resp.Contests))
	for i, c := range resp.Contests {
		items[i] = Item{
			ID:                    c.ID,
			Key:                   c.Key,
			Name:                  c.Name,
			StartTime:             c.StartTime,
			EndTime:               c.EndTime,
			IsVisible:             c.IsVisible,
			IsRated:               c.IsRated,
			UserCount:             c.UserCount,
			FormatName:            c.FormatName,
			IsOrganizationPrivate: c.IsOrganizationPrivate,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     resp.Total,
		"page":      resp.Page,
		"page_size": resp.PageSize,
	})
}

// AdminContestDetail - GET /api/v2/admin/contest/:key
func AdminContestDetail(c *gin.Context) {
	key := c.Param("key")

	resp, err := getContestService().GetContest(contest.GetContestRequest{
		ContestKey: key,
	})
	if err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiError("contest not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	problems := make([]gin.H, len(resp.Problems))
	for i, p := range resp.Problems {
		problems[i] = gin.H{
			"id":         p.ID,
			"problem_id": p.ProblemID,
			"code":       p.Code,
			"name":       p.Name,
			"points":     p.Points,
			"order":      p.Order,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"contest":  resp.Contest,
		"problems": problems,
	})
}

// AdminContestCreate - POST /api/v2/admin/contests
func AdminContestCreate(c *gin.Context) {
	var input struct {
		Key                   string  `json:"key" binding:"required"`
		Name                  string  `json:"name" binding:"required"`
		Description           string  `json:"description" binding:"required"`
		Summary               string  `json:"summary"`
		StartTime             string  `json:"start_time" binding:"required"`
		EndTime               string  `json:"end_time" binding:"required"`
		TimeLimit             *int64  `json:"time_limit"`
		IsVisible             bool    `json:"is_visible"`
		IsRated               bool    `json:"is_rated"`
		FormatName            string  `json:"format_name"`
		FormatConfig          string  `json:"format_config"`
		AccessCode            string  `json:"access_code"`
		HideProblemTags       bool    `json:"hide_problem_tags"`
		RunPretestsOnly       bool    `json:"run_pretests_only"`
		IsOrganizationPrivate bool    `json:"is_organization_private"`
		MaxSubmissions        *int    `json:"max_submissions"`
		AuthorIDs             []uint  `json:"author_ids"`
		CuratorIDs            []uint  `json:"curator_ids"`
		TesterIDs             []uint  `json:"tester_ids"`
		ProblemIDs            []uint  `json:"problem_ids"`
		TagIDs                []uint  `json:"tag_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Parse timestamps
	startTime, err := parseRFC3339(input.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid start_time format"))
		return
	}
	endTime, err := parseRFC3339(input.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid end_time format"))
		return
	}

	profile, err := getContestService().CreateContest(contest.CreateContestRequest{
		Key:                   input.Key,
		Name:                  input.Name,
		Description:           input.Description,
		Summary:               input.Summary,
		StartTime:             startTime,
		EndTime:               endTime,
		TimeLimit:             input.TimeLimit,
		IsVisible:             input.IsVisible,
		IsRated:               input.IsRated,
		FormatName:            input.FormatName,
		FormatConfig:          input.FormatConfig,
		AccessCode:            input.AccessCode,
		HideProblemTags:       input.HideProblemTags,
		RunPretestsOnly:       input.RunPretestsOnly,
		IsOrganizationPrivate: input.IsOrganizationPrivate,
		MaxSubmissions:        input.MaxSubmissions,
		AuthorIDs:             input.AuthorIDs,
		CuratorIDs:            input.CuratorIDs,
		TesterIDs:             input.TesterIDs,
		ProblemIDs:            input.ProblemIDs,
		TagIDs:                input.TagIDs,
	})
	if err != nil {
		if errors.Is(err, contest.ErrInvalidFormatConfig) {
			c.JSON(http.StatusBadRequest, apiError("invalid format_config JSON"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"contest": gin.H{
			"id":   profile.ID,
			"key":  profile.Key,
			"name": profile.Name,
		},
	})
}

// AdminContestUpdate - PATCH /api/v2/admin/contest/:key
func AdminContestUpdate(c *gin.Context) {
	key := c.Param("key")

	var input struct {
		Name                  *string `json:"name"`
		Description           *string `json:"description"`
		Summary               *string `json:"summary"`
		StartTime             *string `json:"start_time"`
		EndTime               *string `json:"end_time"`
		IsVisible             *bool   `json:"is_visible"`
		IsRated               *bool   `json:"is_rated"`
		AccessCode            *string `json:"access_code"`
		HideProblemTags       *bool   `json:"hide_problem_tags"`
		RunPretestsOnly       *bool   `json:"run_pretests_only"`
		IsOrganizationPrivate *bool   `json:"is_organization_private"`
		AddProblemIDs         []uint  `json:"add_problem_ids"`
		RemoveProblemIDs      []uint  `json:"remove_problem_ids"`
		UpdateTimeLimit       *int64  `json:"update_time_limit"`
		MaxSubmissions        *int    `json:"max_submissions"`
		AddTagIDs             []uint  `json:"add_tag_ids"`
		RemoveTagIDs          []uint  `json:"remove_tag_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	profile, err := getContestService().UpdateContest(contest.UpdateContestRequest{
		ContestKey:            key,
		Name:                  input.Name,
		Description:           input.Description,
		Summary:               input.Summary,
		StartTime:             input.StartTime,
		EndTime:               input.EndTime,
		IsVisible:             input.IsVisible,
		IsRated:               input.IsRated,
		AccessCode:            input.AccessCode,
		HideProblemTags:       input.HideProblemTags,
		RunPretestsOnly:       input.RunPretestsOnly,
		IsOrganizationPrivate: input.IsOrganizationPrivate,
		TimeLimit:             input.UpdateTimeLimit,
		MaxSubmissions:        input.MaxSubmissions,
		AddProblemIDs:         input.AddProblemIDs,
		RemoveProblemIDs:      input.RemoveProblemIDs,
		AddTagIDs:             input.AddTagIDs,
		RemoveTagIDs:          input.RemoveTagIDs,
	})
	if err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiError("contest not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"contest": profile,
	})
}

// AdminContestDelete - DELETE /api/v2/admin/contest/:key
func AdminContestDelete(c *gin.Context) {
	key := c.Param("key")

	if err := getContestService().DeleteContest(contest.DeleteContestRequest{
		ContestKey: key,
	}); err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiError("contest not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Contest hidden (soft deleted)",
	})
}

// AdminContestLock - POST /api/v2/admin/contest/:key/lock
func AdminContestLock(c *gin.Context) {
	key := c.Param("key")

	var input struct {
		LockedAfter *string `json:"locked_after"` // ISO 8601 format, or null to unlock
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	locked, err := getContestService().LockContest(contest.LockContestRequest{
		ContestKey:  key,
		LockedAfter: input.LockedAfter,
	})
	if err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiError("contest not found"))
			return
		}
		if errors.Is(err, contest.ErrInvalidLockedAfter) {
			c.JSON(http.StatusBadRequest, apiError("invalid locked_after format, use ISO 8601"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if input.LockedAfter == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Contest unlocked - submissions are now allowed",
			"locked":  false,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"success":      true,
			"message":      "Contest lock time updated",
			"locked":       locked,
			"locked_after": input.LockedAfter,
		})
	}
}

// AdminContestClone - POST /api/v2/admin/contest/:key/clone
func AdminContestClone(c *gin.Context) {
	key := c.Param("key")

	var input struct {
		NewKey       string `json:"new_key" binding:"required"`
		NewName      string `json:"new_name" binding:"required"`
		CopyProblems bool   `json:"copy_problems"`
		CopySettings bool   `json:"copy_settings"`
		NewStartTime string `json:"new_start_time"`
		NewEndTime   string `json:"new_end_time"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	profile, err := getContestService().CloneContest(contest.CloneContestRequest{
		SourceKey:    key,
		NewKey:       input.NewKey,
		NewName:      input.NewName,
		CopyProblems: input.CopyProblems,
		CopySettings: input.CopySettings,
		NewStartTime: input.NewStartTime,
		NewEndTime:   input.NewEndTime,
	})
	if err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiError("contest not found"))
			return
		}
		if errors.Is(err, contest.ErrContestKeyExists) {
			c.JSON(http.StatusBadRequest, apiError("contest key already exists"))
			return
		}
		if errors.Is(err, contest.ErrInvalidStartTime) {
			c.JSON(http.StatusBadRequest, apiError("invalid new_start_time format"))
			return
		}
		if errors.Is(err, contest.ErrInvalidEndTime) {
			c.JSON(http.StatusBadRequest, apiError("invalid new_end_time format"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "contest cloned successfully",
		"new_contest": gin.H{
			"id":   profile.ID,
			"key":  profile.Key,
			"name": profile.Name,
		},
	})
}

// AdminContestParticipationDisqualify - POST /api/v2/admin/contest/:key/participation/:id/disqualify
func AdminContestParticipationDisqualify(c *gin.Context) {
	key := c.Param("key")
	participationIDStr := c.Param("id")

	var participationID uint
	if err := parseUint(participationIDStr, &participationID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid participation id"))
		return
	}

	if err := getContestService().DisqualifyParticipation(contest.DisqualifyParticipationRequest{
		ContestKey:      key,
		ParticipationID: participationID,
	}); err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiError("contest not found"))
			return
		}
		if errors.Is(err, contest.ErrParticipationNotFound) {
			c.JSON(http.StatusNotFound, apiError("participation not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "participation disqualified",
		"participation": gin.H{
			"id":              participationID,
			"is_disqualified": true,
		},
	})
}

// AdminContestParticipationUndisqualify - POST /api/v2/admin/contest/:key/participation/:id/undisqualify
func AdminContestParticipationUndisqualify(c *gin.Context) {
	key := c.Param("key")
	participationIDStr := c.Param("id")

	var participationID uint
	if err := parseUint(participationIDStr, &participationID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid participation id"))
		return
	}

	if err := getContestService().UndisqualifyParticipation(contest.UndisqualifyParticipationRequest{
		ContestKey:      key,
		ParticipationID: participationID,
	}); err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiError("contest not found"))
			return
		}
		if errors.Is(err, contest.ErrParticipationNotFound) {
			c.JSON(http.StatusNotFound, apiError("participation not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "participation undisqualified",
		"participation": gin.H{
			"id":              participationID,
			"is_disqualified": false,
		},
	})
}
