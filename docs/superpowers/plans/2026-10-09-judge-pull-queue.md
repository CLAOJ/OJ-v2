# Judge Pull-Queue Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the v2 backend and v2 judges plain replicated services: backends enqueue submissions to a shared asynq queue, judge agents pull from it, grade with the unmodified v1 engine (tiervnoj), and POST results back over HTTP.

**Architecture:** `jobs.EnqueueJudgeSubmission` puts the DMOJ `submission-request` packet on asynq queue `judge` (task ID `sub-<id>`). `claoj-judge` is rewritten as a ~400-line Go agent: an asynq worker (concurrency 1) that drives one local tiervnoj child over a loopback DMOJ bridge and forwards every packet to `POST /internal/judge/event` on the backend. The backend's existing packet handlers become plain functions behind that route; the TCP bridge listener is deleted.

**Tech Stack:** Go 1.25, gin, GORM (MySQL prod / sqlite tests), asynq v0.26.0, go-redis v9, miniredis (tests), testify, Docker; tiervnoj (VNOJ fork of DMOJ judge, Python) as the grading engine.

**Spec:** `docs/superpowers/specs/2026-10-09-judge-pull-queue-design.md` — read it first.

## Global Constraints

- **No DB schema changes.** v1 Django owns the shared MySQL schema; production runs a DDL guard (`db/ddl_guard.go`) that panics on DDL. Only DML on existing tables/columns.
- **v1 untouched.** No edits to `OJ/`, v1 `bridged`, v1 judges, or v1 compose base files (`claoj-docker/claoj/docker-compose.yml`, `docker-compose.local.yml`). Only `docker-compose.v2.yml` changes in `claoj-docker`.
- **Grading engine = tiervnoj, unmodified.** No edits under `claoj-docker/judge/tiervnoj/`.
- asynq queue name `judge`; task type `judge_submission`; task ID `sub-<submission id>`; `MaxRetry(2)`; `Timeout(30m)`.
- Judge → backend auth: HTTP Basic auth, username = `judge_judge.name`, password = `judge_judge.auth_key`; blocked judges rejected.
- Route `POST /internal/judge/event` lives **outside** `/api/` (public nginx only proxies `/api/`).
- Agent and backend must use the **same** `REDIS_ADDR` and `REDIS_DB` (v2 uses DB `2` in compose).
- All judge replicas share one `judge_judge` row (`JUDGE_NAME`/`JUDGE_KEY`).
- v2 problem-data upload code (`api/v2/admin/problem_data*.go`) is **not** changed (deferred; see spec).
- Commit messages: Git Bash heredoc `git commit -F - <<'EOF' … EOF`, never PowerShell here-strings; end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`; verify with `git log -1 --format=%s`.

## Review Focus

1. **Rejudge while the submission is still being graded** — expected: no error, no duplicate task; the active grading finishes and its result lands. Test: Task 1 `TestEnqueueJudgeSubmission_ConflictWhileActiveIsNoop`.
2. **Judge feedback longer than the 50-char `feedback` column** (MySQL strict mode rejects it → 500 → agent retries forever) — expected: truncated, 204. Test: Task 3 `TestEvent_LongFeedbackIsTruncated`.
3. **Non-ASCII judge name** (prod row is `Vịt Toàn Năng`) — expected: authenticates. Tests: Task 3 `TestEvent_Auth` (uses that name), Task 6 `TestForwarder_SendsBasicAuthAndBody`.
4. **A packet for a different submission id arrives mid-grade** (late packet after an abort race) — expected: not forwarded. Test: Task 7 `TestGrade_HappyPathForwardsInOrderAndSkipsStale`.
5. **tiervnoj idle > 300 s** (its socket timeout) reconnects to the agent — expected: next submission still grades on the new connection. Tests: Task 5 `TestChild_NewestConnectionWins`; Task 10 step "idle reconnect".

---

## File Structure

Backend (`OJ-v2/claoj`):

| File | Responsibility |
|---|---|
| `jobs/judge.go` (rewrite) | Build `submission-request` packet; enqueue to queue `judge`; abort; `JudgeAborter` |
| `jobs/judge_status.go` (new) | Mirror asynq worker heartbeats into `judge_judge.online/load/start_time` |
| `jobs/server.go` (modify) | asynq client + `Inspector`; backend worker no longer serves judge work |
| `bridge/packet.go` (rewrite) | `Packet` type only (TCP framing removed) |
| `bridge/handler.go` (rewrite) | `HandleEvent`: apply one judge packet to the DB, replay-safe |
| `bridge/http.go` (new) | `EventHandler` gin handler + Basic-auth judge lookup |
| `bridge/events.go` (unchanged) | Redis pub/sub posting |
| `bridge/server.go`, `bridge/submit.go`, `bridge/bridge_test.go` | **delete** |
| `api/router.go` (modify) | register `/internal/judge/event` |
| `config/config.go` (modify) | drop `bridge.addr`; add `judge.name` (`JUDGE_NAME`) |
| `cache/cache.go` (modify) | exit if Redis does not answer PING |
| `main.go` (modify) | wiring |
| `integration/helpers.go` (modify) | mock enqueue signature |

Agent (`OJ-v2/claoj-judge`, package `main`, flat):

| File | Responsibility |
|---|---|
| `dmoj.go` | DMOJ framing + `Child` (loopback bridge the tiervnoj child dials) |
| `forward.go` | `Forwarder`: POST packets to backend with retry |
| `grade.go` | `Agent`: asynq handler, grading loop, abort/shutdown |
| `main.go` | env config, spawn tiervnoj, run asynq server |
| `Dockerfile`, `README.md`, `go.mod` | image `FROM claoj/judge-tiervnoj` + agent |
| everything else in `claoj-judge/` and the nested `claoj/judge/` module | **delete** (dead prototypes) |

Deployment: `CLAOJ/claoj-docker/claoj/docker-compose.v2.yml` (separate git repo), `OJ-v2/docs/deployment.md`, `OJ-v2/docs/development.md`, `OJ-v2/.github/workflows/ci-cd.yml`.

---

### Task 0: Branches and baseline

**Files:** none

- [x] **Step 1: Create branches**

```bash
cd /f/Coding/CLAOJ/OJ-v2 && git switch -c feat/judge-pull-queue
cd /f/Coding/CLAOJ/CLAOJ/claoj-docker && git switch -c feat/judge-pull-queue
```

`OJ-v2` has unrelated uncommitted `claoj-web` changes on `main`; leave them alone — never `git add -A`, always add explicit paths.

- [x] **Step 2: Record the backend test baseline**

```bash
cd /f/Coding/CLAOJ/OJ-v2/claoj && go vet ./... && go test ./... 2>&1 | tail -40
```

Write down any package that already fails. Later tasks must not add failures; pre-existing ones are not yours to fix.

---

### Task 1: Enqueue to the `judge` queue, abort via Inspector

**Files:**
- Modify (rewrite): `claoj/jobs/judge.go`
- Modify: `claoj/jobs/server.go`
- Modify: `claoj/main.go` (delete `jobs.SetBridge(judgeBridge)` line)
- Modify: `claoj/integration/helpers.go:108-113`
- Test: `claoj/jobs/judge_test.go` (new)

**Interfaces:**
- Produces:
  - `const jobs.QueueJudge = "judge"`, `const jobs.TypeJudgeSubmission = "judge_submission"`
  - `func jobs.BuildSubmissionRequest(subID uint) (map[string]interface{}, error)`
  - `func jobs.EnqueueJudgeSubmission(subID uint) error` (signature unchanged; 6 existing callers)
  - `func jobs.AbortJudgeSubmission(subID uint) error`
  - `type jobs.JudgeAborter struct{}` with `Abort(subID uint) error`
  - `var jobs.Inspector *asynq.Inspector` (set by `InitClient`)
  - `func jobs.SetMockEnqueue(fn func(*asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error))`
  - Removed: `jobs.SetBridge`, `jobs.BridgeRouter`, `jobs.HandleJudgeSubmissionTask`

- [x] **Step 1: Write the failing tests**

Create `claoj/jobs/judge_test.go`:

```go
package jobs

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupJudgeDB points db.DB at an in-memory sqlite with one C++17 submission (id 7) on problem "aplusb".
func setupJudgeDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.Language{}, &models.Problem{}, &models.Submission{},
		&models.SubmissionSource{}, &models.LanguageLimit{},
	))
	db.DB = database
	require.NoError(t, db.DB.Create(&models.Language{ID: 1, Key: "CPP17", Name: "C++17", CommonName: "C++", Ace: "c_cpp", Pygments: "cpp", Extension: "cpp"}).Error)
	require.NoError(t, db.DB.Create(&models.Problem{ID: 1, Code: "aplusb", Name: "A+B", GroupID: 1, TimeLimit: 1, MemoryLimit: 65536, Points: 100}).Error)
	require.NoError(t, db.DB.Create(&models.Submission{ID: 7, UserID: 1, ProblemID: 1, LanguageID: 1, Date: time.Now(), Status: "QU"}).Error)
	require.NoError(t, db.DB.Create(&models.SubmissionSource{SubmissionID: 7, Source: "int main(){}"}).Error)
}

// stubQueue replaces the asynq seams for one test and records calls.
type stubQueue struct {
	enqueues  [][]asynq.Option
	payloads  [][]byte
	enqErrs   []error // returned by successive enqueue calls; nil when exhausted
	state     asynq.TaskState
	infoErr   error
	deleted   []string
	cancelled []string
}

func useStubQueue(t *testing.T) *stubQueue {
	t.Helper()
	s := &stubQueue{}
	oldEnq, oldInfo, oldDel, oldCancel := enqueueFunc, getTaskInfo, deleteTask, cancelProcessing
	t.Cleanup(func() { enqueueFunc, getTaskInfo, deleteTask, cancelProcessing = oldEnq, oldInfo, oldDel, oldCancel })
	enqueueFunc = func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
		s.enqueues = append(s.enqueues, opts)
		s.payloads = append(s.payloads, task.Payload())
		var err error
		if len(s.enqErrs) > 0 {
			err, s.enqErrs = s.enqErrs[0], s.enqErrs[1:]
		}
		return &asynq.TaskInfo{ID: "sub-7"}, err
	}
	getTaskInfo = func(queue, id string) (*asynq.TaskInfo, error) {
		if s.infoErr != nil {
			return nil, s.infoErr
		}
		return &asynq.TaskInfo{ID: id, Queue: queue, State: s.state}, nil
	}
	deleteTask = func(queue, id string) error { s.deleted = append(s.deleted, queue+"/"+id); return nil }
	cancelProcessing = func(id string) error { s.cancelled = append(s.cancelled, id); return nil }
	return s
}

