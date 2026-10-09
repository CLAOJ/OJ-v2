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
