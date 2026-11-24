// Package contest provides contest management services.
package contest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CLAOJ/claoj/contest_format"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
	"github.com/CLAOJ/claoj/service/relations"
	"gorm.io/gorm"
)

// ContestService provides contest management operations.
type ContestService struct {
	problems *ContestProblemService
}

// NewContestService creates a new ContestService instance.
func NewContestService() *ContestService {
	return &ContestService{
		problems: NewContestProblemService(),
	}
}

// GetProblemService returns the contest problem service.
func (s *ContestService) GetProblemService() *ContestProblemService {
	return s.problems
}

// ListContests retrieves a paginated list of contests.
func (s *ContestService) ListContests(req ListContestsRequest) (*ListContestsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	filter := func(q *gorm.DB) (*gorm.DB, error) {
		if req.Search != "" {
			p := db.ContainsPattern(req.Search)
			q = q.Where("(`key` LIKE ? ESCAPE '!' OR name LIKE ? ESCAPE '!')", p, p)
		}
		switch req.Status {
		case "":
		case "upcoming":
			q = q.Where("start_time > ?", now)
		case "ongoing":
			q = q.Where("start_time <= ? AND end_time > ?", now, now)
		case "past":
			q = q.Where("end_time <= ?", now)
		default:
			return nil, ErrInvalidStatus
		}
		return q, nil
	}

	countQ, err := filter(db.DB.Model(&models.Contest{}))
	if err != nil {
		return nil, err
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, err
	}

	listQ, _ := filter(db.DB.Model(&models.Contest{}))
	var contests []models.Contest
	if err := listQ.Order("start_time DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&contests).Error; err != nil {
		return nil, err
	}

	result := make([]ContestProfile, len(contests))
	for i, c := range contests {
		result[i] = contestToProfile(c)
	}

	return &ListContestsResponse{
		Contests: result,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetContest retrieves a contest by key with full details.
func (s *ContestService) GetContest(req GetContestRequest) (*ContestDetailResponse, error) {
	var contest models.Contest
	if err := db.DB.Preload("Authors.User").
		Preload("Curators.User").
		Preload("Testers.User").
		Preload("BannedUsers.User").
		Preload("PrivateContestants.User").
		Preload("ViewContestScoreboard.User").
		Preload("Tags").
		Preload("Organizations").
		Preload("JoinOrganizations").
		Preload("Classes").
		Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	problems, err := s.problems.GetContestProblems(contest.ID)
	if err != nil {
		return nil, err
	}

	return &ContestDetailResponse{ContestProfile: contestToProfile(contest), Problems: problems}, nil
}

// CreateContest creates a new contest.
func (s *ContestService) CreateContest(req CreateContestRequest) (*ContestProfile, error) {
	visibility := req.ScoreboardVisibility
	if visibility == "" {
		visibility = "V"
	}
	if !validScoreboardVisibility(visibility) {
		return nil, ErrInvalidScoreboardVisibility
	}
	if err := checkFieldLength(req.OgImage, 150, "og_image"); err != nil {
		return nil, err
	}
	if err := checkFieldLength(req.LogoOverrideImage, 150, "logo_override_image"); err != nil {
		return nil, err
	}
	if req.PointsPrecision != nil && !validPointsPrecision(*req.PointsPrecision) {
		return nil, ErrInvalidPointsPrecision
	}
	// v1 defaults: clarifications on, three digits of points. Written
	// explicitly so the returned profile matches the stored row.
	useClarifications := true
	if req.UseClarifications != nil {
		useClarifications = *req.UseClarifications
	}
	pointsPrecision := 3
	if req.PointsPrecision != nil {
		pointsPrecision = *req.PointsPrecision
	}

	contest := models.Contest{
		Key:                        req.Key,
		Name:                       sanitization.SanitizeTitle(req.Name),
		Description:                sanitization.SanitizeProblemContent(req.Description),
		Summary:                    sanitization.SanitizeBlogSummary(req.Summary),
		StartTime:                  req.StartTime,
		EndTime:                    req.EndTime,
		TimeLimit:                  req.TimeLimit,
		IsVisible:                  req.IsVisible,
		IsRated:                    req.IsRated,
		ScoreboardVisibility:       visibility,
		ScoreboardCacheTimeout:     req.ScoreboardCacheTimeout,
		UseClarifications:          useClarifications,
		PushAnnouncements:          req.PushAnnouncements,
		RatingFloor:                req.RatingFloor,
		RatingCeiling:              req.RatingCeiling,
		PerformanceCeilingOverride: req.PerformanceCeilingOverride,
		RateAll:                    req.RateAll,
		HideProblemAuthors:         req.HideProblemAuthors,
		ShowShortDisplay:           req.ShowShortDisplay,
		IsOrganizationPrivate:      req.IsOrganizationPrivate,
		LimitJoinOrganizations:     req.LimitJoinOrganizations,
		OgImage:                    req.OgImage,
		LogoOverrideImage:          req.LogoOverrideImage,
		ProblemLabelScript:         req.ProblemLabelScript,
		PointsPrecision:            pointsPrecision,
		FormatName:                 req.FormatName,
		AccessCode:                 req.AccessCode,
		HideProblemTags:            req.HideProblemTags,
		RunPretestsOnly:            req.RunPretestsOnly,
	}

	if req.FormatConfig != "" {
		contest.FormatConfig = models.JSONField{}
		if err := contest.FormatConfig.Scan(req.FormatConfig); err != nil {
			return nil, ErrInvalidFormatConfig
		}
	}

	// Insert and relations in one transaction so a rejected relation id
	// cannot leave an orphan contest row.
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&contest).Error; err != nil {
			return err
		}
		// GORM substitutes the model default for zero-value fields, so the
		// legal v1 values false/0 must be written explicitly.
		explicit := map[string]interface{}{}
		if !useClarifications {
			explicit["use_clarifications"] = false
		}
		if pointsPrecision == 0 {
			explicit["points_precision"] = 0
		}
		if len(explicit) > 0 {
			if err := tx.Model(&contest).Updates(explicit).Error; err != nil {
				return err
			}
		}
		if err := relations.Sync[models.Profile](tx, &contest, "Authors", fullList(req.AuthorIDs), nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "Curators", fullList(req.CuratorIDs), nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "Testers", fullList(req.TesterIDs), nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.ContestTag](tx, &contest, "Tags", fullList(req.TagIDs), nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Organization](tx, &contest, "Organizations", fullList(req.OrganizationIDs), nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "BannedUsers", fullList(req.BannedUserIDs), nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "PrivateContestants", fullList(req.PrivateContestantIDs), nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "ViewContestScoreboard", fullList(req.ViewScoreboardUserIDs), nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Organization](tx, &contest, "JoinOrganizations", fullList(req.JoinOrganizationIDs), nil, nil); err != nil {
			return err
		}
		return relations.Sync[models.Class](tx, &contest, "Classes", fullList(req.ClassIDs), nil, nil)
	})
	if err != nil {
		return nil, err
	}

	if len(req.ProblemIDs) > 0 {
		// Use problem service to add problems
		if err := s.problems.AddProblemsToContest(contest.ID, req.ProblemIDs); err != nil {
			return nil, err
		}
	}

	profile := contestToProfile(contest)
	return &profile, nil
}

