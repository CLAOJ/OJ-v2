package submission

import (
	"errors"
	"log"

	"github.com/CLAOJ/claoj/contest_format"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"gorm.io/gorm"
)

// DeleteRequest holds the parameters for deleting a submission.
type DeleteRequest struct {
	SubmissionID uint
}

// recomputeParticipation re-runs the contest format over a participation
// (loaded with its Contest). It is a variable so tests can replace it: the
// contest formats use MySQL-only SQL.
var recomputeParticipation = func(p *models.ContestParticipation) error {
	cf := contest_format.GetFormat(p.Contest.FormatName, &p.Contest, p.Contest.FormatConfig)
	return cf.UpdateParticipation(p)
}

// Delete permanently removes a finished submission together with its source,
// test cases and contest link — the rows Django's cascade removes — and then
// recalculates what the submission fed: the owner's points, the owner's
// organizations and, for a contest submission, the participation's score.
//
// A submission that is still being judged is refused: the judge would keep
// writing results for a row that no longer exists.
func (s *SubmissionService) Delete(req DeleteRequest) error {
	var sub models.Submission
	if err := db.DB.First(&sub, req.SubmissionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSubmissionNotFound
		}
		return err
	}
	switch sub.Status {
	case "QU", "P", "G":
		return ErrSubmissionInProgress
	}

	var cSub models.ContestSubmission
	inContest := db.DB.Preload("Participation.Contest").
		Where("submission_id = ?", sub.ID).First(&cSub).Error == nil

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		return deleteSubmissionRows(tx, &sub)
	})
	if err != nil {
		return err
	}

	participations := map[uint]*models.ContestParticipation{}
	if inContest {
		participations[cSub.ParticipationID] = &cSub.Participation
	}
	recalculateAffected([]models.Submission{sub}, participations)
	return nil
}

// BulkDeleteRequest holds the parameters for deleting several submissions.
type BulkDeleteRequest struct {
	SubmissionIDs []uint
}

// BulkDelete removes the given submissions in one transaction. Unknown ids are
// ignored; any submission still being judged aborts the whole batch. Points
// are recalculated after the commit, once per affected user, organization and
// participation.
func (s *SubmissionService) BulkDelete(req BulkDeleteRequest) (int, error) {
	if len(req.SubmissionIDs) == 0 {
		return 0, nil
	}
	var subs []models.Submission
	if err := db.DB.Where("id IN ?", req.SubmissionIDs).Find(&subs).Error; err != nil {
		return 0, err
	}
	for _, sub := range subs {
		switch sub.Status {
		case "QU", "P", "G":
			return 0, ErrSubmissionInProgress
		}
	}

	type contestLink struct {
		cSub models.ContestSubmission
		ok   bool
	}
	links := make(map[uint]contestLink, len(subs))
	for _, sub := range subs {
		var cSub models.ContestSubmission
		ok := db.DB.Preload("Participation.Contest").
			Where("submission_id = ?", sub.ID).First(&cSub).Error == nil
		links[sub.ID] = contestLink{cSub: cSub, ok: ok}
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		for i := range subs {
			if err := deleteSubmissionRows(tx, &subs[i]); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return 0, err
	}

	participations := make(map[uint]*models.ContestParticipation, len(subs))
	for i := range subs {
		link := links[subs[i].ID]
		if link.ok {
			participations[link.cSub.ParticipationID] = &link.cSub.Participation
		}
	}
	recalculateAffected(subs, participations)
	return len(subs), nil
}

// deleteSubmissionRows removes the rows Django's cascade removes for one
// submission. The caller owns the transaction.
func deleteSubmissionRows(tx *gorm.DB, sub *models.Submission) error {
	if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.SubmissionTestCase{}).Error; err != nil {
		return err
	}
	if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.SubmissionSource{}).Error; err != nil {
		return err
	}
	if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.ContestSubmission{}).Error; err != nil {
		return err
	}
	return tx.Delete(sub).Error
}

// recalculateAffected refreshes what a batch of deleted submissions fed: the
// owners' points, their organizations' points and the contest participations'
// scores — once per distinct user, organization and participation, not once
// per submission. A failure is logged, not returned: retrying the delete could
// not fix it.
func recalculateAffected(subs []models.Submission, participations map[uint]*models.ContestParticipation) {
	users := make(map[uint]bool, len(subs))
	orgs := make(map[uint]bool)
	for _, sub := range subs {
		users[sub.UserID] = true
		var orgIDs []uint
		if err := db.DB.Table("judge_profile_organizations").Where("profile_id = ?", sub.UserID).Pluck("organization_id", &orgIDs).Error; err != nil {
			log.Printf("submission: delete %d: load organizations of user %d: %v", sub.ID, sub.UserID, err)
			continue
		}
		for _, orgID := range orgIDs {
			orgs[orgID] = true
		}
	}
	for userID := range users {
		if err := scoring.CalculateProfilePoints(db.DB, userID); err != nil {
			log.Printf("submission: delete: recalculate points for profile %d: %v", userID, err)
		}
	}
	for orgID := range orgs {
		if err := scoring.CalculateOrganizationPoints(db.DB, orgID); err != nil {
			log.Printf("submission: delete: recalculate points for organization %d: %v", orgID, err)
		}
	}
	for id, p := range participations {
		if err := recomputeParticipation(p); err != nil {
			log.Printf("submission: delete: recompute participation %d: %v", id, err)
		}
	}
}
