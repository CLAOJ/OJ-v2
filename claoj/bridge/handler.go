package bridge

import (
	"errors"
	"fmt"
	"log"
	"time"

	v2 "github.com/CLAOJ/claoj/api/v2"
	"github.com/CLAOJ/claoj/contest_format"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"gorm.io/gorm"
)

// ErrBadPacket marks a packet the judge sent malformed (HTTP 400).
var ErrBadPacket = errors.New("bad judge packet")

// maxFeedback is the width of judge_submissiontestcase.feedback.
const maxFeedback = 50

// statusOrder ranks case verdicts; the worst becomes the submission result.
var statusOrder = map[string]int{"AC": 0, "SC": 1, "WA": 2, "MLE": 3, "TLE": 4, "IR": 5, "RTE": 6, "OLE": 7}

// HandleEvent applies one packet a judge agent forwarded from tiervnoj.
// judge is the authenticated judge_judge row and clientIP is where the request
// came from (dropped when empty). Every branch must be safe to replay: the
// agent re-POSTs a packet whose response it never received.
func HandleEvent(judge *models.Judge, pkt Packet, clientIP string) error {
	// judge-runtimes carries no submission-id: it reports the executors
	// tiervnoj announced in its handshake (v1's RuntimeVersion rewrite).
	if pkt.Name() == "judge-runtimes" {
		return onJudgeRuntimes(judge, pkt)
	}

	// judge-status also carries no submission-id: it reports the latency the
	// agent measured posting to this backend — v2's replacement for v1's
	// websocket ping (load is kept separately by jobs.SyncJudgeStatus).
	if pkt.Name() == "judge-status" {
		return onJudgeStatus(judge, pkt, clientIP)
	}

	subID, ok := pkt.SubmissionID()
	if !ok {
		return fmt.Errorf("%w: %q has no submission-id", ErrBadPacket, pkt.Name())
	}

	switch pkt.Name() {
	case "submission-acknowledged":
		return onSubmissionAcknowledged(subID)
	case "grading-begin":
		return onGradingBegin(judge, subID)
	case "test-case-status":
		return onTestCase(subID, pkt)
	case "grading-end":
		return onGradingEnd(subID)
	case "compile-error":
		return onCompileError(subID, pkt)
	case "compile-message":
		return onCompileMessage(subID, pkt)
	case "internal-error":
		return onInternalError(subID, pkt)
	case "submission-terminated":
		return onSubmissionTerminated(subID)
	default:
		return nil // batch-begin / batch-end and anything new: ignored, as before
	}
}

// RecordLastIP stores the address the judge connected from, the way v1 wrote
// judge_judge.last_ip when the daemon opened its websocket. Called only on the
// handshake; the periodic status beat carries last_ip in its own update.
func RecordLastIP(judgeID uint, ip string) error {
	if ip == "" {
		return nil
	}
	return db.DB.Model(&models.Judge{}).Where("id = ?", judgeID).Update("last_ip", ip).Error
}

// onJudgeStatus records the round trip and machine load the agent measured,
// which are v2's analog of v1's judge_judge.ping/load (v1's daemon reported
// both). Fields that are absent or negative — an older agent, or a failed
// read — are left untouched rather than zeroed. last_ip rides along in the same
// UPDATE (an empty client address never blanks it).
func onJudgeStatus(judge *models.Judge, pkt Packet, clientIP string) error {
	updates := map[string]interface{}{}
	if clientIP != "" {
		updates["last_ip"] = clientIP
	}
	if ping, ok := pkt["ping"].(float64); ok && ping >= 0 {
		updates["ping"] = ping
	}
	if load, ok := pkt["load"].(float64); ok && load >= 0 {
		updates["load"] = load
	}
	if len(updates) == 0 {
		return nil
	}
	return db.DB.Model(&models.Judge{}).Where("id = ?", judge.ID).Updates(updates).Error
}

func onSubmissionAcknowledged(subID uint) error {
	if err := db.DB.Model(&models.Submission{}).Where("id = ?", subID).Update("status", "P").Error; err != nil {
		return err
	}
	PostSubmissionState(subID, "processing", nil)
	return nil
}

