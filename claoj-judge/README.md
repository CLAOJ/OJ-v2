# claoj-judge

Pull-based judge agent for CLAOJ v2. It pulls submissions from the asynq queue
`judge` (Redis), grades them on a private **tiervnoj** child (the v1
DMOJ/VNOJ engine: all languages, archives, file_io, checkers, batches), and
POSTs every judge packet to the backend's `/internal/judge/event`.

Nothing connects to the judge: there is no listen port to open. Scale by
adding replicas; all replicas share one `judge_judge` row (`JUDGE_NAME`).

| Env | Default | |
|---|---|---|
| `REDIS_ADDR` | — | required; same Redis as the backend |
| `REDIS_DB` | `0` | **must equal the backend's `REDIS_DB`** |
| `REDIS_PASSWORD` | | |
| `BACKEND_URL` | — | required, e.g. `http://v2_backend:8081` |
| `JUDGE_NAME` / `JUDGE_KEY` | — | required; a `judge_judge` row's name / auth_key |
| `JUDGE_YML` | `/problems/judge.yml` | tiervnoj config |
| `TIERVNOJ_ENTRY` | `/judge/.docker/entry` | tiervnoj launcher inside the image |
| `CHILD_PORT` | `9999` | loopback port tiervnoj dials |

Needs `SYS_PTRACE`, the problems folder at `/problems` (read-only), and a stop
grace period > 25 s. tiervnoj self-tests every executor before connecting, so
the first `judge "<name>" ready` line can take minutes on a cold container.

```bash
docker build -t claoj/claoj-judge-agent .   # needs claoj/judge-tiervnoj locally
go build ./... && go test ./...
```