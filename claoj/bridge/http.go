package bridge

import (
	"crypto/subtle"
	"errors"
	"log"
	"net/http"

	"github.com/CLAOJ/claoj/db"
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

	if err := HandleEvent(judge, pkt); err != nil {
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
