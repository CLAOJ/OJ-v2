# Judge Pull-Queue Review Fix: Abort a Queued Submission

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make "abort a submission that is still queued" work through the admin API, and correct the production cutover runbook, so `feat/judge-pull-queue` can merge.

**Architecture:** One status check in `SubmissionService.Abort` rejects `QU`, which makes the pending-task delete path in `jobs.AbortJudgeSubmission` unreachable. Allow `QU`, close the lookup→delete race in `AbortJudgeSubmission`, fix three wrong lines in `docs/deployment.md`, and prove it once on the local stack.

**Tech Stack:** Go 1.25, GORM (sqlite in tests), asynq v0.26.0, testify, Docker.

**Spec:** `docs/superpowers/specs/2026-10-09-judge-pull-queue-design.md` (§Design → Backend item 3 "Abort", §Testing "abort a queued and a running submission"). Parent plan: `docs/superpowers/plans/2026-10-09-judge-pull-queue.md`.

## Why (review finding on `5aa6387`)

`claoj/service/submission/submission_service.go:239-242`:

```go
	// Only allow aborting submissions that are being processed
	if sub.Status != "P" && sub.Status != "G" {
		return ErrSubmissionNotProcessing
	}
```

`POST /api/admin/submission/<id>/abort` on a `QU` submission returns 400 "submission is not being processed". Consequences:

- The spec's "abort a queued submission" cannot be done; with one shared queue and a backlog, an admin cannot cancel waiting submissions.
- `jobs.AbortJudgeSubmission`'s `deleteTask` (pending) branch is dead code from the API.
- `docs/deployment.md` §9.6 Step 4 tells the operator to abort a `QU` submission and expect `AB`; in production that returns 400 and reads as a broken cutover.

Everything else in the review was green: backend `go vet` clean + 471 tests, agent `go vet` clean + 19 tests, Linux build, merged tree identical to the branch, and the e2e test rows verified deleted (0 users / 0 profiles / 0 submissions).

## Global Constraints

- No DB schema changes (DML only). No edits to `OJ/`, v1 compose base files, `claoj-docker/judge/tiervnoj/`, or `claoj/api/v2/admin/problem_data*.go`.
- Touch only: `claoj/service/submission/submission_service.go`, `claoj/service/submission/abort_test.go` (new), `claoj/jobs/judge.go`, `claoj/jobs/judge_test.go`, `docs/deployment.md`, `docs/superpowers/follow-ups.md`, this plan (checkboxes).
- Commit messages: Git Bash heredoc `git commit -F - <<'EOF' … EOF`; end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`; verify with `git log -1 --format=%s`. Explicit `git add <paths>` only.

## Review Focus

1. **Abort of a `QU` submission whose asynq task no longer exists** (Redis was flushed, or the task already finished) → expected: still ends `AB`, no error. Test: Task 1 `TestAbort_Queued` (fake bridge returns nil) + existing `TestAbortJudgeSubmission/unknown task is a no-op`.
2. **A judge picks the task up between the lookup and the delete** → expected: the abort falls back to cancelling the active task, not a 500. Test: Task 1 `TestAbortJudgeSubmission/pending that turned active is cancelled`.
3. **Abort of a finished (`D`) submission** → expected: still rejected, row untouched. Test: Task 1 `TestAbort_FinishedIsRejected`.
4. **The bridge/queue call fails** → expected: error returned, row NOT marked `AB`. Test: Task 1 `TestAbort_QueueErrorLeavesRowUntouched`.
5. **Queued abort on the real stack with no judge running** → expected: `AB`, and no `sub-<id>` key left in Redis. Test: Task 3.

---

### Task 1: Allow aborting a queued submission

**Files:**
- Modify: `claoj/service/submission/submission_service.go:239-242`
- Modify: `claoj/jobs/judge.go` (`AbortJudgeSubmission`, last 4 lines of the function)
- Modify: `claoj/jobs/judge_test.go` (`stubQueue`, `useStubQueue`, `TestAbortJudgeSubmission`)
- Test: `claoj/service/submission/abort_test.go` (new)

**Interfaces:**
- Consumes: `submission.NewSubmissionService(bridge BridgeServer) *SubmissionService`; `type BridgeServer interface { Abort(subID uint) error }`; `submission.AbortRequest{SubmissionID uint}`; `submission.ErrSubmissionNotProcessing`; jobs seams `getTaskInfo`, `deleteTask`, `cancelProcessing`.
- Produces: no signature changes. Behaviour: `SubmissionService.Abort` accepts status `QU`, `P`, `G`.

- [x] **Step 1: Write the failing service tests**

Create `claoj/service/submission/abort_test.go`:

```go
package submission

