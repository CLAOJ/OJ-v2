# AI submission detection in the plagiarism check — design

Date: 2026-11-26 · Status: draft for approval · Builds on
`2026-11-25-plagiarism-compare50-design.md` (the compare50 fork, the contest
plagiarism job, the report route).

## Intent

The admin plagiarism check only finds contestants who copied **each other**.
A contestant who pastes LLM output and shares it with nobody never shows up.
We want the same admin page to also surface contestants who likely submitted
AI-written code, so admins can review them after a contest.

Research (2026-11-26, summarised in the conversation) found:

- AI-code classifiers ("is this AI-written?") are not reliable enough: in a
  2026 study dedicated detectors scored F1 0 on every generator; no study
  reports usable false-positive rates, and none covers short CP code.
- What does work, and what the one confirmed Codeforces catch used, is
  evidence the judge already has: similarity to AI-generated solutions of
  the same problem, how a contestant's failed attempts change, and how the
  result compares with their history.
- Every source agrees none of it is proof; interviews/manual review decide.

## Goals

1. `/admin/plagiarism` shows two tabs for a contest: **Code copy** (today's
   compare50 table, unchanged) and **AI submissions** (new).
2. The AI tab is a ranked list of suspect (contestant × problem) rows, each
   showing **why** it was flagged, with a link to the evidence.
3. Three signals in this version (approved by the user):
   - **S1 — match against AI reference solutions.**
   - **S3 — wrong-answer rewrite pattern.**
   - **S4 — performance outlier.**
4. One **Run check** runs both tabs' checks in the existing job.

## Non-goals

- S2, style drift against the contestant's own past code (later version).
- Any "is this AI-written?" classifier or LLM verdict on a single submission.
- Keystroke logging, problem-open tracking, proctoring.
- Automatic penalties or disqualification.
- Aggregating a contestant's flags across contests (later; the Codeforces
  proposal relies on it, so keep results in a form that allows it).
- Changes to v1 (Django).

## Anti-goals (failure even if it "works")

- A flag reads as a verdict. The UI words it as "suspect / review", shows the
  evidence, and says that matches are not proof.
- Easy problems flood the list: on a 10-line problem most honest solutions
  look like the AI's. A row must not be flagged only because the problem has
  one obvious solution (see S1 "trivial problems").
- Contestant usernames or code beyond what already goes out leave the server.
- A failed LLM call or a missing statement fails the run or destroys the
  previous results.

## Constraints

- Everything from the compare50 design still holds: job in
  `claoj/jobs/moss.go`, reports under `/app/data/plagiarism/<contest_id>/`,
  atomic swap of the whole contest's results, `judge.moss_contest`
  permission, report route with sandbox CSP, v2-only tables declared in
  `scripts/v2_runtime_tables.sql` + RETAINED-V2 in `cleanup_v2_tables.sql`,
  no `judge_` prefix, v1 tables untouched.
- LLM access reuses the existing `COMPARE50_LLM_URL` / `_KEY` / `_MODEL` env
  (OpenAI-compatible `/chat/completions`). Without them, generation is off
  and everything else still works.
- **New data leaving the server:** problem statement text goes to the
  configured LLM (approved with approach C). Contestant code still only goes
  out through the existing pair review, never with usernames.
- CLAOJ stores no "problem opened" time. Timing uses submission time
  relative to the participation's start.
- Some problems have their statement only as a PDF (`judge_problem.pdf_url`,
  little or no `description` text).
- Job timeout stays 30 minutes.

## Approaches considered

Signals: S1 + S3 + S4 chosen by the user; S2 deferred (largest effort: a
style fingerprint for C++, Python and Pascal).

Source of AI reference solutions for S1:
- A. Generate automatically from the statement at check time — no admin
  effort; costs LLM calls every run, sends statements out, skips PDF-only
  problems.
- B. Admins paste AI solutions they generated themselves — no statements out,
  no cost, PDF works; tedious and easy to forget.
- **C. A + B (chosen by the user).** Generate when the statement has text,
  store the generated solutions so re-runs reuse them, and let admins add
  their own (this is how PDF-only problems get covered).

