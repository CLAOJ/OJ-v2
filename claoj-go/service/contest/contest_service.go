// Package contest provides contest management services.
package contest

import (
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/sanitization"
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

	var contests []models.Contest
	query := db.DB.Order("start_time DESC")

	// Get total count
	var total int64
	if err := db.DB.Model(&models.Contest{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
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
	if err := db.DB.Preload("Authors").
		Preload("Curators").
		Preload("Testers").
		Preload("Tags").
		Where("key = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	// Get contest problems using problem service
	problems, err := s.problems.GetContestProblems(contest.ID)
	if err != nil {
		return nil, err
	}

	profile := contestToProfile(contest)

	return &ContestDetailResponse{
		Contest:  profile,
		Problems: problems,
	}, nil
}

// CreateContest creates a new contest.
func (s *ContestService) CreateContest(req CreateContestRequest) (*ContestProfile, error) {
	contest := models.Contest{
		Key:                   req.Key,
		Name:                  sanitization.SanitizeTitle(req.Name),
		Description:           sanitization.SanitizeProblemContent(req.Description),
		Summary:               sanitization.SanitizeBlogSummary(req.Summary),
		StartTime:             req.StartTime,
		EndTime:               req.EndTime,
		TimeLimit:             req.TimeLimit,
		IsVisible:             req.IsVisible,
		IsRated:               req.IsRated,
		FormatName:            req.FormatName,
		AccessCode:            req.AccessCode,
		HideProblemTags:       req.HideProblemTags,
		RunPretestsOnly:       req.RunPretestsOnly,
		IsOrganizationPrivate: req.IsOrganizationPrivate,
		MaxSubmissions:        req.MaxSubmissions,
	}

	if req.FormatConfig != "" {
		contest.FormatConfig = models.JSONField{}
		if err := contest.FormatConfig.Scan(req.FormatConfig); err != nil {
			return nil, ErrInvalidFormatConfig
		}
	}

	if err := db.DB.Create(&contest).Error; err != nil {
		return nil, err
	}

	// Handle many-to-many relations
	if len(req.AuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", req.AuthorIDs).Find(&authors)
		db.DB.Model(&contest).Association("Authors").Append(&authors)
	}
	if len(req.CuratorIDs) > 0 {
		var curators []models.Profile
		db.DB.Where("id IN ?", req.CuratorIDs).Find(&curators)
		db.DB.Model(&contest).Association("Curators").Append(&curators)
	}
	if len(req.TesterIDs) > 0 {
		var testers []models.Profile
		db.DB.Where("id IN ?", req.TesterIDs).Find(&testers)
		db.DB.Model(&contest).Association("Testers").Append(&testers)
	}
	if len(req.ProblemIDs) > 0 {
		// Use problem service to add problems
		if err := s.problems.AddProblemsToContest(contest.ID, req.ProblemIDs); err != nil {
			return nil, err
		}
	}
	if len(req.TagIDs) > 0 {
		var tags []models.ContestTag
		db.DB.Where("id IN ?", req.TagIDs).Find(&tags)
		db.DB.Model(&contest).Association("Tags").Append(&tags)
	}

	profile := contestToProfile(contest)
	return &profile, nil
}

// UpdateContest updates an existing contest.
func (s *ContestService) UpdateContest(req UpdateContestRequest) (*ContestProfile, error) {
	var contest models.Contest
	if err := db.DB.Where("key = ?", req.ContestKey).First(&contest).Error; err != nil {
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
	if req.TimeLimit != nil {
		updates["time_limit"] = *req.TimeLimit
	}
	if req.MaxSubmissions != nil {
		updates["max_submissions"] = *req.MaxSubmissions
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&contest).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Add problems using problem service
	if len(req.AddProblemIDs) > 0 {
		if err := s.problems.AddProblemsToContest(contest.ID, req.AddProblemIDs); err != nil {
			return nil, err
		}
	}

	// Remove problems using problem service
	if len(req.RemoveProblemIDs) > 0 {
		if err := s.problems.RemoveProblemsFromContest(contest.ID, req.RemoveProblemIDs); err != nil {
			return nil, err
		}
	}

	// Add tags
	if len(req.AddTagIDs) > 0 {
		var tags []models.ContestTag
		db.DB.Where("id IN ?", req.AddTagIDs).Find(&tags)
		db.DB.Model(&contest).Association("Tags").Append(&tags)
	}

	// Remove tags
	if len(req.RemoveTagIDs) > 0 {
		var tags []models.ContestTag
		db.DB.Where("id IN ?", req.RemoveTagIDs).Find(&tags)
		db.DB.Model(&contest).Association("Tags").Delete(&tags)
	}

	// Reload contest with relations
	db.DB.Preload("Authors").Preload("Curators").Preload("Testers").Preload("Tags").First(&contest, contest.ID)

	profile := contestToProfile(contest)
	return &profile, nil
}

// DeleteContest performs a soft delete by hiding the contest.
func (s *ContestService) DeleteContest(req DeleteContestRequest) error {
	var contest models.Contest
	if err := db.DB.Where("key = ?", req.ContestKey).First(&contest).Error; err != nil {
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
	if err := db.DB.Where("key = ?", req.ContestKey).First(&contest).Error; err != nil {
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
	if err := db.DB.Where("key = ?", req.SourceKey).First(&sourceContest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	// Check if new key already exists
	var existing models.Contest
	if err := db.DB.Where("key = ?", req.NewKey).First(&existing).Error; err == nil {
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
	if err := db.DB.Where("key = ?", req.ContestKey).First(&contest).Error; err != nil {
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
	if err := db.DB.Where("key = ?", req.ContestKey).First(&contest).Error; err != nil {
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
	if err := db.DB.Where("key = ?", req.ContestKey).First(&contest).Error; err != nil {
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
	if err := db.DB.Where("key = ?", req.ContestKey).First(&contest).Error; err != nil {
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
		ID:                     c.ID,
		Key:                    c.Key,
		Name:                   c.Name,
		Description:            c.Description,
		Summary:                c.Summary,
		StartTime:              c.StartTime,
		EndTime:                c.EndTime,
		TimeLimit:              c.TimeLimit,
		IsVisible:              c.IsVisible,
		IsRated:                c.IsRated,
		ScoreboardVisibility:   c.ScoreboardVisibility,
		ScoreboardCacheTimeout: c.ScoreboardCacheTimeout,
		UseClarifications:      c.UseClarifications,
		PushAnnouncements:      c.PushAnnouncements,
		RatingFloor:            c.RatingFloor,
		RatingCeiling:          c.RatingCeiling,
		RateAll:                c.RateAll,
		IsPrivate:              c.IsPrivate,
		HideProblemTags:        c.HideProblemTags,
		HideProblemAuthors:     c.HideProblemAuthors,
		RunPretestsOnly:        c.RunPretestsOnly,
		ShowShortDisplay:       c.ShowShortDisplay,
		IsOrganizationPrivate:  c.IsOrganizationPrivate,
		OgImage:                c.OgImage,
		LogoOverrideImage:      c.LogoOverrideImage,
		UserCount:              c.UserCount,
		VirtualCount:           c.VirtualCount,
		AccessCode:             c.AccessCode,
		FormatName:             c.FormatName,
		FormatConfig:           c.FormatConfig,
		ProblemLabelScript:     c.ProblemLabelScript,
		LockedAfter:            c.LockedAfter,
		PointsPrecision:        c.PointsPrecision,
		MaxSubmissions:         c.MaxSubmissions,
		AuthorIDs:              authorIDs,
		CuratorIDs:             curatorIDs,
		TesterIDs:              testerIDs,
		TagIDs:                 tagIDs,
		OrganizationIDs:        orgIDs,
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
