package bridge

import (
	"encoding/json"
	"log"

	v2 "github.com/CLAOJ/claoj/api/v2"
	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
)

// PostEvent publishes a payload for the websocket hub (events package).
// Channel is sub_<secret>, contest_<key> or submissions.
func PostEvent(channel string, payload map[string]interface{}) {
	b, err := json.Marshal(payload)
	if err == nil {
		err = cache.Publish(channel, b)
	}
	if err != nil {
		log.Printf("bridge: post event %s: %v", channel, err)
	}
}

// PostSubmissionState posts the per-submission channel update.
func PostSubmissionState(subID uint, state string, extra map[string]interface{}) {
	payload := map[string]interface{}{"type": state}
	for k, v := range extra {
		payload[k] = v
	}
	PostEvent(v2.SubmissionEventChannel(subID), payload)
}

// PostGlobalSubmissionUpdate posts to the global 'submissions' channel, marked
// with its audience for the hub: public only for a public, non-org-private
// problem outside any contest; otherwise just the submitter ("user", a
// judge_profile id) and view_all_submission holders get it. A contest
// submission also goes to contest_<key>, whose subscribers were checked
// for that contest's scoreboard.
func PostGlobalSubmissionUpdate(subID uint, state string, done bool) {
	var row struct {
		UserID                uint
		IsPublic              bool
		IsOrganizationPrivate bool
		ContestKey            *string
	}
	db.DB.Table("judge_submission s").
		Select("s.user_id, p.is_public, p.is_organization_private, c.`key` AS contest_key").
		Joins("JOIN judge_problem p ON p.id = s.problem_id").
		Joins("LEFT JOIN judge_contest c ON c.id = s.contest_object_id").
		Where("s.id = ?", subID).Scan(&row)
	public := row.IsPublic && !row.IsOrganizationPrivate && row.ContestKey == nil

	types := []string{"update-submission"}
	if done {
		types = append(types, "done-submission")
	}
	for _, t := range types {
		PostEvent("submissions", map[string]interface{}{
			"type": t, "state": state, "id": subID, "user": row.UserID, "public": public,
		})
	}
	if row.ContestKey != nil {
		PostEvent("contest_"+*row.ContestKey, map[string]interface{}{
			"type": "submission", "state": state, "id": subID,
		})
	}
}
