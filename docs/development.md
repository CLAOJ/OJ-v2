# OJ-v2 Development Guide

How to run the full CLAOJ stack locally — the legacy v1 Django site and the
new v2 (Go API + Next.js web) **side by side against the same database** — and
how the two systems interoperate.

> **Golden rule:** the database schema is owned entirely by Django (v1). v2
> performs only row-level reads/writes and never issues DDL. A runtime guard in
> `db.Connect()` panics on any `CREATE/ALTER/DROP`. All v2-only tables are
> provisioned once, out of band, by `scripts/v2_runtime_tables.sql`.

---

## 1. Topology

```
                     ┌──────────────── shared state ────────────────┐
                     │                                              │
  Browser ──▶ Next.js web (:3000) ──HTTP/WS──▶ v2 Go API (:8081)   │
                                                    │  │            │
                                                    │  └─ POST /internal/judge/event ◀── DMOJ judge (agent)
                                                    │     pulls asynq queue `judge` from Redis
                                                    ▼                │
  Browser ──▶ v1 nginx (:8080) ──▶ Django site ──▶ MariaDB (:3306) ◀┘
                                        │              ▲
                                        └─ celery ─────┘   Redis (:6379)
                                        └─ v1 bridge (:9999/:9998) ──▶ (its own judges)
```

Both stacks read and write the **same MariaDB database and the same Redis**.
A user created in Django logs into v2; a submission made in v2 is visible in
Django; a permission granted in Django's admin is enforced by v2.

### Port map

| Port | Service | Owner |
|------|---------|-------|
| 8080 | nginx → Django site | v1 |
| 8090 | `v2_nginx` → v2 web + `/api` (same-origin) | v2, combined compose (§4) |
| 3306 | MariaDB | shared |
| 6379 | Redis | shared |
| 9999 / 9998 | v1 judge bridge | v1 |
| 8081 | Go HTTP API | v2 |
| 3000 | Next.js web | v2 |

v2 has **no judge port**: judges are pull-based — the `claoj-judge` agent
pulls from the asynq queue `judge` in Redis and POSTs packets to the API's
`/internal/judge/event` (`:8081`), which is internal to the Compose network
(§6). v1's TCP bridge keeps :9999/:9998 for its own judges.

Under the **combined-compose** path (§4, the primary way to run v2 locally),
8081/3000 are internal to the Compose network — nothing outside
`v2_nginx`'s **8090** is published to the host. Those ports only bind
directly to `localhost` when you run the Go backend / Next.js dev server on
the host instead, for fast iteration (§4.1/§5).

---

## 2. Prerequisites

- Docker + Docker Compose
- Go 1.23+
- Node 20+ / npm
- A copy of the v1 production runtime bundle (the `CLAOJ/` folder): the five
  `claoj/claoj-*` image tarballs, the `claoj-data/` directory (MariaDB datadir,
  `problems/`, `media/`, `assets/`), and `claoj-docker/`.

---

## 3. Bring up the v1 stack (Django + MariaDB + Redis)

All commands run from `CLAOJ/claoj-docker/claoj/`.

**3.1 Load the production images** (once):

```bash
cd CLAOJ/images
for img in claoj-base claoj-site claoj-celery claoj-bridged claoj-wsevent; do
  docker load -i "$img"
done
docker pull mariadb:10.11 && docker pull redis:alpine && docker pull nginx:alpine
```

**3.2 Seed the database volume** from the raw datadir (once). The compose file
uses a named volume `claoj_database`; copy the snapshot into it so the original
`claoj-data/database` is never mutated:

```bash
docker volume create claoj_database
docker run --rm \
  -v claoj_database:/target \
  -v "$(pwd)/../../claoj-data/database:/source:ro" \
  mariadb:10.11 bash -c "cp -a /source/. /target/ && chown -R mysql:mysql /target"
```

**3.3 Start it.** The local stack is defined in `docker-compose.local.yml`,
which differs from the production `docker-compose.yml` in four ways:

- **cloudflared is removed** — its tunnel token would attach your local
  instance to the real `claoj.edu.vn` edge.
- `assets/`, `media/`, `problems/` are bind-mounted from `claoj-data/`.
- nginx is published on **8080**, db on **3306**, redis on **6379** (so v2 can
  reach them from the host).
- A `local/local_settings.py` overlay disables real SMTP (console backend),
  sets `ALLOWED_HOSTS` to localhost, and uses Cloudflare Turnstile **test**
  keys.

```bash
docker compose -f docker-compose.local.yml up -d
# health check
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/    # 200
```

