# v1 + v2 Combined Compose (Side-by-Side, Shared Database) — Design

**Goal:** Run the v2 stack (Go API + judge bridge, Next.js web, a judge) in the
**same Docker Compose project as v1**, sharing v1's MySQL and Redis, with **no
resource conflicts**. Prepare and validate everything **locally** first; the
production move to `beta.claoj.edu.vn` on the VPS is a later env/credential swap.

**Architecture:** v1 is left untouched. A new override file
`docker-compose.v2.yml` adds v2 services to the existing Compose project so they
join v1's internal networks and reach `db`/`redis` by hostname. v2's web app and
API are served **same-origin** behind a dedicated `v2_nginx`. Everything that
differs between local and production is confined to a single env file and the
choice of base compose file — the v2 override itself never changes.

**Tech stack:** Docker Compose (override merge), existing v2 Dockerfiles
(`OJ-v2/claoj/Dockerfile` Go, `OJ-v2/claoj-web/Dockerfile` Next.js), the
patched DMOJ judge image `claoj/judge-tiervnoj`, MariaDB 10.11, Redis, nginx.

---

## 1. Current state

- **v1** runs from `CLAOJ/claoj-docker/claoj/`, Compose project `name: claoj`:
  - Local base: `docker-compose.local.yml` — `db` (:3306), `redis` (:6379),
    `base`, `site` (Django), `celery`, `bridged` (v1 judge bridge :9999/:9998),
    `wsevent`, `nginx` (:8080). Data bind-mounted from
    `F:\Coding\CLAOJ\CLAOJ\claoj-data\`.
  - Production base: `docker-compose.yml` — same services **plus** `cloudflared`
    (token tunnel → `claoj.edu.vn`), data bind-mounted from `/root/claoj-data/`,
    `problems`/`media`/`assets`/`database` as named volumes.
  - Internal networks: `db`, `site`, `nginx`, `judge`.
- **v2** currently runs on the host, not in Compose: Go API on `:8081`, judge
  bridge on `:9997`, Next.js on `:3000`. It has working Dockerfiles for the API
  and web. The DMOJ judge image `claoj/judge-tiervnoj` is built and grades
  correctly (verified AC/WA end-to-end).
- The database is **already shared cleanly**: v2 is additive (runtime DDL guard;
  `scripts/v2_runtime_tables.sql` provisions v2-only tables; identity handling
  and schema parity verified). No migrations.

---

## 2. Topology

The v2 override is **env-driven** so the identical file layers onto either base:

| | Local (build & validate now) | Production (later) |
|---|---|---|
| base file | `docker-compose.local.yml` | `docker-compose.yml` |
| command | `docker compose -f docker-compose.local.yml -f docker-compose.v2.yml up -d` | `docker compose -f docker-compose.yml -f docker-compose.v2.yml up -d` |
| v2 entry | `http://localhost:8090` (published) | `https://beta.claoj.edu.vn` (cloudflared, no host port) |
| `SITE_URL` | `http://localhost:8090` | `https://beta.claoj.edu.vn` |
| DB / secrets | snapshot creds | live creds (provided later) |
| cookies | `SameSite=Lax` (HTTP) | `Secure; SameSite` (HTTPS) |

**Services added** (all container names prefixed `claoj_v2_*`):

| Service | Image | Role | Networks | Host port |
|---|---|---|---|---|
| `v2_backend` | `claoj/claoj-go` | Go API (`:8081`) + judge bridge (`:9997`) + Asynq worker + WS hub | `db`, `site`, `judge` | none |
| `v2_web` | `claoj/claoj-web` | Next.js (`:3000`) | `v2` | none |
| `v2_nginx` | `nginx:alpine` | same-origin front: `/`→web, `/api`→backend, `/api/events`→WS | `nginx`, `v2` | local `8090:80`; prod none |
| `v1_judge` | `claoj/judge-tiervnoj` | grades v1 submissions → `bridged:9999` | `judge` | none |
| `v2_judge` | `claoj/judge-tiervnoj` | grades v2 submissions → `v2_backend:9997` | `judge` | none |

A new internal network `v2` wires `v2_web ↔ v2_nginx ↔ v2_backend`. `v2_nginx`
also joins `nginx` so the existing cloudflared (prod) or the host (local) can
reach it. `v2_backend` joins `db`/`site` (to reach `db:3306`, `redis:6379`) and
`judge` (its inbound bridge).

**Same-origin.** The browser hits one origin for both the app and `/api`
(`localhost:8090` local, `beta.claoj.edu.vn` prod). Cookies are first-party;
`SITE_URL` drives both the CORS allow-list and the cookie `Secure`/`SameSite`
flags. `NEXT_PUBLIC_API_URL` is left **unset** so the web image derives `/api`
from the browser origin — one image works in both environments.

