# Replace MOSS with a compare50 fork — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** v2's contest plagiarism check runs on a forked, CP-tuned compare50 inside the backend image instead of Stanford's MOSS server.

**Architecture:** The compare50 fork lives in its own repo (`github.com/CLAOJ/compare50`, branch `claoj`), mounted in OJ-v2 as a git submodule at `claoj/plagiarism` and installed in a venv in the backend image. The existing asynq job (`jobs/moss.go`) writes each qualifying submission to `<user>/main.<ext>`, runs compare50 per problem × language, swaps the finished reports into `data/plagiarism/<contest_id>/`, and records rows in a new v2-only table `plagiarism_result`. A permission-checked route serves the reports in a CSP sandbox. Part 2 adds an offline benchmark (`scripts/plagiarism-bench/`) against a local DB clone and tunes the fork until it meets the spec's targets.

**Tech Stack:** Go 1.25 (gin, gorm, asynq, testify, sqlite for tests), Python 3 (compare50 fork, stdlib `unittest`), Next.js (claoj-web, next-intl), Docker (alpine), MariaDB.

**Spec:** `docs/superpowers/specs/2026-11-25-plagiarism-compare50-design.md`

## Global Constraints

- No network call to any outside service at run time (the MOSS baseline only runs in the benchmark).
- v2 never reads or writes `judge_contestmoss`. v1's MOSS stays in use and untouched.
- v2 never runs DDL. New tables go in `scripts/v2_runtime_tables.sql` (idempotent `CREATE TABLE IF NOT EXISTS`), are listed as RETAINED-V2 in `scripts/cleanup_v2_tables.sql`, and get an entry in `docs/schema-audit.md` section 3.
- Reuse the permission `judge.moss_contest` and the existing API paths `/api/v2/admin/contest/:key/moss`.
- Fork: lives in `github.com/CLAOJ/compare50`, branch `claoj`; **never push `main` there** (its CI publishes to PyPI), never force-push. Keep upstream `LICENSE` (GPLv3); `UPSTREAM.md` in the fork records upstream version 1.2.15 and lists every change we make. OJ-v2 consumes it as the submodule `claoj/plagiarism`, pinned by gitlink commit.
- A failed or partial run keeps the previous rows *and* the previous report files.
- Reports are served with `Content-Security-Policy: sandbox allow-scripts` and `X-Content-Type-Options: nosniff`.
- Benchmark data (contestants' code, DB dumps) is never committed; the MOSS account id comes from env var `MOSS_USER_ID` and is never committed.
- Admin UI text: "MOSS" → "Plagiarism check" (en) / "Kiểm tra đạo văn" (vi).
- Commit messages use a heredoc (`git commit -F - <<'EOF'`); check with `git log -1 --format=%s`.

## Review Focus

1. A problem × language with exactly one qualifying submission: no compare50 run, row stored with `submission_count = 1` and `report_path = NULL` (Task 3 test).
2. Usernames Django allows but a filesystem treats specially (`.`, `..`, Unicode like `tik25_lêthanh`) used as folder names: `.`/`..` are prefixed, Unicode kept (Task 3 test).
3. Report route with traversal (`../`, `%2e%2e/`, absolute paths) or another contest's id: 404, never a file outside the contest's dir (Task 4 test).
4. Contestant source or username containing `</script><script>alert(1)</script>`: appears escaped in the report HTML (Task 1 test), and the route sets the sandbox CSP (Task 4 test).
5. A v2 run on a contest that also has v1 MOSS rows: `judge_contestmoss` rows are unchanged afterwards (Task 3 test).

---

## Part 1 — replace MOSS

### Task 1: Drop lib50 in the compare50 fork, mount it as a submodule

**Files** (fork clone `F:\Coding\CLAOJ\compare50`, remote `origin` = `https://github.com/CLAOJ/compare50`; create branch `claoj` from `main`):
- Modify: `setup.py` (remove `lib50` from `install_requires`), `compare50/__main__.py`, `compare50/_api.py` (and any other file that imports `lib50`)
- Create: `UPSTREAM.md`, `tests/cli_tests.py` (follow the style of the existing `tests/*_tests.py`; they run with `python -m tests`)

Then in the OJ-v2 worktree: `.gitmodules`, gitlink `claoj/plagiarism`.

**Interfaces:**
- Produces: `python -m compare50 <sub_dir>... -o <out_dir> -n <max_pairs>` writes `<out_dir>/index.html` plus `match_N.html`; exit code 0 on success. Each `<sub_dir>` is one submission; its folder name is the label shown in the report. The parent of `<out_dir>` must exist (compare50 creates `<out_dir>` itself).
- Produces: OJ-v2 path `claoj/plagiarism/` = the fork checkout (contains `setup.py`), pinned by the gitlink commit. Task 6 consumes it.

- [ ] **Step 1: Write failing tests** in the fork's `tests/cli_tests.py` (stdlib `unittest`, run compare50 via `subprocess` with `sys.executable -m compare50`):
  - `test_ranks_copied_pair_first`: three dirs `a/main.cpp`, `b/main.cpp` (copy of `a` with renamed identifiers), `c/main.cpp` (unrelated program, >= 40 lines each); assert exit 0, `index.html` exists, and the first ranked pair in its `GRAPH` JSON is `{a, b}`.
  - `test_escapes_source_and_names`: a submission folder named `x<b>y` whose source contains `</script><script>alert(1)</script>`; assert neither raw string appears in any output `.html`.
  - `test_imports_without_lib50`: in a subprocess with `sys.modules['lib50'] = None` inserted before import, `import compare50.__main__` succeeds.
  Make sure `python -m tests` still discovers and runs them (extend `tests/__main__.py` only if needed).
- [ ] **Step 2: Run** in a Linux container: `docker run --rm -v <fork clone>:/w -w /w python:3.12-slim sh -c "pip install -q -e . && python -m tests"`. Expected: `test_imports_without_lib50` FAILS (and the escaping test fails if upstream does not escape). Note which pre-existing tests pass before you change anything.
- [ ] **Step 3: Remove lib50.** Replace `lib50.files(patterns, root=dir)` with a stdlib walk (`pathlib.Path.rglob` with `fnmatch` include/exclude) that returns the same `(included, excluded)` sets; replace `lib50.Error`/`lib50.config.TaggedValue` with local equivalents in `compare50/_api.py`. Fix escaping in the renderer templates if Step 2 showed it missing (Jinja autoescape on for `.html`). Record both changes in `UPSTREAM.md` (upstream version 1.2.15, list of our changes).
- [ ] **Step 4: Run the tests again.** Expected: all new and all pre-existing tests PASS on Linux (container) and on Windows (`python -m venv` + `pip install -e . && python -m tests`). If compare50 needs another Unix-only dependency to run on Windows, report it; Linux is the required platform.
- [ ] **Step 5: Commit and push the fork branch.** In the fork clone: `git commit` (heredoc), subject `feat: drop lib50 so compare50 runs without termios`, then `git push -u origin claoj`. Never push `main`, never force-push. Record the pushed HEAD SHA.
- [ ] **Step 6: Add the submodule in the OJ-v2 worktree.** `git submodule add -b claoj https://github.com/CLAOJ/compare50 claoj/plagiarism`, check it is at the SHA from Step 5 (`git submodule status`), and add one line to `docs/development.md` (clone with `--recurse-submodules`, or run `git submodule update --init`). Commit `build: add the compare50 fork as a submodule` (stage `.gitmodules`, `claoj/plagiarism`, `docs/development.md`).

### Task 2: `plagiarism_result` table and model

**Files:**
- Modify: `scripts/v2_runtime_tables.sql`, `scripts/cleanup_v2_tables.sql` (RETAINED-V2 comment list), `docs/schema-audit.md` (section 3)
- Create: `claoj/models/plagiarism.go`

**Interfaces:**
- Produces:
  ```go
  type PlagiarismResult struct {
      ID              uint      `gorm:"primaryKey;column:id"`
      ContestID       uint      `gorm:"column:contest_id;not null;index"`
      ProblemID       uint      `gorm:"column:problem_id;not null"`
      Language        string    `gorm:"column:language;size:20;not null"`
      SubmissionCount uint      `gorm:"column:submission_count;not null;default:0"`
      ReportPath      *string   `gorm:"column:report_path;size:255"`
      CreatedAt       time.Time `gorm:"column:created_at;not null"`
  }
  func (PlagiarismResult) TableName() string { return "plagiarism_result" }
  ```
  `ReportPath` is relative to the contest's report dir, e.g. `knap/cpp`.

- [ ] **Step 1: Add the model** above.
- [ ] **Step 2: Add the DDL** to `v2_runtime_tables.sql` matching the model (`BIGINT UNSIGNED` id, `KEY idx_plagiarism_result_contest_id (contest_id)`, utf8mb4_unicode_ci, header comment "Plagiarism check results (jobs/moss.go), v2-only; v1 MOSS uses judge_contestmoss"). Add `plagiarism_result` to the RETAINED-V2 list in `cleanup_v2_tables.sql` and to `schema-audit.md` section 3.
- [ ] **Step 3: Verify** `cd claoj && go build ./...` succeeds, and the SQL applies twice cleanly to a scratch MariaDB (`docker run -d --name pr-test -e MARIADB_ALLOW_EMPTY_ROOT_PASSWORD=1 -e MARIADB_DATABASE=t mariadb`, then pipe just the new statement in twice).
- [ ] **Step 4: Commit** `feat(models): add v2-only plagiarism_result table`.

### Task 3: Job runs compare50 instead of MOSS

**Files:**
- Modify: `claoj/jobs/moss.go`, `claoj/jobs/moss_test.go`
- Create: `claoj/jobs/compare50.go`
- Delete: `claoj/moss/` (whole package)

**Interfaces:**
- Consumes: `models.PlagiarismResult` (Task 2).
- Produces:
  ```go
  // Checker writes one HTML report for the submission folders under subsDir into outDir.
  type Checker interface { Check(ctx context.Context, subsDir, outDir string) error }

  var PlagiarismDir = filepath.Join("data", "plagiarism")      // swapped in tests
  var Compare50Python = "/opt/compare50/bin/python"             // swapped in tests

  type Compare50 struct{ MaxPairs int }                         // implements Checker; MaxPairs default 250
  func LanguageExt(commonName string) string                    // "C++"→"cpp", unknown→"txt"
  func LanguageSlug(commonName string) string                   // "C++"→"cpp", "C#"→"csharp"
  func ContestReportDir(contestID uint) string                  // filepath.Join(PlagiarismDir, strconv.Itoa(id))
  func RunContestMoss(tx *gorm.DB, contestID uint, chk Checker) (int, error)
  ```
  `Compare50.Check` runs `Compare50Python -m compare50 <each subdir of subsDir> -o outDir -n MaxPairs` with `exec.CommandContext`; on non-zero exit it returns an error containing the last 2 KB of stderr.

Extension map (`LanguageExt`): C→c, C++→cpp, Python→py, Pascal→pas, Java→java, Golang→go, Rust→rs, Perl→pl, Ruby→rb, Kotlin→kt, C#→cs, PHP→php, JavaScript→js, Haskell→hs, Scala→scala, Swift→swift, Lua→lua, D→d; anything else → txt. `LanguageSlug`: lowercase, `+`→`p`, `#`→`sharp`, any other non-`[a-z0-9]` → `_`.

- [ ] **Step 1: Rewrite tests in `moss_test.go`.** Drop `fakeSubmitter`, `TestMossUserID`, `MiscConfig` and add `PlagiarismResult` to `setupMossDB`. Add a `fakeChecker` that records `subsDir` contents (folder → file name → content) and writes `outDir/index.html`, optionally returning an error. Set `PlagiarismDir = t.TempDir()` in each test. Tests:
  - `TestRunContestMoss_WritesReportsAndRows`: seed as today (alice 100 + 40, bob spectate, carol virtual) plus a dave C++ submission so C++ has 3 users. Assert one checker call, with folders `alice/main.cpp` ("best alice"), `bob/main.cpp`, `dave/main.cpp` (carol excluded); one row per *language present* (C++ only), `submission_count = 3`, `report_path = "knap/cpp"`; file `ContestReportDir(c.ID)/knap/cpp/index.html` exists.
  - `TestRunContestMoss_SingleSubmissionSkipsChecker`: only alice submits Python; no checker call for Python; row with `submission_count = 1`, `report_path = nil`.
  - `TestRunContestMoss_FailureKeepsOldRowsAndFiles`: pre-existing row + `ContestReportDir/old/index.html`; checker returns error; assert error returned, old row unchanged, old file still there, no `.tmp-*` dirs left in `PlagiarismDir`.
  - `TestRunContestMoss_LeavesV1TableAlone`: insert a `models.ContestMoss` row for the contest; run successfully; the row is unchanged.
  - `TestRunContestMoss_SafeFolderNames`: users `.`, `..`, `tik25_lêthanh`; folders are `_.`, `_..`, `tik25_lêthanh`, all inside `subsDir`.
  - `TestLanguageExtAndSlug`: table test over the maps above, including unknown → `txt` and `C#` → `csharp`.
  - Keep `TestMossRunError_*` as they are.
- [ ] **Step 2: Run** `cd claoj && go test ./jobs/ -run 'Moss|Language' -v`. Expected: compile FAIL (`Checker`, `PlagiarismDir` undefined).
- [ ] **Step 3: Implement** `compare50.go` (`Checker`, `Compare50`, `LanguageExt`, `LanguageSlug`, `ContestReportDir`) and rewrite `RunContestMoss`:
  - Same submission query as today, but group by `l.common_name` for every language present (drop `MossLanguages`).
  - Per contest run, build everything in `tmp := PlagiarismDir/.tmp-<contestID>-<unixnano>`: submissions in `tmp/subs/<code>/<slug>/<user>/main.<ext>`, reports in `tmp/report/<code>/<slug>/`. Call the checker only when ≥ 2 users; folder names `.`/`..` get a `_` prefix. `defer os.RemoveAll(tmp)`.
  - On success: rename `ContestReportDir` → `PlagiarismDir/.old-<id>-<nanos>` (if it exists), rename `tmp/report` → `ContestReportDir`, then in a DB transaction delete the contest's `plagiarism_result` rows and insert the new ones. If the transaction fails, rename both back. Finally `RemoveAll` the `.old` dir.
  - `HandleContestMossTask` passes `Compare50{}` and no longer reads `moss_user_id`. Delete `MossUserID`, `ErrMossNotConfigured`, `MossSubmitter` and the `claoj/moss` package. Keep `models.ContestMoss`: no v2 production code uses it any more, but `TestRunContestMoss_LeavesV1TableAlone` does.
- [ ] **Step 4: Run** `go test ./jobs/ -v` and `go vet ./...`. Expected: PASS, and no remaining imports of `github.com/CLAOJ/claoj/moss`.
- [ ] **Step 5: Commit** `feat(jobs): run the compare50 fork for contest plagiarism checks`.

### Task 4: API — list, run, delete, serve report

**Files:**
- Modify: `claoj/api/v2/admin_contest_moss.go`, `claoj/api/v2/admin_contest_moss_test.go`, `claoj/api/router.go:155-158`

**Interfaces:**
- Consumes: `jobs.ContestReportDir`, `models.PlagiarismResult`.
- Produces:
  - `GET /admin/contest/:key/moss` → `{"running": bool, "error": string, "results": [{"problem_code","problem_name","language","submission_count","report_path": string|null}]}` (field `url` replaced by `report_path`).
  - `POST` unchanged response; no config pre-check.
  - `DELETE` → `{"deleted": n}`; also `os.RemoveAll(jobs.ContestReportDir(id))`.
  - New `GET /admin/contest/:key/moss/report/*path` → `AdminContestMossReport`: serves a file from `ContestReportDir(contest.ID)` with the CSP sandbox and nosniff headers.

- [ ] **Step 1: Update tests.** Switch fixtures from `ContestMoss` to `PlagiarismResult`; drop the "needs config" half of `TestContestMoss_RunNeedsConfigAndEnqueues` (rename `TestContestMoss_RunEnqueues`). Add:
  - `TestContestMoss_ListReturnsReportPath`: row with `report_path "knap/cpp"` → JSON has `report_path: "knap/cpp"`.
  - `TestContestMoss_DeleteRemovesReportDir`: file under `ContestReportDir`; after DELETE the dir is gone and the rows are deleted.
  - `TestContestMoss_ReportServesWithSandbox`: `index.html` under `ContestReportDir(c.ID)/knap/cpp/`; `GET .../moss/report/knap/cpp/index.html` → 200, body matches, header `Content-Security-Policy: sandbox allow-scripts`, `X-Content-Type-Options: nosniff`.
  - `TestContestMoss_ReportRejectsTraversal`: for `../1/x.html`, `%2e%2e/%2e%2e/secret`, `/etc/passwd`, and a path into another contest's id dir → 404.
  - `TestContestMoss_ReportRequiresPermission`: staff without the permission → 403.
  - Each test sets `jobs.PlagiarismDir = t.TempDir()`.
- [ ] **Step 2: Run** `go test ./api/v2/ -run ContestMoss -v`. Expected: FAIL.
- [ ] **Step 3: Implement.** Report handler: `filepath.Clean("/"+c.Param("path"))`, join under the contest dir, and reject if `filepath.Rel` starts with `..`; use `c.File`. Register the route next to the others with `auth.RequireContestEdit()`.
- [ ] **Step 4: Run** `go test ./api/... -v` and `go build ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(api): serve plagiarism reports behind the MOSS permission`.

### Task 5: Admin page

**Files:**
- Modify: `claoj-web/src/lib/adminApi.ts:952-970`, `claoj-web/src/app/[locale]/admin/moss/page.tsx:184-186`, `claoj-web/src/i18n/en.json`, `claoj-web/src/i18n/vi.json` (`Admin.moss.*`), `claoj-web/src/app/[locale]/admin/layout.tsx` (nav label only)

**Interfaces:**
- Consumes: Task 4's list response.
- Produces: `ContestMossResult.report_path: string | null` (replaces `url`); `adminMossApi.reportUrl(key: string, reportPath: string): string` → `` `${api.defaults.baseURL}/admin/contest/${key}/moss/report/${reportPath}/index.html` `` (same pattern as `getRssUrl` in `lib/api.ts`).

- [ ] **Step 1: Implement** the type change and `reportUrl`; the page's link uses `adminMossApi.reportUrl(contestKey, r.report_path)` when `report_path` is set. Change the `moss.*` title, subtitle and nav strings to "Plagiarism check" / "Kiểm tra đạo văn"; drop any copy that mentions Stanford or a MOSS account id.
- [ ] **Step 2: Verify** `cd claoj-web && npx tsc --noEmit && npm run lint` pass, and `grep -ri "stanford\|moss_user_id" src/` returns nothing.
- [ ] **Step 3: Commit** `feat(admin-web): plagiarism check page uses local reports`.

### Task 6: Backend image and CI install the fork

**Files:**
- Modify: `claoj/Dockerfile`, `.github/workflows/ci-cd.yml` (the `actions/checkout@v4` step of the backend image-build job, the one before `context: claoj`: add `with: { submodules: true }`)
- Create: `claoj/.dockerignore` only if none exists (exclude `plagiarism/.git`, `plagiarism/tests`, `**/__pycache__`)

**Interfaces:**
- Consumes: `jobs.Compare50Python = "/opt/compare50/bin/python"` (Task 3); the `claoj/plagiarism` submodule (Task 1).

- [ ] **Step 1: Implement.** Builder stage (same alpine base as runtime): `apk add python3`, `COPY plagiarism /src/compare50`, `python3 -m venv /opt/compare50 && /opt/compare50/bin/pip install --no-cache-dir /src/compare50`. Runtime stage: `apk add --no-cache python3` and `COPY --from=<that stage> /opt/compare50 /opt/compare50`. If numpy has no musl wheel for the alpine Python version, switch the runtime base to a Python-capable alternative and note it in the commit.
- [ ] **Step 2: Verify:**
  - `docker build -t claoj-go:plag claoj` (submodule checked out); record the image size change against `ghcr.io/claoj/claoj-go:latest` (`docker images`).
  - Smoke test: `docker run --rm -v <dir with 3 submission folders>:/s claoj-go:plag sh -c '/opt/compare50/bin/python -m compare50 /s/a /s/b /s/c -o /tmp/r -n 250 && ls /tmp/r/index.html'` prints the path.
  - `ci-cd.yml` still parses as YAML (`python -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci-cd.yml'))"`).
- [ ] **Step 3: SKIPPED by user instruction** (end-to-end check through the local compose stack; needs a seeded local DB). The final report must list it as not done. If a later run does it: open a report; match pages must navigate with no sandbox console errors; if links open new tabs and get blocked, add `allow-popups` to the CSP and to Task 4's test.
- [ ] **Step 4: Commit** `build(backend): install the compare50 submodule in the image`.

**Deploy note (for the user, not the executor):** before deploying, apply the `plagiarism_result` statement from `scripts/v2_runtime_tables.sql` to the prod DB.

---

## Part 2 — benchmark and tuning

### Task 7: Local DB clone and export

**Files:**
- Create: `scripts/plagiarism-bench/clone_from_prod.sh`, `scripts/plagiarism-bench/export.py`, `scripts/plagiarism-bench/requirements.txt` (`pymysql`, `mosspy`), `scripts/plagiarism-bench/.gitignore` (`data/`, `dumps/`, `results/raw/`), `scripts/plagiarism-bench/README.md`
- Test: `scripts/plagiarism-bench/tests/test_export.py`

**Interfaces:**
- Produces: `export.py --contest <key> --out data/<key>`, which writes `data/<key>/<problem_code>/<lang_slug>/<user>/main.<ext>` with the same selection rules, ext map and slug as Task 3 (best-points submission per user, participation `virtual IN (0,-1)`). Local DB connection from env `BENCH_DB_DSN` (default `root@127.0.0.1:3307/claoj_bench`).

- [ ] **Step 1: Write `clone_from_prod.sh`** (user-run; never by an executor). It runs read-only over `ssh root@claoj` + `docker exec claoj_db`:
  - `mariadb-dump --single-transaction --skip-lock-tables` of `judge_language judge_problem judge_contest judge_contestproblem judge_contestparticipation judge_contestsubmission`;
  - `judge_submission` with `--where="contest_object_id IS NOT NULL"`;
  - `judge_submissionsource` with `--where="submission_id IN (SELECT id FROM judge_submission WHERE contest_object_id IS NOT NULL)"`;
  - `auth_user` reduced to `id, username` and `judge_profile` to `id, user_id`, exported as TSV via `SELECT`.
  It saves everything to `dumps/`, then starts `mariadb` locally on port 3307 (container `claoj-bench-db`, volume `claoj-bench-db`) and loads it.
- [ ] **Step 2: Write failing test** `test_export.py`: against an in-memory sqlite built from a small fixture with the same columns, assert the folder layout, best-per-user selection, and virtual exclusion (`export.select_rows` takes a DB-API connection, so sqlite works).
- [ ] **Step 3: Implement `export.py`; run** `python -m unittest discover -s scripts/plagiarism-bench/tests -v`. Expected: PASS.
- [ ] **Step 4: Commit** `feat(bench): local DB clone and contest export`.

### Task 8: Disguise generator

**Files:**
- Create: `scripts/plagiarism-bench/disguise.py`
- Test: `scripts/plagiarism-bench/tests/test_disguise.py`

**Interfaces:**
- Produces: `disguise.py --src data/<key> --out data/<key>-disguised --per-problem 3 --seed 1`. It picks up to 3 solutions per problem × language and adds `<user>__copy/main.<ext>` next to the originals, with `manifest.json` listing `{problem, lang, original, copy, transforms}`. Transforms (per language: C++, Python, Pascal): `rename_identifiers`, `reformat` (whitespace/recase; recase only for Pascal), `reorder_functions`, `swap_int_types` (`int`↔`long long`, `integer`↔`longint`), `insert_dead_code`, `for_to_while` (C++ only, simple counted loops).

- [ ] **Step 1: Write failing tests**, one per transform per language, on small fixed inputs. Assert the output differs from the input and is still valid: C++ compiles with `g++ -fsyntax-only` when available (skip otherwise), and Python passes `compile()`.
- [ ] **Step 2: Implement** with Pygments' tokenizer (already a fork dependency) for identifier renaming, so keywords and builtins stay untouched.
- [ ] **Step 3: Run** the tests. Expected: PASS.
- [ ] **Step 4: Commit** `feat(bench): disguised-copy generator`.

### Task 9: Runner, metrics and baseline

**Files:**
- Create: `scripts/plagiarism-bench/run.py`, `scripts/plagiarism-bench/report.py`, `scripts/plagiarism-bench/labels/template_pairs.json`, `scripts/plagiarism-bench/RESULTS.md`
- Test: `scripts/plagiarism-bench/tests/test_report.py`

**Interfaces:**
- Produces:
  - `run.py --data data/<key> --tool {fork,moss} --out results/raw/<tool>/<key>` saves each tool's ranked pairs as `pairs.json`: `[{problem, lang, a, b, score, pct}]`. The fork's pairs are parsed from `index.html` `GRAPH` JSON; MOSS's are fetched with mosspy, using `MOSS_USER_ID`.
  - `report.py` prints the 3 spec metrics per language for both tools:
    - **target 1:** the share of MOSS ≥ 50% pairs that land in the fork's top 10, ignoring pairs listed under `"not_copied"` in `labels/template_pairs.json`;
    - **target 2:** top-3 recall of the `manifest.json` copies;
    - **target 3:** the count of top-10 pairs listed under `"template_only"`.

- [ ] **Step 1: Write failing tests** for the metric functions with hand-made `pairs.json` / manifest / labels fixtures and known answers.
- [ ] **Step 2: Implement; run** the tests. Expected: PASS.
- [ ] **Step 3: Baseline (interactive, with the user).** Export `tet26`, `tonghophsg12tinh`, `claojre19`, run the disguise step, then run both tools and `report.py`. Read the top pairs and fill in `labels/template_pairs.json`, then write the numbers to `RESULTS.md` as "baseline: stock fork vs MOSS". No source code goes in `RESULTS.md`.
- [ ] **Step 4: Commit** `feat(bench): runner, metrics and baseline results`.

### Task 10: Tune the fork

**Files:**
- Modify, in the fork repo (branch `claoj`): `compare50/passes.py`, `compare50/preprocessors.py`, `compare50/comparators/_winnowing.py`, renderer templates (percentage column), `UPSTREAM.md`; in OJ-v2: the `claoj/plagiarism` gitlink (bump after each kept lever), `scripts/plagiarism-bench/RESULTS.md`
- Test: `tests/tuning_tests.py` in the fork repo

Work through one lever at a time. For each: write a unit test, implement the lever, re-run the benchmark, and record the numbers in `RESULTS.md`. Keep the lever only if target 1 or 2 improves without regressing the other. Otherwise revert it and record why.

- [ ] **Lever A — k-gram size.** Test: a 16-line Pascal program and its fully renamed copy share ≥ 1 fingerprint under the structure pass (fails at `k=25`). Try `k` ∈ {12, 15, 18}, with `t = k + 10`, or adaptive to the shorter file's token count.
- [ ] **Lever B — frequency ignore (MOSS `-m`).** Test: with 12 submissions sharing an identical 30-line template plus distinct bodies, the shared template contributes 0 to any pair's score when the threshold is 10. Implement in the winnowing comparator: drop fingerprints present in more than `max_matches` submissions of the run (default 10, CLI flag `--max-matches`).
- [ ] **Lever C — CP preprocessors.** Tests: `#include`, `using namespace`, `#define` lines and `ios::sync_with_stdio`/`cin.tie` statements produce no tokens; `int`/`long long`/`int64_t`/`longint`/`int64` normalize to one token. Add them to the structure pass.
- [ ] **Lever D — percentage in the report.** Test: the rendered `index.html` contains, for each pair, the shared-fingerprint share of each file (MOSS-style %). This doesn't affect ranking; always keep it.
- [ ] **Stop rule:** finish when targets 1 and 2 hold on all three contests. If they still miss after levers A–C plus one combined re-tune, stop, keep the best variant, and record the gap in `RESULTS.md` for the user.
- [ ] **Final:** run `python -m tests` for the fork, Task 1's tests again, and `go test ./jobs/ ./api/...`; update `UPSTREAM.md` with the kept levers; rebuild the image (Task 6 Step 2 smoke test).
- [ ] **Commit** after each kept lever: `perf(plagiarism): <lever> (target1 x%→y%, target2 x%→y%)`.
