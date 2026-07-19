# OJ-v2 Deployment Guide

How to deploy the v2 stack (Go API + Next.js web) **alongside the existing v1
Django deployment**, against the **same production database**, with zero schema
migrations.

> **Core guarantee:** v2 introduces **no Django migrations and no DDL**. The
> schema stays 100% Django-managed. A runtime guard (`db.Connect()`) panics if
> any `CREATE/ALTER/DROP` is ever attempted. The only one-time, out-of-band DDL
> is `scripts/v2_runtime_tables.sql`, which you run manually and review first.

---

## 1. Pre-flight

Before touching production:

1. **Back up the database.** Even though v2 issues no DDL, you are about to run
   one additive provisioning script and add a new writer to a live DB.
   ```bash
   mysqldump --single-transaction --routines --triggers claoj > claoj-backup-$(date +%F).sql
   ```
2. **Confirm the schema-parity test passes against a copy of production.**
   Restore the dump into a scratch DB, apply `v2_runtime_tables.sql`, then:
   ```bash
   CLAOJ_DJANGO_DB_DSN="user:pass@tcp(scratch-host:3306)/claoj_copy?parseTime=true" \
     go test ./integration/ -run TestSchemaParity -v
   ```
   This must be green before you deploy. It is the single best guard against a
   schema drift between what v2 expects and what Django actually has.
3. **Decide the rollout.** v2 is purely additive and read-mostly; you can run it
   in parallel and route only a fraction of traffic (or only `/api/*` and the
   new web app) to it while v1 keeps serving. Nothing about v2 stops v1 working.

---

## 2. Provision the v2-only schema (once)

v2 keeps its bookkeeping out of MySQL where it can (refresh tokens, one-time
tokens, and the audit log live in **Redis**). What remains are additive tables
and columns Django never reads or writes. Apply them once:

```bash
# Review it first — it is idempotent (CREATE TABLE IF NOT EXISTS / ADD COLUMN
# IF NOT EXISTS) and safe to re-run.
less OJ-v2/scripts/v2_runtime_tables.sql
mysql -u <admin> -p claoj < OJ-v2/scripts/v2_runtime_tables.sql
```

Objects created (see `docs/schema-audit.md` §3 for the decision record):

| Object | Backs |
|--------|-------|
| `notification`, `notification_preference` | in-app notifications |
| `totp_device`, `backup_code` | TOTP 2FA |
| `oauth_user_link` | OAuth login linking |
| `moss_result` | MOSS plagiarism-check cache |
| `judge_commentrevision` | comment edit history |
| `judge_contestclarification` | contest Q&A |
| `judge_solution` + `is_official, valid_until, summary, language` | editorial solution fields |

If you ever decommission v1-era experimental tables, `scripts/cleanup_v2_tables.sql`
removes the retired ones — review before running.

Requires MySQL 8.0.29+ / MariaDB 10.3+ for `ADD COLUMN IF NOT EXISTS`.

---

## 3. Deploy the Go API

### 3.1 Build

```bash
cd OJ-v2/claoj
CGO_ENABLED=0 go build -o claoj-server .
```

Produces a single static binary. Ship it with a systemd unit or a container.

### 3.2 Configuration (environment)

v2 reads config from env vars (or an optional `.env`), with Docker-secret file
support. **Never commit secrets.** Minimum production set:

```dotenv
# Database — the SAME database Django uses
DATABASE_DSN=claoj:<MYSQL_PASSWORD>@tcp(<db-host>:3306)/claoj?charset=utf8mb4&parseTime=True&loc=UTC
# or supply components + a Docker secret file:
#   MYSQL_HOST, MYSQL_PORT, MYSQL_USER, MYSQL_DATABASE, MYSQL_PASSWORD_FILE

REDIS_ADDR=<redis-host>:6379
REDIS_DB=0                     # match/segregate from Django's Redis DBs as desired

SERVER_PORT=8081
SERVER_MODE=release            # sets Gin to release mode

# Secrets — generate fresh, do not reuse Django's:
SECRET_KEY=<openssl rand -base64 48>       # or SECRET_KEY_FILE=/run/secrets/...
JWT_SECRET_KEY=<openssl rand -base64 48>   # or JWT_SECRET_KEY_FILE=...

# Public origin of the web app — drives CORS allow-list AND cookie Secure/SameSite:
SITE_URL=https://beta.claoj.edu.vn

# Judge bridge listen address. MUST differ from the v1 bridge (:9999) if both
# run on the same host/interface:
BRIDGE_ADDR=:9997

# Email (optional; set EMAIL_NO_REPLY=false and SMTP_* to actually send)
EMAIL_NO_REPLY=true

# OAuth (optional)
# OAUTH_GOOGLE_CLIENT_ID / _SECRET / _REDIRECT_URL / _ENABLED
# OAUTH_GITHUB_CLIENT_ID / _SECRET / _REDIRECT_URL / _ENABLED
```

**Security-critical settings:**