**Production ingress (manual, out-of-compose).** The cloudflared tunnel is
token-based (remotely managed), so ingress is configured in the Cloudflare Zero
Trust dashboard: add public hostname `beta.claoj.edu.vn` →
`http://claoj_v2_nginx:80`, plus the DNS record. v1's `claoj.edu.vn` route is
untouched. `v2_nginx` on the `nginx` network is reachable by cloudflared by
container name.

---

## 3. Conflict avoidance (the "share without conflict" requirement)

**Database — additive, no migrations.** The DDL guard blocks schema changes;
`scripts/v2_runtime_tables.sql` provisions v2-only tables once; both apps write
the same tables with identical column semantics. Two front-ends, one schema.

**Redis — dedicated DB index.** v1 uses index `0` (Django cache) and `1`
(Celery). v2 is pinned to index **`2`** (`REDIS_DB=2`) for *all* its Redis use
(permission cache, refresh/one-time tokens, audit-log stream, Asynq queue), so
no key collides.
- *Verification point:* confirm Asynq honors `REDIS_DB=2`. If Asynq cannot share
  the index, give it its own index (`3`) and document both. This must be
  checked during implementation, not assumed.

**Judge bridges — distinct ports + identities.** v1 bridge `:9999`/`:9998`; v2
bridge `:9997`. Each judge authenticates as its **own** `judge_judge` row
(`v1_judge` = `Vịt Con`, `v2_judge` = `Vịt Toàn Năng`). Dispatch never crosses:
a Django-created submission is dispatched by v1's bridge to `v1_judge`; a
Go-created submission by v2's bridge to `v2_judge`. No double-grading. Both
judges read the same `/problems` data read-only.

**Names / networks / host ports.** All v2 containers prefixed `claoj_v2_*`; new
internal `v2` network; only one new host port locally (`v2_nginx:8090`), none in
prod.

**Shared files.** `v2_backend` mounts `problems` and `media` **read-only** (for
serving problem PDFs/attachments) at the same paths v1 uses — no write
contention. The data root is parameterized as `${CLAOJ_DATA}` (local
`F:\Coding\CLAOJ\CLAOJ\claoj-data`, prod `/root/claoj-data`) so the one override
file binds correctly in both environments.

---

## 4. Service configuration

### 4.1 `v2_backend`

- Image `claoj/claoj-go`, `build:` context `../../../OJ-v2/claoj` (Compose builds
  and tags it; prod can reuse the tag).
- Env from gitignored `local/v2.local.env` (prod: `local/v2.prod.env`):
  ```
  DATABASE_DSN=claoj:<MYSQL_PASSWORD>@tcp(db:3306)/claoj?charset=utf8mb4&parseTime=True&loc=UTC
  REDIS_ADDR=redis:6379
  REDIS_DB=2
  SERVER_PORT=8081
  SERVER_MODE=release
  BRIDGE_ADDR=:9997
  SECRET_KEY=<fresh; openssl rand -base64 48>
  JWT_SECRET_KEY=<fresh>
  SITE_URL=http://localhost:8090        # prod: https://beta.claoj.edu.vn
  EMAIL_NO_REPLY=true                   # local; configure SMTP for prod as needed
  ```
- Volumes: `${CLAOJ_DATA}/problems:/problems:ro`, `${CLAOJ_DATA}/media:/media:ro`.
- Networks `db`, `site`, `judge`; `depends_on: [db, redis]`;
  `restart: unless-stopped`.

### 4.2 `v2_web`

- Image `claoj/claoj-web`, `build:` context `../../../OJ-v2/claoj-web`.
- `NEXT_PUBLIC_API_URL` **unset** (auto-derives `/api`); `NODE_ENV=production`,
  `PORT=3000`. Network `v2`; `restart: unless-stopped`.

### 4.3 `v2_nginx`

- `nginx:alpine`, config mounted from `local/v2-nginx/conf.d/`:
  - `location /api/`  → `proxy_pass http://v2_backend:8081;`
  - `location /api/events` → `proxy_pass http://v2_backend:8081;` with
    `Upgrade`/`Connection` WebSocket headers and long read timeout.
  - `location /` → `proxy_pass http://v2_web:3000;` (Next serves its own
    `/_next` assets).
  - `client_max_body_size` sized for submissions/uploads.
- Local: `ports: ["8090:80"]`, `server_name _`. Prod: no host port,
  `server_name beta.claoj.edu.vn`. Networks `nginx`, `v2`.

### 4.4 Judges

Both use `claoj/judge-tiervnoj` (already patched for the Python 3.14 `fork`
start method), `cap_add: [SYS_PTRACE]`, `${CLAOJ_DATA}/problems:/problems`,
network `judge`, `restart: unless-stopped`.

- `v1_judge` command: `run -p 9999 -A 0.0.0.0 -a 9998 -c /problems/judge.yml bridged "Vịt Con" <key>`
- `v2_judge` command: `run -p 9997 -A 0.0.0.0 -a 9998 -c /problems/judge.yml v2_backend "Vịt Toàn Năng" <key>`

