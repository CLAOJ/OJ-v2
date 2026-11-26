package bridge

import (
	"crypto/subtle"
	"errors"
	"log"
	"net/http"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// EventHandler is POST /internal/judge/event: one DMOJ packet per request,
// sent by a claoj-judge agent with Basic auth (judge name / judge key).
// It lives outside /api/ so the public nginx/ingress never routes to it.
func EventHandler(c *gin.Context) {
	judge, ok := authenticateJudge(c.Request)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var pkt Packet
	if err := c.ShouldBindJSON(&pkt); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// The handshake (judge-runtimes) is the one packet that says "this judge is
	// here for the first time"; the status beat folds last_ip into its own
	// update so a beat stays a single UPDATE.
	if pkt.Name() == "judge-runtimes" {
		if err := RecordLastIP(judge.ID, c.ClientIP()); err != nil {
			log.Printf("bridge: %s from %q: record last_ip: %v", pkt.Name(), judge.Name, err)
		}
	}
	// Handshake and the 30 s status beat keep this judge's row online.
	if pkt.Name() == "judge-runtimes" || pkt.Name() == "judge-status" {
		if err := jobs.MarkJudgeAlive(judge.ID); err != nil {
			log.Printf("bridge: %s from %q: mark alive: %v", pkt.Name(), judge.Name, err)
		}
	}

	if err := HandleEvent(judge, pkt, c.ClientIP()); err != nil {
		if errors.Is(err, ErrBadPacket) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		log.Printf("bridge: %s from %q: %v", pkt.Name(), judge.Name, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}

func authenticateJudge(r *http.Request) (*models.Judge, bool) {
	name, key, ok := r.BasicAuth()
	if !ok || name == "" {
		return nil, false
	}
	var judge models.Judge
	if err := db.DB.Where("name = ?", name).First(&judge).Error; err != nil {
		return nil, false
	}
	if subtle.ConstantTimeCompare([]byte(judge.AuthKey), []byte(key)) != 1 || judge.IsBlocked {
		return nil, false
	}
	return &judge, true
}
