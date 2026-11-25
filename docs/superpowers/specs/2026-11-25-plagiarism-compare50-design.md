# Replace MOSS with a self-hosted compare50 fork — design

Date: 2026-11-25 · Status: draft for approval · Branch: `feat/plagiarism-compare50`

## Intent

v2's contest plagiarism check (admin → MOSS) depends on Stanford's remote MOSS
server. We want no dependency on any outside server, while still checking
(nearly) every language CLAOJ judges, with detection as good as MOSS or better.

The chosen tool is **compare50** (CS50, GPLv3), forked to `github.com/CLAOJ/compare50` (used here as a submodule) and tuned
on its source for competitive-programming code.

## Evidence (spike, 2026-11-25)

Run on prod data (read-only), scratch only, nothing committed:

- Prod language mix (all submissions): C++ 161k (85%), Python 24.5k (13%),
  Pascal 2.5k (1.3%), C 269, Java 136, Go 16, Rust 13, Perl 1.
- Contest `tet26`, 6 C++ problems, 12–57 submissions each (same rows the MOSS
  job picks). Old MOSS report URLs had all expired (404), so MOSS was re-run.
  - Time per problem: MOSS 3–9 s (good day), stock compare50 3–8 s locally.
  - 16 of 20 pairs MOSS scored ≥50% were in compare50's top 5. The other 4 all
    involve one 27-line textbook knapsack file that MOSS inflates (MOSS % is
    relative to the smaller file); compare50 ranked them 56–102.
- Pascal problem `hsg12_2021_1` (9 short AC solutions): both found the same real
  pair (97%) at #1. A 16-line copy with every identifier renamed + case/whitespace
  changed was flagged by **neither**.
- Why stock compare50 is weak on short code: its passes use
  `Winnowing(k=25, t=35)` (25-token k-grams), sized for long homework files.
  It also has no "ignore code that appears in many submissions" (MOSS `-m`).
- Prod has no `moss_user_id` misc config, so v2's MOSS run is not usable in
  prod today anyway (v1 keeps `MOSS_API_KEY` in `local_settings.py`).

## Goals

1. Admin contest plagiarism check runs entirely inside our infrastructure.
2. Every language a contest's submissions use gets checked: proper token-level
   comparison for anything Pygments can lex (C, C++, Python, Pascal, Java, Go,
   Rust, Perl, Ruby, Kotlin, …), text-level fallback for the rest.
3. Detection on CLAOJ-style code (short, template-heavy) is **at least MOSS's**,
   proven by a repeatable benchmark (targets below).