S3 and S4 have one sensible build: computed in Go inside the job from the
database; they need neither compare50 nor an LLM.

## Design

### Reference solutions (S1 input)

New v2-only table `plagiarism_ai_reference`:
`id, problem_id, language VARCHAR(20) (= judge_language.common_name),
source LONGTEXT, origin VARCHAR(10) ('generated' | 'admin'),
model VARCHAR(100) NULL, created_by (profile id) NULL, created_at`.
Keyed by **problem**, not contest, because problems are reused across
contests (my call).

**Generation** (in the Go job, before matching; my call to do it in Go rather
than in the Python fork, because the job owns the DB and the problems):
- For each contest problem × language that has qualifying submissions: if
  the LLM is configured, the statement has usable text, and fewer than
  **5** `generated` references exist for that problem × language, ask for the
  missing ones.
- Prompt: the statement text (`description`, markdown/LaTeX as stored), the
  language, time/memory limits; "write a complete accepted solution, code
  only". Vary the sampling (temperature > 0 and/or separate calls) so the 5
  are not identical. Strip code fences.
- "Usable text": the builder picks a sensible minimum length; PDF-only
  problems fall below it and are reported as "no statement text".
- Parallel calls like the fork (4 at a time), with the same retry/backoff on
  429/5xx. A failed generation is recorded for the report and never fails
  the run.
- Generated code is not verified by judging it (my call: wrong AI code is
  still what a cheater would paste; judging it is a later improvement).

**Admin references**: admins can add (paste, pick language) and delete
references per problem, and delete all `generated` ones for a problem to
force regeneration. API under the existing contest plagiarism routes, same
permission, scoped to the contest's problems.

### S1 — match against references

- For each problem × language with ≥ 1 submission and ≥ 1 reference, run
  compare50 a **second time**: contestants' folders as submissions, each
  reference as an **archive** submission (`-a`; archives are compared with
  submissions, not with each other). Report goes to
  `<problem>/<lang>/ai/` inside the contest report dir, so admins can open
  the side-by-side match page. LLM pair verdicts are off for this run.
- Separate run rather than adding references to the code-copy run (my call),
  so AI matches don't push contestant-vs-contestant pairs out of the
  code-copy top N.