`<key>` values are the existing `auth_key`s in `judge_judge`. (Keys may be
passed via env-substituted variables rather than inline, to keep them out of the
committed file.)

---

## 5. Files created / changed

- **Create** `CLAOJ/claoj-docker/claoj/docker-compose.v2.yml` — the override
  (all five v2 services, the `v2` network, `${CLAOJ_DATA}` binds).
- **Create** `CLAOJ/claoj-docker/claoj/local/v2-nginx/conf.d/v2.conf` — the
  same-origin nginx config.
- **Create** `CLAOJ/claoj-docker/claoj/local/v2.local.env` — gitignored v2 env
  (local creds, `REDIS_DB=2`, fresh secrets, `SITE_URL=http://localhost:8090`).
- **Create** `CLAOJ/claoj-docker/claoj/.env` (or documented shell exports) —
  sets `CLAOJ_DATA` and any judge-key substitutions for Compose interpolation.
- **No change** to v1's `docker-compose.local.yml` / `docker-compose.yml`, to the
  v2 Go/web source, or to the Dockerfiles.
- **Docs:** extend `OJ-v2/docs/deployment.md` with the combined-compose
  procedure and the `beta.claoj.edu.vn` cloudflared/DNS step.

The CLAOJ tree is a separate git repo from OJ-v2; the new compose/env/nginx
files live under `CLAOJ/claoj-docker/claoj/` and are committed there (env files
gitignored).

---

## 6. Rollout & testing

### 6.1 Local (now)

1. Build v2 images (`docker compose ... build v2_backend v2_web`, or direct
   `docker build`).
2. Write `local/v2.local.env` + the v2 nginx conf; set `CLAOJ_DATA`.
3. `docker compose -f docker-compose.local.yml -f docker-compose.v2.yml up -d`.
4. **Verify:**
   - v1 unaffected at `http://localhost:8080`.
   - v2 same-origin at `http://localhost:8090` — login sets first-party cookies,
     pages render, mutations pass CSRF.
   - Submit on v2 → `v2_judge` grades to AC/WA; submit on v1 → `v1_judge` grades;
     both rows land in the one shared DB.
   - Redis isolation: v2 keys appear only in DB `2`; v1 keys in DB `0`/`1`.
   - `docker compose ... down` and re-`up` is clean and idempotent.

### 6.2 Production (later, on credential handover)

1. **Pre-flight:** back up the live DB; restore a copy; run the schema-parity
   test against the copy; apply `scripts/v2_runtime_tables.sql` to prod.
2. Provide `local/v2.prod.env` (live DSN, `REDIS_DB=2`, fresh secrets,
   `SITE_URL=https://beta.claoj.edu.vn`); set `CLAOJ_DATA=/root/claoj-data`.
3. Cloudflare: add `beta.claoj.edu.vn` public-hostname ingress →
   `http://claoj_v2_nginx:80` and the DNS record.
4. `docker compose -f docker-compose.yml -f docker-compose.v2.yml up -d v2_backend v2_web v2_nginx v2_judge`
   (v1 services already running are untouched).
5. Smoke test on `beta.claoj.edu.vn` (public reads, auth round-trip, permission
   parity, a graded submission).
6. **Rollback:** `docker compose ... stop v2_backend v2_web v2_nginx v2_judge` —
   v1 is never affected; the additive tables can stay or be removed with
   `cleanup_v2_tables.sql`.

---

## 7. Error handling

- `v2_backend` fails fast and restarts (`restart: unless-stopped`) if it cannot
  reach `db`/`redis`; the DDL guard aborts on any accidental schema write.
- A judge with a wrong/duplicate key simply fails its handshake and never
  grades — no effect on the other bridge or on v1.
- `v2_nginx` returns 502 if the backend/web are still starting; `depends_on`
  orders startup but readiness is eventual (health of the shared DB gates
  `v2_backend`).
- None of these failure modes touch v1: separate containers, separate Redis
  index, additive schema.

---

## 8. Out of scope / non-goals

- No changes to v1 Django, its compose files, or its judges' behavior.
- No schema migrations; no DDL beyond the one-time additive provisioning script.
- Not building a unified single-nginx edge for both apps (rejected: needs
  host-based routing and more setup for no local benefit).
- Not replacing the DMOJ judge with the in-repo Go judge (unverified).
- Real production deployment is deferred until VPS credentials are provided; this
  spec prepares and validates the setup locally and defines the prod procedure.

---

## 9. Open items to resolve during implementation

1. **Asynq Redis index** — confirm it honors `REDIS_DB=2`; if not, assign it a
   separate index and document. (Isolation-critical.)
2. **Judge key handling** — decide inline-with-gitignore vs env-substituted
   variables for the two `judge_judge` auth keys in the override.
3. **`v2_backend` file mounts** — confirm which paths the API actually reads
   (problem PDFs, submission attachments) and mount exactly those read-only.