- `SITE_URL` **must** be the exact public HTTPS origin of the web app. It is the
  sole entry in the CORS allow-list, and it decides cookie flags: an
  `https://` origin makes auth cookies `Secure; SameSite=None` (required for the
  web app on a different subdomain). If `SITE_URL` is `http://`, cookies fall
  back to `SameSite=Lax` — **do not run production over plain HTTP.**
- The server **refuses to start** if `SECRET_KEY` is empty or the literal
  `changeme` — this is intentional.
- Prefer `*_FILE` env vars pointing at Docker/Kubernetes secrets over inline
  values.

### 3.3 Run

```bash
./claoj-server           # honours the env above
```

Behind nginx, proxy `/api/` and the events WebSocket to `:8081`:

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:8081;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
location /api/events {          # WebSocket for live submission updates
    proxy_pass http://127.0.0.1:8081;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 86400;
}
```

---

## 4. Deploy the Next.js web app

### 4.1 Build

```bash
cd OJ-v2/claoj-web
npm ci
npm run build
```

Build-time / runtime env (`.env.production` or the process environment):

```dotenv
NEXT_PUBLIC_API_URL=https://beta.claoj.edu.vn/api
NEXT_PUBLIC_WS_URL=wss://beta.claoj.edu.vn/api/events
SITE_URL=https://beta.claoj.edu.vn
```

`NEXT_PUBLIC_*` values are inlined at build time — rebuild if they change.

### 4.2 Run

```bash
npm run start            # Next.js server, default :3000
```

Serve it behind the same nginx (or a CDN). Because the API and web share the
public origin (`beta.claoj.edu.vn` → `/api` to Go, everything else to Next),
cookies are first-party and CSRF/CORS "just work".

---

## 5. Judges

The judge servers are unchanged from the v1/DMOJ model — they connect over TCP
to a **bridge**. Point judges at v2's bridge (`BRIDGE_ADDR`, default `:9997`)
instead of (or in addition to) the v1 bridge:

- Each judge needs a row in `judge_judge` with a matching `name` + `auth_key`
  (managed in Django admin).
- Open the bridge port to the judge hosts only (firewall it off the public net).
- Build the judge image from `claoj-docker/judge/tiervnoj`. **Note the Python
  3.14 `fork` start-method patch** in `dmoj/judge.py` (see `docs/development.md`
  §6) — required for grading to work on the current runtimes base.

```bash
docker run -d --name claoj_judge --cap-add SYS_PTRACE \
  -v /srv/problems:/problems \
  claoj/judge-tiervnoj \
  run -p 9997 -A 0.0.0.0 -a 9998 -c /problems/judge.yml <bridge-host>
```

Verify: `GET /api/judges` lists the judge `online`; a test submission
transitions `QU → P → G → D`.

---

## 6. Post-deploy smoke test

Run against the live deployment (read-only except the one test submission):

```bash
# public reads
for u in /api/problems /api/contests /api/users /api/stats/overall /api/judges; do
  curl -s -o /dev/null -w "$u %{http_code}\n" https://beta.claoj.edu.vn$u
done

# auth round-trip (a throwaway test account created in Django)
#  1. POST /api/auth/login  → sets access/refresh cookies
#  2. GET  /api/user/me     → 200 + plants csrf_token cookie
#  3. POST a mutation with X-CSRF-Token header → succeeds

# permission parity: grant judge.see_private_problem to the test user via a
# Django group, confirm a hidden problem becomes visible in v2 within ~60s,
# revoke it, confirm it 404s again.

# grading: submit to a known problem, confirm it reaches D/AC.
```

---

## 7. Rollback

v2 is additive, so rollback is low-risk:

- **Stop v2** (API + web). v1 Django is entirely unaffected — it never depended
  on v2, the v2-only tables, or Redis keys v2 owns.
- The `v2_runtime_tables.sql` objects can be left in place (Django ignores them)
  or removed with `cleanup_v2_tables.sql` if you want a clean teardown.
- No Django migration was applied, so there is nothing to reverse on the schema.

---

## 8. Operational notes

- **Redis usage:** v2 stores refresh-token families, password-reset/email-verify
  one-time tokens, the permission cache (`perm:v{N}:{user_id}`, 60s TTL), and a
  capped audit-log stream (`XADD audit:log MAXLEN ~100000`). Sizing is modest;
  use a dedicated Redis DB index (`REDIS_DB`) to keep it separate from Django's
  cache/celery if they share a server.
- **Permission cache:** changes made in Django admin propagate to v2 within the
  60s TTL. v2-side admin writes bump the version key for immediate invalidation.
  If you change permissions in Django and need instant effect in v2, wait out
  the TTL or flush the `perm:*` keys.
- **Clock/timezone:** the DB uses `USE_TZ`; the DSN pins `loc=UTC`. Keep the Go
  host clock in sync (NTP) — contest timing and rating windows depend on it.
- **Monitoring:** v2 exposes Prometheus metrics at `/metrics` and a health probe
  at `/health`.
- **Two writers, one DB:** both Django and Go now write submissions, votes,
  participations, etc. All v2 writes use the same tables and column semantics as
  Django (verified by the parity test and the identity-handling rules in
  `docs/development.md` §7.1). There is no dual-write reconciliation to manage —
  it is one database with two application front-ends.