// UpdateContest updates an existing contest in one transaction.
func (s *ContestService) UpdateContest(req UpdateContestRequest) (*ContestProfile, error) {
	var contest models.Contest
	if err := db.DB.Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = sanitization.SanitizeTitle(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = sanitization.SanitizeProblemContent(*req.Description)
	}
	if req.Summary != nil {
		updates["summary"] = sanitization.SanitizeBlogSummary(*req.Summary)
	}
	if req.IsVisible != nil {
		updates["is_visible"] = *req.IsVisible
	}
	if req.IsRated != nil {
		updates["is_rated"] = *req.IsRated
	}
	if req.AccessCode != nil {
		updates["access_code"] = *req.AccessCode
	}
	if req.HideProblemTags != nil {
		updates["hide_problem_tags"] = *req.HideProblemTags
	}
	if req.RunPretestsOnly != nil {
		updates["run_pretests_only"] = *req.RunPretestsOnly
	}
	if req.IsOrganizationPrivate != nil {
		updates["is_organization_private"] = *req.IsOrganizationPrivate
	}
	if req.LimitJoinOrganizations != nil {
		updates["limit_join_organizations"] = *req.LimitJoinOrganizations
	}
	if req.ScoreboardVisibility != nil {
		if !validScoreboardVisibility(*req.ScoreboardVisibility) {
			return nil, ErrInvalidScoreboardVisibility
		}
		updates["scoreboard_visibility"] = *req.ScoreboardVisibility
	}
	if req.ScoreboardCacheTimeout != nil {
		updates["scoreboard_cache_timeout"] = *req.ScoreboardCacheTimeout
	}
	if req.UseClarifications != nil {
		updates["use_clarifications"] = *req.UseClarifications
	}
	if req.PushAnnouncements != nil {
		updates["push_announcements"] = *req.PushAnnouncements
	}
	if req.HideProblemAuthors != nil {
		updates["hide_problem_authors"] = *req.HideProblemAuthors
	}
	if req.ShowShortDisplay != nil {
		updates["show_short_display"] = *req.ShowShortDisplay
	}
	if req.RateAll != nil {
		updates["rate_all"] = *req.RateAll
	}
	if req.PointsPrecision != nil {
		if !validPointsPrecision(*req.PointsPrecision) {
			return nil, ErrInvalidPointsPrecision
		}
		updates["points_precision"] = *req.PointsPrecision
	}
	if req.ClearRatingFloor {
		updates["rating_floor"] = nil
	} else if req.RatingFloor != nil {
		updates["rating_floor"] = *req.RatingFloor
	}
	if req.ClearRatingCeiling {
		updates["rating_ceiling"] = nil
	} else if req.RatingCeiling != nil {
		updates["rating_ceiling"] = *req.RatingCeiling
	}
	if req.ClearPerformanceCeilingOverride {
		updates["performance_ceiling_override"] = nil
	} else if req.PerformanceCeilingOverride != nil {
		updates["performance_ceiling_override"] = *req.PerformanceCeilingOverride
	}
	if req.OgImage != nil {
		if err := checkFieldLength(*req.OgImage, 150, "og_image"); err != nil {
			return nil, err
		}
		updates["og_image"] = *req.OgImage
	}
	if req.LogoOverrideImage != nil {
		if err := checkFieldLength(*req.LogoOverrideImage, 150, "logo_override_image"); err != nil {
			return nil, err
		}
		updates["logo_override_image"] = *req.LogoOverrideImage
	}
	if req.ProblemLabelScript != nil {
		updates["problem_label_script"] = *req.ProblemLabelScript
	}
	if req.TimeLimit != nil {
		if *req.TimeLimit <= 0 {
			updates["time_limit"] = gorm.Expr("NULL")
		} else {
			updates["time_limit"] = *req.TimeLimit
		}
	}

	start, end := contest.StartTime, contest.EndTime
	if req.StartTime != nil {
		t, err := time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			return nil, ErrInvalidTime
		}
		start = t
		updates["start_time"] = t
	}
	if req.EndTime != nil {
		t, err := time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			return nil, ErrInvalidTime
		}
		end = t
		updates["end_time"] = t
	}
	if !end.After(start) {
		return nil, ErrInvalidSchedule
	}

	if req.FormatName != nil {
		if !contest_format.IsRegistered(*req.FormatName) {
			return nil, ErrUnknownFormat
		}
		updates["format_name"] = *req.FormatName
	}
	// The config belongs to the format that will be in effect after this
	// update, which may itself be changing.
	effectiveFormat := contest.FormatName
	if req.FormatName != nil {
		effectiveFormat = *req.FormatName
	}
	if req.FormatConfig != nil {
		if strings.TrimSpace(*req.FormatConfig) == "" {
			updates["format_config"] = gorm.Expr("NULL")
		} else {
			var cfg models.JSONField
			if err := cfg.Scan(*req.FormatConfig); err != nil {
				return nil, ErrInvalidFormatConfig
			}
			raw, err := json.Marshal(cfg)
			if err != nil {
				return nil, ErrInvalidFormatConfig
			}
			var decoded map[string]interface{}
			if err := json.Unmarshal(raw, &decoded); err != nil {
				return nil, ErrInvalidFormatConfig
			}
			if err := contest_format.ValidateConfig(effectiveFormat, decoded); err != nil {
				return nil, ErrInvalidFormatConfig
			}
			updates["format_config"] = cfg
		}
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&contest).Updates(updates).Error; err != nil {
				return err
			}
		}
		if err := relations.Sync[models.Profile](tx, &contest, "Authors", req.AuthorIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "Curators", req.CuratorIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "Testers", req.TesterIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.ContestTag](tx, &contest, "Tags", req.TagIDs, req.AddTagIDs, req.RemoveTagIDs); err != nil {
			return err
		}
		if err := relations.Sync[models.Organization](tx, &contest, "Organizations", req.OrganizationIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "BannedUsers", req.BannedUserIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "PrivateContestants", req.PrivateContestantIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "ViewContestScoreboard", req.ViewScoreboardUserIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Organization](tx, &contest, "JoinOrganizations", req.JoinOrganizationIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Class](tx, &contest, "Classes", req.ClassIDs, nil, nil); err != nil {
			return err
		}
		return s.applyProblems(tx, contest.ID, req)
	})
	if err != nil {
		return nil, err
	}

	detail, err := s.GetContest(GetContestRequest{ContestKey: contest.Key})
	if err != nil {
		return nil, err
	}
	return &detail.ContestProfile, nil
}

