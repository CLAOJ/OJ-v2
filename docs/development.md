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
                                                    │  └─ judge bridge (:9997) ──▶ DMOJ judge
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
| 3306 | MariaDB | shared |
| 6379 | Redis | shared |
| 9999 / 9998 | v1 judge bridge | v1 |
| 8081 | Go HTTP API | v2 |
| 9997 | Go judge bridge (`BRIDGE_ADDR`) | v2 |
| 3000 | Next.js web | v2 |

v2's judge bridge listens on **:9997** specifically so it does not collide with
the v1 bridge on :9999. This is configurable — see `BRIDGE_ADDR` below.

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

## 4. Run the v2 Go backend

**4.1 Configure** `claoj/.env` (gitignored — never commit it):

```dotenv
DATABASE_DSN=claoj:<MYSQL_PASSWORD>@tcp(127.0.0.1:3306)/claoj?charset=utf8mb4&parseTime=True&loc=UTC
REDIS_ADDR=127.0.0.1:6379
REDIS_DB=0
SERVER_PORT=8081
SERVER_MODE=release
SECRET_KEY=<generate: openssl rand -base64 48>
SITE_URL=http://localhost:3000
EMAIL_NO_REPLY=true
# v1 bridged owns :9999 on this host; v2's judge bridge listens here instead:
BRIDGE_ADDR=:9997
```

**4.2 Build and run:**

```bash
cd OJ-v2/claoj
go build -o claoj-server.exe .
./claoj-server.exe
```

Expected log lines: `db: connected successfully`, `cache: connected to Redis`,
`claoj-go HTTP API starting on :8081`, `bridge: TCP server listening on :9997`.

**4.3 Verify against the live schema.** The env-gated parity test confirms every
GORM model maps cleanly onto the real Django schema:

```bash
CLAOJ_DJANGO_DB_DSN="claoj:<MYSQL_PASSWORD>@tcp(127.0.0.1:3306)/claoj?parseTime=true" \
  go test ./integration/ -run TestSchemaParity -v
```

---

## 5. Run the v2 frontend

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

**6.1 Build** (once, ~6 GB, downloads all language runtimes):

```bash
cd CLAOJ/claoj-docker/judge
docker build -t claoj/judge-tiervnoj ./tiervnoj
```

**6.2 Run**, pointing it at v2's bridge on the host (`host.docker.internal`
resolves to the host from inside the container). The judge name/key must match a
row in `judge_judge`:

```bash
docker run -d --name claoj_judge_local --cap-add SYS_PTRACE \
  -v "F:\Coding\CLAOJ\CLAOJ\claoj-data\problems:/problems" \
  claoj/judge-tiervnoj \
  run -p 9997 -A 0.0.0.0 -a 9998 -c /problems/judge.yml host.docker.internal
```

`/problems/judge.yml` carries the judge id (`Vịt Con`) and auth key. On connect
you'll see `bridge [...]: authenticated as Vịt Con` in the v2 log, and
`GET /api/judges` reports it `online`.

**6.3 Submit and watch it grade:**

```bash
# (after logging in and obtaining the csrf_token cookie — see §7.2)
curl -b cookies.txt -X POST http://localhost:8081/api/problem/aplusb/submit \
  -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" \
  -d '{"source":"...","language":"PY3"}'
# poll GET /api/submission/<id> — status goes QU → P → G → D, result AC/WA/...
```

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
| `bridge: listen tcp :9999: bind: ...` | v1 bridged already owns :9999 | set `BRIDGE_ADDR=:9997` |
| Web login "succeeds" but stays logged out | auth cookie `SameSite=None` over HTTP rejected | already fixed (Lax over HTTP); ensure `SITE_URL` scheme matches |
| Every POST from the web UI 403s | missing/mismatched CSRF header | client must echo `csrf_token` cookie as `X-CSRF-Token` |
| Submission stuck at `QU` | no judge connected, or judge's problem list empty | check `GET /api/judges` online; check judge log for handshake |
| `ERROR 1064` in a query | MySQL reserved word unquoted (`key`, `order`, `read`, `load`) | backtick-quote it in the raw SQL / GORM clause |
| Data attributed to wrong user | `auth_user.id` used where `judge_profile.id` expected | resolve via `auth.CurrentProfileID(c)` |
