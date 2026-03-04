package bridge

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/CLAOJ/claoj-go/cache"
	"github.com/CLAOJ/claoj-go/config"
)

// getIDSecret mimics Submission.get_id_secret() in Django.
func getIDSecret(id uint) string {
	secret := config.C.App.SecretKey
	h := sha1.New()
	h.Write([]byte(strconv.FormatUint(uint64(id), 10)))
	h.Write([]byte(secret))
	return hex.EncodeToString(h.Sum(nil))
}

// PostEvent posts a payload to the Redis pub/sub event daemon.
// Channel is usually sub_<secret> or contest_<id> or submissions.
func PostEvent(channel string, payload map[string]interface{}) {
	cache.Publish(channel, payload)
}

// PostSubmissionState posts the per-submission channel update.
func PostSubmissionState(subID uint, state string, extra map[string]interface{}) {
	secret := getIDSecret(subID)
	channel := fmt.Sprintf("sub_%s", secret)

	payload := map[string]interface{}{
		"type": state,
	}
	for k, v := range extra {
		payload[k] = v
	}

	PostEvent(channel, payload)
}

// PostGlobalSubmissionUpdate posts to the global 'submissions' channel.
func PostGlobalSubmissionUpdate(subID uint, state string, done bool) {
	// In the real system, this fetches problem, user, organizations, etc.
	// For now, we stub it out or just send minimal info.
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
