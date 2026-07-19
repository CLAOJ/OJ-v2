# v1+v2 Combined Compose Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run the v2 stack (Go API+bridge, Next.js web behind a same-origin nginx, and a judge) in the same Docker Compose project as v1, sharing v1's MySQL and Redis without conflict, validated locally now and deployable to `beta.claoj.edu.vn` later by swapping env files.

**Architecture:** v1's compose files are untouched. A new override `docker-compose.v2.yml` (in `CLAOJ/claoj-docker/claoj/`) adds five services — `v2_backend`, `v2_web`, `v2_nginx`, `v1_judge`, `v2_judge` — that join v1's existing networks and reach `db:3306`/`redis:6379` by hostname. Everything env-specific lives in two gitignored env files.

**Tech Stack:** Docker Compose override merge, existing Dockerfiles (`OJ-v2/claoj`, `OJ-v2/claoj-web`), `claoj/judge-tiervnoj` DMOJ judge image, nginx:alpine, MariaDB 10.11, Redis.

**Spec:** `OJ-v2/docs/superpowers/specs/2026-07-19-v1-v2-combined-compose-design.md`. Two spec details are superseded by code findings (see Task 3 notes): v2_backend gets a `v2_data` volume instead of read-only `problems`/`media` mounts (all its file I/O is under its own relative `data/` dir), and the local port publish binds to `127.0.0.1` so the same override is safe on the VPS.

## Global Constraints

- **Never modify** `docker-compose.local.yml`, `docker-compose.yml`, or any v1 service definition.
- All v2 Redis usage (cache, tokens, audit log, Asynq queue) uses **`REDIS_DB=2`** — verified: `cache.Connect()` and both `asynq.RedisClientOpt` sites read `config.C.Redis.DB`.
- v2 judge bridge listens on **`:9997`** (`BRIDGE_ADDR=:9997`); v1's `bridged` keeps `:9999`/`:9998`.
- Same-origin: browser reaches web AND `/api` through `v2_nginx` only. `NEXT_PUBLIC_API_URL` stays **unset** at build time.
- Container names exactly: `claoj_v2_backend`, `claoj_v2_web`, `claoj_v2_nginx`, `claoj_v1_judge`, `claoj_v2_judge`.
- Judge identities: `v1_judge` = `Vịt Con` → `bridged:9999`; `v2_judge` = `Vịt Toàn Năng` → `v2_backend:9997`. Keys come from `judge_judge` via env substitution (`${V1_JUDGE_KEY}`, `${V2_JUDGE_KEY}`) — never committed.
- Local entry point: `http://localhost:8090` (published as `127.0.0.1:${V2_HTTP_PORT:-8090}:80`).
- No DDL against the shared DB (the runtime guard enforces this; nothing in this plan touches the schema).
- Two git repos are involved: compose/nginx/env work commits in **`CLAOJ/claoj-docker`** (paths below relative to `CLAOJ/claoj-docker/claoj/`); code + docs commit in **`OJ-v2`**.

---

### Task 1: OJ-v2 branch setup + build the Go backend image

The v2 images MUST include the live-stack compatibility fixes, which are on the
unmerged branch `feat/live-stack-compatibility` (5 commits: schema/identity,
auth cookies/CSRF, bridge, frontend contract, docs). `main` currently has only
the spec/docs commits on top of the old code. Merge it first.

**Files:**
- No source changes. Git + `docker build` only.

**Interfaces:**
- Produces: local image **`claoj/claoj-go`** (Go API + bridge, binary at `/app/main`, workdir `/app`), built from `OJ-v2/claoj/Dockerfile`; branch `feat/combined-compose` in OJ-v2 for later tasks.