Visit <http://localhost:8080> — the Django site loads with production data.

> This starts v1 alone. §4 brings up v2 (backend + web + nginx + both judges)
> layered on top of it with one combined command — you don't need to run v1
> and v2 as two separate steps day-to-day, this section just covers the
> one-time v1 image/DB setup that has to happen first either way.

**3.4 Provision the v2-only tables** (once per database). These are additive
tables/columns that Django never touches (refresh-token bookkeeping lives in
Redis, but notifications, 2FA, OAuth links, MOSS cache, comment revisions,
contest clarifications, and four editorial columns on `judge_solution` are
additive DB objects):

```bash
docker exec -i claoj_db mariadb -uclaoj -p<MYSQL_PASSWORD> claoj \
  < ../../OJ-v2/scripts/v2_runtime_tables.sql
```

`<MYSQL_PASSWORD>` is the `MYSQL_PASSWORD` value in
`environment/mysql.env`. See `docs/schema-audit.md` for the full rationale
behind every object this script creates.

---

## 4. Run v2 (combined compose — primary)

The verified way to run v2 locally is as Docker Compose services in v1's
**same** Compose project (`name: claoj`), sharing v1's `db`/`redis` containers
by hostname. One command brings up the Go backend, the Next.js web app, the
same-origin nginx front, and both judges (§6), layered on top of the v1 stack
from §3. See `docs/deployment.md` §9 for the full env-file contract and the
production equivalent.

**4.0 Configure** the two gitignored env files under
`CLAOJ/claoj-docker/claoj/` once (`.env` for `CLAOJ_DATA`/`V2_HTTP_PORT`/judge
keys, `local/v2.local.env` for the backend's `DATABASE_DSN`/`REDIS_DB=2`/fresh
secrets — see `docs/deployment.md` §9.2 for the exact keys and where the
values come from).

**Bring it up**, from `CLAOJ/claoj-docker/claoj/`:

```bash
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml up -d
```

Visit **<http://localhost:8090>** — same-origin web app + `/api`, served
through `claoj_v2_nginx`. v1 is unaffected and stays at
<http://localhost:8080>.

```bash
docker ps --format '{{.Names}}\t{{.Status}}'
# claoj_v2_backend  Up (healthy)
# claoj_v2_web      Up (healthy)
# claoj_v2_nginx    Up
# claoj_v1_judge    Up
# claoj_v2_judge    Up
# ...plus the seven v1 containers from §3
```

Rebuild after a source change (Go or web):

```bash
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml \
  up -d --build v2_backend v2_web
```

`docker compose ... down` and re-`up` is clean and idempotent — v1's data
lives in its own named volumes, untouched by v2 coming and going.

### 4.1 Fast iteration: run the Go backend on the host

For a tight edit/rebuild/test loop on Go code, a host binary beats a
container rebuild. Point it at the same `db`/`redis` v1 already publishes on
`127.0.0.1:3306`/`127.0.0.1:6379` (§3.3). Use `REDIS_DB=0` (not `2`) and stop
`v2_backend` (and `v2_judge`, see §6) in compose before running this — with
the pull-queue model nothing collides on a port, but you don't want a second
backend racing the compose one on the same Redis queues, or a second judge
agent presenting the same `Vịt Toàn Năng` identity.

**Configure** `claoj/.env` (gitignored — never commit it):

```dotenv
DATABASE_DSN=claoj:<MYSQL_PASSWORD>@tcp(127.0.0.1:3306)/claoj?charset=utf8mb4&parseTime=True&loc=UTC
REDIS_ADDR=127.0.0.1:6379
REDIS_DB=0
SERVER_PORT=8081
SERVER_MODE=release
SECRET_KEY=<generate: openssl rand -base64 48>
SITE_URL=http://localhost:3000
EMAIL_NO_REPLY=true
```

**Build and run:**

```bash
cd OJ-v2/claoj
go build -o claoj-server.exe .
./claoj-server.exe
```

Expected log lines: `db: connected successfully`, `cache: connected to Redis`,
`claoj-go HTTP API starting on :8081`. There is no judge listener line — the
backend serves no bridge; when an agent connects (§6), its own log ends with
`judge "Vịt Toàn Năng" ready`.

**Verify against the live schema.** The env-gated parity test confirms every
GORM model maps cleanly onto the real Django schema:

```bash
CLAOJ_DJANGO_DB_DSN="claoj:<MYSQL_PASSWORD>@tcp(127.0.0.1:3306)/claoj?parseTime=true" \
  go test ./integration/ -run TestSchemaParity -v
```

