# CLAOJ v2

The Go + Next.js rewrite of the CLAOJ online judge (<https://claoj.edu.vn>).
It runs alongside the legacy Django site (v1) against the **same MariaDB
database and Redis**, and owns no schema of its own: Django's models are the
source of truth and v2 only reads and writes rows.

## Services

| Directory | Service | Stack |
|---|---|---|
| `claoj/` | HTTP API (`:8081`) | Go 1.25, Gin, GORM, asynq |
| `claoj-web/` | Web app (`:3000`) | Next.js 16 App Router, React 19, Tailwind 4, next-intl |
| `claoj-judge/` | Judge agent | Go agent driving a private **tiervnoj** (DMOJ/VNOJ) child process |
| `CLAOJ/claoj-docker/` (separate repo) | v1 site + compose stack | Django, MariaDB, Redis, nginx |

## Architecture

```
Browser ──▶ v2_nginx :8090 ─┬─ /     ──▶ claoj-web :3000
                            └─ /api  ──▶ claoj :8081 ──┬─▶ MariaDB :3306   ◀─ shared with v1
                                                       ├─▶ Redis :6379     ◀─ shared with v1
                                                       └─▶ /internal/judge/event ◀── claoj-judge
Browser ──▶ v1 nginx :8080 ──▶ Django site ──▶ same MariaDB / Redis
                                  └─ v1 bridge :9999 ──▶ v1 judges
```

Invariants worth knowing before changing anything:

- **No DDL from v2.** `db.Connect()` panics on `CREATE/ALTER/DROP`. Additive
  v2-only tables/columns are provisioned once from
  `scripts/v2_runtime_tables.sql`; the decision record is
  [`docs/schema-audit.md`](docs/schema-audit.md).
- **Two user id spaces.** `auth_user.id` (Django auth) and `judge_profile.id`
  (DMOJ profile) differ for most users. `judge_*` tables key by profile id —
  resolve with `auth.CurrentProfileID(c)`. v2-only tables key by auth id.
- **Judges pull, nothing dials them.** Submissions go to the asynq `judge`
  queue in Redis; agents grade and POST packets to `/internal/judge/event`.
  There is no bridge/listen port on the v2 backend.
- **Same-origin API.** Browsers call `/api` on the web origin; nginx proxies
  it to the Go API. The base is `/api` (not `/api/v2`).
- **Auth is cookie + CSRF.** httpOnly `access_token`/`refresh_token` cookies,
  plus a non-httpOnly `csrf_token` echoed as `X-CSRF-Token` on mutations.
- **Redis is split by DB index.** In the compose stack v2 uses DB 2 (tokens,
  permission cache, audit log, asynq); v1 keeps DB 0 (cache) and DB 1 (Celery).
- **Permissions are Django's.** `auth_group` / `auth_permission` codenames are
  enforced by v2, cached in Redis for 60 s.

## Repository layout

```
claoj/                Go API (module github.com/CLAOJ/claoj)
  api/                HTTP router and handlers (api/v2/*)
  auth/               JWT, password, profile-id helpers
  bridge/             v1 graded-submission watcher (shared DB)
  jobs/               asynq client/worker, judge status, MOSS/plagiarism
  models/             GORM models over the Django schema
  service/            domain services (user, contest, problem, submission, ...)
  integration/        live-schema parity test (needs a database)
claoj-web/            Next.js web app (locale-prefixed /vi, /en)
claoj-judge/          pull-based judge agent wrapping tiervnoj
scripts/              v2_runtime_tables.sql, v2 nginx conf, ops scripts
docs/                 development, deployment, schema, follow-ups
.github/workflows/    CI: vet/test per component, GHCR images on main
```

## Getting started

Full local stack (v1 + v2 + both judges, one database) — see
[`docs/development.md`](docs/development.md). It needs the v1 runtime bundle
and is the only way to grade submissions locally.

### API only

Prerequisites: Go 1.25+, MariaDB 10.6+/MySQL 8+, Redis 7+.

```bash
cd claoj
go mod download
# create .env (gitignored) — see the env table below
go run .
# claoj-go HTTP API starting on :8081
```

### Web only

```bash
cd claoj-web
npm install
npm run dev          # http://localhost:3000/vi
```

Server-side fetches reach the API through `API_URL_INTERNAL`
(default `http://localhost:8081/api`); the browser uses the page origin.

### Backend environment

| Variable | Default | Notes |
|---|---|---|
| `DATABASE_DSN` | — | `user:pass@tcp(host:3306)/claoj?charset=utf8mb4&parseTime=True&loc=UTC` |
| `REDIS_ADDR` / `REDIS_DB` | `127.0.0.1:6379` / `0` | judge agents must use the same DB |
| `SERVER_PORT` / `SERVER_MODE` | `8081` / `debug` | |
| `SECRET_KEY` | — | **required**; boot fails on empty/`changeme` |
| `JWT_SECRET_KEY` | falls back to `SECRET_KEY` | |
| `SITE_URL` | `http://localhost:8081` | public web origin; drives CORS and cookie flags |
| `JUDGE_NAME` | — | comma-separated `judge_judge` rows to keep online |
| `EMAIL_NO_REPLY` | `true` | set `false` + `SMTP_*` to send mail |

`*_FILE` variants exist for Docker secrets; `CLAOJ_*`-prefixed and legacy
names (`DATABASE_URL`, `REDIS_URL`) are accepted too.

### Frontend environment

All optional — the defaults derive from the browser origin:

| Variable | Default | |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `<origin>/api` | browser API base |
| `NEXT_PUBLIC_WS_URL` | `<origin>/api/events` | WebSocket base |
| `API_URL_INTERNAL` | `http://localhost:8081/api` | SSR fetches |
| `SITE_URL` | `https://beta.claoj.edu.vn` | canonical URLs, sitemap |

## Testing

```bash
# Go API (integration/ skips itself when CLAOJ_DJANGO_DB_DSN is unset)
cd claoj && go vet ./... && go test ./...

# judge agent
cd claoj-judge && go vet ./... && go test ./...

# web (Jest)
cd claoj-web && npm test

# web checks
cd claoj-web && npm run i18n:check && npm run ui:check
```

The live-schema parity test is the guard against drift from the Django
database:

```bash
CLAOJ_DJANGO_DB_DSN="user:pass@tcp(host:3306)/claoj?parseTime=true" \
  go test ./integration/ -run TestSchemaParity -v
```

## Documentation

| Doc | Contents |
|---|---|
| [`docs/development.md`](docs/development.md) | full local stack, ports, judge setup, id spaces, CSRF, API response shapes |
| [`docs/deployment.md`](docs/deployment.md) | production deploy, v2-only schema, compose overlay, cutover |
| [`docs/schema-audit.md`](docs/schema-audit.md) | model-by-model audit and every retained v2-only DB object |
| [`docs/follow-ups.md`](docs/follow-ups.md) | parked open issues, none of which block a deploy |
| [`claoj-web/README.md`](claoj-web/README.md) | web app routes, admin panel, scripts |
| [`claoj-judge/README.md`](claoj-judge/README.md) | judge env and image |

## License

Based on DMOJ — see [LICENSE](LICENSE). Developed by IT-CLA Productions;
powered by [DMOJ](https://github.com/DMOJ/site) and VNOI.