- [ ] **Step 1: Merge the fixes branch and create the working branch**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git checkout main
git merge feat/live-stack-compatibility -m "merge: live-stack compatibility fixes (schema/identity, auth, bridge, web contract)"
git checkout -b feat/combined-compose
```

Expected: merge commits cleanly (the branch and main only diverge by doc commits; no overlapping files). If a conflict appears, stop and report BLOCKED — do not resolve by hand-picking.

- [ ] **Step 2: Verify the merged tree is green**

```bash
cd claoj && go build ./... && go vet ./... && go test ./... 2>&1 | grep -c "^ok"
```

Expected: build+vet silent, test output ends with 18 `ok` package lines, zero `FAIL`.

- [ ] **Step 3: Build the backend image**

```bash
cd /f/Coding/CLAOJ/OJ-v2/claoj
docker build -t claoj/claoj-go .
```

Expected: `naming to docker.io/claoj/claoj-go` and exit 0. (The Dockerfile's `EXPOSE 8080` is stale documentation only — `SERVER_PORT` env controls the real port; do not edit it in this task.)

- [ ] **Step 4: Smoke-run the image against the running local DB/Redis**

The local v1 stack publishes db on `3306` and redis on `6379`. From a container, the host is `host.docker.internal`:

```bash
MYSQL_PW=$(grep MYSQL_PASSWORD /f/Coding/CLAOJ/CLAOJ/claoj-docker/claoj/environment/mysql.env | cut -d= -f2)
docker run --rm -e DATABASE_DSN="claoj:${MYSQL_PW}@tcp(host.docker.internal:3306)/claoj?charset=utf8mb4&parseTime=True&loc=UTC" \
  -e REDIS_ADDR=host.docker.internal:6379 -e REDIS_DB=2 \
  -e SECRET_KEY=smoketest-only-not-a-real-secret-0123456789 \
  -e SERVER_PORT=8081 -e BRIDGE_ADDR=:9997 -e SITE_URL=http://localhost:8090 \
  claoj/claoj-go sh -c './main & sleep 5; wget -q -O- http://127.0.0.1:8081/health'
```

Expected output contains `db: connected successfully`, `bridge: TCP server listening on :9997`, and the health probe prints `{"status":"ok"}`.

- [ ] **Step 5: Commit (nothing to commit in OJ-v2 beyond the merge; verify state)**

```bash
cd /f/Coding/CLAOJ/OJ-v2 && git status --short
```

Expected: clean tree on `feat/combined-compose`.

---

### Task 2: Web image — runtime internal API base for SSR + buildable image

Two changes make one web image work in any environment: (a) server-side
rendering (metadata fetchers) must reach the backend **inside** the compose
network at runtime — add a non-public `API_URL_INTERNAL` env read (runtime, not
build-baked); (b) `next start` should see `next.config.ts` in the runtime
stage.

**Files:**
- Modify: `OJ-v2/claoj-web/src/lib/api.ts:9-12`
- Modify: `OJ-v2/claoj-web/Dockerfile:28-30`

**Interfaces:**
- Consumes: nothing from other tasks.
- Produces: local image **`claoj/claoj-web`** (Next.js server on `:3000`); env contract: `API_URL_INTERNAL` (optional, server-side only, e.g. `http://v2_backend:8081/api`).

- [ ] **Step 1: Change the SSR fallback in `api.ts`**

Replace the current `getApiBaseUrl` (lines 9–12):

```ts
function getApiBaseUrl(): string {
    if (typeof window === 'undefined') {
        // Server-side (SSR metadata fetchers): reach the backend over the
        // internal network. Read at runtime — NOT inlined at build — so one
        // image works in every environment.
        return process.env.API_URL_INTERNAL || 'http://localhost:8081/api';
    }
    return `${window.location.origin}/api`;
}
```

