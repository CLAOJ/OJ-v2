package v2

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	judgeNameMaxLen    = 50  // judge_judge.name varchar(50)
	judgeAuthKeyMaxLen = 100 // judge_judge.auth_key varchar(100)
)

// generateJudgeAuthKey returns a 64-character URL-safe random key.
func generateJudgeAuthKey() (string, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// judgeNameTaken reports whether another judge already uses name.
func judgeNameTaken(name string, exceptID uint) (bool, error) {
	var count int64
	err := db.DB.Model(&models.Judge{}).Where("name = ? AND id <> ?", name, exceptID).Count(&count).Error
	return count > 0, err
}

// AdminJudgeCreate - POST /api/v2/admin/judges
// Registers a judge. The agent authenticates with name + auth_key; when no
// key is supplied one is generated and returned in the response.
func AdminJudgeCreate(c *gin.Context) {
	var input struct {
		Name        string `json:"name"`
		AuthKey     string `json:"auth_key"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" || utf8.RuneCountInString(name) > judgeNameMaxLen {
		c.JSON(http.StatusBadRequest, apiUserError("name is required and must be at most 50 characters"))
		return
	}
	if utf8.RuneCountInString(input.AuthKey) > judgeAuthKeyMaxLen {
		c.JSON(http.StatusBadRequest, apiUserError("auth_key must be at most 100 characters"))
		return
	}

	taken, err := judgeNameTaken(name, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if taken {
		c.JSON(http.StatusConflict, apiUserError("a judge with this name already exists"))
		return
	}

	key := input.AuthKey
	if key == "" {
		if key, err = generateJudgeAuthKey(); err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	judge := models.Judge{
		Name:        name,
		Created:     time.Now(),
		AuthKey:     key,
		Description: input.Description,
	}
	if err := db.DB.Create(&judge).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       judge.ID,
		"name":     judge.Name,
		"auth_key": judge.AuthKey,
	})
}

// AdminJudgeDelete - DELETE /api/v2/admin/judge/:id
// Hard-deletes a judge the way Django's cascade does: runtime versions and
// the problem/runtime join rows go with it, and submissions it judged keep
// existing with judged_on cleared (Submission.judged_on is SET_NULL).
func AdminJudgeDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid id"))
		return
	}

	var judge models.Judge
	if err := db.DB.First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("judge not found"))
		return
	}
	// Deleting a connected judge would strand whatever it is grading: its
	// next event fails authentication and the submission never finishes.
	if judge.Online {
		c.JSON(http.StatusConflict, apiUserError("judge is online; stop the judge agent and wait for it to go offline first"))
		return
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Submission{}).Where("judged_on_id = ?", judge.ID).
			Update("judged_on_id", gorm.Expr("NULL")).Error; err != nil {
			return err
		}
		if err := tx.Where("judge_id = ?", judge.ID).Delete(&models.RuntimeVersion{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&judge).Association("Problems").Clear(); err != nil {
			return err
		}
		if err := tx.Model(&judge).Association("Runtimes").Clear(); err != nil {
			return err
		}
		return tx.Delete(&judge).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
