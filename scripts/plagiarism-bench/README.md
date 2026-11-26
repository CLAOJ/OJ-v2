# Plagiarism benchmark (compare50 vs MOSS)

Offline tooling to compare the compare50 fork against MOSS on past contests,
using a local copy of a few production tables.

## 1. Clone the tables (user-run only)

```sh
scripts/plagiarism-bench/clone_from_prod.sh
```

This script is run by the user, never by an automated executor. It only reads
prod, over `ssh root@claoj` and `docker exec claoj_db`:

- `judge_language judge_problem judge_contest judge_contestproblem judge_contestparticipation judge_contestsubmission judge_rating`, full dump
- `judge_submission`, only rows with `contest_object_id IS NOT NULL`
- `judge_submissionsource`, only rows of those submissions
- `judge_submission` rows with `contest_object_id IS NULL`, reduced to `id, user_id, problem_id, language_id, date, result, points, status` as TSV, loaded with `INSERT IGNORE` (other NOT NULL columns take defaults); used by the AI history signals
- `auth_user` reduced to `id, username` and `judge_profile` reduced to `id, user_id, rating`, exported as TSV

It also applies `scripts/v2_runtime_tables.sql` to `claoj_bench` (needed for `plagiarism_ai_reference`).

Dumps go to `scripts/plagiarism-bench/dumps/`. The script then starts a local
`mariadb:11` container named `claoj-bench-db` (volume `claoj-bench-db`, bound
to `127.0.0.1:3307`) and loads everything into database `claoj_bench`. Each run
drops and recreates that database.

`dumps/` and `data/` are gitignored. They contain contestants' code, so keep
them off shared storage.

## 2. Export one contest to the compare50 layout

```sh
BENCH_DB_DSN='root@127.0.0.1:3307/claoj_bench' \
  python scripts/plagiarism-bench/export.py --contest tik25 --out scripts/plagiarism-bench/data/tik25
```

`BENCH_DB_DSN` defaults to `root@127.0.0.1:3307/claoj_bench`. The output is
`data/<key>/<problem_code>/<lang_slug>/<user>/main.<ext>`, using the same
selection, extension map, slug and folder-name rules as the Go job
(`claoj/jobs/compare50.go`, `claoj/jobs/moss.go`). Selection: each contestant's
best-points submission per problem and language, from LIVE and SPECTATE
participations only (virtual IN (0,-1)).

Re-running overwrites files but does not delete folders from earlier runs.
Delete `data/<key>` first if you want a clean export.

Install the Python dependencies with `pip install -r scripts/plagiarism-bench/requirements.txt`.
`export.py` imports `pymysql` only in the CLI path.

## Tests

```sh
python -m unittest discover -s scripts/plagiarism-bench/tests -v
```

The tests use an in-memory sqlite fixture, not real data.

## AI detection benchmark

Run the job on past contests, then measure false positives and recall:

```sh
cd claoj && go run ./cmd/plagiarism-bench -dsn 'root@tcp(127.0.0.1:3307)/claoj_bench?parseTime=true' \
  -contests claojre1,claojre2 -out ../scripts/plagiarism-bench/data/reports -python /path/to/compare50/venv/python
python scripts/plagiarism-bench/ai_bench.py fp --out scripts/plagiarism-bench/data/reports --contests <contest ids>
python scripts/plagiarism-bench/ai_bench.py recall --contest claojre1 --per-problem 3 --python /path/to/compare50/venv/python
```

The Go command refuses any DSN host other than 127.0.0.1/localhost and forces `parseTime=true`.
`-out` holds one report dir per contest id (`<out>/<id>/ai.json`). Both the job and
`recall` call the LLM through `COMPARE50_LLM_URL`, `_KEY`, `_MODEL`; they cost tokens.
`fp` prints flagged rows over contestant x problem rows, per signal and combined.
`recall` generates fresh solutions, also disguises them (`disguise.py`), and prints the
share whose match against the stored references is at least `aiMatchPct`.
Tests: `python -m unittest scripts/plagiarism-bench/test_ai_bench.py -v`.