// applyProblems writes the contest problem list: the full list when given,
// otherwise the legacy add/remove lists folded into one.
func (s *ContestService) applyProblems(tx *gorm.DB, contestID uint, req UpdateContestRequest) error {
	if req.Problems != nil {
		return s.problems.ReplaceContestProblems(tx, contestID, *req.Problems)
	}
	if len(req.AddProblemIDs) == 0 && len(req.RemoveProblemIDs) == 0 {
		return nil
	}
	items, err := s.problems.mergeLegacyProblemEdits(tx, contestID, req.AddProblemIDs, req.RemoveProblemIDs)
	if err != nil {
		return err
	}
	return s.problems.ReplaceContestProblems(tx, contestID, items)
}

// DeleteContest performs a soft delete by hiding the contest.
func (s *ContestService) DeleteContest(req DeleteContestRequest) error {
	var contest models.Contest
	if err := db.DB.Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrContestNotFound
		}
		return err
	}

	return db.DB.Model(&contest).Update("is_visible", false).Error
}

// LockContest locks or unlocks a contest.
func (s *ContestService) LockContest(req LockContestRequest) (bool, error) {
	var contest models.Contest
	if err := db.DB.Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, ErrContestNotFound
		}
		return false, err
	}

	if req.LockedAfter == nil {
		// Unlock
		if err := db.DB.Model(&contest).Update("locked_after", gorm.Expr("NULL")).Error; err != nil {
			return false, err
		}
		return false, nil
	}

	// Lock at specified time
	lockedAt, err := time.Parse(time.RFC3339, *req.LockedAfter)
	if err != nil {
		return false, ErrInvalidLockedAfter
	}

	if err := db.DB.Model(&contest).Update("locked_after", lockedAt).Error; err != nil {
		return false, err
	}

	return true, nil
}