- The fork needs a machine-readable result of each run: for every pair, the
  two submission names, the match page, and an **absolute** similarity
  (share of the submission's fingerprints found in the reference, 0–100%),
  not compare50's score that is relative to the best pair of the report.
  The fork already writes `llm.json` next to reports; add a similar
  `results.json`.
- **Which attempts** (my call): each contestant contributes two folders per
  problem × language — their best-scored submission (as the code-copy run)
  and their **first** attempt, which is often the raw paste before edits
  (one folder if they are the same). This roughly doubles the AI run, not
  more; match pages stay capped as in the code-copy run.
- A row's S1 value is its best match over all references and both attempts.
- **Trivial problems**: per problem × language, the report shows how many
  contestants matched a reference above the threshold. If a large share
  matches, the problem is marked trivial: there S1 lists a row only
  together with S3 (S1 + S4 is not enough), and S1-only rows are not listed;
  the admin still sees the matched share and can open that problem's AI
  report (my call). Threshold and "large share" are tuned with the
  benchmark.

### S3 — wrong-answer rewrite pattern

- For each live/spectating participant × problem, take their contest
  submissions in time order (same language; a language switch starts a new
  chain).
- For consecutive attempts, compute a cheap similarity in Go (e.g. Jaccard of
  normalised lines or token shingles; builder's choice). An attempt that
  shares almost nothing with the previous one — and both are not tiny — is a
  **full rewrite**.
- The row records the number of full rewrites and links to the submissions.
  Thresholds (similarity cut-off, minimum size, rewrites needed to flag) are
  tuned with the benchmark.

### S4 — performance outlier

Two facts per participant × problem, computed from the DB:
- **Above their history**: the problem's points exceed the highest-points
  problem they had AC'd before the contest started, by a margin; also show
  their rating before the contest (`judge_rating` / `judge_profile.rating`).
- **Speed inversion**: they AC'd this problem faster (minutes from
  participation start / previous AC) than a lower-points problem in the same
  contest.

S4 is weak alone: it **never lists a row by itself**; it adds evidence and
rank to rows listed by S1 or S3 (my call).

### Combining and output

- A row (participant × problem) is listed when S1 or S3 fires. Sort: number
  of signals fired, then S1 strength. The admin can filter by user/problem.
- Results are written as one `ai.json` in the contest report dir and swapped
  in with the reports, so the existing "failed run keeps the previous
  results" rule covers them and no new results table is needed (my call).
  It holds: the rows (username, problem code, language, submission id(s),
  per-signal values and evidence links), and per problem × language: number
  of references by origin, matched share, and notes ("no statement text",
  "generation failed: …", "no references").
- The frontend reads `ai.json` through the existing report route (as it reads
  `llm.json` today).

### Admin page

`/admin/plagiarism`, contest picked as today; tabs **Code copy | AI
submissions** (en + vi strings).

```
Contest: tet26                              [Run check] [Clear]
[ Code copy ] [ AI submissions ]
────────────────────────────────────────────────────────────────
Suspects — for review, not proof                    filter: [   ]
User    Problem  Lang  AI match      Rewrites  Outlier
alice   knap     C++   82% → ref 3   2         +900 pts, faster
bob     path     C++   —             3         —
…
Reference solutions
knap  C++  5 generated · 1 admin   [add] [regenerate]
path  C++  none — no statement text (PDF)   [add]
```

- "AI match" links to the match page in `<problem>/<lang>/ai/`.
- "Rewrites" links to the contestant's submissions of that problem.
- Clear removes the AI results along with the code-copy results (they share
  the report dir); it does **not** delete reference solutions.

### Benchmark (extends `scripts/plagiarism-bench/`)

Uses the local DB clone; data never committed.
- **Wider clone (approved by the user 2026-11-26, user-run as before):**
  `clone_from_prod.sh` additionally copies, for **all** submissions (not only
  contest ones), `id, user_id, problem_id, language_id, date, result, points`
  without source; `judge_rating`; and `judge_profile` with `rating` added
  (`id, user_id, rating`). Needed for S4.
- **Known-human set (user: official rated contests from the past):** the 9
  rated contests that ended before ChatGPT (2022-11-30), checked on prod
  2026-11-26 (read-only): `claojre1`, `claojre2`, `claojre3`,
  `claojre4_div1`, `claojre4_div2`, `prelaoi22d1`, `prelaoi22d2`,
  `prelaoi22thcs`, `prelaoi23v1` — 41 problems, all with statement text,
  ~114 live participants, ~760 submissions. Every row flagged there is a
  false positive. Small set: a 2% target means ~5 rows, so read the result
  as a rough rate. (Copilot existed from mid-2021; a flag there is still
  treated as a false positive.)
- **LLM for the benchmark:** the prod config — `deepseek-flash` at
  `api.deepseek.com` (`COMPARE50_LLM_*`), non-thinking. Estimated
  ~430 calls, ~1.3M tokens, approved by the user.
- **Known-AI set:** for benchmark problems, generate extra AI solutions
  (not the stored references) and lightly disguised versions (reuse
  `disguise.py`); S1 recall = share that matches a reference.
- S3 has no labelled AI data; check its false positives on the known-human
  set only.
- Targets (my call, user may change): on the known-human set, at most ~2% of
  contestant × problem rows flagged; S1 recall on undisguised AI solutions
  ≥ 80%. Report the numbers to the user with the thresholds chosen.
- **Stop rule** (my call, same as the compare50 design): tune until the
  targets hold; if they don't after a reasonable number of rounds, ship the
  best variant measured and report the gap to the user. The benchmark does
  not gate shipping the tab; no per-signal on/off switch.

## Left to the builder

- Prompt wording, sampling settings, "usable text" length.
- The S3 similarity measure and all thresholds (from the benchmark).
- `results.json` exact format in the fork.
- Exact API paths for references and the tab component structure.
- Whether reference generation also runs from a separate "generate now"
  button (fine either way).