import (
	"errors"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// fakeBridge records Abort calls instead of talking to the judge queue.
type fakeBridge struct {
	aborted []uint
	err     error
}

func (f *fakeBridge) Abort(subID uint) error {
	f.aborted = append(f.aborted, subID)
	return f.err
}

// setupAbortDB points db.DB at an in-memory sqlite holding submission 7 in the given status.
func setupAbortDB(t *testing.T, status string) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.Submission{}))
	db.DB = database
	require.NoError(t, db.DB.Create(&models.Submission{ID: 7, UserID: 1, ProblemID: 1, LanguageID: 1, Date: time.Now(), Status: status}).Error)
}

func loadAbortSub(t *testing.T) models.Submission {
	t.Helper()
	var s models.Submission
	require.NoError(t, db.DB.First(&s, 7).Error)
	return s
}

func TestAbort_Queued(t *testing.T) {
	setupAbortDB(t, "QU")
	bridge := &fakeBridge{}

	require.NoError(t, NewSubmissionService(bridge).Abort(AbortRequest{SubmissionID: 7}))

	assert.Equal(t, []uint{7}, bridge.aborted)
	s := loadAbortSub(t)
	assert.Equal(t, "AB", s.Status)
	require.NotNil(t, s.Result)
	assert.Equal(t, "AB", *s.Result)
}

func TestAbort_ProcessingAndGradingStillWork(t *testing.T) {
	for _, status := range []string{"P", "G"} {
		t.Run(status, func(t *testing.T) {
			setupAbortDB(t, status)
			bridge := &fakeBridge{}
			require.NoError(t, NewSubmissionService(bridge).Abort(AbortRequest{SubmissionID: 7}))
			assert.Equal(t, []uint{7}, bridge.aborted)
			assert.Equal(t, "AB", loadAbortSub(t).Status)
		})
	}
}

func TestAbort_FinishedIsRejected(t *testing.T) {
	for _, status := range []string{"D", "CE", "IE", "AB"} {
		t.Run(status, func(t *testing.T) {
			setupAbortDB(t, status)
			bridge := &fakeBridge{}
			err := NewSubmissionService(bridge).Abort(AbortRequest{SubmissionID: 7})
			assert.ErrorIs(t, err, ErrSubmissionNotProcessing)
			assert.Empty(t, bridge.aborted)
			assert.Equal(t, status, loadAbortSub(t).Status)
		})
	}
}

func TestAbort_QueueErrorLeavesRowUntouched(t *testing.T) {
	setupAbortDB(t, "QU")
	bridge := &fakeBridge{err: errors.New("redis down")}

	assert.Error(t, NewSubmissionService(bridge).Abort(AbortRequest{SubmissionID: 7}))
	assert.Equal(t, "QU", loadAbortSub(t).Status)
}
```

- [x] **Step 2: Run to verify the right ones fail**

Run: `cd F:/Coding/CLAOJ/OJ-v2-wt/feat/judge-pull-queue/claoj && go test ./service/submission/ -run 'TestAbort_' -v`
Expected: `TestAbort_Queued` FAILS (`submission is not being processed`). The other three pass already: they pin behaviour that must not change (`TestAbort_QueueErrorLeavesRowUntouched` starts guarding once Step 3 lets `QU` through).

- [x] **Step 3: Allow `QU` in `claoj/service/submission/submission_service.go`**

Replace lines 239-242:

```go
	// Only allow aborting submissions that are being processed
	if sub.Status != "P" && sub.Status != "G" {
		return ErrSubmissionNotProcessing
	}
```

with:

```go
	// Abortable while queued (QU), acknowledged (P) or grading (G); the
	// judge queue drops a waiting task and cancels a running one.
	if sub.Status != "QU" && sub.Status != "P" && sub.Status != "G" {
		return ErrSubmissionNotProcessing
	}