// CloneContest creates a copy of an existing contest.
func (s *ContestService) CloneContest(req CloneContestRequest) (*ContestProfile, error) {
	// Get source contest
	var sourceContest models.Contest
	if err := db.DB.Where("`key` = ?", req.SourceKey).First(&sourceContest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	// Check if new key already exists
	var existing models.Contest
	if err := db.DB.Where("`key` = ?", req.NewKey).First(&existing).Error; err == nil {
		return nil, ErrContestKeyExists
	}

	// Parse times
	var startTime, endTime time.Time
	if req.NewStartTime != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, req.NewStartTime)
		if err != nil {
			return nil, ErrInvalidStartTime
		}
		endTime, err = time.Parse(time.RFC3339, req.NewEndTime)
		if err != nil {
			return nil, ErrInvalidEndTime
		}
	} else {
		startTime = sourceContest.StartTime
		endTime = sourceContest.EndTime
	}

	// Create new contest
	newContest := models.Contest{
		Key:                    req.NewKey,
		Name:                   sanitization.SanitizeTitle(req.NewName),
		Description:            sourceContest.Description,
		Summary:                sourceContest.Summary,
		StartTime:              startTime,
		EndTime:                endTime,
		TimeLimit:              sourceContest.TimeLimit,
		IsVisible:              false,
		IsRated:                sourceContest.IsRated,
		ScoreboardVisibility:   sourceContest.ScoreboardVisibility,
		ScoreboardCacheTimeout: sourceContest.ScoreboardCacheTimeout,
		UseClarifications:      sourceContest.UseClarifications,
		PushAnnouncements:      sourceContest.PushAnnouncements,
		RatingFloor:            sourceContest.RatingFloor,
		RatingCeiling:          sourceContest.RatingCeiling,
		RateAll:                sourceContest.RateAll,
		IsPrivate:              sourceContest.IsPrivate,
		HideProblemTags:        sourceContest.HideProblemTags,
		HideProblemAuthors:     sourceContest.HideProblemAuthors,
		RunPretestsOnly:        sourceContest.RunPretestsOnly,
		ShowShortDisplay:       sourceContest.ShowShortDisplay,
		IsOrganizationPrivate:  sourceContest.IsOrganizationPrivate,
		OgImage:                sourceContest.OgImage,
		LogoOverrideImage:      sourceContest.LogoOverrideImage,
		AccessCode:             sourceContest.AccessCode,
		FormatName:             sourceContest.FormatName,
		FormatConfig:           sourceContest.FormatConfig,
		ProblemLabelScript:     sourceContest.ProblemLabelScript,
		LockedAfter:            nil,
		PointsPrecision:        sourceContest.PointsPrecision,
	}

	if err := db.DB.Create(&newContest).Error; err != nil {
		return nil, err
	}

	// Copy authors, curators, testers, organizations if requested
	if req.CopySettings {
		copyAssociations(db.DB, sourceContest.ID, newContest.ID)
	}

	// Copy problems if requested
	if req.CopyProblems {
		if err := s.problems.CopyProblemsToContest(sourceContest.ID, newContest.ID); err != nil {
			return nil, err
		}
	}

	// Reload with relations
	db.DB.Preload("Authors").Preload("Curators").Preload("Testers").Preload("Tags").First(&newContest, newContest.ID)

	profile := contestToProfile(newContest)
	return &profile, nil
}