func optValue(opts []asynq.Option, typ asynq.OptionType) interface{} {
	for _, o := range opts {
		if o.Type() == typ {
			return o.Value()
		}
	}
	return nil
}

func TestBuildSubmissionRequest_UsesLanguageLimitOverride(t *testing.T) {
	setupJudgeDB(t)
	require.NoError(t, db.DB.Create(&models.LanguageLimit{ProblemID: 1, LanguageID: 1, TimeLimit: 2.5, MemoryLimit: 131072}).Error)

	pkt, err := BuildSubmissionRequest(7)
	require.NoError(t, err)
	assert.Equal(t, "submission-request", pkt["name"])
	assert.Equal(t, uint(7), pkt["submission-id"])
	assert.Equal(t, "aplusb", pkt["problem-id"])
	assert.Equal(t, "CPP17", pkt["language"])
	assert.Equal(t, "int main(){}", pkt["source"])
	assert.Equal(t, 2.5, pkt["time-limit"])
	assert.Equal(t, uint(131072), pkt["memory-limit"])
}

func TestBuildSubmissionRequest_MissingSubmission(t *testing.T) {
	setupJudgeDB(t)
	_, err := BuildSubmissionRequest(999)
	assert.Error(t, err)
}

func TestEnqueueJudgeSubmission_QueueTaskIDAndPayload(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)

	require.NoError(t, EnqueueJudgeSubmission(7))
	require.Len(t, s.enqueues, 1)
	assert.Equal(t, "judge", optValue(s.enqueues[0], asynq.QueueOpt))
	assert.Equal(t, "sub-7", optValue(s.enqueues[0], asynq.TaskIDOpt))
	assert.Equal(t, 2, optValue(s.enqueues[0], asynq.MaxRetryOpt))

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(s.payloads[0], &payload))
	assert.Equal(t, "aplusb", payload["problem-id"])
}

func TestEnqueueJudgeSubmission_ConflictWhilePendingIsNoop(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)
	s.enqErrs = []error{asynq.ErrTaskIDConflict}
	s.state = asynq.TaskStatePending

	require.NoError(t, EnqueueJudgeSubmission(7))
	assert.Len(t, s.enqueues, 1)
	assert.Empty(t, s.deleted)
}

func TestEnqueueJudgeSubmission_ConflictWhileActiveIsNoop(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)
	s.enqErrs = []error{asynq.ErrTaskIDConflict}
	s.state = asynq.TaskStateActive

	require.NoError(t, EnqueueJudgeSubmission(7))
	assert.Len(t, s.enqueues, 1)
	assert.Empty(t, s.deleted)
}

func TestEnqueueJudgeSubmission_ConflictWithArchivedRequeues(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)
	s.enqErrs = []error{asynq.ErrTaskIDConflict}
	s.state = asynq.TaskStateArchived

	require.NoError(t, EnqueueJudgeSubmission(7))
	assert.Equal(t, []string{"judge/sub-7"}, s.deleted)
	assert.Len(t, s.enqueues, 2)
}

func TestEnqueueJudgeSubmission_ConflictButTaskGoneRequeues(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)
	s.enqErrs = []error{asynq.ErrTaskIDConflict}
	s.infoErr = asynq.ErrTaskNotFound

	require.NoError(t, EnqueueJudgeSubmission(7))
	assert.Empty(t, s.deleted)
	assert.Len(t, s.enqueues, 2)
}

func TestAbortJudgeSubmission(t *testing.T) {
	t.Run("pending is deleted", func(t *testing.T) {
		s := useStubQueue(t)
		s.state = asynq.TaskStatePending
		require.NoError(t, AbortJudgeSubmission(7))
		assert.Equal(t, []string{"judge/sub-7"}, s.deleted)
		assert.Empty(t, s.cancelled)
	})
	t.Run("active is cancelled", func(t *testing.T) {
		s := useStubQueue(t)
		s.state = asynq.TaskStateActive
		require.NoError(t, AbortJudgeSubmission(7))
		assert.Equal(t, []string{"sub-7"}, s.cancelled)
		assert.Empty(t, s.deleted)
	})
	t.Run("unknown task is a no-op", func(t *testing.T) {
		s := useStubQueue(t)
		s.infoErr = asynq.ErrTaskNotFound
		require.NoError(t, AbortJudgeSubmission(7))
		assert.Empty(t, s.deleted)
		assert.Empty(t, s.cancelled)
	})
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && go test ./jobs/ -run 'Judge|Build|Abort' -v`
Expected: compile FAIL — `undefined: getTaskInfo`, `undefined: BuildSubmissionRequest`, etc.

- [x] **Step 3: Rewrite `claoj/jobs/judge.go`**

Replace the whole file with:

```go
package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/hibiken/asynq"
)

const (
	TypeJudgeSubmission = "judge_submission"
	// QueueJudge is consumed only by judge agents (claoj-judge), never by
	// the backend's own asynq worker.
	QueueJudge    = "judge"
	judgeMaxRetry = 2
)

// enqueueFunc and the inspector funcs below are seams overridden in tests.
var enqueueFunc = defaultEnqueue

var (
	getTaskInfo      = func(queue, id string) (*asynq.TaskInfo, error) { return Inspector.GetTaskInfo(queue, id) }
	deleteTask       = func(queue, id string) error { return Inspector.DeleteTask(queue, id) }
	cancelProcessing = func(id string) error { return Inspector.CancelProcessing(id) }
)

func defaultEnqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return Client.Enqueue(task, opts...)
}

// SetMockEnqueue sets a mock enqueue function for testing.
func SetMockEnqueue(fn func(*asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error)) {
	enqueueFunc = fn
}

func judgeTaskID(subID uint) string { return fmt.Sprintf("sub-%d", subID) }

// BuildSubmissionRequest builds the DMOJ "submission-request" packet a judge
// grades from. Ported from the old bridge.Server.Submit.
func BuildSubmissionRequest(subID uint) (map[string]interface{}, error) {
	var sub models.Submission
	if err := db.DB.Preload("Problem").Preload("Language").First(&sub, subID).Error; err != nil {
		return nil, fmt.Errorf("submission %d not found: %w", subID, err)
	}

	var source models.SubmissionSource
	if err := db.DB.Where("submission_id = ?", subID).First(&source).Error; err != nil && !sub.Language.FileOnly {
		return nil, fmt.Errorf("submission %d source missing: %w", subID, err)
	}

	timeLimit, memLimit := sub.Problem.TimeLimit, sub.Problem.MemoryLimit
	var override models.LanguageLimit
	if err := db.DB.Where("problem_id = ? AND language_id = ?", sub.ProblemID, sub.LanguageID).First(&override).Error; err == nil {
		timeLimit, memLimit = override.TimeLimit, uint(override.MemoryLimit)
	}

	src := source.Source
	if sub.Language.FileOnly {
		// e.g. /media/submissions/file.zip -> https://site/media/...
		src = config.C.App.SiteFullURL + src
	}

	return map[string]interface{}{
		"name":          "submission-request",
		"submission-id": subID,
		"problem-id":    sub.Problem.Code,
		"language":      sub.Language.Key,
		"source":        src,
		"time-limit":    timeLimit,
		"memory-limit":  memLimit,
		"short-circuit": sub.Problem.ShortCircuit,
		"meta": map[string]interface{}{
			"pretests-only":   sub.IsPretested,
			"in-contest":      0, // stub, as before
			"attempt-no":      1, // stub, as before
			"user":            sub.UserID,
			"file-only":       sub.Language.FileOnly,
			"file-size-limit": sub.Language.FileSizeLimit,
		},
	}, nil
}

// EnqueueJudgeSubmission queues a submission for any judge agent to grade.
// The task ID makes it idempotent: re-enqueueing a submission that is
// still queued or being graded is a no-op.
func EnqueueJudgeSubmission(subID uint) error {
	pkt, err := BuildSubmissionRequest(subID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(pkt)
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeJudgeSubmission, payload)
	id := judgeTaskID(subID)
	opts := []asynq.Option{
		asynq.Queue(QueueJudge),
		asynq.TaskID(id),
		asynq.MaxRetry(judgeMaxRetry),
		asynq.Timeout(30 * time.Minute),
	}

	_, err = enqueueFunc(task, opts...)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		info, ierr := getTaskInfo(QueueJudge, id)
		switch {
		case errors.Is(ierr, asynq.ErrTaskNotFound):
			// finished between the two calls: just enqueue again
		case ierr != nil:
			return ierr
		case info.State != asynq.TaskStateArchived && info.State != asynq.TaskStateCompleted:
			return nil // already queued or being graded
		default:
			if err := deleteTask(QueueJudge, id); err != nil {
				return err
			}
		}
		_, err = enqueueFunc(task, opts...)
	}
	if err != nil {
		return err
	}

	log.Printf("jobs: queued judge task %s", id)
	return nil
}

