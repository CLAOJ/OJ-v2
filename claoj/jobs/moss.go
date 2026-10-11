package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/moss"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

const TypeContestMoss = "contest_moss"

// MossLanguages is v1's ContestMoss.LANG_MAPPING: Language.common_name →
// MOSS language.
var MossLanguages = []struct{ CommonName, Moss string }{
	{"C", "c"},
	{"C++", "cc"},
	{"Java", "java"},
	{"Python", "python"},
	{"Pascal", "pascal"},
}

var (
	ErrMossNotConfigured  = errors.New(`moss: set misc config "moss_user_id" to your MOSS account id`)
	ErrMossAlreadyRunning = errors.New("moss: a run for this contest is already queued or running")
)

// MossSubmitter is the part of moss.Client the job needs (faked in tests).
type MossSubmitter interface {
	Submit(lang, comment string, files []moss.File) (string, error)
}

// MossUserID reads the MOSS account id from judge_miscconfig.
func MossUserID(tx *gorm.DB) (string, error) {
	var cfg models.MiscConfig
	err := tx.Where("`key` = ?", "moss_user_id").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrMossNotConfigured
	}
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(cfg.Value)
	if id == "" {
		return "", ErrMossNotConfigured
	}
	return id, nil
}

// RunContestMoss mirrors v1 judge.tasks.contest.run_moss: for every contest
// problem × MOSS language, send each live/spectating contestant's
// best-scored source and record the report. Rows are replaced only after
// every call succeeded, so a failed run keeps the previous results.
func RunContestMoss(tx *gorm.DB, contestID uint, sub MossSubmitter) (int, error) {
	var contest models.Contest
	if err := tx.First(&contest, contestID).Error; err != nil {
		return 0, err
	}
	var problems []models.Problem
	if err := tx.Table("judge_problem").
		Select("judge_problem.*").
		Joins("JOIN judge_contestproblem cp ON cp.problem_id = judge_problem.id").
		Where("cp.contest_id = ?", contestID).
		Order("cp.`order` ASC").
		Find(&problems).Error; err != nil {
		return 0, err
	}

	results := make([]models.ContestMoss, 0, len(problems)*len(MossLanguages))
	for _, p := range problems {
		for _, lang := range MossLanguages {
			row := models.ContestMoss{ContestID: contestID, ProblemID: p.ID, Language: lang.CommonName}

			var subs []struct {
				Username string
				Source   string
			}
			if err := tx.Table("judge_submission s").
				Select("u.username AS username, src.source AS source").
				Joins("JOIN judge_submissionsource src ON src.submission_id = s.id").
				Joins("JOIN judge_language l ON l.id = s.language_id").
				Joins("JOIN judge_profile pr ON pr.id = s.user_id").
				Joins("JOIN auth_user u ON u.id = pr.user_id").
				Joins("JOIN judge_contestsubmission cs ON cs.submission_id = s.id").
				Joins("JOIN judge_contestparticipation part ON part.id = cs.participation_id").
				Where("s.contest_object_id = ? AND s.problem_id = ? AND l.common_name = ?", contestID, p.ID, lang.CommonName).
				Where("part.virtual IN ?", []int{0, -1}). // LIVE, SPECTATE
				Order("s.points DESC").
				Scan(&subs).Error; err != nil {
				return 0, err
			}

			seen := make(map[string]bool, len(subs))
			files := make([]moss.File, 0, len(subs))
			for _, s := range subs {
				if seen[s.Username] {
					continue
				}
				seen[s.Username] = true
				files = append(files, moss.File{Name: s.Username, Content: []byte(s.Source)})
			}
			if len(files) > 0 {
				url, err := sub.Submit(lang.Moss, fmt.Sprintf("%s - %s", contest.Key, p.Code), files)
				if err != nil {
					return 0, fmt.Errorf("moss %s/%s: %w", p.Code, lang.CommonName, err)
				}
				row.URL = &url
				row.SubmissionCount = uint(len(files))
			}
			results = append(results, row)
		}
	}

	err := tx.Transaction(func(t *gorm.DB) error {
		if err := t.Where("contest_id = ?", contestID).Delete(&models.ContestMoss{}).Error; err != nil {
			return err
		}
		if len(results) == 0 {
			return nil
		}
		return t.Create(&results).Error
	})
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

type contestMossPayload struct {
	ContestID uint `json:"contest_id"`
}

func mossRunningKey(contestID uint) string { return fmt.Sprintf("claoj:moss:running:%d", contestID) }

func mossErrorKey(contestID uint) string { return fmt.Sprintf("claoj:moss:error:%d", contestID) }

// MossRunError returns the last failed run's error message ("" if none).
func MossRunError(contestID uint) string {
	if cache.Client == nil {
		return ""
	}
	s, err := cache.Client.Get(cache.Ctx, mossErrorKey(contestID)).Result()
	if err != nil {
		return ""
	}
	return s
}

// setMossRunError records a failed run so the admin list endpoint can show it.
func setMossRunError(contestID uint, msg string) {
	if cache.Client != nil {
		cache.Client.Set(cache.Ctx, mossErrorKey(contestID), msg, 24*time.Hour)
	}
}

// clearMossRunError forgets a previous failure (called when a new run starts).
func clearMossRunError(contestID uint) {
	if cache.Client != nil {
		cache.Client.Del(cache.Ctx, mossErrorKey(contestID))
	}
}

// IsContestMossRunning reports whether a run is queued or in progress.
func IsContestMossRunning(contestID uint) bool {
	if cache.Client == nil {
		return false
	}
	n, err := cache.Client.Exists(cache.Ctx, mossRunningKey(contestID)).Result()
	return err == nil && n > 0
}

// EnqueueContestMoss queues one run per contest; a second request while one
// is pending returns ErrMossAlreadyRunning.
func EnqueueContestMoss(contestID uint) error {
	payload, err := json.Marshal(contestMossPayload{ContestID: contestID})
	if err != nil {
		return err
	}
	taskID := fmt.Sprintf("contest_moss:%d", contestID)
	task := asynq.NewTask(TypeContestMoss, payload,
		asynq.MaxRetry(0),
		asynq.Timeout(30*time.Minute),
		asynq.TaskID(taskID),
	)
	// A run that failed stays archived/completed under the fixed TaskID, which
	// would make every later Enqueue collide (409 "already running") even
	// though IsContestMossRunning is false. Drop the stale entry first.
	if Inspector != nil {
		for _, q := range []string{"archived", "completed"} {
			_ = Inspector.DeleteTask(q, taskID) // ignore not-found
		}
	}
	// Set the running key BEFORE enqueueing so a very fast worker cannot
	// finish (and clear the key) before we set it; clear it again on failure.
	if cache.Client != nil {
		cache.Client.Set(cache.Ctx, mossRunningKey(contestID), "1", 30*time.Minute)
	}
	if _, err := Client.Enqueue(task, asynq.Queue("low")); err != nil {
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			// A real run is already queued or in progress, so the running key
			// belongs to that run — keep it.
			return ErrMossAlreadyRunning
		}
		if cache.Client != nil {
			cache.Client.Del(cache.Ctx, mossRunningKey(contestID))
		}
		return err
	}
	return nil
}

// HandleContestMossTask runs a queued contest MOSS job.
func HandleContestMossTask(ctx context.Context, t *asynq.Task) error {
	var p contestMossPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	defer func() {
		if cache.Client != nil {
			cache.Client.Del(cache.Ctx, mossRunningKey(p.ContestID))
		}
	}()
	clearMossRunError(p.ContestID)
	userID, err := MossUserID(db.DB)
	if err != nil {
		setMossRunError(p.ContestID, err.Error())
		return err
	}
	n, err := RunContestMoss(db.DB, p.ContestID, &moss.Client{UserID: userID, Show: 200})
	if err != nil {
		log.Printf("jobs: MOSS for contest %d failed: %v", p.ContestID, err)
		setMossRunError(p.ContestID, err.Error())
		return err
	}
	log.Printf("jobs: MOSS for contest %d wrote %d rows", p.ContestID, n)
	return nil
}