```

- [x] **Step 4: Run the service tests**

Run: `cd F:/Coding/CLAOJ/OJ-v2-wt/feat/judge-pull-queue/claoj && go test ./service/submission/ -v`
Expected: PASS (all, including any pre-existing tests in the package).

- [x] **Step 5: Write the failing race test in `claoj/jobs/judge_test.go`**

Add a field to `stubQueue` (after `infoErr error`):

```go
	delErr    error // returned by deleteTask
```

In `useStubQueue`, replace the `deleteTask = …` line with:

```go
	deleteTask = func(queue, id string) error { s.deleted = append(s.deleted, queue+"/"+id); return s.delErr }
```

Add a subtest at the end of `TestAbortJudgeSubmission` (after "missing queue is a no-op"):

```go
	t.Run("pending that turned active is cancelled", func(t *testing.T) {
		s := useStubQueue(t)
		s.state = asynq.TaskStatePending
		s.delErr = errors.New("task is in active state") // a judge picked it up after the lookup
		require.NoError(t, AbortJudgeSubmission(7))
		assert.Equal(t, []string{"judge/sub-7"}, s.deleted)
		assert.Equal(t, []string{"sub-7"}, s.cancelled)
	})
```

Add `"errors"` to that file's imports if it is not already there.

Run: `cd F:/Coding/CLAOJ/OJ-v2-wt/feat/judge-pull-queue/claoj && go test ./jobs/ -run TestAbortJudgeSubmission -v`
Expected: the new subtest FAILS (`task is in active state` returned).

- [x] **Step 6: Close the race in `claoj/jobs/judge.go`**

In `AbortJudgeSubmission`, replace the final statement

```go
	return deleteTask(QueueJudge, id)
```

with:

```go
	if err := deleteTask(QueueJudge, id); err != nil {
		// A judge picked the task up between the lookup and the delete (or
		// it just finished): cancel whatever is running; a no-op otherwise.
		return cancelProcessing(id)
	}
	return nil
```

- [x] **Step 7: Run the suites**

Run: `cd F:/Coding/CLAOJ/OJ-v2-wt/feat/judge-pull-queue/claoj && go vet ./... && go test ./...`
Expected: vet clean; every package passes with 0 FAIL; the total is above the 471 the branch had (new: 4 tests + 6 subtests in `service/submission`, 1 subtest in `jobs`).

- [x] **Step 8: Commit**

```bash
cd F:/Coding/CLAOJ/OJ-v2-wt/feat/judge-pull-queue
git add claoj/service/submission/submission_service.go claoj/service/submission/abort_test.go claoj/jobs/judge.go claoj/jobs/judge_test.go
git commit -F - <<'EOF'
fix(submission): allow aborting a queued submission

Abort rejected QU, so a submission waiting in the judge queue could not be
cancelled and the pending-task delete path was unreachable. Also fall back
to cancelling when a judge picks the task up between lookup and delete.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 2: Correct the cutover runbook

**Files:**
- Modify: `docs/deployment.md` (lines ~217, ~441, ~488-500)
- Modify: `docs/superpowers/follow-ups.md` (remove the two entries this fixes)

- [x] **Step 1: §5 judge example — stop hardcoding the Redis DB**

In `docs/deployment.md` (~line 217) replace

```
  -e REDIS_ADDR=<redis-host>:6379 -e REDIS_DB=0 \
```

with

```
  -e REDIS_ADDR=<redis-host>:6379 -e REDIS_DB=<same as the backend's REDIS_DB> \
```

- [x] **Step 2: §9.6 intro — make the provenance command runnable**

Replace `` `git show 05946a7:claoj/jobs/judge.go:56` `` with `` `git show 05946a7:claoj/jobs/judge.go | sed -n '56p'` ``.

- [x] **Step 3: §9.6 Step 4 — drop the `archive:` gate, fix the AC sample**

The local e2e run did exercise `archive:` problems: `aplusb` (`archive: aplusb.zip`) graded AC/WA/CE/TLE and `01_01` (`archive: cake.zip`) graded WA — confirm with `head -1 F:/Coding/CLAOJ/CLAOJ/claoj-data/problems/aplusb/init.yml` and `…/01_01/init.yml`. So:

- In the Step 4 lead-in, change `never exercised these four` to `never exercised these three`.
- Delete the whole `- **\`archive:\` problem:** …` bullet (2 lines).
- In the "AC/WA/CE/TLE attribution" bullet, replace `` `print(sum(...))`→AC `` with `a correct solution→AC (on \`aplusb\` the input is N then N pairs, so a one-line \`print(sum(...))\` is WA)`.

