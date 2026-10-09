# Judge pull-queue redesign (horizontal scaling on k3s / Docker Swarm)

Date: 2026-10-09
Status: implemented — Tasks 0–10 built and reviewed on branch `feat/judge-pull-queue` (whole-branch review 2026-10-10: 0 Critical / 4 Important, all four closed by its fix wave, approved on scoped re-review). Open: Task 10 Step 8, the production cutover (human), and Task 9 Step 2's `BRIDGE_ADDR` removal from the local `v2.local.env` (human).

## Goal

Run the v2 backend and the v2 judges as plain replicated services
(`replicas: N`) on k3s now and Docker Swarm later, so capacity scales
horizontally instead of vertically. Same YAML shape on both platforms.

## Constraints

- **No DB schema changes.** v1 (Django) and v2 (Go) share one MySQL schema,
  which Django owns. Only existing tables/columns may be read or written.
- **v1 untouched.** The v1 site, its `bridged`, and its Python judges keep
  running as they are.
- **Language/feature parity with v1 judge.** v2 must grade every existing
  problem the v1 judge can.

## Findings that drive this design

### 1. The Go judge (`claoj-judge`) cannot replace the v1 judge

v1 judge = `tiervnoj` (VNOJ fork of DMOJ judge, Python). Production v2
(`claoj-docker/claoj/docker-compose.v2.yml`, service `v2_judge`) already
runs the **tiervnoj image**, not `claoj-judge`.

Languages: tiervnoj ships ~60 executors (image adds Java 21, Go, PyPy2/3,
Kotlin, Rust, FPC, Scratch). `claoj-judge` has 9: CPP17, CPP20, C11, PY3,
PY2, JAVA8, GO, NODEJS, RUST. Missing even from v1's minimal fixture set:
C, CPP03, CPP11, CPP14, PAS, PERL, AWK, SED, BF, GAS32, GAS64, TEXT.

Engine features used by the 1,630 problems in `claoj-data/problems`:

| Feature | Problems | `claoj-judge` |
|---|---|---|
| `archive:` zip test data | 1,627 | no (reads only `tests/1.in`…`100.in`) |
| `file_io` (VNOJ) | 421 | no |
| `bridged` C++ checkers | 52 | no |
| custom Python checkers (`*.py`) | ~20 | no; cannot run in Go |
| float / linecount checkers | ~24 | no |
| batches / pretests | 17 / 45 | no |
| output_only / signature grader / generators | 3 / 1 / 3 | no |

It also has no sandbox (`sh -c`, no memory limit, no seccomp/ptrace), no
`main.go`, broken Python/Node run (script path never passed), and an empty
`grading-end`. **Decision: keep tiervnoj as the grading engine; delete the
Go executors/sandbox.**

### 2. The current bridge breaks with more than one backend replica

`bridge.Server` listens in every backend process; a judge dials one address
and lands on one pod. The asynq `judge_submission` task runs on whichever
pod pulls it; if that pod holds no judge, `Submit` returns "no available
judge", retries 3×, then the submission is stuck `QU` forever. Abort only
works on the pod holding the judge. `working` flags are per-process.

### 3. Pre-existing protocol bugs fixed along the way

- Backend sends `submission-abort`; DMOJ judges only understand
  `terminate-submission` (tiervnoj `packet.py:279`). Abort never worked.
- `submission-terminated` is mapped to `IE`; v1 maps it to `AB`
  (`OJ/judge/bridge/judge_handler.py:501`).
- `submission.judged_on_id` is never set.

## Design

```
backend pods (N) ──enqueue──> Redis / asynq queue "judge" <──pull── judge pods (M)
      ^                                                        agent (Go) + 1 tiervnoj child
      └──────── POST /internal/judge/event (backend Service VIP) ──────┘
```

Nothing dials into a judge. Judges are stateless pullers; backends are
stateless HTTP servers. The only shared state is MySQL, Redis, and the
`/problems` volume.

### Components

**Backend (`OJ-v2/claoj`)**

1. `jobs.EnqueueJudgeSubmission(subID)` — same signature, same 6 call
   sites. Loads the submission and builds the existing `submission-request`
   packet (today in `bridge/submit.go`: problem code, language key, source
   or file URL, time/memory limit with `LanguageLimit` override,
   short-circuit, meta). Enqueues it to asynq queue `judge` with
   `asynq.TaskID("sub-<id>")`, `MaxRetry(2)`, `Timeout(30m)`.
   - `ErrTaskIDConflict`: look the task up with the Inspector. If
     `pending`/`active`/`scheduled`/`retry` → no-op (already queued or
     grading). If `archived`/`completed` → `DeleteTask`, enqueue again.