func onGradingBegin(judge *models.Judge, subID uint) error {
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Submission{}).Where("id = ?", subID).Updates(map[string]interface{}{
			"status":           "G",
			"current_testcase": 1,
			"batch":            false,
			"judged_date":      time.Now(),
			"judged_on_id":     judge.ID,
		}).Error; err != nil {
			return err
		}
		// a regrade after a judge crash starts clean
		return tx.Where("submission_id = ?", subID).Delete(&models.SubmissionTestCase{}).Error
	})
	if err != nil {
		return err
	}
	PostSubmissionState(subID, "grading-begin", nil)
	return nil
}

func onTestCase(subID uint, pkt Packet) error {
	casesRaw, ok := pkt["cases"].([]interface{})
	if !ok {
		return fmt.Errorf("%w: cases must be an array", ErrBadPacket)
	}

	var rows []models.SubmissionTestCase
	var positions []int
	maxPos := 0
	for _, raw := range casesRaw {
		c, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		posVal, ok := c["position"].(float64)
		if !ok {
			continue
		}
		statusVal, ok := c["status"].(float64)
		if !ok {
			continue
		}
		pos := int(posVal)
		if pos > maxPos {
			maxPos = pos
		}
		t, m, p, total := num(c["time"]), num(c["memory"]), num(c["points"]), num(c["total-points"])
		rows = append(rows, models.SubmissionTestCase{
			SubmissionID:     subID,
			Case:             pos,
			Status:           caseStatus(int(statusVal)),
			Time:             &t,
			Memory:           &m,
			Points:           &p,
			Total:            &total,
			Feedback:         truncate(str(c["feedback"]), maxFeedback),
			ExtendedFeedback: str(c["extended-feedback"]),
			Output:           str(c["output"]),
		})
		positions = append(positions, pos)
	}
	if len(rows) == 0 {
		return nil
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// delete-then-insert per case: a replayed packet never duplicates rows
		if err := tx.Where("submission_id = ? AND `case` IN ?", subID, positions).Delete(&models.SubmissionTestCase{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
		return tx.Model(&models.Submission{}).Where("id = ?", subID).Update("current_testcase", maxPos+1).Error
	})
	if err != nil {
		return err
	}
	PostSubmissionState(subID, "test-case", map[string]interface{}{"id": maxPos})
	return nil
}

func onGradingEnd(subID uint) error {
	var cases []models.SubmissionTestCase
	if err := db.DB.Where("submission_id = ?", subID).Find(&cases).Error; err != nil {
		return err
	}

	var maxMem, sumTime, pts, total float64
	worstStatus := "AC"
	for _, c := range cases {
		if c.Time != nil {
			sumTime += *c.Time
		}
		if c.Memory != nil && *c.Memory > maxMem {
			maxMem = *c.Memory
		}
		if c.Points != nil {
			pts += *c.Points
		}
		if c.Total != nil {
			total += *c.Total
		}
		if statusOrder[c.Status] > statusOrder[worstStatus] {
			worstStatus = c.Status
		}
	}

	var sub models.Submission
	if err := db.DB.Preload("Problem").Preload("User").Where("id = ?", subID).First(&sub).Error; err != nil {
		return err
	}

	finalPts := 0.0
	if total > 0 {
		finalPts = (pts / total) * sub.Problem.Points
	}
	if !sub.Problem.Partial && finalPts != sub.Problem.Points {
		finalPts = 0
	}

	if err := db.DB.Model(&sub).Updates(map[string]interface{}{
		"status":      "D",
		"result":      worstStatus,
		"time":        sumTime,
		"memory":      maxMem,
		"points":      finalPts,
		"case_points": pts,
		"case_total":  total,
	}).Error; err != nil {
		return err
	}

	PostSubmissionState(subID, "grading-end", map[string]interface{}{
		"time": sumTime, "memory": maxMem, "points": pts, "total": total, "result": worstStatus,
	})
	PostGlobalSubmissionUpdate(subID, "grading-end", true)

	// v1 refreshes the submitter's stored points on every grading end
	// (judge_handler.on_grading_end -> Profile.calculate_points).
	recalcProfilePoints(&sub)

	// notification.user_id is keyed by auth_user.id; sub.UserID is a
	// judge_profile.id — pass the profile's owning auth user id.
	authUserID := sub.User.UserID
	go func() {
		_, _ = v2.CreateSubmissionResultNotification(authUserID, int(subID), sub.Problem.Code, sub.Problem.Name, worstStatus)
	}()

	// Recompute contest participation if this was a contest submission.
	var cSub models.ContestSubmission
	if err := db.DB.Preload("Participation.Contest").Where("submission_id = ?", subID).First(&cSub).Error; err == nil {
		cSub.Points = finalPts
		db.DB.Save(&cSub)
		cf := contest_format.GetFormat(cSub.Participation.Contest.FormatName, &cSub.Participation.Contest, cSub.Participation.Contest.FormatConfig)
		if err := cf.UpdateParticipation(&cSub.Participation); err != nil {
			log.Printf("bridge: sub %d: update contest participation: %v", subID, err)
		}
	}
	return nil
}

// recalcProfilePoints refreshes the stored points columns after a grading end,
// the way v1 does (Profile.calculate_points also refreshes the user's
// organizations). It runs unconditionally: hiding a problem or archiving the
// submission AFTER it was graded only drops the points when a rejudge
// recalculates, which is exactly v1's behaviour. The calculation itself skips
// non-public, organization-private and archived rows. Failures are logged and
// never returned — a points problem must not fail grading.
func recalcProfilePoints(sub *models.Submission) {
	if err := scoring.CalculateProfilePoints(db.DB, sub.UserID); err != nil {
		log.Printf("bridge: sub %d: recalculate profile %d points: %v", sub.ID, sub.UserID, err)
		return
	}

	var orgIDs []uint
	if err := db.DB.Table("judge_profile_organizations").
		Where("profile_id = ?", sub.UserID).Pluck("organization_id", &orgIDs).Error; err != nil {
		log.Printf("bridge: sub %d: list organizations of profile %d: %v", sub.ID, sub.UserID, err)
		return
	}
	for _, orgID := range orgIDs {
		if err := scoring.CalculateOrganizationPoints(db.DB, orgID); err != nil {
			log.Printf("bridge: sub %d: recalculate organization %d points: %v", sub.ID, orgID, err)
		}
	}
}

func onCompileError(subID uint, pkt Packet) error {
	logRaw := str(pkt["log"])
	if err := db.DB.Model(&models.Submission{}).Where("id = ?", subID).Updates(map[string]interface{}{
		"status": "CE", "result": "CE", "error": logRaw,
	}).Error; err != nil {
		return err
	}
	PostSubmissionState(subID, "compile-error", map[string]interface{}{"log": logRaw})
	PostGlobalSubmissionUpdate(subID, "compile-error", true)
	return nil
}

// onCompileMessage stores compiler warnings; grading continues (v1 parity).
func onCompileMessage(subID uint, pkt Packet) error {
	return db.DB.Model(&models.Submission{}).Where("id = ?", subID).Update("error", str(pkt["log"])).Error
}

func onInternalError(subID uint, pkt Packet) error {
	if err := db.DB.Model(&models.Submission{}).Where("id = ?", subID).Updates(map[string]interface{}{
		"status": "IE", "result": "IE", "error": str(pkt["message"]),
	}).Error; err != nil {
		return err
	}
	PostSubmissionState(subID, "internal-error", nil)
	PostGlobalSubmissionUpdate(subID, "internal-error", true)
	return nil
}

// onSubmissionTerminated: an aborted submission is AB, not IE (v1 parity).
func onSubmissionTerminated(subID uint) error {
	if err := db.DB.Model(&models.Submission{}).Where("id = ?", subID).Updates(map[string]interface{}{
		"status": "AB", "result": "AB",
	}).Error; err != nil {
		return err
	}
	PostSubmissionState(subID, "aborted-submission", nil)
	PostGlobalSubmissionUpdate(subID, "aborted-submission", true)
	return nil
}

// caseStatus maps the DMOJ result bitmask to a verdict code.
func caseStatus(n int) string {
	switch {
	case n&4 != 0:
		return "TLE"
	case n&8 != 0:
		return "MLE"
	case n&64 != 0:
		return "OLE"
	case n&2 != 0:
		return "RTE"
	case n&16 != 0:
		return "IR"
	case n&1 != 0:
		return "WA"
	case n&32 != 0:
		return "SC"
	default:
		return "AC"
	}
}

func num(v interface{}) float64 { f, _ := v.(float64); return f }
func str(v interface{}) string  { s, _ := v.(string); return s }

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
