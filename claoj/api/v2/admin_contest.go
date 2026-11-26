package v2

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/contest"
	"github.com/CLAOJ/claoj/service/relations"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN CONTEST MANAGEMENT API
// ============================================================

// secondsToMicros converts the admin API's seconds to Django's
// DurationField microseconds. nil stays nil; 0 is passed through so the
// update path can clear the limit.
func secondsToMicros(seconds *int64) *int64 {
	if seconds == nil {
		return nil
	}
	us := *seconds * 1_000_000
	return &us
}

func positiveOrNil(v *int64) *int64 {
	if v == nil || *v <= 0 {
		return nil
	}
	return v
}

// requireElevatedContestFields mirrors v1's readonly_fields gates
// (judge/admin/contest.py): these two fields need their own Django permission
// even for a contest author. Aborts with 403 when a supplied field is denied.
func requireElevatedContestFields(c *gin.Context, problemLabelScript, performanceCeiling bool) bool {
	if (problemLabelScript && !auth.HasPerm(c, "judge.contest_problem_label")) ||
		(performanceCeiling && !auth.HasPerm(c, "judge.override_performance_ceiling")) {
		c.JSON(http.StatusForbidden, apiUserError("permission denied"))
		return false
	}
	return true
}

// storedGatedContestFields reads the two v1-gated fields so an update that
// leaves them as they are needs no extra permission. ok is false when the
// contest cannot be read; the service then reports that error itself.
func storedGatedContestFields(key string) (script string, ceiling *int, ok bool) {
	var stored models.Contest
	err := db.DB.Select("problem_label_script", "performance_ceiling_override").
		Where(&models.Contest{Key: key}).First(&stored).Error
	return stored.ProblemLabelScript, stored.PerformanceCeilingOverride, err == nil
}

// AdminContestList - GET /api/v2/admin/contests
func AdminContestList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getContestService().ListContests(contest.ListContestsRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
		Status:   c.Query("status"),
		Sort:     c.Query("sort"),
		Order:    c.Query("order"),
	})
	if err != nil {
		if errors.Is(err, contest.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
			return
		}
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
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// adminContestCreateInput is the body of POST /admin/contests and of the
// organization-scoped POST /organization/:id/contests.
type adminContestCreateInput struct {
	Key                        string `json:"key" binding:"required"`
	Name                       string `json:"name" binding:"required"`
	Description                string `json:"description" binding:"required"`
	Summary                    string `json:"summary"`
	StartTime                  string `json:"start_time" binding:"required"`
	EndTime                    string `json:"end_time" binding:"required"`
	TimeLimit                  *int64 `json:"time_limit"`
	IsVisible                  bool   `json:"is_visible"`
	IsRated                    bool   `json:"is_rated"`
	ScoreboardVisibility       string `json:"scoreboard_visibility"`
	ScoreboardCacheTimeout     uint   `json:"scoreboard_cache_timeout"`
	UseClarifications          *bool  `json:"use_clarifications"`
	PushAnnouncements          bool   `json:"push_announcements"`
	RatingFloor                *int   `json:"rating_floor"`
	RatingCeiling              *int   `json:"rating_ceiling"`
	PerformanceCeilingOverride *int   `json:"performance_ceiling_override"`
	RateAll                    bool   `json:"rate_all"`
	IsPrivate                  bool   `json:"is_private"`
	HideProblemAuthors         bool   `json:"hide_problem_authors"`
	ShowShortDisplay           bool   `json:"show_short_display"`
	LimitJoinOrganizations     bool   `json:"limit_join_organizations"`
	OgImage                    string `json:"og_image"`
	LogoOverrideImage          string `json:"logo_override_image"`
	ProblemLabelScript         string `json:"problem_label_script"`
	PointsPrecision            *int   `json:"points_precision"`
	FormatName                 string `json:"format_name"`
	FormatConfig               string `json:"format_config"`
	AccessCode                 string `json:"access_code"`
	HideProblemTags            bool   `json:"hide_problem_tags"`
	RunPretestsOnly            bool   `json:"run_pretests_only"`
	IsOrganizationPrivate      bool   `json:"is_organization_private"`
	AuthorIDs                  []uint `json:"author_ids"`
	CuratorIDs                 []uint `json:"curator_ids"`
	TesterIDs                  []uint `json:"tester_ids"`
	ProblemIDs                 []uint `json:"problem_ids"`
	TagIDs                     []uint `json:"tag_ids"`
	OrganizationIDs            []uint `json:"organization_ids"`
	BannedUserIDs              []uint `json:"banned_user_ids"`
	PrivateContestantIDs       []uint `json:"private_contestant_ids"`
	ViewScoreboardUserIDs      []uint `json:"view_scoreboard_user_ids"`
	JoinOrganizationIDs        []uint `json:"join_organization_ids"`
	ClassIDs                   []uint `json:"class_ids"`
}

// AdminContestCreate - POST /api/v2/admin/contests
func AdminContestCreate(c *gin.Context) {
	var input adminContestCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}
	createContestFromInput(c, input, nil, 0)
}

