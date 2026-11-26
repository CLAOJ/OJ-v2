package bridge

import (
	"encoding/json"
	"log"

	v2 "github.com/CLAOJ/claoj/api/v2"
	"github.com/CLAOJ/claoj/cache"
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

// PostGlobalSubmissionUpdate posts to the global 'submissions' channel.
func PostGlobalSubmissionUpdate(subID uint, state string, done bool) {
	PostEvent("submissions", map[string]interface{}{
		"type":  "update-submission",
		"state": state,
		"id":    subID,
	})
	if done {
		PostEvent("submissions", map[string]interface{}{
			"type":  "done-submission",
			"state": state,
			"id":    subID,
		})
	}
}
