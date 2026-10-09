# Follow-ups

Open items parked during implementation, one line each, naming the file each lives in.
Nothing here blocks the branch; each was triaged at a review gate and parked with a reason.

## 2026-10-09 → 2026-10-10 — `feat/judge-pull-queue` (judge pull-queue)

Carried from the batch reviews and the whole-branch review (plan
`docs/superpowers/plans/2026-10-09-judge-pull-queue.md`, spec
`docs/superpowers/specs/2026-10-09-judge-pull-queue-design.md`):

- `claoj/jobs/judge_test.go`, `claoj/jobs/judge_status_test.go` — test-log noise: gorm
  `record not found`/SELECT dumps and `jobs: queued judge task` prints interleave with PASS.
- `claoj/jobs/judge_status.go` — `RunJudgeStatusLoop` calls `time.NewTicker(every)` and panics
  on `every <= 0`; the single caller passes 30 s. Guard if a second caller appears.
- `claoj/jobs/judge_test.go` — tests reassign `db.DB` with no `t.Cleanup` restore (harmless
  while the package has no other tests).
- `claoj-judge/dmoj.go:39,137` — no read deadline on the loopback read path; a peer stalling
  mid-frame leaks the handle goroutine (the deleted TCP bridge set 60 s). Plan-mandated code,
  loopback + co-container.
- `claoj-judge/dmoj.go:177` — `KeepAlive` uses an unstoppable `time.Tick`; moot today because
  `main.go` ends in `os.Exit(0)`, but wire a `NewTicker` if the agent ever restarts in-process.
- `claoj-judge/*_test.go` — test output not pristine (package `log.Printf` lines interleave).
- `claoj-judge/grade_test.go` — `require`/`FailNow` called from spawned goroutines.
- `claoj-judge/forward.go:36,52` — `Backoff: 0` in the zero value hot-spins a retry loop.
- `claoj-judge/README.md` — no `TIERVNOJ_ENTRY` row in the env table.
- `claoj-judge/grade.go` — after a post-failure mid-grade abandon no `terminate-submission` is
  sent (needs a >5 min outage to bite); asynq's task-timeout race is inherent to the design.
- `claoj-judge/grade.go`, `claoj-judge/forward.go` — untested arms: 403 branch, network-error
  retry, `ProcessTask` itself.
- `claoj/bridge/handler.go` (`handler_test.go`) — no test drives `EventHandler`'s 500 (DB
  error) arm or the malformed-body 400; `readPacket` oversized/corrupt-deflate paths untested.
- `claoj/bridge/handler.go` — truncation guards only `feedback`; `output`,
  `extended_feedback`, `log`/`message` are written unlength-checked (spec scopes the fix to
  `feedback`; the column widths are unverified).
- `claoj/bridge/handler.go` — narrow parked race: an abort-timeout's own
  `submission-terminated` can race a leftover terminal packet into an immediate rejudge's
  event channel; replay-corrected by any later rejudge. Plan-mandated design.
- `docs/deployment.md` (§9.6 step 1 example) — the §5 `docker run` sample hardcodes
  `-e REDIS_DB=0` while this document's own env table says 2; prefer
  `REDIS_DB=<same as backend>`.
- `docs/deployment.md` (§9.6 provenance note) — `git show 05946a7:claoj/jobs/judge.go:56` is
  not runnable git syntax; use `git show 05946a7:claoj/jobs/judge.go | sed -n '56p'`.
- `docs/deployment.md` (§9.6 step 1) — the asynq inspection loop omits `asynq:{q}:active`;
  in-flight tasks self-heal into `retry`, which is inspected.
- `local/v2.local.env` (claoj-docker main checkout, gitignored) — still sets `BRIDGE_ADDR=:9997`;
  harmless, the backend no longer reads it. **Human step**: delete it.
- `.superpowers/sdd/…/task-1-report.md`, `task-2-report.md`, `task-6-report.md`,
  `task-9-report.md`, `task-10-report.md` — record-only wording errors (wrong line counts, a
  "verbatim" claim vs an extra comment line, `load: 0` framed as a pairing error when it is
  correct idle behaviour). Not re-runnable; noted here so nobody re-derives them.
- Pre-existing backlog, not introduced by this branch, sharpened for the cutover:
  `claoj/main.go` has no SIGTERM shutdown path (`docker stop` hangs to SIGKILL — documented in
  `docs/deployment.md` §9.6), `claoj/api/v2/submission.go:126-129` 404s
  `GET /api/submission/<id>` for non-public problems even for superusers, and
  `claoj/jobs/judge_status.go` never writes `judge_judge.ping`.