// DisqualifyParticipation disqualifies a contest participation.
func (s *ContestService) DisqualifyParticipation(req DisqualifyParticipationRequest) error {
	var contest models.Contest
	if err := db.DB.Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrContestNotFound
		}
		return err
	}

	var participation models.ContestParticipation
	if err := db.DB.Where("contest_id = ? AND id = ?", contest.ID, req.ParticipationID).First(&participation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrParticipationNotFound
		}
		return err
	}

	return db.DB.Model(&participation).Update("is_disqualified", true).Error
}

// UndisqualifyParticipation undisqualifies a contest participation.
func (s *ContestService) UndisqualifyParticipation(req UndisqualifyParticipationRequest) error {
	var contest models.Contest
	if err := db.DB.Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrContestNotFound
		}
		return err
	}

	var participation models.ContestParticipation
	if err := db.DB.Where("contest_id = ? AND id = ?", contest.ID, req.ParticipationID).First(&participation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrParticipationNotFound
		}
		return err
	}

	return db.DB.Model(&participation).Update("is_disqualified", false).Error
}

// AddTag adds a tag to a contest.
func (s *ContestService) AddTag(req AddTagRequest) error {
	var contest models.Contest
	if err := db.DB.Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrContestNotFound
		}
		return err
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, req.TagID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrTagNotFound
		}
		return err
	}

	return db.DB.Model(&contest).Association("Tags").Append(&tag)
}