// createContestFromInput validates and inserts a contest. A non-nil org forces
// the organization onto the object (v1 ContestCreateOrganization: organization
// private, organizations = [org], caller is an author).
func createContestFromInput(c *gin.Context, input adminContestCreateInput, org *models.Organization, callerProfileID uint) {
	if !requireElevatedContestFields(c, input.ProblemLabelScript != "", input.PerformanceCeilingOverride != nil) {
		return
	}

	authorIDs := input.AuthorIDs
	if org != nil {
		input.IsOrganizationPrivate = true
		input.OrganizationIDs = []uint{org.ID}
		authorIDs = withProfileID(authorIDs, callerProfileID)
	}

	// Parse timestamps
	startTime, err := parseRFC3339(input.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid start_time format"))
		return
	}
	endTime, err := parseRFC3339(input.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid end_time format"))
		return
	}

	profile, err := getContestService().CreateContest(contest.CreateContestRequest{
		Key:                        input.Key,
		Name:                       input.Name,
		Description:                input.Description,
		Summary:                    input.Summary,
		StartTime:                  startTime,
		EndTime:                    endTime,
		TimeLimit:                  positiveOrNil(secondsToMicros(input.TimeLimit)),
		IsVisible:                  input.IsVisible,
		IsRated:                    input.IsRated,
		ScoreboardVisibility:       input.ScoreboardVisibility,
		ScoreboardCacheTimeout:     input.ScoreboardCacheTimeout,
		UseClarifications:          input.UseClarifications,
		PushAnnouncements:          input.PushAnnouncements,
		RatingFloor:                input.RatingFloor,
		RatingCeiling:              input.RatingCeiling,
		PerformanceCeilingOverride: input.PerformanceCeilingOverride,
		RateAll:                    input.RateAll,
		IsPrivate:                  input.IsPrivate,
		HideProblemAuthors:         input.HideProblemAuthors,
		ShowShortDisplay:           input.ShowShortDisplay,
		LimitJoinOrganizations:     input.LimitJoinOrganizations,
		OgImage:                    input.OgImage,
		LogoOverrideImage:          input.LogoOverrideImage,
		ProblemLabelScript:         input.ProblemLabelScript,
		PointsPrecision:            input.PointsPrecision,
		FormatName:                 input.FormatName,
		FormatConfig:               input.FormatConfig,
		AccessCode:                 input.AccessCode,
		HideProblemTags:            input.HideProblemTags,
		RunPretestsOnly:            input.RunPretestsOnly,
		IsOrganizationPrivate:      input.IsOrganizationPrivate,
		AuthorIDs:                  authorIDs,
		CuratorIDs:                 input.CuratorIDs,
		TesterIDs:                  input.TesterIDs,
		ProblemIDs:                 input.ProblemIDs,
		TagIDs:                     input.TagIDs,
		OrganizationIDs:            input.OrganizationIDs,
		BannedUserIDs:              input.BannedUserIDs,
		PrivateContestantIDs:       input.PrivateContestantIDs,
		ViewScoreboardUserIDs:      input.ViewScoreboardUserIDs,
		JoinOrganizationIDs:        input.JoinOrganizationIDs,
		ClassIDs:                   input.ClassIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, contest.ErrInvalidFormatConfig):
			c.JSON(http.StatusBadRequest, apiUserError("invalid format_config JSON"))
		case errors.Is(err, contest.ErrInvalidScoreboardVisibility),
			errors.Is(err, contest.ErrInvalidPointsPrecision),
			errors.Is(err, contest.ErrFieldTooLong),
			errors.Is(err, relations.ErrUnknownID):
			c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		default:
			c.JSON(http.StatusBadRequest, apiClientError(err))
		}
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
		Name                            *string                        `json:"name"`
		Description                     *string                        `json:"description"`
		Summary                         *string                        `json:"summary"`
		StartTime                       *string                        `json:"start_time"`
		EndTime                         *string                        `json:"end_time"`
		TimeLimit                       *int64                         `json:"time_limit"` // seconds; 0 clears
		IsVisible                       *bool                          `json:"is_visible"`
		IsRated                         *bool                          `json:"is_rated"`
		ScoreboardVisibility            *string                        `json:"scoreboard_visibility"`
		ScoreboardCacheTimeout          *uint                          `json:"scoreboard_cache_timeout"`
		UseClarifications               *bool                          `json:"use_clarifications"`
		PushAnnouncements               *bool                          `json:"push_announcements"`
		HideProblemAuthors              *bool                          `json:"hide_problem_authors"`
		ShowShortDisplay                *bool                          `json:"show_short_display"`
		LimitJoinOrganizations          *bool                          `json:"limit_join_organizations"`
		RateAll                         *bool                          `json:"rate_all"`
		IsPrivate                       *bool                          `json:"is_private"`
		PointsPrecision                 *int                           `json:"points_precision"`
		RatingFloor                     *int                           `json:"rating_floor"`
		RatingCeiling                   *int                           `json:"rating_ceiling"`
		PerformanceCeilingOverride      *int                           `json:"performance_ceiling_override"`
		ClearRatingFloor                bool                           `json:"clear_rating_floor"`
		ClearRatingCeiling              bool                           `json:"clear_rating_ceiling"`
		ClearPerformanceCeilingOverride bool                           `json:"clear_performance_ceiling_override"`
		OgImage                         *string                        `json:"og_image"`
		LogoOverrideImage               *string                        `json:"logo_override_image"`
		ProblemLabelScript              *string                        `json:"problem_label_script"`
		AccessCode                      *string                        `json:"access_code"`
		HideProblemTags                 *bool                          `json:"hide_problem_tags"`
		RunPretestsOnly                 *bool                          `json:"run_pretests_only"`
		IsOrganizationPrivate           *bool                          `json:"is_organization_private"`
		FormatName                      *string                        `json:"format_name"`
		FormatConfig                    *string                        `json:"format_config"`
		AuthorIDs                       *[]uint                        `json:"author_ids"`
		CuratorIDs                      *[]uint                        `json:"curator_ids"`
		TesterIDs                       *[]uint                        `json:"tester_ids"`
		TagIDs                          *[]uint                        `json:"tag_ids"`
		OrganizationIDs                 *[]uint                        `json:"organization_ids"`
		BannedUserIDs                   *[]uint                        `json:"banned_user_ids"`
		PrivateContestantIDs            *[]uint                        `json:"private_contestant_ids"`
		ViewScoreboardUserIDs           *[]uint                        `json:"view_scoreboard_user_ids"`
		JoinOrganizationIDs             *[]uint                        `json:"join_organization_ids"`
		ClassIDs                        *[]uint                        `json:"class_ids"`
		Problems                        *[]contest.ContestProblemInput `json:"problems"`
		AddProblemIDs                   []uint                         `json:"add_problem_ids"`
		RemoveProblemIDs                []uint                         `json:"remove_problem_ids"`
		AddTagIDs                       []uint                         `json:"add_tag_ids"`
		RemoveTagIDs                    []uint                         `json:"remove_tag_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}
	// A nil pointer means "unchanged", and a value equal to the stored one is
	// not a write either: the form PATCHes every field, so a plain author must
	// still be able to save. Only a real change needs the permission.
	if script, ceiling, stored := storedGatedContestFields(key); stored {
		scriptChanged := input.ProblemLabelScript != nil && *input.ProblemLabelScript != script
		ceilingChanged := (input.PerformanceCeilingOverride != nil &&
			(ceiling == nil || *input.PerformanceCeilingOverride != *ceiling)) ||
			(input.ClearPerformanceCeilingOverride && ceiling != nil)
		if !requireElevatedContestFields(c, scriptChanged, ceilingChanged) {
			return
		}
	}

	profile, err := getContestService().UpdateContest(contest.UpdateContestRequest{
		ContestKey:                      key,
		Name:                            input.Name,
		Description:                     input.Description,
		Summary:                         input.Summary,
		StartTime:                       input.StartTime,
		EndTime:                         input.EndTime,
		TimeLimit:                       secondsToMicros(input.TimeLimit),
		IsVisible:                       input.IsVisible,
		IsRated:                         input.IsRated,
		ScoreboardVisibility:            input.ScoreboardVisibility,
		ScoreboardCacheTimeout:          input.ScoreboardCacheTimeout,
		UseClarifications:               input.UseClarifications,
		PushAnnouncements:               input.PushAnnouncements,
		HideProblemAuthors:              input.HideProblemAuthors,
		ShowShortDisplay:                input.ShowShortDisplay,
		LimitJoinOrganizations:          input.LimitJoinOrganizations,
		RateAll:                         input.RateAll,
		IsPrivate:                       input.IsPrivate,
		PointsPrecision:                 input.PointsPrecision,
		RatingFloor:                     input.RatingFloor,
		RatingCeiling:                   input.RatingCeiling,
		PerformanceCeilingOverride:      input.PerformanceCeilingOverride,
		ClearRatingFloor:                input.ClearRatingFloor,
		ClearRatingCeiling:              input.ClearRatingCeiling,
		ClearPerformanceCeilingOverride: input.ClearPerformanceCeilingOverride,
		OgImage:                         input.OgImage,
		LogoOverrideImage:               input.LogoOverrideImage,
		ProblemLabelScript:              input.ProblemLabelScript,
		AccessCode:                      input.AccessCode,
		HideProblemTags:                 input.HideProblemTags,
		RunPretestsOnly:                 input.RunPretestsOnly,
		IsOrganizationPrivate:           input.IsOrganizationPrivate,
		FormatName:                      input.FormatName,
		FormatConfig:                    input.FormatConfig,
		AuthorIDs:                       input.AuthorIDs,
		CuratorIDs:                      input.CuratorIDs,
		TesterIDs:                       input.TesterIDs,
		TagIDs:                          input.TagIDs,
		OrganizationIDs:                 input.OrganizationIDs,
		BannedUserIDs:                   input.BannedUserIDs,
		PrivateContestantIDs:            input.PrivateContestantIDs,
		ViewScoreboardUserIDs:           input.ViewScoreboardUserIDs,
		JoinOrganizationIDs:             input.JoinOrganizationIDs,
		ClassIDs:                        input.ClassIDs,
		Problems:                        input.Problems,
		AddProblemIDs:                   input.AddProblemIDs,
		RemoveProblemIDs:                input.RemoveProblemIDs,
		AddTagIDs:                       input.AddTagIDs,
		RemoveTagIDs:                    input.RemoveTagIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, contest.ErrContestNotFound):
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
		case errors.Is(err, contest.ErrContestProblemHasSubmissions):
			c.JSON(http.StatusConflict, apiUserError(err.Error()))
		case errors.Is(err, contest.ErrInvalidTime),
			errors.Is(err, contest.ErrInvalidSchedule),
			errors.Is(err, contest.ErrUnknownFormat),
			errors.Is(err, contest.ErrInvalidFormatConfig),
			errors.Is(err, contest.ErrUnknownProblem),
			errors.Is(err, contest.ErrInvalidPoints),
			errors.Is(err, contest.ErrInvalidScoreboardVisibility),
			errors.Is(err, contest.ErrInvalidPointsPrecision),
			errors.Is(err, contest.ErrFieldTooLong),
			errors.Is(err, relations.ErrUnknownID):
			c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		default:
			c.JSON(http.StatusBadRequest, apiClientError(err))
		}
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
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiClientError(err))
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
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}

	locked, err := getContestService().LockContest(contest.LockContestRequest{
		ContestKey:  key,
		LockedAfter: input.LockedAfter,
	})
	if err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
			return
		}
		if errors.Is(err, contest.ErrInvalidLockedAfter) {
			c.JSON(http.StatusBadRequest, apiUserError("invalid locked_after format, use ISO 8601"))
			return
		}
		c.JSON(http.StatusBadRequest, apiClientError(err))
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
		c.JSON(http.StatusBadRequest, apiBindError(err))
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
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
			return
		}
		if errors.Is(err, contest.ErrContestKeyExists) {
			c.JSON(http.StatusBadRequest, apiUserError("contest key already exists"))
			return
		}
		if errors.Is(err, contest.ErrInvalidStartTime) {
			c.JSON(http.StatusBadRequest, apiUserError("invalid new_start_time format"))
			return
		}
		if errors.Is(err, contest.ErrInvalidEndTime) {
			c.JSON(http.StatusBadRequest, apiUserError("invalid new_end_time format"))
			return
		}
		c.JSON(http.StatusBadRequest, apiClientError(err))
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
		c.JSON(http.StatusBadRequest, apiUserError("invalid participation id"))
		return
	}

	if err := getContestService().DisqualifyParticipation(contest.DisqualifyParticipationRequest{
		ContestKey:      key,
		ParticipationID: participationID,
	}); err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
			return
		}
		if errors.Is(err, contest.ErrParticipationNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("participation not found"))
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
		c.JSON(http.StatusBadRequest, apiUserError("invalid participation id"))
		return
	}

	if err := getContestService().UndisqualifyParticipation(contest.UndisqualifyParticipationRequest{
		ContestKey:      key,
		ParticipationID: participationID,
	}); err != nil {
		if errors.Is(err, contest.ErrContestNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
			return
		}
		if errors.Is(err, contest.ErrParticipationNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("participation not found"))
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