// AbortJudgeSubmission drops a queued judge task, or cancels the one a
// judge is grading (the agent then sends terminate-submission to tiervnoj).
func AbortJudgeSubmission(subID uint) error {
	id := judgeTaskID(subID)
	info, err := getTaskInfo(QueueJudge, id)
	if errors.Is(err, asynq.ErrTaskNotFound) || errors.Is(err, asynq.ErrQueueNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.State == asynq.TaskStateActive {
		return cancelProcessing(id)
	}
	return deleteTask(QueueJudge, id)
}

// JudgeAborter satisfies submission.BridgeServer / v2.SetBridgeServer.
type JudgeAborter struct{}

func (JudgeAborter) Abort(subID uint) error { return AbortJudgeSubmission(subID) }
```

- [x] **Step 4: Update `claoj/jobs/server.go`**

Add the Inspector next to `Client`, create it in `InitClient`, and stop serving judge work in `StartWorker`:

```go
var Client *asynq.Client

// Inspector queries and manages queued tasks (judge abort / rejudge).
var Inspector *asynq.Inspector
```

In `InitClient`, after `Client = asynq.NewClient(redisOpt)`:

```go
	Inspector = asynq.NewInspector(redisOpt)
```

In `StartWorker`, change the `Queues` map and remove the judge handler:

```go
			Queues: map[string]int{
				"default": 3,
				"low":     1,
			},
```

```go
	// Register all task handlers. Judge work (queue "judge") is consumed
	// by claoj-judge agents, not here.
	mux.HandleFunc(TypeRescoreProblem, HandleRescoreTask)
	mux.HandleFunc(TypeRateContest, HandleRateContestTask)
	mux.HandleFunc(TypeUserExport, HandleUserExportTask)
```

- [x] **Step 5: Remove the bridge injection from `claoj/main.go`**

Delete the line:

```go
	jobs.SetBridge(judgeBridge)
```

(`judgeBridge` is still used by `v2.SetBridgeServer` until Task 3.)

- [x] **Step 6: Update the integration mock in `claoj/integration/helpers.go`**

```go
	jobs.SetMockEnqueue(func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
		// Mock successful enqueue
		return &asynq.TaskInfo{ID: "mock-task-id"}, nil
	})
```

- [x] **Step 7: Run tests to verify they pass**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && go build ./... && go test ./jobs/ -v && go test ./integration/...`
Expected: jobs tests PASS; integration no new failures vs Task 0 baseline.

- [x] **Step 8: Commit**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add claoj/jobs/judge.go claoj/jobs/judge_test.go claoj/jobs/server.go claoj/main.go claoj/integration/helpers.go
git commit -F - <<'EOF'
feat(jobs): queue judge work on asynq "judge" for pull-based judges

EnqueueJudgeSubmission now builds the DMOJ submission-request packet and
enqueues it with task ID sub-<id> (idempotent rejudge). The backend worker
no longer consumes judge tasks; abort goes through the asynq Inspector.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 2: Judge online status from asynq heartbeats

**Files:**
- Create: `claoj/jobs/judge_status.go`
- Test: `claoj/jobs/judge_status_test.go`

**Interfaces:**
- Consumes: `jobs.Inspector`, `jobs.QueueJudge` (Task 1)
- Produces:
  - `func jobs.SyncJudgeStatus(judgeName string) error`
  - `func jobs.RunJudgeStatusLoop(ctx context.Context, judgeName string, every time.Duration)`

- [x] **Step 1: Write the failing test**

Create `claoj/jobs/judge_status_test.go`:

```go
package jobs

import (
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupJudgeRow(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.Judge{}))
	db.DB = database
	require.NoError(t, db.DB.Create(&models.Judge{Name: "Vịt Toàn Năng", AuthKey: "k", Created: time.Now()}).Error)
}

func useServers(t *testing.T, servers []*asynq.ServerInfo) {
	t.Helper()
	old := listServers
	t.Cleanup(func() { listServers = old })
	listServers = func() ([]*asynq.ServerInfo, error) { return servers, nil }
}

func loadJudge(t *testing.T) models.Judge {
	t.Helper()
	var j models.Judge
	require.NoError(t, db.DB.Where("name = ?", "Vịt Toàn Năng").First(&j).Error)
	return j
}

func TestSyncJudgeStatus_OnlineWhenAJudgeWorkerIsUp(t *testing.T) {
	setupJudgeRow(t)
	started := time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)
	useServers(t, []*asynq.ServerInfo{
		{Queues: map[string]int{"default": 3}, Concurrency: 10, Started: started.Add(-time.Hour)}, // a backend
		{Queues: map[string]int{"judge": 1}, Concurrency: 1, Started: started, ActiveWorkers: []*asynq.WorkerInfo{{}}},
		{Queues: map[string]int{"judge": 1}, Concurrency: 1, Started: started.Add(time.Minute)},
	})

	require.NoError(t, SyncJudgeStatus("Vịt Toàn Năng"))
	j := loadJudge(t)
	assert.True(t, j.Online)
	require.NotNil(t, j.Load)
	assert.InDelta(t, 0.5, *j.Load, 1e-9)
	require.NotNil(t, j.StartTime)
	assert.True(t, j.StartTime.Equal(started))
}

func TestSyncJudgeStatus_OfflineWhenNoJudgeWorker(t *testing.T) {
	setupJudgeRow(t)
	require.NoError(t, db.DB.Model(&models.Judge{}).Where("name = ?", "Vịt Toàn Năng").Update("online", true).Error)
	useServers(t, []*asynq.ServerInfo{{Queues: map[string]int{"default": 3}, Concurrency: 10}})

	require.NoError(t, SyncJudgeStatus("Vịt Toàn Năng"))
	assert.False(t, loadJudge(t).Online)
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && go test ./jobs/ -run SyncJudgeStatus -v`
Expected: compile FAIL — `undefined: listServers`, `undefined: SyncJudgeStatus`.

- [x] **Step 3: Implement `claoj/jobs/judge_status.go`**

```go
package jobs

import (
	"context"
	"log"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/hibiken/asynq"
)

// listServers is a seam overridden in tests.
var listServers = func() ([]*asynq.ServerInfo, error) { return Inspector.Servers() }

// SyncJudgeStatus mirrors asynq's worker heartbeats into the shared v2
// judge_judge row: online if any server consumes queue "judge", load =
// busy slots / total slots, start_time = oldest such server. Every backend
// replica may run it; the writes are idempotent.
func SyncJudgeStatus(judgeName string) error {
	servers, err := listServers()
	if err != nil {
		return err
	}

	var judges, busy, slots int
	var started time.Time
	for _, s := range servers {
		if _, ok := s.Queues[QueueJudge]; !ok {
			continue
		}
		judges++
		busy += len(s.ActiveWorkers)
		slots += s.Concurrency
		if started.IsZero() || s.Started.Before(started) {
			started = s.Started
		}
	}

	updates := map[string]interface{}{"online": judges > 0}
	if judges > 0 && slots > 0 {
		updates["load"] = float64(busy) / float64(slots)
		updates["start_time"] = started
	}
	return db.DB.Model(&models.Judge{}).Where("name = ?", judgeName).Updates(updates).Error
}

// RunJudgeStatusLoop calls SyncJudgeStatus now and then every interval
// until ctx is done.
func RunJudgeStatusLoop(ctx context.Context, judgeName string, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if err := SyncJudgeStatus(judgeName); err != nil {
			log.Printf("jobs: judge status sync: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && go test ./jobs/ -v`
Expected: PASS (all jobs tests).

- [x] **Step 5: Commit**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add claoj/jobs/judge_status.go claoj/jobs/judge_status_test.go
git commit -F - <<'EOF'
feat(jobs): derive v2 judge online/load from asynq worker heartbeats

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 3: Judge events over HTTP; delete the TCP bridge

**Files:**
- Modify (rewrite): `claoj/bridge/packet.go`, `claoj/bridge/handler.go`
- Create: `claoj/bridge/http.go`
- Delete: `claoj/bridge/server.go`, `claoj/bridge/submit.go`, `claoj/bridge/bridge_test.go`
- Modify: `claoj/api/router.go` (after the `/metrics` route, ~line 49)
- Modify: `claoj/main.go`
- Modify: `claoj/config/config.go` (remove `Bridge` config)
- Test: `claoj/bridge/handler_test.go` (new)

**Interfaces:**
- Consumes: `jobs.JudgeAborter` (Task 1)
- Produces:
  - `type bridge.Packet map[string]interface{}` with `Name() string`, `SubmissionID() (uint, bool)`
  - `var bridge.ErrBadPacket`
  - `func bridge.HandleEvent(judge *models.Judge, pkt Packet) error`
  - `func bridge.EventHandler(c *gin.Context)` — route `POST /internal/judge/event`; 204 ok, 400 malformed, 401 bad/blocked credentials, 500 DB error
  - Removed: `bridge.Server`, `bridge.NewServer`, `bridge.Manager`, `bridge.Handler`, `bridge.Connection`, `config.C.Bridge`

- [x] **Step 1: Write the failing tests**

Delete `claoj/bridge/bridge_test.go` (it only tests the TCP `Manager`/`Handler` being removed). Create `claoj/bridge/handler_test.go`:

```go
package bridge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const judgeName = "Vịt Toàn Năng" // prod name: non-ASCII on purpose

// setupEventDB: sqlite with judge "Vịt Toàn Năng"/"secret", a blocked judge,
// problem aplusb (100 pts, partial) and submission 7; Redis via miniredis.
func setupEventDB(t *testing.T) *models.Judge {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.Judge{}, &models.Problem{}, &models.Profile{},
		&models.Submission{}, &models.SubmissionTestCase{},
	))
	db.DB = database

	mr := miniredis.RunT(t)
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})

	judge := models.Judge{Name: judgeName, AuthKey: "secret", Created: time.Now()}
	require.NoError(t, db.DB.Create(&judge).Error)
	require.NoError(t, db.DB.Create(&models.Judge{Name: "blocked", AuthKey: "secret", Created: time.Now(), IsBlocked: true}).Error)
	require.NoError(t, db.DB.Create(&models.Problem{ID: 1, Code: "aplusb", Name: "A+B", GroupID: 1, TimeLimit: 1, MemoryLimit: 65536, Points: 100, Partial: true}).Error)
	require.NoError(t, db.DB.Create(&models.Profile{ID: 1, UserID: 1, Timezone: "UTC", LanguageID: 1}).Error)
	require.NoError(t, db.DB.Create(&models.Submission{ID: 7, UserID: 1, ProblemID: 1, LanguageID: 1, Date: time.Now(), Status: "QU"}).Error)
	return &judge
}

func newEventRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/internal/judge/event", EventHandler)
	return r
}

func post(t *testing.T, r *gin.Engine, user, pass string, pkt Packet) int {
	t.Helper()
	body, err := json.Marshal(pkt)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/internal/judge/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func loadSub(t *testing.T) models.Submission {
	t.Helper()
	var s models.Submission
	require.NoError(t, db.DB.First(&s, 7).Error)
	return s
}

func TestEvent_Auth(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	ack := Packet{"name": "submission-acknowledged", "submission-id": 7}

	assert.Equal(t, http.StatusUnauthorized, post(t, r, "", "", ack))
	assert.Equal(t, http.StatusUnauthorized, post(t, r, judgeName, "wrong", ack))
	assert.Equal(t, http.StatusUnauthorized, post(t, r, "blocked", "secret", ack))
	assert.Equal(t, http.StatusUnauthorized, post(t, r, "nobody", "secret", ack))
	assert.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", ack))
	assert.Equal(t, "P", loadSub(t).Status)
}

func TestEvent_MissingSubmissionIDIsBadRequest(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	assert.Equal(t, http.StatusBadRequest, post(t, r, judgeName, "secret", Packet{"name": "grading-begin"}))
}

func TestEvent_FullGradingFlowIsReplaySafe(t *testing.T) {
	judge := setupEventDB(t)
	r := newEventRouter()
	send := func(p Packet) { require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", p)) }

	send(Packet{"name": "grading-begin", "submission-id": 7, "pretested": false})
	s := loadSub(t)
	assert.Equal(t, "G", s.Status)
	require.NotNil(t, s.JudgedOnID)
	assert.Equal(t, judge.ID, *s.JudgedOnID)

	cases := Packet{"name": "test-case-status", "submission-id": 7, "cases": []interface{}{
		map[string]interface{}{"position": 1, "status": 0, "time": 0.1, "memory": 1024, "points": 10, "total-points": 10, "feedback": "", "output": ""},
		map[string]interface{}{"position": 2, "status": 1, "time": 0.2, "memory": 2048, "points": 0, "total-points": 10, "feedback": "wrong", "output": ""},
	}}
	send(cases)
	send(cases) // replayed after a lost response: must not duplicate rows

	var n int64
	require.NoError(t, db.DB.Model(&models.SubmissionTestCase{}).Where("submission_id = ?", 7).Count(&n).Error)
	assert.Equal(t, int64(2), n)

	send(Packet{"name": "grading-end", "submission-id": 7})
	s = loadSub(t)
	assert.Equal(t, "D", s.Status)
	require.NotNil(t, s.Result)
	assert.Equal(t, "WA", *s.Result)
	require.NotNil(t, s.Points)
	assert.InDelta(t, 50.0, *s.Points, 1e-9) // 10/20 of 100, partial
	assert.InDelta(t, 0.3, *s.Time, 1e-9)
}

func TestEvent_CompileMessageKeepsGrading(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "grading-begin", "submission-id": 7}))
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "compile-message", "submission-id": 7, "log": "warning: unused"}))
	s := loadSub(t)
	assert.Equal(t, "G", s.Status)
	require.NotNil(t, s.Error)
	assert.Equal(t, "warning: unused", *s.Error)
}

func TestEvent_CompileErrorAndInternalError(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "compile-error", "submission-id": 7, "log": "boom"}))
	s := loadSub(t)
	assert.Equal(t, "CE", s.Status)
	assert.Equal(t, "CE", *s.Result)

	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "internal-error", "submission-id": 7, "message": "judge crashed"}))
	s = loadSub(t)
	assert.Equal(t, "IE", s.Status)
	assert.Equal(t, "IE", *s.Result)
}

func TestEvent_SubmissionTerminatedIsAborted(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "submission-terminated", "submission-id": 7}))
	s := loadSub(t)
	assert.Equal(t, "AB", s.Status)
	assert.Equal(t, "AB", *s.Result)
}

func TestEvent_LongFeedbackIsTruncated(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	long := strings.Repeat("ố", 80) // multibyte: truncate by runes, not bytes
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "test-case-status", "submission-id": 7, "cases": []interface{}{
		map[string]interface{}{"position": 1, "status": 1, "feedback": long},
	}}))
	var tc models.SubmissionTestCase
	require.NoError(t, db.DB.Where("submission_id = ?", 7).First(&tc).Error)
	assert.Equal(t, 50, len([]rune(tc.Feedback)))
}

func TestEvent_UnknownPacketIsIgnored(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	assert.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "batch-begin", "submission-id": 7}))
	assert.Equal(t, "QU", loadSub(t).Status)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && go test ./bridge/ -v`
Expected: compile FAIL — `undefined: EventHandler`.

- [x] **Step 3: Delete the TCP bridge files**

```bash
cd /f/Coding/CLAOJ/OJ-v2 && git rm claoj/bridge/server.go claoj/bridge/submit.go claoj/bridge/bridge_test.go
```

- [x] **Step 4: Rewrite `claoj/bridge/packet.go`**

```go
package bridge

// Packet is one DMOJ judge protocol message: a JSON object with a "name"
// field. Judge agents POST them to /internal/judge/event unchanged. JSON
// numbers decode as float64.
type Packet map[string]interface{}

// Name returns the "name" field of the packet (the event type).
func (p Packet) Name() string {
	if n, ok := p["name"].(string); ok {
		return n
	}
	return ""
}

// SubmissionID returns the packet's "submission-id".
func (p Packet) SubmissionID() (uint, bool) {
	v, ok := p["submission-id"].(float64)
	if !ok || v <= 0 {
		return 0, false
	}
	return uint(v), true
}
```

- [x] **Step 5: Rewrite `claoj/bridge/handler.go`**

```go
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
	"gorm.io/gorm"
)

// ErrBadPacket marks a packet the judge sent malformed (HTTP 400).
var ErrBadPacket = errors.New("bad judge packet")

// maxFeedback is the width of judge_submissiontestcase.feedback.
const maxFeedback = 50

// statusOrder ranks case verdicts; the worst becomes the submission result.
var statusOrder = map[string]int{"AC": 0, "SC": 1, "WA": 2, "MLE": 3, "TLE": 4, "IR": 5, "RTE": 6, "OLE": 7}

// HandleEvent applies one packet a judge agent forwarded from tiervnoj.
// judge is the authenticated judge_judge row. Every branch must be safe to
// replay: the agent re-POSTs a packet whose response it never received.
func HandleEvent(judge *models.Judge, pkt Packet) error {
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
```

- [x] **Step 6: Create `claoj/bridge/http.go`**

```go
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
```

- [x] **Step 7: Register the route in `claoj/api/router.go`**

Add `"github.com/CLAOJ/claoj/bridge"` to the imports, and right after the `/metrics` route:

```go
	// Judge agents POST grading packets here. Deliberately outside /api/:
	// the public nginx/ingress only forwards /api/ to the backend.
	r.POST("/internal/judge/event", bridge.EventHandler)
```

- [x] **Step 8: Update `claoj/main.go`**

Delete the whole block:

```go
	// 4. Start the Judge Bridge TCP server
	judgeBridge := bridge.NewServer()
	go func() {
		if err := judgeBridge.Start(); err != nil {
			log.Fatalf("bridge: %v", err)
		}
	}()
```

Replace `v2.SetBridgeServer(judgeBridge)` with:

```go
	v2.SetBridgeServer(jobs.JudgeAborter{})
```

Remove the `"github.com/CLAOJ/claoj/bridge"` import.

- [x] **Step 9: Remove the bridge config from `claoj/config/config.go`**

Delete the `Bridge BridgeConfig \`mapstructure:"bridge"\`` field, the `BridgeConfig` type, `v.SetDefault("bridge.addr", ":9999")`, and `v.BindEnv("bridge.addr", "BRIDGE_ADDR", "CLAOJ_BRIDGE_ADDR")`. Then confirm nothing else uses it:

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && grep -rn "C.Bridge\|BridgeConfig\|bridge.addr" --include=*.go .`
Expected: no output.

- [x] **Step 10: Run tests to verify they pass**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && go vet ./... && go test ./bridge/ -v && go test ./...`
Expected: bridge tests PASS; no new failures vs Task 0 baseline.

- [x] **Step 11: Commit**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add claoj/bridge/packet.go claoj/bridge/handler.go claoj/bridge/http.go claoj/bridge/handler_test.go claoj/api/router.go claoj/main.go claoj/config/config.go
git commit -F - <<'EOF'
feat(bridge): accept judge packets over HTTP, drop the TCP bridge

POST /internal/judge/event (Basic auth = judge_judge name/key) applies one
DMOJ packet. Handlers are replay-safe, set judged_on_id, map
submission-terminated to AB, keep compile-message non-terminal and
truncate feedback to the column width.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 4: `JUDGE_NAME` config, status loop, mandatory Redis

**Files:**
- Modify: `claoj/config/config.go`
- Modify: `claoj/cache/cache.go:23-27`
- Modify: `claoj/main.go`
- Test: `claoj/config/judge_config_test.go` (new)

**Interfaces:**
- Consumes: `jobs.RunJudgeStatusLoop` (Task 2), `jobs.JudgeAborter` (Task 1)
- Produces: `config.C.Judge.Name string` (env `JUDGE_NAME` / `CLAOJ_JUDGE_NAME`)

- [x] **Step 1: Write the failing test**

Create `claoj/config/judge_config_test.go`:

```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_JudgeName(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-only-secret") // Load() log.Fatal()s on a missing/default key
	t.Setenv("JUDGE_NAME", "Vịt Toàn Năng")
	Load()
	assert.Equal(t, "Vịt Toàn Năng", C.Judge.Name)
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && go test ./config/ -run JudgeName -v`
Expected: compile FAIL — `C.Judge undefined`.

- [x] **Step 3: Add the config**

In `claoj/config/config.go`, add the field to `Config` (after `Server`):

```go
	Judge    JudgeConfig    `mapstructure:"judge"`
```

Add the type next to `ServerConfig`:

```go
type JudgeConfig struct {
	// Name of the judge_judge row all v2 judge agents share; its online/load
	// columns are kept in sync from asynq heartbeats. Empty = don't sync.
	Name string `mapstructure:"name"`
}
```

Add with the other defaults and env bindings:

```go
	v.SetDefault("judge.name", "")
```

```go
	v.BindEnv("judge.name", "JUDGE_NAME", "CLAOJ_JUDGE_NAME")
```

- [x] **Step 4: Make Redis mandatory in `claoj/cache/cache.go`**

First check no test calls `cache.Connect`:

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && grep -rn "cache.Connect()" --include=*_test.go .`
Expected: no output.

Replace the ping branch in `Connect`:

```go
	if _, err := Client.Ping(Ctx).Result(); err != nil {
		// The judge queue, live events, sessions and the audit log all live
		// in Redis; with several replicas there is no safe fallback.
		log.Fatalf("cache: Redis at %s unreachable: %v", cfg.Addr, err)
	}
	log.Printf("cache: connected to Redis at %s", cfg.Addr)