// RemoveTag removes a tag from a contest.
func (s *ContestService) RemoveTag(req RemoveTagRequest) error {
	var contest models.Contest
	if err := db.DB.Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrContestNotFound
		}
		return err
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, req.TagID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrTagNotFound
		}
		return err
	}

	return db.DB.Model(&contest).Association("Tags").Delete(&tag)
}

// Helper functions

func contestToProfile(c models.Contest) ContestProfile {
	authorIDs := getProfileIDs(c.Authors)
	curatorIDs := getProfileIDs(c.Curators)
	testerIDs := getProfileIDs(c.Testers)
	tagIDs := getTagIDs(c.Tags)
	orgIDs := getOrgIDs(c.Organizations)

	return ContestProfile{
		ID:                         c.ID,
		Key:                        c.Key,
		Name:                       c.Name,
		Description:                c.Description,
		Summary:                    c.Summary,
		StartTime:                  c.StartTime,
		EndTime:                    c.EndTime,
		TimeLimit:                  c.TimeLimit,
		TimeLimitSeconds:           microsToSeconds(c.TimeLimit),
		IsVisible:                  c.IsVisible,
		IsRated:                    c.IsRated,
		ScoreboardVisibility:       c.ScoreboardVisibility,
		ScoreboardCacheTimeout:     c.ScoreboardCacheTimeout,
		UseClarifications:          c.UseClarifications,
		PushAnnouncements:          c.PushAnnouncements,
		RatingFloor:                c.RatingFloor,
		RatingCeiling:              c.RatingCeiling,
		PerformanceCeilingOverride: c.PerformanceCeilingOverride,
		RateAll:                    c.RateAll,
		IsPrivate:                  c.IsPrivate,
		HideProblemTags:            c.HideProblemTags,
		HideProblemAuthors:         c.HideProblemAuthors,
		RunPretestsOnly:            c.RunPretestsOnly,
		ShowShortDisplay:           c.ShowShortDisplay,
		IsOrganizationPrivate:      c.IsOrganizationPrivate,
		LimitJoinOrganizations:     c.LimitJoinOrganizations,
		OgImage:                    c.OgImage,
		LogoOverrideImage:          c.LogoOverrideImage,
		UserCount:                  c.UserCount,
		VirtualCount:               c.VirtualCount,
		AccessCode:                 c.AccessCode,
		FormatName:                 c.FormatName,
		FormatConfig:               c.FormatConfig,
		ProblemLabelScript:         c.ProblemLabelScript,
		LockedAfter:                c.LockedAfter,
		PointsPrecision:            c.PointsPrecision,
		AuthorIDs:                  authorIDs,
		CuratorIDs:                 curatorIDs,
		TesterIDs:                  testerIDs,
		TagIDs:                     tagIDs,
		OrganizationIDs:            orgIDs,
		BannedUserIDs:              getProfileIDs(c.BannedUsers),
		PrivateContestantIDs:       getProfileIDs(c.PrivateContestants),
		ViewScoreboardUserIDs:      getProfileIDs(c.ViewContestScoreboard),
		JoinOrganizationIDs:        getOrgIDs(c.JoinOrganizations),
		ClassIDs:                   getClassIDs(c.Classes),
		Authors:                    relations.UserRefs(c.Authors),
		Curators:                   relations.UserRefs(c.Curators),
		Testers:                    relations.UserRefs(c.Testers),
		BannedUsers:                relations.UserRefs(c.BannedUsers),
		PrivateContestants:         relations.UserRefs(c.PrivateContestants),
		ViewScoreboardUsers:        relations.UserRefs(c.ViewContestScoreboard),
		JoinOrganizations:          orgRefs(c.JoinOrganizations),
		Classes:                    classRefs(c.Classes),
	}
}