4. Reports stay available as long as we keep them (MOSS's expire).

## Non-goals

- Changing v1 (Django) — v1 keeps its own MOSS integration and its
  `judge_contestmoss` table untouched and still in use; "no outside server"
  applies to v2 only.
- Cross-contest / internet-archive comparison (compare50 "archive" support
  exists; not wired now).
- Cross-language detection (C++ vs Python copies).
- Building a v2-native report UI; we serve compare50's HTML report.
- Contributing the fork back upstream.

## Anti-goals (failure even if it "works")

- Contestant source code reachable without the `judge.moss_contest` permission,
  or able to run script in the admin's session (reports render contestant code).
- A failed or partial run destroying the previous results.
- Template/boilerplate code (`bits/stdc++`, fast I/O, personal templates)
  flooding the top of the ranking with innocent pairs.
- Any network call to an outside service at run time.

## Constraints

- Backend image `ghcr.io/claoj/claoj-go` (alpine, Go binary) runs the asynq
  worker that executes the job (`jobs/server.go`). Docker build context is
  `OJ-v2/claoj`, so the fork is mounted inside it as a submodule
  (`claoj/plagiarism`).
- `v2_data` volume is already mounted at `/app/data` in `v2_backend`.
- DB schema is shared with v1, and **v1's MOSS feature is still in use**
  (user, 2026-11-25). v1's run deletes and rewrites all `judge_contestmoss` rows
  of a contest, so v2 must not write that table at all. v2 gets its own table
  (v2-only tables are declared in `scripts/v2_runtime_tables.sql`, no `judge_`
  prefix, and listed as RETAINED-V2 in `cleanup_v2_tables.sql`). The permission
  `judge.moss_contest` is reused as-is.
- Prod `judge_language.common_name` values are all ≤ 6 chars today (longest
  `Golang`); the new table's language column is sized like `common_name`
  (20), so no truncation is needed.
- License: compare50 GPLv3 → fork keeps its LICENSE and a notice of changes;
  compatible with v2's AGPL.
- compare50 imports `lib50`, which imports `termios` (Unix-only) and is only
  used for file globbing and error types.

## Approaches considered

Tool:
- **compare50 (chosen)** — MOSS-style winnowing over Pygments tokens (~500
  languages incl. Pascal), static HTML report, spike showed parity on C++.
- Dolos — best report UI, but no parser for Pascal/Ruby/Perl/Kotlin
  (char-level fallback only). Rejected: Pascal is a real share of CLAOJ.
- JPlag — no Pascal. SIM — Pascal but weak C++/Python. Rejected.
- Own Go winnowing (chroma lexer) — no new runtime, but must build tokenizer
  glue, tuning and a report UI from scratch. Rejected: most work.

Tuning:
- A. Wrapper script registering a custom pass, compare50 stock — rejected by
  the user in favour of B.
- **B. Fork compare50 (own repo, submodule in OJ-v2) and tune its source (chosen by the user)** —
  full control over preprocessors, winnowing, scoring and report.
- C. Ship stock, tune later — rejected: short solutions stay weak.

## Design

Two deliverables, shipped in this order. Part 1 must work with the fork
imported essentially as-is; Part 2 tunes it, gated by the benchmark.

### Part 1 — replace MOSS with the fork

**Fork location.** The fork lives in its own repository,
`github.com/CLAOJ/compare50` (a GitHub fork of `cs50/compare50`; its `main` =
upstream 1.2.15). All our changes go on the branch `claoj` (never `main`: its
CI publishes to PyPI and cuts releases on `main`). OJ-v2 mounts it as a **git
submodule at `claoj/plagiarism`** (`.gitmodules`: https URL, `branch = claoj`),
so the Docker build context `claoj/` contains it and the pinned gitlink commit
is exactly what ships; bumping the submodule commit is how tuning reaches OJ-v2.
`UPSTREAM.md` in the fork lists our changes. First change: drop the `lib50`
dependency (replace its file-glob use with stdlib, its error types with our
own), so it runs on any OS including Windows.

Considered and deferred: running compare50 as its own container behind an HTTP
API. It keeps Python out of the backend image, but needs a new compose service
(in the separate claoj-docker repo), a published image, an authenticated API
for contestants' code and shared storage for reports. The `Checker` interface
in the job makes that a drop-in replacement later.

**Image.** The Dockerfile copies `plagiarism/` and builds a venv at
`/opt/compare50` (`pip install /opt/compare50-src`), with no network access to
GitHub needed at build time. CI's image-build checkout and local builds must
fetch submodules (`submodules: true`; `git submodule update --init`). Keep the
size growth modest (spike estimate ~ +60 MB) and report the actual number.

**Job** (`jobs/moss.go`, keep names/table; internals change):
- For each contest problem × **each `judge_language.common_name` present** in
  qualifying submissions (no fixed 5-language list): same row selection as
  today — live/spectating participations, best-points submission per user.
- Write each submission as `<workdir>/<username>/main.<ext>`, one folder per
  submission (compare50 treats a folder as one submission). `ext` comes from a
  `common_name → extension` map so Pygments picks the right lexer; unknown →
  `.txt` (text-level only).
- Run the fork via `os/exec` with a context timeout, output to a temp dir, all
  ranked pairs (no top-50 cap; builder may set a sane upper bound such as 250
  like MOSS's `show`).
- Reports live at `/app/data/plagiarism/<contest_id>/<problem_code>/<language>/`.
  The whole contest run is written to a temp dir and swapped in only after every
  problem×language succeeded, together with the DB row replacement — same
  "failed run keeps previous results" rule as today.
- Results go to a new v2-only table, `plagiarism_result` (contest_id,
  problem_id, language VARCHAR(20), submission_count, report_path NULL,
  created_at), declared in `scripts/v2_runtime_tables.sql` and added to the
  RETAINED-V2 list in `cleanup_v2_tables.sql`. One row per problem × language,
  `report_path = NULL` when there were no submissions. `judge_contestmoss`
  (v1) is never read or written by v2 after this change.
- Existing running/error Redis keys, single-run-per-contest TaskID, and the
  30-minute task timeout stay.
- Delete: `claoj/moss/` client, `MossUserID`, `ErrMossNotConfigured`, the
  config pre-check in `AdminContestMossRun`.
- Clear (DELETE endpoint) deletes the contest's `plagiarism_result` rows and
  its report directory.
- `models.ContestMoss` stays (v1 table model) only if something else in v2
  still uses it; otherwise remove it from v2.

**Serving reports.**
`GET /api/v2/admin/contest/:key/moss/report/*path` (exact path builder's call),
behind the existing `contestForMoss` permission check. Serves files only from
that contest's report dir (clean the path, reject traversal). Because reports
contain contestant code and inline JS, respond with
`Content-Security-Policy: sandbox allow-scripts` (opaque origin, no access to
the admin's cookies/storage) and `X-Content-Type-Options: nosniff`. The builder
must also confirm the fork HTML-escapes all source/filenames (usernames go in
filenames) and fix it in the fork if not.

**Admin page.** `/admin/moss` keeps its URL and layout; text changes from
"MOSS" to "Plagiarism check" (en + vi); "open report" links go to the new route.
API paths stay `/contest/:key/moss` to keep the diff small. The page shows only
v2's results; v1's MOSS results stay visible in v1's own admin (not mirrored —
add later if admins ask).

### Part 2 — tune the fork + benchmark

**Benchmark** `scripts/plagiarism-bench/` (committed; data never committed,
gitignored, local only — it is contestants' code):
- **Local DB clone** (user, 2026-11-25): the benchmark never talks to prod.
  A one-time, user-approved step dumps only the tables the export needs
  (`judge_submission`, `judge_submissionsource`, `judge_language`,
  `judge_problem`, `judge_contest`, `judge_contestproblem`,
  `judge_contestsubmission`, `judge_contestparticipation`, `judge_profile`,
  and `auth_user` reduced to `id, username` — no password hashes or emails)
  from prod read-only, and loads them into a local MariaDB container. The dump
  file and the container's volume stay outside the repo. Everything after
  that, including tuning by an automated executor, runs against the local
  clone with no prod access.
- `export`: read contests' qualifying submissions from the local clone into
  the same folder layout the job uses.
- `disguise`: from real AC solutions, generate disguised copies per language
  (C++, Python, Pascal at minimum): rename identifiers, reformat/recase,
  reorder functions, swap equivalent types (`int`/`long long`/`int64`), insert
  dead code, swap `for`/`while` where trivial.
- `run`: run the fork and (benchmark-only) MOSS on the same sets; MOSS account
  id taken from an env var, never committed.
- `report`: per language, print the metrics below for MOSS vs fork.

**Targets:**
1. *Firm.* Real-contest agreement: ≥ 90% of pairs MOSS scores ≥ 50% appear in
   the fork's top 10 for that problem, on contests `tet26`,
   `tonghophsg12tinh`, `claojre19`. A MOSS ≥ 50% pair the fork ranks low only
   counts as a miss if a human reading it agrees it looks copied (the spike's
   knapsack case is the example of a pair that should not count).
2. *Firm.* Disguised copies: fork's top-3 recall ≥ MOSS's, per language
   (C++, Python, Pascal).
3. *Goal.* Template noise: the fork's top 10 contains no more template-only
   pairs than MOSS's. "Template-only" is labelled by reading the pairs during
   the interactive tuning session; the labelled list is kept with the
   benchmark results.

**Stop rule.** Tuning iterates until targets 1–2 hold. If they still don't
after a reasonable number of rounds, ship the best variant measured so far and
report the remaining gap to the user rather than continuing open-ended.

**Who runs what.** Building the benchmark tooling and the fork changes can be
handed to an automated executor. The prod dump is a one-time step done by the
user (or Claude in an interactive session with the user's approval). MOSS
baseline runs need the MOSS account id from an env var (never committed);
they send code to Stanford, as v1 already does, and only run during
benchmarking.

**Tuning levers in the fork** (builder iterates with the benchmark; keep
what moves the numbers, drop what doesn't):
- Smaller k-gram / window for the structure pass (k=25 is too large for
  15–40-line solutions); possibly adaptive to file length.
- Frequency ignore like MOSS `-m`: per problem run, fingerprints present in
  more than N submissions (MOSS uses 10) don't count.
- CP preprocessors: drop `#include` / `using` / `#define` lines and common
  fast-I/O statements; normalize equivalent integer/float types across
  languages (`int`, `long long`, `int64`, `longint`, …).
- Score shown in the report: add a MOSS-style per-file percentage next to
  compare50's score, so admins can judge short files.

## Left to the builder

- apk vs pip for Python deps; exact image size result.
- Exact report route path and how the admin page links to it.
- Upper bound on ranked pairs per report.
- Disguise transformations' implementation per language.
- Exact tuned parameters — whatever the benchmark supports.
