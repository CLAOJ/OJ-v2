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
		if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.SubmissionTestCase{}).Error; err != nil {
			return err
		}
		if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.SubmissionSource{}).Error; err != nil {
			return err
		}
		if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.ContestSubmission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&sub).Error
	})
	if err != nil {
		return err
	}

	// The submission is gone; a failed recalculation is logged, not returned,
	// because retrying the delete could not fix it.
	if err := scoring.CalculateProfilePoints(db.DB, sub.UserID); err != nil {
		log.Printf("submission: delete %d: recalculate points for profile %d: %v", sub.ID, sub.UserID, err)
	}
	var orgIDs []uint
	if err := db.DB.Table("judge_profile_organizations").Where("profile_id = ?", sub.UserID).Pluck("organization_id", &orgIDs).Error; err != nil {
		log.Printf("submission: delete %d: load organizations of user %d: %v", sub.ID, sub.UserID, err)
	}
	for _, orgID := range orgIDs {
		if err := scoring.CalculateOrganizationPoints(db.DB, orgID); err != nil {
			log.Printf("submission: delete %d: recalculate points for organization %d: %v", sub.ID, orgID, err)
		}
	}
	if inContest {
		if err := recomputeParticipation(&cSub.Participation); err != nil {
			log.Printf("submission: delete %d: recompute participation %d: %v", sub.ID, cSub.ParticipationID, err)
		}
	}
	return nil
}