Leave the "Abort of a *queued* submission" bullet as it is: Task 1 makes it true.

- [x] **Step 4: Remove the fixed follow-ups**

In `docs/superpowers/follow-ups.md` delete these two bullets (each 3 lines), which Steps 1–2 fix:

- the one starting ``- `docs/deployment.md` (§9.6 step 1 example) — the §5 `docker run` sample hardcodes``
- the one starting ``- `docs/deployment.md` (§9.6 provenance note) — `git show 05946a7:claoj/jobs/judge.go:56` is``

- [x] **Step 5: Verify and commit**

Run: `cd F:/Coding/CLAOJ/OJ-v2-wt/feat/judge-pull-queue && grep -nF -e 'REDIS_ADDR=<redis-host>:6379 -e REDIS_DB=0' -e 'judge.go:56`' -e 'these four' -e 'archive:` problem:' docs/deployment.md`
Expected: no output.

```bash
git add docs/deployment.md docs/superpowers/follow-ups.md
git commit -F - <<'EOF'
docs: correct the judge cutover runbook gate

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 3: Prove queued abort on the local stack

**Files:** none (verification only). A failure → fix in Task 1's files with a regression test, commit, rerun this task.

Uses the same overrides as the parent task's end-to-end run: build from the worktree, compose from the claoj-docker **main** checkout with the worktree overlay, a throwaway superuser, teardown without `-v`. If `docker kill` / `docker stop` / `docker compose down` are blocked by the command-approval policy, use the equivalents the parent run used (`docker rm -f` on the project's containers + `docker network rm`) and say so.

- [x] **Step 1: Build and start the stack with no judge**

```sh
docker build -t ghcr.io/claoj/claoj-go:judge-pull-queue F:/Coding/CLAOJ/OJ-v2-wt/feat/judge-pull-queue/claoj
docker image inspect claoj/claoj-judge-agent >/dev/null || docker build -t claoj/claoj-judge-agent F:/Coding/CLAOJ/OJ-v2-wt/feat/judge-pull-queue/claoj-judge
cd F:/Coding/CLAOJ/CLAOJ/claoj-docker/claoj
V2_BACKEND_TAG=judge-pull-queue docker compose -f docker-compose.local.yml \
  -f F:/Coding/CLAOJ/CLAOJ/claoj-docker-wt/feat/judge-pull-queue/claoj/docker-compose.v2.yml \
  up -d --no-build --scale v2_judge=0
```

Expected: stack up, **no** `v2_judge` container. If ports 8080 / 8090 / 3306 / 6379 are taken by another project, do not stop that container: block the task with the reason.

- [x] **Step 2: Abort a queued submission**

Create the throwaway superuser `e2e-judge` (as in the parent run: `docker exec claoj_site python3 manage.py shell -c …`, `create_superuser` + its `judge.Profile` row), log in through `localhost:8090` (`docs/development.md` §7.2), then:

1. Submit `print(0)` (PY3) to `aplusb`; note `<id>`. With no judge it stays `QU`.
2. `docker exec claoj_redis redis-cli -n 2 --scan --pattern 'asynq:{judge}:t:sub-<id>'` → prints that key (the task is queued).
3. `POST /api/admin/submission/<id>/abort` → **200**.
4. Final row: `status = AB`, `result = AB` (admin submissions endpoint, or SQL on `judge_submission`).
5. The same redis scan → **no output** (the task is gone).

- [x] **Step 3: The queue still grades afterwards**

```sh
V2_BACKEND_TAG=judge-pull-queue docker compose -f docker-compose.local.yml \
  -f F:/Coding/CLAOJ/CLAOJ/claoj-docker-wt/feat/judge-pull-queue/claoj/docker-compose.v2.yml \
  up -d --no-build --scale v2_judge=1
```

Wait for `judge "Vịt Toàn Năng" ready` in the judge log. The aborted submission must stay `AB` (not get graded). Submit `print(0)` to `aplusb` again → ends `D` / `WA`. Rejudge the aborted submission from admin → ends `D` / `WA` (an aborted submission can still be re-queued).

- [x] **Step 4: Tear down**

Delete the throwaway user, its profile and its submissions (with source/testcase rows), and verify the counts are 0. Then the same compose prefix with `down` (**never `-v`**). Confirm no `claoj` containers remain and the ports are free.