---

## 5. Run the v2 frontend

The §4 combined-compose command already runs the built `v2_web` container
against the whole stack — nothing extra to do for the primary path.

**Fast iteration (hot reload):** run Next.js on the host, pointed at the
host-run backend from §4.1 (the compose-managed backend's `:8081` isn't
published to the host — see the port-map note in §1).

**5.1 Configure** `claoj-web/.env.local` (gitignored):

```dotenv
NEXT_PUBLIC_API_URL=http://localhost:8081/api
NEXT_PUBLIC_WS_URL=ws://localhost:8081/api/events
SITE_URL=http://localhost:3000
```

**5.2 Run:**

```bash
cd OJ-v2/claoj-web
npm install
npm run dev        # http://localhost:3000
```

The app is locale-prefixed: <http://localhost:3000/vi> (Vietnamese) or `/en`.

---

## 6. Run a judge (real grading)

Submissions stay in state `QU` (queued) until a judge is connected. The judge is
a separate Docker image built from `CLAOJ/claoj-docker/judge/tiervnoj`.

**Combined compose (primary):** the `docker compose ... up -d` in §4 already
starts both judges — nothing extra to run:

- `claoj_v1_judge`, identity `Vịt Con`, dials `bridged:9999` and grades v1
  (Django) submissions.
- `claoj_v2_judge`, identity `Vịt Toàn Năng`, pulls asynq `judge` tasks from
  Redis and POSTs results to `v2_backend:8081`, grading v2 (Go) submissions.

Confirm they're online:

```bash
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml logs v2_judge | grep ready
# judge "Vịt Toàn Năng" ready
curl -s http://localhost:8090/api/judges   # both judges report "online": true
```

**6.1 Build the judge image** (once, ~6 GB, downloads all language runtimes —
needed before either compose judge or a standalone one can run):

```bash
cd CLAOJ/claoj-docker/judge
docker build -t claoj/judge-tiervnoj ./tiervnoj
```

### Fast iteration: a standalone judge for the host-run backend (§4.1)

When running the Go backend directly on the host instead of in compose, point
a standalone judge container at the host instead of at `v2_backend`
(`host.docker.internal` resolves to the host from inside the container).
Stop the compose-managed `v2_judge` first — otherwise two judge agents
present the same `Vịt Toàn Năng` identity and race for the same asynq
`judge` queue.

**6.2 Run a judge** against the host-run backend (`:8081`) and host Redis:

```bash
docker run -d --name claoj_judge_local --cap-add SYS_PTRACE \
  -v "F:\Coding\CLAOJ\CLAOJ\claoj-data\problems:/problems:ro" \
  -e REDIS_ADDR=host.docker.internal:6379 -e REDIS_DB=2 \
  -e BACKEND_URL=http://host.docker.internal:8081 \
  -e JUDGE_NAME="Vịt Toàn Năng" -e JUDGE_KEY=<V2_JUDGE_KEY, from .env> \
  claoj/claoj-judge-agent
```

The backend needs `JUDGE_NAME="Vịt Toàn Năng"`; within 30 s `GET /api/judges`
reports it `online`. Nothing connects to the judge: it pulls from Redis.

**6.3 Submit and watch it grade** (against the host-run backend on `:8081`):

```bash
# (after logging in and obtaining the csrf_token cookie — see §7.2)
curl -b cookies.txt -X POST http://localhost:8081/api/problem/aplusb/submit \
  -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" \
  -d '{"source":"...","language":"PY3"}'
# poll GET /api/submission/<id> — status goes QU → P → G → D, result AC/WA/...
```

Against the combined-compose stack, submit through `localhost:8090` instead
(same shape, same-origin through `v2_nginx`).

> **Judge note:** the runtimes base image ships Python 3.14, which changed the
> default multiprocessing start method from `fork` to `forkserver`. The judge's
> per-submission workers rely on `fork` to inherit the problem cache, so
> `dmoj/judge.py` forces `multiprocessing.set_start_method('fork')`. Without it
> every grade dies with an `AssertionError` in `get_problem_roots`.

---

## 7. How v1 and v2 share identity, permissions, and sessions

### 7.1 The two id spaces (critical)

Django has **two distinct id sequences** that are easy to conflate:

- `auth_user.id` — the Django user id. This is what v2's JWT/auth middleware
  puts in the Gin context: `c.GetUint("user_id")`.