```

- [x] **Step 5: Rewrite `claoj/main.go`**

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/CLAOJ/claoj/api"
	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auditlog"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/events"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()
	gin.SetMode(config.C.Server.Mode)

	db.Connect()
	cache.Connect() // exits if Redis is unreachable

	// Session, one-time-token and audit-log state live in Redis so every
	// replica shares them and the MySQL schema stays Django-owned.
	authHandlers.RefreshStore = tokenstore.NewRedisStore(cache.Client)
	authHandlers.OneTimeTokens = tokenstore.NewRedisOneTime(cache.Client)
	auditlog.Default = auditlog.NewRedisStore(cache.Client)

	// Background jobs. Submissions go to asynq queue "judge", consumed only
	// by claoj-judge agents, which POST results to /internal/judge/event.
	jobs.InitClient()
	go jobs.StartWorker()
	if config.C.Judge.Name != "" {
		go jobs.RunJudgeStatusLoop(context.Background(), config.C.Judge.Name, 30*time.Second)
	} else {
		log.Printf("warning: JUDGE_NAME unset; judge online status is not synced")
	}

	events.InitHub()
	v2.SetBridgeServer(jobs.JudgeAborter{})

	r := api.NewRouter()
	addr := ":" + config.C.Server.Port
	log.Printf("claoj-go HTTP API starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
```

- [x] **Step 6: Run tests to verify they pass**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj && go vet ./... && go test ./...`
Expected: PASS; no new failures vs Task 0 baseline.

- [x] **Step 7: Commit**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add claoj/config/config.go claoj/config/judge_config_test.go claoj/cache/cache.go claoj/main.go
git commit -F - <<'EOF'
feat(backend): JUDGE_NAME status sync; refuse to start without Redis

The in-memory fallbacks in main.go were dead code (cache.Client is never
nil) and unsafe with several replicas.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 5: Agent — clean slate, DMOJ framing and loopback `Child`

**Files:**
- Delete: `claoj-judge/config/`, `claoj-judge/core/`, `claoj-judge/executors/`, `claoj-judge/protocol/`, `claoj-judge/INTEGRATION.md`, `claoj-judge/MIGRATION_SUMMARY.md`, `claoj-judge/Makefile`, `claoj-judge/docker-compose.yml`, `claoj-judge/judge.example.yml`, `claoj-judge/setup.sh`, `claoj-judge/Dockerfile`, `claoj-judge/README.md`, and the nested dead module `claoj/judge/`
- Modify: `claoj-judge/go.mod`
- Create: `claoj-judge/dmoj.go`
- Test: `claoj-judge/dmoj_test.go`

**Interfaces:**
- Produces (package `main`):
  - `type Packet map[string]interface{}` with `Name() string`, `SubmissionID() int64`
  - `func readPacket(r io.Reader) (Packet, error)`, `func writePacket(w io.Writer, p Packet) error`
  - `func NewChild(key string) *Child`; `(*Child).Serve(ln net.Listener) error`; `Send(p Packet) error`; `Events() <-chan Packet`; `Ready() bool`; `KeepAlive(every time.Duration)`
  - `var errNoChild`
  - test helpers in `dmoj_test.go`: `startChild(t, key) (*Child, string)`, `fakeJudge(t, addr, key) net.Conn`

- [x] **Step 1: Delete the dead prototypes and reset the module**

```bash
cd /f/Coding/CLAOJ/OJ-v2
grep -rn "claoj-go/judge\|claoj-judge/" --include=*.go claoj | grep -v "^claoj/judge/"   # expect no output
git rm -r -q claoj-judge/config claoj-judge/core claoj-judge/executors claoj-judge/protocol \
  claoj-judge/INTEGRATION.md claoj-judge/MIGRATION_SUMMARY.md claoj-judge/Makefile \
  claoj-judge/docker-compose.yml claoj-judge/judge.example.yml claoj-judge/setup.sh \
  claoj-judge/Dockerfile claoj-judge/README.md claoj/judge
```

Replace `claoj-judge/go.mod` with:

```
module github.com/CLAOJ/claoj-judge

go 1.25.0
```

```bash
cd /f/Coding/CLAOJ/OJ-v2/claoj-judge && rm -f go.sum && go get github.com/hibiken/asynq@v0.26.0 github.com/stretchr/testify@v1.11.1
```

- [x] **Step 2: Write the failing tests**

Create `claoj-judge/dmoj_test.go`:

```go
package main

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startChild runs a Child on a random loopback port.
func startChild(t *testing.T, key string) (*Child, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	c := NewChild(key)
	go c.Serve(ln)
	t.Cleanup(func() { ln.Close() })
	return c, ln.Addr().String()
}

// fakeJudge dials the Child the way tiervnoj does and completes the handshake.
func fakeJudge(t *testing.T, addr, key string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, writePacket(conn, Packet{
		"name": "handshake", "id": "Vịt Toàn Năng", "key": key,
		"problems": []interface{}{}, "executors": map[string]interface{}{},
	}))
	resp, err := readPacket(conn)
	require.NoError(t, err)
	require.Equal(t, "handshake-success", resp.Name())
	return conn
}

func nextEvent(t *testing.T, c *Child) Packet {
	t.Helper()
	select {
	case p := <-c.Events():
		return p
	case <-time.After(2 * time.Second):
		t.Fatal("no event from child")
		return nil
	}
}

func TestFraming_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	in := Packet{"name": "grading-end", "submission-id": float64(7), "note": "Vịt"}
	require.NoError(t, writePacket(&buf, in))
	out, err := readPacket(&buf)
	require.NoError(t, err)
	assert.Equal(t, in, out)
	assert.Equal(t, int64(7), out.SubmissionID())
}

func TestChild_RejectsBadKey(t *testing.T) {
	c, addr := startChild(t, "secret")
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, writePacket(conn, Packet{"name": "handshake", "id": "x", "key": "wrong"}))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = readPacket(conn)
	assert.Error(t, err) // closed without handshake-success
	assert.False(t, c.Ready())
}

func TestChild_ForwardsEventsAndFiltersNoise(t *testing.T) {
	c, addr := startChild(t, "secret")
	conn := fakeJudge(t, addr, "secret")
	assert.True(t, c.Ready())

	require.NoError(t, writePacket(conn, Packet{"name": "ping-response", "when": 1.0}))
	require.NoError(t, writePacket(conn, Packet{"name": "supported-problems", "problems": []interface{}{}}))
	require.NoError(t, writePacket(conn, Packet{"name": "grading-begin", "submission-id": 7}))

	assert.Equal(t, "grading-begin", nextEvent(t, c).Name())
}

func TestChild_SendReachesJudge(t *testing.T) {
	c, addr := startChild(t, "secret")
	conn := fakeJudge(t, addr, "secret")

	require.NoError(t, c.Send(Packet{"name": "terminate-submission"}))
	p, err := readPacket(conn)
	require.NoError(t, err)
	assert.Equal(t, "terminate-submission", p.Name())
}

func TestChild_NewestConnectionWins(t *testing.T) {
	c, addr := startChild(t, "secret")
	old := fakeJudge(t, addr, "secret")
	fresh := fakeJudge(t, addr, "secret") // tiervnoj reconnected after its 300 s timeout
	_ = old

	require.NoError(t, c.Send(Packet{"name": "ping", "when": 1.0}))
	fresh.SetReadDeadline(time.Now().Add(2 * time.Second))
	p, err := readPacket(fresh)
	require.NoError(t, err)
	assert.Equal(t, "ping", p.Name())
}

func TestChild_SendWithoutJudge(t *testing.T) {
	c := NewChild("secret")
	assert.ErrorIs(t, c.Send(Packet{"name": "ping"}), errNoChild)
}
```

- [x] **Step 3: Run tests to verify they fail**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj-judge && go test -timeout 60s ./... -v`
Expected: compile FAIL — `undefined: NewChild`, `undefined: writePacket`.

- [x] **Step 4: Implement `claoj-judge/dmoj.go`**

```go
package main

import (
	"bytes"
	"compress/zlib"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const maxPacket = 64 << 20

// Packet is one DMOJ judge protocol message (JSON object with a "name").
type Packet map[string]interface{}

func (p Packet) Name() string { s, _ := p["name"].(string); return s }

// SubmissionID reads "submission-id" (a float64 after JSON decoding).
func (p Packet) SubmissionID() int64 {
	switch v := p["submission-id"].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

// readPacket reads one frame: 4-byte big-endian length + zlib(JSON).
func readPacket(r io.Reader) (Packet, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	if size > maxPacket {
		return nil, fmt.Errorf("packet too large: %d bytes", size)
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	zr, err := zlib.NewReader(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	var p Packet
	if err := json.NewDecoder(zr).Decode(&p); err != nil {
		return nil, err
	}
	return p, nil
}

// writePacket writes one frame in a single Write so frames never interleave.
func writePacket(w io.Writer, p Packet) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	if _, err := zw.Write(raw); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	frame := make([]byte, 4+z.Len())
	binary.BigEndian.PutUint32(frame, uint32(z.Len()))
	copy(frame[4:], z.Bytes())
	_, err = w.Write(frame)
	return err
}

var errNoChild = errors.New("tiervnoj is not connected")

// Child is the agent's loopback DMOJ bridge: the local tiervnoj process
// dials it exactly as it would dial the site's bridged. tiervnoj reconnects
// on its own (e.g. after its 300 s read timeout), so the newest
// authenticated connection always wins.
type Child struct {
	key    string
	events chan Packet

	mu   sync.Mutex // guards conn and serialises writes
	conn net.Conn
}

func NewChild(key string) *Child {
	return &Child{key: key, events: make(chan Packet, 1024)}
}

// Serve accepts tiervnoj connections until ln is closed.
func (c *Child) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go c.handle(conn)
	}
}

func (c *Child) handle(conn net.Conn) {
	defer conn.Close()

	hello, err := readPacket(conn)
	if err != nil || hello.Name() != "handshake" {
		log.Printf("child: bad handshake: %v", err)
		return
	}
	key, _ := hello["key"].(string)
	if subtle.ConstantTimeCompare([]byte(key), []byte(c.key)) != 1 {
		log.Printf("child: handshake rejected: wrong key")
		return
	}

	c.mu.Lock()
	err = writePacket(conn, Packet{"name": "handshake-success"})
	if err == nil {
		c.conn = conn
	}
	c.mu.Unlock()
	if err != nil {
		return
	}
	log.Printf("child: tiervnoj connected")

	for {
		p, err := readPacket(conn)
		if err != nil {
			c.mu.Lock()
			if c.conn == conn {
				c.conn = nil
			}
			c.mu.Unlock()
			log.Printf("child: connection closed: %v", err)
			return
		}
		switch p.Name() {
		case "ping-response", "supported-problems", "current-submission-id", "handshake":
			// bookkeeping the backend never needed
		default:
			c.events <- p
		}
	}
}

// Send writes a packet to the connected tiervnoj.
func (c *Child) Send(p Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return errNoChild
	}
	return writePacket(c.conn, p)
}

func (c *Child) Events() <-chan Packet { return c.events }

func (c *Child) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// KeepAlive pings tiervnoj so its 300 s socket timeout never fires while
// the queue is idle.
func (c *Child) KeepAlive(every time.Duration) {
	for range time.Tick(every) {
		_ = c.Send(Packet{"name": "ping", "when": float64(time.Now().UnixNano()) / 1e9})
	}
}
```

