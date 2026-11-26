package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

const TypeContestMoss = "contest_moss"

var safeProblemCode = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

var ErrMossAlreadyRunning = errors.New("moss: a run for this contest is already queued or running")

// RunContestMoss runs compare50 for every contest problem × language that has
// submissions, and records one plagiarism_result row per problem × language.
// Each live/spectating contestant's best-scored source is used. Reports and
// rows are replaced only after every check succeeded, so a failed run keeps
// the previous results.
func RunContestMoss(ctx context.Context, tx *gorm.DB, contestID uint, chk Checker) (int, error) {
	var problems []models.Problem
	if err := tx.Table("judge_problem").
		Select("judge_problem.*").
		Joins("JOIN judge_contestproblem cp ON cp.problem_id = judge_problem.id").
		Where("cp.contest_id = ?", contestID).
		Order("cp.`order` ASC").
		Find(&problems).Error; err != nil {
		return 0, err
	}

	if err := os.MkdirAll(PlagiarismDir, 0o755); err != nil {
		return 0, err
	}
	tmp := filepath.Join(PlagiarismDir, fmt.Sprintf(".tmp-%d-%d", contestID, time.Now().UnixNano()))
	defer os.RemoveAll(tmp)
	subsRoot := filepath.Join(tmp, "subs")
	reportRoot := filepath.Join(tmp, "report")

	now := time.Now()
	var rows []models.PlagiarismResult
	for _, p := range problems {
		if !safeProblemCode.MatchString(p.Code) {
			log.Printf("jobs: plagiarism: skipping problem with unsafe code %q", p.Code)
			continue
		}
		var subs []struct {
			Language string
			Username string
			Source   string
		}
		if err := tx.Table("judge_submission s").
			Select("l.common_name AS language, u.username AS username, src.source AS source").
			Joins("JOIN judge_submissionsource src ON src.submission_id = s.id").
			Joins("JOIN judge_language l ON l.id = s.language_id").
			Joins("JOIN judge_profile pr ON pr.id = s.user_id").
			Joins("JOIN auth_user u ON u.id = pr.user_id").
			Joins("JOIN judge_contestsubmission cs ON cs.submission_id = s.id").
			Joins("JOIN judge_contestparticipation part ON part.id = cs.participation_id").
			Where("s.contest_object_id = ? AND s.problem_id = ?", contestID, p.ID).
			Where("part.virtual IN ?", []int{0, -1}). // LIVE, SPECTATE
			Order("s.points DESC, s.id ASC").
			Scan(&subs).Error; err != nil {
			return 0, err
		}

		// subs is ordered by points, so the first entry per (language, user) is the best one.
		type entry struct{ user, source string }
		byLang := map[string][]entry{}
		var langs []string
		seen := map[[2]string]bool{}
		for _, s := range subs {
			k := [2]string{s.Language, s.Username}
			if seen[k] {
				continue
			}
			seen[k] = true
			if _, ok := byLang[s.Language]; !ok {
				langs = append(langs, s.Language)
			}
			byLang[s.Language] = append(byLang[s.Language], entry{s.Username, s.Source})
		}
		sort.Strings(langs)

		for _, lang := range langs {
			ents := byLang[lang]
			slug := LanguageSlug(lang)
			subsDir := filepath.Join(subsRoot, p.Code, slug)
			for _, e := range ents {
				dir := filepath.Join(subsDir, SubmissionFolder(e.user))
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return 0, err
				}
				if err := os.WriteFile(filepath.Join(dir, "main."+LanguageExt(lang)), []byte(e.source), 0o644); err != nil {
					return 0, err
				}
			}
			row := models.PlagiarismResult{ContestID: contestID, ProblemID: p.ID, Language: lang, SubmissionCount: uint(len(ents)), CreatedAt: now}
			if len(ents) >= 2 {
				if err := chk.Check(ctx, subsDir, "", filepath.Join(reportRoot, p.Code, slug)); err != nil {
					return 0, fmt.Errorf("plagiarism %s/%s: %w", p.Code, lang, err)
				}
				// compare50 exits 0 without creating -o when there are no similar pairs.
				if _, err := os.Stat(filepath.Join(reportRoot, p.Code, slug, "index.html")); err == nil {
					reportPath := p.Code + "/" + slug
					row.ReportPath = &reportPath
				}
			}
			rows = append(rows, row)
		}
	}

	// Swap the new report dir in. The old one is moved aside first because
	// Windows cannot rename onto an existing directory.
	final := ContestReportDir(contestID)
	var oldDir string
	if _, err := os.Stat(final); err == nil {
		oldDir = filepath.Join(PlagiarismDir, fmt.Sprintf(".old-%d-%d", contestID, time.Now().UnixNano()))
		if err := os.Rename(final, oldDir); err != nil {
			return 0, err
		}
	}
	movedIn := false
	rollback := func(err error) (int, error) {
		if movedIn {
			_ = os.Rename(final, reportRoot)
		}
		if oldDir != "" {
			_ = os.Rename(oldDir, final)
		}
		return 0, err
	}
	if _, err := os.Stat(reportRoot); err == nil {
		if err := os.Rename(reportRoot, final); err != nil {
			return rollback(err)
		}
		movedIn = true
	}

	err := tx.Transaction(func(t *gorm.DB) error {
		if err := t.Where("contest_id = ?", contestID).Delete(&models.PlagiarismResult{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return t.Create(&rows).Error
	})
	if err != nil {
		return rollback(err)
	}
	if oldDir != "" {
		_ = os.RemoveAll(oldDir)
	}
	return len(rows), nil
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

// HandleContestMossTask runs a queued contest plagiarism job.
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
	n, err := RunContestMoss(ctx, db.DB, p.ContestID, Compare50{})
	if err != nil {
		log.Printf("jobs: plagiarism check for contest %d failed: %v", p.ContestID, err)
		setMossRunError(p.ContestID, err.Error())
		return err
	}
	log.Printf("jobs: plagiarism check for contest %d wrote %d rows", p.ContestID, n)
	return nil
}