(The old server-side default was `http://localhost:8080/api` — v1's nginx, always wrong for v2 SSR.)

- [ ] **Step 2: Type-check**

```bash
cd /f/Coding/CLAOJ/OJ-v2/claoj-web && npx tsc --noEmit 2>&1 | grep -v "__tests__" | head -5
```

Expected: no errors outside `__tests__/` (those matcher-type errors are pre-existing).

- [ ] **Step 3: Copy `next.config.ts` into the runtime image**

In `OJ-v2/claoj-web/Dockerfile`, after the two existing `COPY --from=builder` lines, add:

```dockerfile
COPY --from=builder /app/next.config.ts ./
```

- [ ] **Step 4: Build the web image (no NEXT_PUBLIC_API_URL build arg)**

```bash
cd /f/Coding/CLAOJ/OJ-v2/claoj-web
docker build -t claoj/claoj-web .
```

Expected: `npm run build` succeeds inside the image; exit 0.

- [ ] **Step 5: Smoke-run**

```bash
docker run --rm -d --name v2web_smoke -p 127.0.0.1:3999:3000 claoj/claoj-web
sleep 8
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3999/vi
docker rm -f v2web_smoke
```

Expected: `200` (the page renders even without a reachable API; widgets degrade gracefully).

- [ ] **Step 6: Commit**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add claoj-web/src/lib/api.ts claoj-web/Dockerfile
git commit -m "feat(web): runtime API_URL_INTERNAL for SSR + ship next.config.ts in image"
```

---

### Task 3: The compose override, nginx config, and env files

All paths in this task are relative to `CLAOJ/claoj-docker/claoj/` (its repo is
`CLAOJ/claoj-docker`).

**Spec deviations locked in here (from code findings):** v2_backend writes all
its uploads under its own relative `data/` dir (`/app/data` in-container) —
mount a named volume `v2_data` there; do NOT mount `problems`/`media` into
v2_backend. The nginx publish binds `127.0.0.1` so the same file is
VPS-safe.

**Files:**
- Create: `docker-compose.v2.yml`
- Create: `local/v2-nginx/conf.d/v2.conf`
- Create: `local/v2.local.env` (gitignored)
- Create: `.env` (gitignored; compose interpolation)
- Modify: `.gitignore` (repo `CLAOJ/claoj-docker`)

**Interfaces:**
- Consumes: images `claoj/claoj-go`, `claoj/claoj-web` (Tasks 1–2), `claoj/judge-tiervnoj` (already built).
- Produces: services `v2_backend`, `v2_web`, `v2_nginx`, `v1_judge`, `v2_judge`; network `v2`; volume `v2_data`; env contract `${CLAOJ_DATA}`, `${V2_HTTP_PORT}`, `${V1_JUDGE_KEY}`, `${V2_JUDGE_KEY}`.

- [ ] **Step 1: Write `docker-compose.v2.yml`**

```yaml
# v2 services overlay. Never edit the v1 base files.
# Local:  docker compose -f docker-compose.local.yml -f docker-compose.v2.yml up -d
# Prod:   docker compose -f docker-compose.yml       -f docker-compose.v2.yml up -d
# Env-specific values live in .env (compose interpolation) and local/v2.local.env
# (backend env). See docs/superpowers/specs/2026-07-19-v1-v2-combined-compose-design.md.

name: claoj

services:
  v2_backend:
    container_name: claoj_v2_backend
    image: claoj/claoj-go
    build:
      context: ../../../OJ-v2/claoj
    restart: unless-stopped
    env_file: [local/v2.local.env]
    volumes:
      # v2's own uploads (problem data, PDFs) live under /app/data — its own
      # volume, NOT the shared v1 problems/media trees.
      - v2_data:/app/data
    networks: [db, site, judge, v2]
    depends_on: [db, redis]
    healthcheck:
      test: ["CMD-SHELL", "wget -q -O /dev/null http://127.0.0.1:8081/health || exit 1"]
      interval: 30s
      timeout: 10s

  v2_web:
    container_name: claoj_v2_web
    image: claoj/claoj-web
    build:
      context: ../../../OJ-v2/claoj-web
    restart: unless-stopped
    environment:
      NODE_ENV: production
      PORT: 3000
      # runtime-read (not build-baked): SSR metadata fetchers reach the backend
      # over the compose network
      API_URL_INTERNAL: http://v2_backend:8081/api
    networks: [v2]
    healthcheck:
      test: ["CMD-SHELL", "wget -q -O /dev/null http://127.0.0.1:3000/ || exit 1"]
      interval: 30s
      timeout: 10s

  v2_nginx:
    container_name: claoj_v2_nginx
    image: nginx:alpine
    restart: unless-stopped
    volumes:
      - ./local/v2-nginx/conf.d/:/etc/nginx/conf.d/:ro
    networks: [nginx, v2]
    ports:
      # loopback-bound: local browsing via localhost:8090; on the VPS this is
      # host-only (cloudflared reaches claoj_v2_nginx over the nginx network)
      - "127.0.0.1:${V2_HTTP_PORT:-8090}:80"
    depends_on: [v2_backend, v2_web]

  v1_judge:
    container_name: claoj_v1_judge
    image: claoj/judge-tiervnoj
    restart: unless-stopped
    cap_add: [SYS_PTRACE]
    volumes:
      - ${CLAOJ_DATA}/problems:/problems
    networks: [judge]
    command: ["run", "-p", "9999", "-A", "0.0.0.0", "-a", "9998", "-c", "/problems/judge.yml", "bridged", "Vịt Con", "${V1_JUDGE_KEY}"]
    depends_on: [bridged]

  v2_judge:
    container_name: claoj_v2_judge
    image: claoj/judge-tiervnoj
    restart: unless-stopped
    cap_add: [SYS_PTRACE]
    volumes:
      - ${CLAOJ_DATA}/problems:/problems
    networks: [judge]
    command: ["run", "-p", "9997", "-A", "0.0.0.0", "-a", "9998", "-c", "/problems/judge.yml", "v2_backend", "Vịt Toàn Năng", "${V2_JUDGE_KEY}"]
    depends_on: [v2_backend]

networks:
  v2:

volumes:
  v2_data:
```

- [ ] **Step 2: Write `local/v2-nginx/conf.d/v2.conf`**

```nginx
server {
    listen 80 default_server;
    server_name _;
    client_max_body_size 256m;
    charset utf-8;

    # WebSocket (live submission updates) — must precede the /api/ prefix block
    location /api/events {
        proxy_pass http://v2_backend:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $http_host;
        proxy_read_timeout 86400;
    }

    location /api/ {
        proxy_pass http://v2_backend:8081;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        proxy_pass http://v2_web:3000;
        proxy_set_header Host $http_host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

(For prod the only change later is `server_name beta.claoj.edu.vn;` — noted in Task 6 docs.)

- [ ] **Step 3: Create `local/v2.local.env`** (gitignored — real values, never committed)

```bash
cd /f/Coding/CLAOJ/CLAOJ/claoj-docker/claoj
MYSQL_PW=$(grep MYSQL_PASSWORD environment/mysql.env | cut -d= -f2)
SK=$(openssl rand -base64 48 | tr -d '\n')
JK=$(openssl rand -base64 48 | tr -d '\n')
cat > local/v2.local.env <<EOF
DATABASE_DSN=claoj:${MYSQL_PW}@tcp(db:3306)/claoj?charset=utf8mb4&parseTime=True&loc=UTC
REDIS_ADDR=redis:6379
REDIS_DB=2
SERVER_PORT=8081
SERVER_MODE=release
BRIDGE_ADDR=:9997
SECRET_KEY=${SK}
JWT_SECRET_KEY=${JK}
SITE_URL=http://localhost:8090
EMAIL_NO_REPLY=true
EOF
```

- [ ] **Step 4: Create `.env`** (gitignored; compose variable interpolation)

```bash
V1K=$(docker exec claoj_db mariadb -uclaoj -p"$MYSQL_PW" claoj -N -e "SELECT auth_key FROM judge_judge WHERE name='Vịt Con';")
V2K=$(docker exec claoj_db mariadb -uclaoj -p"$MYSQL_PW" claoj -N -e "SELECT auth_key FROM judge_judge WHERE name='Vịt Toàn Năng';")
cat > .env <<EOF
CLAOJ_DATA=F:\\Coding\\CLAOJ\\CLAOJ\\claoj-data
V2_HTTP_PORT=8090
V1_JUDGE_KEY=${V1K}
V2_JUDGE_KEY=${V2K}
EOF
```

Expected: both keys are 64-char hex strings (query them, don't guess).

- [ ] **Step 5: Gitignore the env files**

Append to `CLAOJ/claoj-docker/.gitignore`:

```
claoj/.env
claoj/local/v2.local.env
```

- [ ] **Step 6: Validate the merged config**

```bash
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml config --services | sort
```

Expected output (13 lines): `base bridged celery db nginx redis site v1_judge v2_backend v2_judge v2_nginx v2_web wsevent`.

Also confirm no v1 service was altered:

```bash
docker compose -f docker-compose.local.yml config > /tmp/v1-only.yml
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml config > /tmp/merged.yml
diff <(grep -A5 "claoj_site" /tmp/v1-only.yml) <(grep -A5 "claoj_site" /tmp/merged.yml)
```

Expected: empty diff.

- [ ] **Step 7: Commit (CLAOJ/claoj-docker repo)**

```bash
cd /f/Coding/CLAOJ/CLAOJ/claoj-docker
git add claoj/docker-compose.v2.yml claoj/local/v2-nginx/conf.d/v2.conf .gitignore
git commit -m "feat: v2 overlay compose (backend+web+nginx same-origin, dual judges)"
```

---

### Task 4: Bring up the combined stack and verify isolation

**Files:** none (operations + verification).

**Interfaces:**
- Consumes: everything from Tasks 1–3.
- Produces: the running combined stack all later verification depends on.

- [ ] **Step 1: Stop the ad-hoc host processes from the earlier session**

The host-run backend/web/judge must not fight the compose ones:

```bash
powershell -Command "Stop-Process -Name claoj-server -Force -ErrorAction SilentlyContinue"
powershell -Command "Get-Process node -ErrorAction SilentlyContinue | Where-Object {\$_.MainWindowTitle -eq ''} | Out-Null"  # leave IDE node alone; kill the dev server terminal manually if running
docker rm -f claoj_judge_local 2>/dev/null || true
```

Expected: `claoj-server.exe` gone; old judge container removed. (The old Next dev server on :3000 doesn't conflict — compose publishes nothing on 3000 — but stop it anyway to avoid confusion.)

- [ ] **Step 2: Up the full stack**

```bash
cd /f/Coding/CLAOJ/CLAOJ/claoj-docker/claoj
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml up -d
docker compose -f docker-compose.local.yml -f docker-compose.v2.yml ps --format '{{.Name}} {{.Status}}'
```

Expected: 12 containers up (`base` exits — that's normal); `claoj_v2_backend`, `claoj_v2_web`, `claoj_v2_nginx`, `claoj_v1_judge`, `claoj_v2_judge` all `Up`.

- [ ] **Step 3: v1 untouched**

```bash
curl -s -o /dev/null -w "v1 %{http_code}\n" http://localhost:8080/
```

Expected: `v1 200`.

- [ ] **Step 4: v2 same-origin through nginx**

```bash
curl -s -o /dev/null -w "web %{http_code}\n" http://localhost:8090/vi
curl -s http://localhost:8090/api/problems | head -c 80; echo
curl -s -o /dev/null -w "ssr-meta %{http_code}\n" http://localhost:8090/vi/problems/aplusb
curl -s http://localhost:8090/vi/problems/aplusb | grep -o "<title>[^<]*</title>"
```

Expected: `web 200`; the API returns `{"data":[...`; `ssr-meta 200`; the title contains `aplusb: A cộng B` (proves SSR reached the backend via `API_URL_INTERNAL`).

- [ ] **Step 5: Same-origin auth round-trip (cookies + CSRF through nginx)**

```bash
JAR=$(mktemp)
curl -s -c "$JAR" -o /dev/null -w "login %{http_code}\n" -X POST http://localhost:8090/api/auth/login \
  -H 'Content-Type: application/json' -d '{"username":"v2test_user","password":"V2test!Claude99"}'
curl -s -b "$JAR" -c "$JAR" -o /dev/null -w "me %{http_code}\n" http://localhost:8090/api/user/me
CSRF=$(awk '$6=="csrf_token" {print $7}' "$JAR")
echo "csrf present: ${CSRF:+yes}"
```

Expected: `login 200`, `me 200`, `csrf present: yes`.

- [ ] **Step 6: Redis isolation**

```bash
docker exec claoj_redis redis-cli -n 2 keys '*' | head -5
docker exec claoj_redis redis-cli -n 2 dbsize
docker exec claoj_redis redis-cli -n 0 keys 'perm:*' | wc -l
docker exec claoj_redis redis-cli -n 0 keys 'asynq*' | wc -l
```

Expected: DB 2 is non-empty (asynq/tokens/perm keys); DB 0 contains **zero** `perm:*` and zero `asynq*` keys (v2 writes nothing outside DB 2).

- [ ] **Step 7: Commit any config corrections made while stabilizing**

```bash
cd /f/Coding/CLAOJ/CLAOJ/claoj-docker && git status --short
# if changes: git add -A claoj/docker-compose.v2.yml claoj/local/v2-nginx/ && git commit -m "fix: combined-stack startup corrections"
```

---

### Task 5: Dual-judge grading end-to-end

**Files:** none (operations + verification).

**Interfaces:**
- Consumes: running stack (Task 4); test accounts `v2test_user`/`v2test_admin` (password `V2test!Claude99`) already in the local DB; submissions 156143 (AC) / 156144 (WA) from the earlier session.

- [ ] **Step 1: Both judges authenticated**

```bash
docker logs claoj_v2_backend 2>&1 | grep "authenticated as"
docker exec claoj_db mariadb -uclaoj -p"$(grep MYSQL_PASSWORD claoj/environment/mysql.env | cut -d= -f2)" claoj \
  -N -e "SELECT name, online FROM judge_judge WHERE online=1;"
```

Expected: backend log shows `authenticated as Vịt Toàn Năng`; the DB shows BOTH `Vịt Con` (via v1 bridged) and `Vịt Toàn Năng` online. If `Vịt Con` is not online, check `docker logs claoj_v1_judge` for its handshake against `bridged`.

- [ ] **Step 2: v2 path — fresh submission grades to AC**

```bash
JAR=$(mktemp)
curl -s -c "$JAR" -o /dev/null -X POST http://localhost:8090/api/auth/login \
  -H 'Content-Type: application/json' -d '{"username":"v2test_user","password":"V2test!Claude99"}'
curl -s -b "$JAR" -c "$JAR" -o /dev/null http://localhost:8090/api/user/me
CSRF=$(awk '$6=="csrf_token" {print $7}' "$JAR")
SUB=$(curl -s -b "$JAR" -X POST http://localhost:8090/api/problem/aplusb/submit \
  -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" \
  -d '{"source":"import sys\ndata=sys.stdin.buffer.read().split()\nt=int(data[0])\nprint(\"\\n\".join(str(int(data[1+2*i])+int(data[2+2*i])) for i in range(t)))","language":"PY3"}')
echo "$SUB"
ID=$(echo "$SUB" | python -c "import json,sys; print(json.loads(sys.stdin.read())['submission_id'])")
for i in $(seq 1 20); do
  S=$(curl -s http://localhost:8090/api/submission/$ID | python -c "import json,sys; d=json.loads(sys.stdin.read()); print(d['status'], d.get('result'))")
  echo "[$i] $S"; case "$S" in D\ *|CE\ *|IE\ *) break;; esac; sleep 3
done
```

Expected: final line `D AC`.

- [ ] **Step 3: v1 path — rejudge an existing submission through Django**

```bash
docker exec -i claoj_site python manage.py shell <<'EOF'
from judge.models import Submission
s = Submission.objects.get(id=156144)
s.judge(rejudge=True)
print('requeued', s.id)
EOF
sleep 12
docker exec -i claoj_site python manage.py shell <<'EOF'
from judge.models import Submission
s = Submission.objects.get(id=156144)
print('status:', s.status, 'result:', s.result)
EOF
```

Expected: after the wait, `status: D result: WA` (the deliberately wrong `print(42)` submission, regraded by `v1_judge` via v1's bridged). If still `QU`, check `docker logs claoj_v1_judge` and `docker logs claoj_bridged`.

- [ ] **Step 4: Cross-visibility sanity**

```bash
curl -s "http://localhost:8090/api/submission/$ID" | python -c "import json,sys; d=json.loads(sys.stdin.read()); print(d['user'], d['result'])"
curl -s "http://localhost:8080/submissions/" -o /dev/null -w "v1 submissions page %{http_code}\n"
```

Expected: v2 API shows `v2test_user AC`; v1 page 200 (the new submission appears there too — same DB).

---

### Task 6: Documentation

**Files:**
- Modify: `OJ-v2/docs/deployment.md` (add combined-compose section)
- Modify: `OJ-v2/docs/development.md` (replace the "run v2 on the host" flow with the combined command; keep the host flow as an alternative for iterating on Go/TS code)

**Interfaces:**
- Consumes: everything validated in Tasks 4–5.

- [ ] **Step 1: `deployment.md` — add a "Combined compose (v1+v2, one project)" section** containing, concretely:
  - The two commands (local base vs prod base) with the shared `docker-compose.v2.yml`.
  - The env contract table: `.env` (`CLAOJ_DATA` — local `F:\Coding\CLAOJ\CLAOJ\claoj-data`, prod `/root/claoj-data`; `V2_HTTP_PORT`; judge keys) and `local/v2.local.env` → prod `local/v2.prod.env` (live DSN, `REDIS_DB=2`, fresh secrets, `SITE_URL=https://beta.claoj.edu.vn`).
  - Prod nginx tweak: `server_name beta.claoj.edu.vn;`.
  - The manual Cloudflare step: Zero Trust dashboard → tunnel → add public hostname `beta.claoj.edu.vn` → service `http://claoj_v2_nginx:80`, plus the DNS record; v1's `claoj.edu.vn` route untouched.
  - Prod bring-up limited to the v2 services: `docker compose -f docker-compose.yml -f docker-compose.v2.yml up -d v2_backend v2_web v2_nginx v2_judge` (and `v1_judge` if the VPS should also grade v1 with this image).
  - Rollback: `docker compose ... stop v2_backend v2_web v2_nginx v1_judge v2_judge`.
- [ ] **Step 2: `development.md` — update §4/§5/§6** to present the combined compose as the primary way to run v2 locally (`http://localhost:8090`), with the old host-run flow (`go run` + `npm run dev`) kept as the "fast iteration" alternative. Update the port map table (add 8090; note 8081/3000/9997 are now internal).
- [ ] **Step 3: Commit (OJ-v2 repo)**

```bash
cd /f/Coding/CLAOJ/OJ-v2
git add docs/deployment.md docs/development.md
git commit -m "docs: combined v1+v2 compose usage (local + beta.claoj.edu.vn deploy)"
```

---

## Self-review

- **Spec coverage:** topology/override (§2 spec → Tasks 1–3), conflict rules (§3 → Task 3 config + Task 4 Step 6 verification), service configs (§4 → Task 3), files (§5 → Tasks 3, 6), local rollout+testing (§6.1 → Tasks 4–5), prod procedure documented (§6.2 → Task 6), error handling is config-level (`restart`, healthchecks — Task 3). Spec open items resolved: Asynq index (verified in code — Global Constraints), judge keys (env-substituted, Task 3 Step 4), backend mounts (superseded: `v2_data`, Task 3 note).
- **Placeholders:** none — all configs/commands are complete and copy-pasteable; secrets are generated or queried, never invented.
- **Consistency:** service names, container names, ports (8090/8081/3000/9997/9999), env var names (`API_URL_INTERNAL`, `CLAOJ_DATA`, `V1_JUDGE_KEY`, `V2_JUDGE_KEY`, `REDIS_DB=2`) match across all tasks.