- `judge_profile.id` — the DMOJ Profile id, its **own** sequence.
  `judge_profile.user_id` is a foreign key *to* `auth_user.id`.

On real production data these differ for ~95% of users. Most `judge_*` tables
(`judge_submission.user_id`, `judge_comment.author_id`,
`judge_contestparticipation.user_id`, …) key by **profile id**, not auth id.
Conflating them silently attributes data to the wrong person.

**Rule:** when a `judge_*` column wants a user, resolve the context auth id to a
profile id first — use `auth.CurrentProfileID(c)` (cached per request) or
`db.Where("user_id = ?", authID).First(&profile)` then `profile.ID`. The v2-only
tables (`notification`, `totp_device`, `backup_code`, `oauth_user_link`) key by
**auth id** by convention.

### 7.2 Session cookies + CSRF (browser vs curl)

v2 issues httpOnly `access_token` / `refresh_token` cookies on login and enforces
a **double-submit CSRF** check on mutations:

- The backend plants a **non-httpOnly** `csrf_token` cookie on authenticated GETs.
- The browser client (`lib/api.ts`) reads it and echoes it in the
  `X-CSRF-Token` header on every non-GET request.
- Over plain HTTP (local dev), auth cookies use `SameSite=Lax` (not `None`,
  which browsers silently reject without `Secure`). Over HTTPS they use
  `SameSite=None; Secure`.

So a `curl` flow must: (1) POST `/api/auth/login` saving cookies, (2) GET
`/api/user/me` to receive the `csrf_token` cookie, (3) send that value as
`X-CSRF-Token` on subsequent POSTs.

### 7.3 Permissions

v2 reads Django's own `auth_group` / `auth_permission` / `auth_user_groups`
tables and enforces Django codenames (`judge.see_private_problem`,
`judge.edit_all_contest`, …), mirroring Django's `is_editable_by` /
`is_accessible_by`. Grants made in Django admin apply to v2 within a 60-second
Redis cache TTL (invalidated immediately on v2-side admin writes). The admin
"Groups" page (`/admin/groups`) manages these; mutations are superuser-only.

---

## 8. API response-shape conventions (frontend contract)

The Go API is **not uniform** about list envelopes — match each endpoint's
actual shape:

| Shape | Endpoints |
|-------|-----------|
| `{ "results": [...] }` | `/contests`, `/notifications`, `/contest/:key/clarifications` |
| `{ "data": [...] }` | everything else (`/problems`, `/users`, `/blogs`, `/submissions`, `/comments`, `/problem/:code/clarifications`, …) |
| flat named object | `/contest/:key/ranking`, `/contest/:key/stats/public`, detail endpoints |

Some query params the UI may want don't exist server-side (e.g. `/users` has no
`order`; `/problems` has no date sort). Where the backend can't sort/filter,
the frontend does it client-side (see `HomePageContent.tsx`). Params that **do**
exist: `/ratings/leaderboard?limit=`, `/comments?page_type=`, `page`/`page_size`
on paginated lists.

---

## 9. Verification checklist

```bash
# backend
cd OJ-v2/claoj && go build ./... && go vet ./... && go test ./...   # 364 pass
# live-schema parity (needs the DB up)
CLAOJ_DJANGO_DB_DSN="..." go test ./integration/ -run TestSchemaParity
# frontend
cd OJ-v2/claoj-web && node scripts/check-i18n.mjs                   # 1509 keys, both locales
```

---

## 10. Common issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| Backend exits at boot: `cache: Redis at … unreachable` | Redis is down/unreachable — v2 has no safe fallback without it | start Redis / fix `REDIS_ADDR` (and `REDIS_DB`, which must match the judge agent's) |
| Web login "succeeds" but stays logged out | auth cookie `SameSite=None` over HTTP rejected | already fixed (Lax over HTTP); ensure `SITE_URL` scheme matches |
| Every POST from the web UI 403s | missing/mismatched CSRF header | client must echo `csrf_token` cookie as `X-CSRF-Token` |
| Submission stuck at `QU` | no judge agent connected, empty problem list, or a task stranded in asynq across a backend cutover | check `GET /api/judges` online (~30 s status ticker); check the agent log for `judge "<name>" ready`; if it predates a cutover, see `docs/deployment.md` §9.6 |
| `ERROR 1064` in a query | MySQL reserved word unquoted (`key`, `order`, `read`, `load`) | backtick-quote it in the raw SQL / GORM clause |
| Data attributed to wrong user | `auth_user.id` used where `judge_profile.id` expected | resolve via `auth.CurrentProfileID(c)` |