- [x] **Step 5: Run tests to verify they pass**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj-judge && go mod tidy && go vet ./... && go test -timeout 60s ./... -v`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add claoj-judge/go.mod claoj-judge/go.sum claoj-judge/dmoj.go claoj-judge/dmoj_test.go
git commit -F - <<'EOF'
refactor(judge): replace the prototype Go judge with a DMOJ loopback bridge

The native executors/sandbox could not grade the existing problem set
(no archive, file_io, checkers, batches, sandbox). The agent now drives
the v1 tiervnoj engine; this adds framing and the loopback connection.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 6: Agent — `Forwarder`

**Files:**
- Create: `claoj-judge/forward.go`
- Test: `claoj-judge/forward_test.go`

**Interfaces:**
- Consumes: `Packet` (Task 5)
- Produces: `type Forwarder struct{ URL, Name, Key string; Client *http.Client; MaxWait, Backoff time.Duration }`; `(*Forwarder).Post(p Packet) error`; `var errUnauthorized`

- [x] **Step 1: Write the failing tests**

Create `claoj-judge/forward_test.go`:

```go
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newForwarder(url string) *Forwarder {
	return &Forwarder{URL: url, Name: "Vịt Toàn Năng", Key: "secret", Client: http.DefaultClient, MaxWait: time.Second, Backoff: time.Millisecond}
}

func TestForwarder_SendsBasicAuthAndBody(t *testing.T) {
	var got Packet
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "Vịt Toàn Năng", user)
		assert.Equal(t, "secret", pass)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	require.NoError(t, newForwarder(srv.URL).Post(Packet{"name": "grading-end", "submission-id": float64(7)}))
	assert.Equal(t, "grading-end", got.Name())
}

func TestForwarder_RetriesServerErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	require.NoError(t, newForwarder(srv.URL).Post(Packet{"name": "grading-begin"}))
	assert.Equal(t, int32(3), calls.Load())
}

func TestForwarder_UnauthorizedIsFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	assert.ErrorIs(t, newForwarder(srv.URL).Post(Packet{"name": "grading-begin"}), errUnauthorized)
}

func TestForwarder_BadRequestIsDroppedNotRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	assert.NoError(t, newForwarder(srv.URL).Post(Packet{"name": "grading-begin"}))
	assert.Equal(t, int32(1), calls.Load())
}

func TestForwarder_GivesUpAfterMaxWait(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	f := newForwarder(srv.URL)
	f.MaxWait = 30 * time.Millisecond
	start := time.Now()
	assert.Error(t, f.Post(Packet{"name": "grading-begin"}))
	assert.Less(t, time.Since(start), time.Second)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj-judge && go test -timeout 60s -run Forwarder -v`
Expected: compile FAIL — `undefined: Forwarder`.

- [x] **Step 3: Implement `claoj-judge/forward.go`**

```go
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var errUnauthorized = errors.New("backend rejected judge credentials")

// Forwarder POSTs judge packets to the backend's /internal/judge/event.
// Callers send one packet at a time and wait, which keeps per-submission
// order even when the backend Service load-balances across replicas.
type Forwarder struct {
	URL     string // e.g. http://backend:8081/internal/judge/event
	Name    string // judge_judge.name
	Key     string // judge_judge.auth_key
	Client  *http.Client
	MaxWait time.Duration // give up after this long; the asynq task is then retried
	Backoff time.Duration // first retry delay; doubles up to 10 s
}

// Post delivers p, retrying network errors and 5xx. 401/403 return
// errUnauthorized; 400 (malformed packet) is logged and dropped.
func (f *Forwarder) Post(p Packet) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(f.MaxWait)
	wait := f.Backoff
	for {
		status, err := f.postOnce(body)
		switch {
		case err == nil && status < 300:
			return nil
		case status == http.StatusUnauthorized || status == http.StatusForbidden:
			return errUnauthorized
		case status == http.StatusBadRequest:
			log.Printf("forward: backend rejected %q as malformed; dropping it", p.Name())
			return nil
		}
		if time.Now().Add(wait).After(deadline) {
			return fmt.Errorf("forward %q: giving up: status %d, err %v", p.Name(), status, err)
		}
		time.Sleep(wait)
		if wait *= 2; wait > 10*time.Second {
			wait = 10 * time.Second
		}
	}
}

func (f *Forwarder) postOnce(body []byte) (int, error) {
	req, err := http.NewRequest(http.MethodPost, f.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(f.Name, f.Key)
	resp, err := f.Client.Do(req)
	if err != nil {
		return 0, err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode, nil
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj-judge && go vet ./... && go test -timeout 60s ./... -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add claoj-judge/forward.go claoj-judge/forward_test.go
git commit -F - <<'EOF'
feat(judge): forward judge packets to the backend over HTTP with retry

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 7: Agent — grading loop (`Agent`)

**Files:**
- Create: `claoj-judge/grade.go`
- Test: `claoj-judge/grade_test.go`

**Interfaces:**
- Consumes: `Child`, `Packet`, test helpers `startChild`/`fakeJudge` (Task 5)
- Produces:
  - `type Agent struct{ child *Child; post func(Packet) error; stopping atomic.Bool; abortWait time.Duration }`
  - `(*Agent).ProcessTask(ctx context.Context, t *asynq.Task) error` (implements `asynq.Handler`)
  - `(*Agent).grade(ctx context.Context, req Packet, giveUp bool) error`

- [x] **Step 1: Write the failing tests**

Create `claoj-judge/grade_test.go`:

```go
package main

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recorder collects posted packets.
type recorder struct {
	mu  sync.Mutex
	got []Packet
	err error
}

func (r *recorder) post(p Packet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, p)
	return r.err
}

func (r *recorder) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, p := range r.got {
		out = append(out, p.Name())
	}
	return out
}

func newAgent(t *testing.T) (*Agent, *recorder, net.Conn) {
	t.Helper()
	child, addr := startChild(t, "secret")
	judge := fakeJudge(t, addr, "secret")
	rec := &recorder{}
	return &Agent{child: child, post: rec.post, abortWait: 200 * time.Millisecond}, rec, judge
}

var request = Packet{"name": "submission-request", "submission-id": float64(7), "problem-id": "aplusb"}

// expectPacket reads the next packet the agent sent tiervnoj.
func expectPacket(t *testing.T, judge net.Conn, name string) Packet {
	t.Helper()
	judge.SetReadDeadline(time.Now().Add(2 * time.Second))
	p, err := readPacket(judge)
	require.NoError(t, err)
	require.Equal(t, name, p.Name())
	return p
}

func TestGrade_HappyPathForwardsInOrderAndSkipsStale(t *testing.T) {
	a, rec, judge := newAgent(t)
	go func() {
		expectPacket(t, judge, "submission-request")
		for _, p := range []Packet{
			{"name": "submission-acknowledged", "submission-id": 7},
			{"name": "grading-begin", "submission-id": 7},
			{"name": "test-case-status", "submission-id": 6}, // stale: another submission
			{"name": "test-case-status", "submission-id": 7, "cases": []interface{}{}},
			{"name": "grading-end", "submission-id": 7},
		} {
			writePacket(judge, p)
		}
	}()

	require.NoError(t, a.grade(context.Background(), request, false))
	assert.Equal(t, []string{"submission-acknowledged", "grading-begin", "test-case-status", "grading-end"}, rec.names())
}

func TestGrade_CompileErrorIsTerminal(t *testing.T) {
	a, rec, judge := newAgent(t)
	go func() {
		expectPacket(t, judge, "submission-request")
		writePacket(judge, Packet{"name": "compile-error", "submission-id": 7, "log": "boom"})
	}()

	require.NoError(t, a.grade(context.Background(), request, false))
	assert.Equal(t, []string{"compile-error"}, rec.names())
}

func TestGrade_AbortSendsTerminateAndForwardsTermination(t *testing.T) {
	a, rec, judge := newAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		expectPacket(t, judge, "submission-request")
		writePacket(judge, Packet{"name": "grading-begin", "submission-id": 7})
		time.Sleep(50 * time.Millisecond)
		cancel() // asynq CancelProcessing
		expectPacket(t, judge, "terminate-submission")
		writePacket(judge, Packet{"name": "submission-terminated", "submission-id": 7})
	}()

	require.NoError(t, a.grade(ctx, request, false))
	assert.Equal(t, []string{"grading-begin", "submission-terminated"}, rec.names())
}

func TestGrade_AbortWithoutAnswerRecordsTermination(t *testing.T) {
	a, rec, judge := newAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		expectPacket(t, judge, "submission-request")
		cancel()
		expectPacket(t, judge, "terminate-submission") // ...and never answer
	}()

	require.NoError(t, a.grade(ctx, request, false))
	assert.Equal(t, []string{"submission-terminated"}, rec.names())
}

func TestGrade_ShutdownLeavesTaskForRequeue(t *testing.T) {
	a, rec, judge := newAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		expectPacket(t, judge, "submission-request")
		a.stopping.Store(true)
		cancel()
	}()

	assert.ErrorIs(t, a.grade(ctx, request, false), context.Canceled)
	assert.Empty(t, rec.names())

	judge.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, err := readPacket(judge)
	assert.Error(t, err) // no terminate-submission was sent
}

func TestGrade_GiveUpReportsInternalErrorWithoutGrading(t *testing.T) {
	a, rec, judge := newAgent(t)

	require.NoError(t, a.grade(context.Background(), request, true))
	assert.Equal(t, []string{"internal-error"}, rec.names())
	assert.Equal(t, int64(7), rec.got[0].SubmissionID())

	judge.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, err := readPacket(judge)
	assert.Error(t, err) // tiervnoj never saw the submission
}

func TestGrade_PostErrorFailsTheTask(t *testing.T) {
	a, rec, judge := newAgent(t)
	rec.err = errors.New("backend down")
	go func() {
		expectPacket(t, judge, "submission-request")
		writePacket(judge, Packet{"name": "grading-begin", "submission-id": 7})
	}()

	assert.Error(t, a.grade(context.Background(), request, false))
}

func TestGrade_NoChildFailsTheTask(t *testing.T) {
	a := &Agent{child: NewChild("secret"), post: (&recorder{}).post}
	assert.ErrorIs(t, a.grade(context.Background(), request, false), errNoChild)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj-judge && go test -timeout 60s -run Grade -v`
Expected: compile FAIL — `undefined: Agent`.

- [x] **Step 3: Implement `claoj-judge/grade.go`**

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hibiken/asynq"
)

// terminal packets end a submission on tiervnoj's side.
var terminal = map[string]bool{
	"grading-end":           true,
	"compile-error":         true,
	"internal-error":        true,
	"submission-terminated": true,
}

// Agent grades one asynq task at a time on the local tiervnoj child.
type Agent struct {
	child     *Child
	post      func(Packet) error
	stopping  atomic.Bool   // set on SIGTERM: asynq re-queues the task, so don't abort it
	abortWait time.Duration // how long to wait for tiervnoj to confirm an abort
}

// ProcessTask implements asynq.Handler.
func (a *Agent) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var req Packet
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		return fmt.Errorf("bad judge task payload: %v: %w", err, asynq.SkipRetry)
	}
	retried, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	// A failed attempt is not necessarily a crash: asynq counts every handler
	// error and lease expiry, so SIGTERM mid-grade, an errNoChild reconnect
	// gap and a >5min backend outage all land here too. On the last attempt
	// the retry budget is exhausted — report IE instead of retrying forever.
	return a.grade(ctx, req, retried > 0 && retried >= maxRetry)
}