2. Backend's asynq worker (`jobs.StartWorker`) stops handling
   `judge_submission` and does **not** list queue `judge`. Rescore, rating
   and export tasks are unchanged.
3. `Abort(subID)`: Inspector `DeleteTask("judge", "sub-<id>")` if pending;
   `CancelProcessing` if active; otherwise mark `AB` in DB (current
   behaviour).
4. `POST /internal/judge/event` — body is one raw DMOJ packet from the
   judge. Dispatches to the existing handlers in `bridge/handler.go`
   (`onGradingBegin`, `onTestCase`, `onGradingEnd`, `onCompileError`,
   `onInternalError`, plus `submission-terminated` → `AB`,
   `submission-acknowledged` → `P`). Handlers become plain functions
   (no `*Handler` connection state; `free()` disappears).
   - `grading-begin` also sets `judged_on_id` to the authenticated judge row.
   - `test-case-status` inserts become upserts on the existing unique key
     `(submission_id, case)` so a retried POST does not duplicate rows.
   - Auth: HTTP Basic auth, username = judge name, password = judge key
     (Basic auth base64-encodes, so non-ASCII names like "Vịt Toàn Năng"
     are safe), checked against `judge_judge.name` / `auth_key`
     (constant-time) and `is_blocked`. Same trust model as DMOJ today (key over the internal
     network). The `/internal/` prefix is never routed by nginx/ingress.
5. Judge status: a 30 s ticker in each backend pod reads asynq
   `Inspector.Servers()`, counts servers consuming queue `judge`, and
   writes `judge_judge.online` / `load` / `start_time` for the configured
   v2 judge row (`JUDGE_NAME`). Writes are idempotent, so every pod doing
   it is harmless. No heartbeat endpoint.
6. Delete: `bridge/server.go`, `Manager`, `Handler` connection loop,
   `Submit`, `FindAvailableJudge`, the `bridge.addr` config and the
   9999/9998 port exposure, and `bridge/packet.go` (the backend no longer
   speaks DMOJ framing; the agent keeps its own copy).
7. Redis becomes mandatory: `cache.Connect` exits at startup if Redis does
   not answer PING. (Today it only logs a warning, and `cache.Client` is
   never nil, so the in-memory fallback branches in `main.go` are dead code
   — delete them. A backend that starts without Redis silently cannot
   queue, publish events or keep sessions.)

**Judge agent (`OJ-v2/claoj-judge`, rewritten)**

One Go binary, the container entrypoint, ~300 lines:

1. Starts a loopback DMOJ bridge on `127.0.0.1:9999`, spawns one tiervnoj
   child (`/judge/.docker/entry run -p 9999 -c <judge.yml> 127.0.0.1
   <name> <key>`), accepts its `handshake` when the key matches its own,
   answers `ping`. If the child exits, the agent exits non-zero; the
   orchestrator restarts the container.
2. Runs an asynq server: `Concurrency: 1`, `Queues: {"judge": 1}`.
   Handler per task:
   - If `asynq.GetRetryCount(ctx) == MaxRetry` (the judge crashed on this
     submission every previous time), POST `internal-error` "judge crashed
     while grading" and return `nil` — never leave a submission stuck in `G`.
   - Send `submission-request` to the child; forward every packet the child
     emits to `POST /internal/judge/event`, **sequentially**, retrying each
     with backoff until 2xx. Sequential + wait-for-2xx keeps per-submission
     order across load-balanced pods.
   - Stop after a terminal packet: `grading-end`, `compile-error`,
     `internal-error`, `submission-terminated`.
   - On `ctx.Done()` (abort via `CancelProcessing`, or shutdown): send
     `terminate-submission` to the child; still forward the resulting
     terminal packet (use a fresh context for that POST).
3. Config (env): `REDIS_ADDR`, `REDIS_PASSWORD`, `BACKEND_URL`,
   `JUDGE_NAME`, `JUDGE_KEY`, `JUDGE_YML` (default `/problems/judge.yml`).
4. Image: `FROM claoj/judge-tiervnoj` + agent binary as entrypoint.
5. Delete: `executors/`, `core/`, `protocol/` (except the framing code it
   reuses), `INTEGRATION.md`, `MIGRATION_SUMMARY.md`, stale README parts.

All judge replicas share **one** `judge_judge` row and key. Scaling is
`replicas: M`; no admin step per replica.

### Failure behaviour

