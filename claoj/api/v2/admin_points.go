package v2

import (
	"log"
	"net/http"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"github.com/gin-gonic/gin"
)

// recalcBatchSize bounds how many profiles the backfill holds in memory.
const recalcBatchSize = 200

// AdminRecalculatePoints - POST /api/v2/admin/recalculate-points
// One-off backfill of the stored points columns: every profile first (in id
// batches), then every organization (which reads its members' points). Rows
// written before grading end refreshed them by itself are stale until this
// runs once. Superuser only; the admin audit middleware records the call.
func AdminRecalculatePoints(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}
	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiUserError("admin access required"))
		return
	}

	profiles, failed := 0, 0
	var lastID uint
	for {
		var ids []uint
		if err := db.DB.Model(&models.Profile{}).Where("id > ?", lastID).
			Order("id").Limit(recalcBatchSize).Pluck("id", &ids).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			if err := scoring.CalculateProfilePoints(db.DB, id); err != nil {
				log.Printf("recalculate-points: profile %d: %v", id, err)
				failed++
				continue
			}
			profiles++
		}
		lastID = ids[len(ids)-1]
	}

	var orgIDs []uint
	if err := db.DB.Model(&models.Organization{}).Pluck("id", &orgIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	organizations := 0
	for _, id := range orgIDs {
		if err := scoring.CalculateOrganizationPoints(db.DB, id); err != nil {
			log.Printf("recalculate-points: organization %d: %v", id, err)
			failed++
			continue
		}
		organizations++
	}

	c.JSON(http.StatusOK, gin.H{
		"profiles":      profiles,
		"organizations": organizations,
		"failed":        failed,
	})
}