func (a *Agent) grade(ctx context.Context, req Packet, giveUp bool) error {
	id := req.SubmissionID()
	if giveUp {
		return a.post(Packet{"name": "internal-error", "submission-id": id, "message": "judge task failed after exhausting its retries"})
	}
	if err := a.child.Send(req); err != nil {
		return err
	}

	done := ctx.Done()
	var abortTimeout <-chan time.Time
	for {
		select {
		case p := <-a.child.Events():
			if p.SubmissionID() != id {
				continue // late packet from an earlier submission
			}
			if err := a.post(p); err != nil {
				return err
			}
			if terminal[p.Name()] {
				return nil
			}
		case <-done:
			if a.stopping.Load() {
				return ctx.Err() // shutting down: asynq re-queues for another judge
			}
			done = nil // aborted via CancelProcessing (or task timeout)
			if err := a.child.Send(Packet{"name": "terminate-submission"}); err != nil {
				return a.post(Packet{"name": "submission-terminated", "submission-id": id})
			}
			abortTimeout = time.After(a.abortWait)
		case <-abortTimeout:
			// tiervnoj never confirmed; record the abort ourselves
			return a.post(Packet{"name": "submission-terminated", "submission-id": id})
		}
	}
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj-judge && go vet ./... && go test -timeout 60s ./... -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add claoj-judge/grade.go claoj-judge/grade_test.go
git commit -F - <<'EOF'
feat(judge): grade asynq judge tasks on the local tiervnoj child

Forwards packets in order, ignores stale ones, turns CancelProcessing
into terminate-submission, leaves tasks for re-queue on shutdown, and
reports IE instead of re-grading a submission that keeps killing tiervnoj.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 8: Agent — `main.go`, image, README, CI

**Files:**
- Create: `claoj-judge/main.go`, `claoj-judge/Dockerfile`, `claoj-judge/README.md`
- Modify: `.github/workflows/ci-cd.yml`

**Interfaces:**
- Consumes: `NewChild`, `Child.Serve/KeepAlive/Ready`, `Forwarder`, `errUnauthorized`, `Agent` (Tasks 5–7)
- Produces: binary `claoj-judge`; env contract `REDIS_ADDR` (required), `REDIS_PASSWORD`, `REDIS_DB` (default 0), `BACKEND_URL` (required), `JUDGE_NAME` (required), `JUDGE_KEY` (required), `JUDGE_YML` (default `/problems/judge.yml`), `CHILD_PORT` (default 9999), `TIERVNOJ_ENTRY` (default `/judge/.docker/entry`); image built `FROM claoj/judge-tiervnoj`

- [x] **Step 1: Implement `claoj-judge/main.go`**

```go
// Command claoj-judge pulls submissions from the asynq "judge" queue,
// grades them on a local tiervnoj (DMOJ) process and POSTs every judge
// packet to the backend's /internal/judge/event.
package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
)

func main() {
	redisAddr := mustEnv("REDIS_ADDR")
	backendURL := mustEnv("BACKEND_URL")
	name := mustEnv("JUDGE_NAME")
	key := mustEnv("JUDGE_KEY")
	redisDB, err := strconv.Atoi(env("REDIS_DB", "0"))
	if err != nil {
		log.Fatalf("REDIS_DB: %v", err)
	}
	port := env("CHILD_PORT", "9999")

	// 1. Loopback bridge for tiervnoj.
	child := NewChild(key)
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	go child.Serve(ln)
	go child.KeepAlive(60 * time.Second)

	// 2. Spawn tiervnoj; if it dies, die with it so the orchestrator restarts us.
	cmd := exec.Command(env("TIERVNOJ_ENTRY", "/judge/.docker/entry"), "run",
		"-p", port, "-c", env("JUDGE_YML", "/problems/judge.yml"), "--no-watchdog",
		"127.0.0.1", name, key)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("start tiervnoj: %v", err)
	}
	go func() { log.Fatalf("tiervnoj exited: %v", cmd.Wait()) }()

	// tiervnoj self-tests every executor before connecting; that takes a while.
	for deadline := time.Now().Add(15 * time.Minute); !child.Ready(); time.Sleep(time.Second) {
		if time.Now().After(deadline) {
			log.Fatal("tiervnoj did not connect within 15 minutes")
		}
	}
	log.Printf("judge %q ready", name)

	// 3. Pull work.
	fwd := &Forwarder{
		URL:     strings.TrimRight(backendURL, "/") + "/internal/judge/event",
		Name:    name,
		Key:     key,
		Client:  &http.Client{Timeout: 30 * time.Second},
		MaxWait: 5 * time.Minute,
		Backoff: 500 * time.Millisecond,
	}
	agent := &Agent{child: child, abortWait: 30 * time.Second}
	agent.post = func(p Packet) error {
		err := fwd.Post(p)
		if errors.Is(err, errUnauthorized) {
			log.Fatalf("backend rejected judge %q: JUDGE_NAME/JUDGE_KEY must match a judge_judge row", name)
		}
		return err
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: os.Getenv("REDIS_PASSWORD"), DB: redisDB},
		asynq.Config{
			Concurrency:     1, // one tiervnoj grades one submission at a time; scale with replicas
			Queues:          map[string]int{"judge": 1},
			ShutdownTimeout: 25 * time.Second, // keep the container stop grace period above this
			RetryDelayFunc:  func(int, error, *asynq.Task) time.Duration { return time.Second },
		},
	)
	if err := srv.Start(agent); err != nil {
		log.Fatalf("asynq: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
	<-sig
	agent.stopping.Store(true)
	srv.Shutdown()
	os.Exit(0)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("%s is required", k)
	}
	return v
}
```

- [x] **Step 2: Build and vet**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj-judge && go mod tidy && go vet ./... && go build -o /dev/null . && go test ./...`
Expected: no errors; tests PASS.

- [x] **Step 3: Create `claoj-judge/Dockerfile`**

```dockerfile
# Agent + the v1 grading engine. Build the tiervnoj image first
# (CLAOJ/claoj-docker/judge/tiervnoj -> claoj/judge-tiervnoj).
ARG TIERVNOJ_IMAGE=claoj/judge-tiervnoj

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /claoj-judge .

FROM ${TIERVNOJ_IMAGE}
COPY --from=build /claoj-judge /usr/local/bin/claoj-judge
# Runs as root: the tiervnoj entry script drops to the judge user itself.
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/claoj-judge"]
```

- [x] **Step 4: Build the image**

Run: `cd /f/Coding/CLAOJ/OJ-v2/claoj-judge && docker image inspect claoj/judge-tiervnoj >/dev/null && docker build -t claoj/claoj-judge-agent .`
Expected: `naming to docker.io/claoj/claoj-judge-agent` (if the inspect fails, build tiervnoj first: `docker build -t claoj/judge-tiervnoj /f/Coding/CLAOJ/CLAOJ/claoj-docker/judge/tiervnoj`).

- [x] **Step 5: Create `claoj-judge/README.md`**

```markdown
# claoj-judge

Judge agent for CLAOJ v2. Pulls submissions from the asynq queue `judge`
(Redis), grades them on a local **tiervnoj** process (the v1 DMOJ/VNOJ
engine: all languages, archives, file_io, checkers, batches), and POSTs
every judge packet to the backend's `/internal/judge/event`.

Nothing connects to the judge. Scale by adding replicas; all replicas share
one `judge_judge` row.

| Env | Default | |
|---|---|---|
| `REDIS_ADDR` | — | same Redis as the backend |
| `REDIS_DB` | `0` | **must equal the backend's `REDIS_DB`** |
| `REDIS_PASSWORD` | | |
| `BACKEND_URL` | — | e.g. `http://v2_backend:8081` |
| `JUDGE_NAME` / `JUDGE_KEY` | — | a `judge_judge` row's name / auth_key |
| `JUDGE_YML` | `/problems/judge.yml` | tiervnoj config |
| `CHILD_PORT` | `9999` | loopback port tiervnoj dials |

Needs `SYS_PTRACE`, the problems folder at `/problems` (read-only), and a
stop grace period > 25 s.

    docker build -t claoj/claoj-judge-agent .   # needs claoj/judge-tiervnoj locally
    go test ./...
```

- [x] **Step 6: Add a judge test job to `.github/workflows/ci-cd.yml`**

In `changes.outputs` add `judge: ${{ steps.filter.outputs.judge }}`; in the paths filter add:

```yaml
            judge:
              - 'claoj-judge/**'
              - '.github/workflows/ci-cd.yml'
```

Add the job after `test-backend`:

```yaml
  test-judge:
    needs: changes
    if: needs.changes.outputs.judge == 'true'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: claoj-judge
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache-dependency-path: claoj-judge/go.sum
      - run: go vet ./...
      - run: go test ./...
```

(No image push: the base image `claoj/judge-tiervnoj` is built locally, not published.)

- [x] **Step 7: Commit**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add claoj-judge/main.go claoj-judge/Dockerfile claoj-judge/README.md claoj-judge/go.mod claoj-judge/go.sum .github/workflows/ci-cd.yml
git commit -F - <<'EOF'
feat(judge): agent entrypoint and image on top of tiervnoj

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 9: Compose wiring and docs

**Files:**
- Modify: `CLAOJ/claoj-docker/claoj/docker-compose.v2.yml` (separate repo)
- Modify (not committed — gitignored): `CLAOJ/claoj-docker/claoj/local/v2.local.env`
- Modify: `OJ-v2/docs/deployment.md`, `OJ-v2/docs/development.md` (§6.2–6.3)

**Interfaces:**
- Consumes: agent env contract (Task 8), backend `JUDGE_NAME` (Task 4)

- [x] **Step 1: Update `docker-compose.v2.yml`**

In `v2_backend`: change `networks: [db, site, judge, v2]` to `networks: [db, site, v2]`, and add to its `environment:`:

```yaml
      # judge_judge row shared by every v2 judge replica (online/load sync)
      JUDGE_NAME: ${V2_JUDGE_NAME:-Vịt Toàn Năng}
```

Replace the whole `v2_judge` service with (no `container_name`, so `--scale v2_judge=N` works):

```yaml
  v2_judge:
    image: claoj/claoj-judge-agent
    build:
      context: ../../../OJ-v2/claoj-judge
    restart: unless-stopped
    cap_add: [SYS_PTRACE]
    stop_grace_period: 30s
    volumes:
      - ${CLAOJ_DATA}/problems:/problems:ro
    environment:
      REDIS_ADDR: redis:6379
      REDIS_DB: "2" # must equal v2_backend's REDIS_DB (local/v2.local.env)
      BACKEND_URL: http://v2_backend:8081
      JUDGE_NAME: ${V2_JUDGE_NAME:-Vịt Toàn Năng}
      JUDGE_KEY: ${V2_JUDGE_KEY}
    networks: [site, v2]
    depends_on: [redis, v2_backend]
```

Leave `v1_judge` exactly as it is.

- [ ] **Step 2: Update the local backend env (not committed)**

In `CLAOJ/claoj-docker/claoj/local/v2.local.env` delete `BRIDGE_ADDR=:9997`. Do the same in the production env file on the VPS during cutover (Task 10 Step 8).

- [x] **Step 3: Validate compose**

Run: `cd /f/Coding/CLAOJ/CLAOJ/claoj-docker/claoj && docker compose -f docker-compose.local.yml -f docker-compose.v2.yml config --quiet && echo OK`
Expected: `OK`.

- [x] **Step 4: Update docs**

In `OJ-v2/docs/development.md`, replace the §6.2 "run a judge against the host-run backend" block (the `docker run … claoj/judge-tiervnoj … host.docker.internal` command and the `bridge [...]: authenticated` note) with:

````markdown
**6.2 Run a judge** against the host-run backend (`:8081`) and host Redis:

```bash
docker run -d --name claoj_judge_local --cap-add SYS_PTRACE \
  -v "F:\Coding\CLAOJ\CLAOJ\claoj-data\problems:/problems:ro" \
  -e REDIS_ADDR=host.docker.internal:6379 -e REDIS_DB=2 \
  -e BACKEND_URL=http://host.docker.internal:8081 \
  -e JUDGE_NAME="Vịt Toàn Năng" -e JUDGE_KEY=<V2_JUDGE_KEY, from .env> \
  claoj/claoj-judge-agent
```

The backend needs `JUDGE_NAME="Vịt Toàn Năng"`; within 30 s `GET /api/judges`
reports it `online`. Nothing connects to the judge: it pulls from Redis.
````

Append to `OJ-v2/docs/deployment.md`:

```markdown
## Judges and horizontal scaling (k3s / Docker Swarm)

Submissions are queued in Redis (asynq queue `judge`); `claoj-judge`
replicas pull from it and POST results to the backend's
`/internal/judge/event`. Nothing dials into a judge, so backend and judge
are both plain replicated services with the same shape on k3s and Swarm:

- `backend`: N replicas, env `JUDGE_NAME`. Route only `/api/` to it from the
  public ingress/nginx; `/internal/` must stay cluster-internal.
- `judge`: M replicas of `claoj/claoj-judge-agent`, `SYS_PTRACE`, stop grace
  ≥ 30 s, env `REDIS_ADDR`, `REDIS_DB` (= backend's), `BACKEND_URL` (the
  backend Service), `JUDGE_NAME`, `JUDGE_KEY`. No Service or ports.
- `problems`: the single v1 problems folder, mounted read-only on judges.
  Multi-node: NFS-export it from the host that owns it.
- backend `/app/data` (v2 PDFs): one shared RWX volume across backend replicas.
- The backend refuses to start without Redis.
- Scale judges with replicas; there is no per-judge admin step (all share one
  `judge_judge` row).
```

- [x] **Step 5: Commit both repos**

```bash
cd /f/Coding/CLAOJ/CLAOJ/claoj-docker
git add claoj/docker-compose.v2.yml
git commit -F - <<'EOF'
deploy(v2): run claoj-judge agent; backend no longer on the judge network

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s

cd /f/Coding/CLAOJ/OJ-v2
git add docs/deployment.md docs/development.md
git commit -F - <<'EOF'
docs: judge pull-queue deployment and local judge setup

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 10: End-to-end verification (local stack), then cutover

**Files:** none (verification only). Any bug found → fix in the owning task's files with a regression test, commit, rerun this task.

- [x] **Step 1: Build and start the stack**

```bash
cd /f/Coding/CLAOJ/CLAOJ/claoj-docker/claoj
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml build v2_backend v2_judge
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml up -d
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml logs -f v2_judge
```

Expected in judge logs (after tiervnoj self-tests): `child: tiervnoj connected`, then `judge "Vịt Toàn Năng" ready`. Backend log has no `bridge:` listener line.

- [x] **Step 2: Judge shows online**

Within 30 s: `curl -s localhost:8090/api/judges` lists `Vịt Toàn Năng` as online. If not: compare `REDIS_DB` of `v2_backend` and `v2_judge` first.

- [x] **Step 3: Grade one of each verdict on `aplusb`**

Log in and submit through `localhost:8090` as in `docs/development.md` §6.3 / §7.2, with language `PY3`:

| Source | Expected final `result` |
|---|---|
| `print(sum(map(int, open(0).read().split())))` | `AC` (if `WA`, read the aplusb statement; the pipeline is still fine) |
| `print(0)` | `WA` |
| `def (:` | `CE` |
| `while True: pass` | `TLE` |

Poll `GET /api/submission/<id>`: status goes `QU → P → G → D` (or `CE`), and the live page updates without reload (websocket).

- [x] **Step 4: Engine-feature problems reach a verdict, never IE**

Submit `print(0)` (PY3) to `01_01` (file_io), `alphabet` (bridged checker), `halloween_tree` (batched). Expected: each ends `D` with a result other than `IE`.

- [x] **Step 5: Abort, rejudge, crash**

- Submit `while True: pass` and abort it from admin while `G` → ends `AB`.
- Rejudge a finished submission from admin twice quickly → graded once, ends `D`.
- Submit `while True: pass`, then `docker kill` the judge container while it is `G` (`docker compose … ps -q v2_judge`). Compose restarts it; within ~1–2 min the submission is graded again and ends `TLE` (not stuck at `G`).

- [x] **Step 6: Two backends, two judges**

```bash
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml up -d --scale v2_judge=2
# create, attach all networks, THEN start: the backend exits if Redis is unreachable at boot
docker create --name claoj_v2_backend_2 --network claoj_v2 --network-alias v2_backend \
  --env-file local/v2.local.env -e JUDGE_NAME="Vịt Toàn Năng" \
  ghcr.io/claoj/claoj-go:latest
docker network connect claoj_db claoj_v2_backend_2
docker network connect claoj_site claoj_v2_backend_2
docker start claoj_v2_backend_2
```

(`ghcr.io/claoj/claoj-go:latest` is the tag Step 1 built; the compose project is `claoj`, so networks are `claoj_v2`, `claoj_db`, `claoj_site` — confirm with `docker network ls`.) Submit 6 submissions in a row → all reach `D`; `docker compose … logs v2_judge` shows both judge replicas grading. Stop `claoj_v2_backend_2` mid-grade → the submission still finishes. Clean up: `docker rm -f claoj_v2_backend_2`.

- [x] **Step 7: Idle reconnect**

Leave the stack idle for 6 minutes (longer than tiervnoj's 300 s socket timeout), then submit `print(0)` to `aplusb` → ends `D`.

- [ ] **Step 8: Production cutover (only after Steps 1–7 pass; confirm with the user first)**

On the VPS: build `claoj/judge-tiervnoj` and `claoj/claoj-judge-agent`, remove `BRIDGE_ADDR` from the prod v2 env, deploy the backend image from `main`, then `docker compose -f docker-compose.yml -f docker-compose.v2.yml up -d v2_backend v2_judge`. Watch one real submission finish. Rollback = previous backend tag (`scripts/rollback.sh backend <sha>`) plus restoring the old `v2_judge` service from git history; no DB changes to undo.