| Event | Result |
|---|---|
| No judge running | Tasks wait in Redis; submissions stay `QU` (DMOJ semantics). |
| Judge pod killed mid-grade | asynq lease expires (~30 s), task re-queued, regraded from scratch; `grading-begin` wipes old test cases. |
| Same submission kills the judge every time | After `MaxRetry`, agent posts `IE` instead of grading. |
| Backend pod dies mid-POST | Agent retries; Service routes to another pod; upserts keep it idempotent. |
| Graceful judge shutdown (SIGTERM) | asynq re-queues the active task after `ShutdownTimeout`; set container stop grace ≥ that. |
| Redis down | Submit fails loudly (as today); backend refuses to start. |
| Rejudge while queued/grading | No-op (TaskID conflict). |

`ponytail:` a submission regraded after a mid-grade crash may briefly show
partial test cases from the dead attempt until the new `grading-begin`
arrives. Acceptable; same as a manual rejudge.

## Deployment

Both k3s and Swarm:

- `redis`: 1 replica (existing).
- `backend`: Deployment / service, `replicas: N`, no judge ports. Env adds
  `JUDGE_NAME`.
- `judge`: Deployment / service, `replicas: M`, `SYS_PTRACE`, env
  `REDIS_ADDR`, `BACKEND_URL=http://backend:8081`, `JUDGE_NAME`,
  `JUDGE_KEY`. No Service/ports needed.
- `problems` volume: one shared RWX volume (NFS works on both; Longhorn RWX
  on k3s), mounted read-only on judges and read-write on backends.
  - **v2 problem-data upload stays where it is (deferred).** Verified while
    planning: the backend writes uploads to relative
    `data/problems/<code>` (`api/v2/admin/problem_data.go:166`) with a
    generated `init.yml` (`utils.GenerateInitYML`) that has **no
    `test_cases:` and no `archive:`**, and files are `0600` (unreadable by
    tiervnoj's `judge` user). Pointing it at the shared folder would
    overwrite v1-valid `init.yml` files with ungradable ones. Until a
    follow-up makes v2 upload emit a DMOJ-valid problem dir (port of v1's
    `ProblemDataCompiler`), problem data is edited through v1 admin, as
    today.
  - The backend's own `data/` dir (v2 PDFs statements, v2 uploads) must be
    one shared RWX volume across backend replicas, otherwise a PDF uploaded
    via pod A 404s on pod B. Deployment-only change (`v2_data` volume).
  - It is the **same single folder v1 uses** (`/root/claoj-data/problems`;
    same DB, same problem codes). No copies anywhere. Single node:
    bind-mount the host path. Multi-node: NFS-export that path from the
    host that owns it.
  - (For the deferred upload follow-up:) uploads must write to a temp dir
    inside the volume and `rename` it into place, so a judge never reads a
    half-written problem.
  - No problem watcher (`judge_notify`) for v2: tiervnoj's
    `get_problem_root` (`dmoj/judgeenv.py:266`) checks
    `<root>/<code>/init.yml` on disk at grade time, so a new upload is
    gradable on its next submission. (inotify would not fire across NFS
    clients anyway.)
- docker-compose (`claoj-docker/claoj/docker-compose.v2.yml`): `v2_judge`
  switches to the agent image with the env above; `v2_backend` drops the
  `judge` network and the bridge listener. `v1_judge` and v1 `bridged`
  unchanged.

Cutover is a single switch (no dual run: old bridge and queue cannot both
own dispatch). No DB migration, so rollback = previous image tags.

## Testing

- **Agent unit tests** with a fake child that speaks DMOJ packets over
  loopback: forwards packets in order, retries a failing POST, stops on
  each terminal packet, sends `terminate-submission` on cancel, posts `IE`
  on final retry.
- **Backend unit tests**: keep `bridge/bridge_test.go` handler coverage
  (adapted to plain functions); add enqueue (TaskID conflict paths),
  abort (pending / active / none), event endpoint auth (bad key, blocked
  judge), test-case upsert idempotency.
- **One end-to-end compose test**: 2 backend replicas behind nginx, 1
  judge replica with real tiervnoj, problems volume with a few real
  problems (one `archive`, one `file_io`, one `bridged` checker, one
  batched). Submit AC / WA / CE / TLE through both backends; abort a
  queued and a running submission; kill the judge mid-grade and assert the
  submission still finishes; assert final `judge_submission` rows.

## Out of scope

- Native Go grading engine (the agent's task contract keeps the engine
  swappable later).
- Per-problem/language judge routing (all judges run the same image and
  mount the same problems; one queue suffices).
- Syncing `judge_judge_problems` / `judge_judge_runtimes` (v2 never wrote
  them).
- Pre-existing scoring gaps in `onGradingEnd`: batch handling
  (`batch-begin`/`batch-end` ignored) and contest pretest handling. Logged
  here so they are not mistaken for regressions; fix separately.
- Object storage for problem data.
- v1 site and v1 judges.