func getProfileIDs(profiles []models.Profile) []uint {
	ids := make([]uint, len(profiles))
	for i, p := range profiles {
		ids[i] = p.ID
	}
	return ids
}

func getTagIDs(tags []models.ContestTag) []uint {
	ids := make([]uint, len(tags))
	for i, t := range tags {
		ids[i] = t.ID
	}
	return ids
}

func getOrgIDs(orgs []models.Organization) []uint {
	ids := make([]uint, len(orgs))
	for i, o := range orgs {
		ids[i] = o.ID
	}
	return ids
}

func getClassIDs(classes []models.Class) []uint {
	ids := make([]uint, len(classes))
	for i, c := range classes {
		ids[i] = c.ID
	}
	return ids
}

func orgRefs(orgs []models.Organization) []NamedRef {
	refs := make([]NamedRef, len(orgs))
	for i, o := range orgs {
		refs[i] = NamedRef{ID: o.ID, Name: o.Name}
	}
	return refs
}

func classRefs(classes []models.Class) []NamedRef {
	refs := make([]NamedRef, len(classes))
	for i, c := range classes {
		refs[i] = NamedRef{ID: c.ID, Name: c.Name}
	}
	return refs
}

func validScoreboardVisibility(v string) bool {
	return v == "V" || v == "C" || v == "P"
}

// validPointsPrecision mirrors v1's 0..10 validators on points_precision.
func validPointsPrecision(v int) bool {
	return v >= 0 && v <= 10
}

// checkFieldLength rejects values over the model's varchar limit, which MySQL
// would otherwise surface as a raw driver error.
func checkFieldLength(value string, max int, field string) error {
	if utf8.RuneCountInString(value) > max {
		return fmt.Errorf("%w: %s must be at most %d characters", ErrFieldTooLong, field, max)
	}
	return nil
}

// fullList maps an empty create-time list to nil so relations.Sync is a no-op.
func fullList(ids []uint) *[]uint {
	if len(ids) == 0 {
		return nil
	}
	return &ids
}

func copyAssociations(tx *gorm.DB, sourceID, targetID uint) {
	var authors, curators, testers []models.Profile
	var orgs []models.Organization

	tx.Model(&models.Contest{}).Where("id = ?", sourceID).Association("Authors").Find(&authors)
	tx.Model(&models.Contest{}).Where("id = ?", sourceID).Association("Curators").Find(&curators)
	tx.Model(&models.Contest{}).Where("id = ?", sourceID).Association("Testers").Find(&testers)
	tx.Model(&models.Contest{}).Where("id = ?", sourceID).Association("Organizations").Find(&orgs)

	if len(authors) > 0 {
		tx.Model(&models.Contest{}).Where("id = ?", targetID).Association("Authors").Append(&authors)
	}
	if len(curators) > 0 {
		tx.Model(&models.Contest{}).Where("id = ?", targetID).Association("Curators").Append(&curators)
	}
	if len(testers) > 0 {
		tx.Model(&models.Contest{}).Where("id = ?", targetID).Association("Testers").Append(&testers)
	}
	if len(orgs) > 0 {
		tx.Model(&models.Contest{}).Where("id = ?", targetID).Association("Organizations").Append(&orgs)
	}
}

// microsToSeconds converts Django DurationField microseconds to seconds.
func microsToSeconds(us *int64) *int64 {
	if us == nil {
		return nil
	}
	s := *us / 1_000_000
	return &s
}
