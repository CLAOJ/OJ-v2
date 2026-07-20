# OJ-v2 CI/CD — Design

**Date:** 2026-07-21
**Status:** Approved design, pending spec review
**Scope:** Continuous integration + auto-deploy of the OJ-v2 backend and web apps to the `beta.claoj.edu.vn` VPS.

---

## 1. Goal

Every change that lands on `main` is tested, packaged as a container image, and
deployed to the beta VPS automatically — with no inbound access to the box and no
source compilation on the box. Day-to-day work happens on a `dev` branch where CI
proves the code and the image build, but publishes nothing.

Put plainly: **merge to `main` → beta updates itself within a couple of minutes.**

---

## 2. Current state (context)

Two separate Git repositories cooperate to run v2:

- **`OJ-v2`** (`github.com/CLAOJ/OJ-v2`) — the source. Three apps:
  `claoj/` (Go API), `claoj-web/` (Next.js), `claoj-judge/` (Go judge, **not**
  the production judge). Dockerfiles exist for `claoj` and `claoj-web`. No CI today.
- **`claoj-docker`** (`github.com/bachtam2001/claoj-docker`) — the deployment.
  `claoj/docker-compose.v2.yml` overlays five `claoj_v2_*` services onto v1's
  existing Compose project and currently **builds** the backend/web images on the
  box from OJ-v2 source at a sibling path (`build: context: ../../../OJ-v2/claoj`).

Production is a single VPS running the v1 Django stack + the v2 overlay, fronted by
a **Cloudflare tunnel** (`cloudflared`). The VPS has no exposed inbound ports for
deployment. Env files (`local/v2.prod.env`, `.env`) live only on the box and are
gitignored. v2 issues **no DDL**; the one-time `scripts/v2_runtime_tables.sql` is
applied out-of-band.

---

## 3. Decisions

Each row was chosen deliberately during design; the rationale is the trade-off that
won.

| Decision | Choice | Why |
|---|---|---|
| Deploy path to VPS | **Pull-based** (VPS reaches out) | No inbound ports; works behind the Cloudflare tunnel. |
| Build location | **CI builds & pushes images; VPS pulls** | VPS stays light; only tested commits become images, so CI gates CD. |
| Registry | **GitHub Container Registry (ghcr.io)**, private | Free, native to the repo, CI pushes with the built-in `GITHUB_TOKEN`. |
| Poller on VPS | **Watchtower**, label-scoped to v2 only | Off-the-shelf; recreates a container when its image digest changes. |
| CI checks | **Unit tests + builds** (skip DB-backed integration) | Fast, no service containers; catches common breakage. |
| Trigger / target | **Push to `main` → auto-deploy to beta** | Continuous deployment while v2 is beta-only. |
| Dev workflow | **`dev` branch**; images publish **only on `main`** | Safe iteration space; the deployable tag moves only on merge to `main`. |

---

## 4. Branch & trigger model

```
feature/* ──PR──► dev ──────PR/merge──────► main
                   │                          │
         CI: test + build-VALIDATE   CI: test + build + PUSH
              (push: false)            :latest + :sha-<short>  → GHCR
                   │                          │
              nothing deploys         Watchtower → beta updates
```

- **`dev`** — integration branch for local development and testing.
  On push/PR to `dev`: run the **test jobs** and **build the images to validate**
  they still build (`push: false`). Nothing is published; beta is untouched.
- **`main`** — release/beta branch.
  On push to `main` (i.e. a merge): run tests, then **build and push** the images
  tagged `:latest` (moving) and `:sha-<short>` (immutable) to GHCR. Watchtower on
  the VPS then updates beta.
- **Pull requests** into `dev` or `main`: run tests + build-validate (no push) so
  review is gated on a green pipeline.

This satisfies the rule **"image only applies when merged to `main`."** A `:dev`
tag / dedicated dev box can be added later if a separate dev deployment is wanted;
it is intentionally out of scope now.

---

## 5. CI pipeline — `OJ-v2/.github/workflows/ci-cd.yml`

- **Triggers:** `push` to `[main, dev]`, `pull_request` to `[main, dev]`,
  `workflow_dispatch` (manual). Concurrency group per-ref cancels superseded runs.
- **Permissions:** `contents: read`, `packages: write`. No new secret needed —
  GHCR push uses the automatic `GITHUB_TOKEN`.
- **Path filtering** (`dorny/paths-filter`): only rebuild what changed.
  `backend` ← `claoj/**` (+ the workflow file); `web` ← `claoj-web/**` (+ the
  workflow file). A docs-only push builds nothing.

**Jobs**

| Job | Runs when | Steps |
|---|---|---|
| `changes` | always | path-filter → outputs `backend`, `web` booleans |
| `test-backend` | `backend` changed | Go 1.25: `go vet ./...`; `go test` over all packages **except `./integration/`** (that suite needs a live DB) |
| `test-web` | `web` changed | Node 20: `npm ci`; ESLint; `jest`. (`next build` is exercised in the image build.) |
| `build-push-backend` | `backend` changed, `test-backend` green | `docker/build-push-action`, context `claoj/`, buildx + GHA layer cache, platform `linux/amd64`, `push` = *(ref is `main`)* |
| `build-push-web` | `web` changed, `test-web` green | same, context `claoj-web/`, target `ghcr.io/claoj/claoj-web` |

`push: ${{ github.ref == 'refs/heads/main' }}` — on `dev`/PRs the build runs but
publishes nothing.

Representative skeleton (final YAML produced in the implementation plan):

```yaml
name: ci-cd
on:
  push: { branches: [main, dev] }
  pull_request: { branches: [main, dev] }
  workflow_dispatch:
concurrency: { group: ci-cd-${{ github.ref }}, cancel-in-progress: true }
permissions: { contents: read, packages: write }
jobs:
  changes: ...            # dorny/paths-filter → backend / web
  test-backend: ...       # go vet + go test (exclude ./integration)
  test-web: ...           # npm ci + eslint + jest
  build-push-backend:     # needs test-backend
    steps:
      - uses: docker/metadata-action   # tags: latest (main only) + sha-<short>
      - uses: docker/login-action      # ghcr.io, GITHUB_TOKEN
      - uses: docker/build-push-action
        with:
          context: claoj
          platforms: linux/amd64
          push: ${{ github.ref == 'refs/heads/main' }}
  build-push-web: ...     # needs test-web, context: claoj-web
```

---

## 6. Registry & image tags

- Images: `ghcr.io/claoj/claoj-go` and `ghcr.io/claoj/claoj-web`, **private**.
- Tags on a `main` merge:
  - `:latest` — moving; the tag Watchtower watches.
  - `:sha-<short>` — immutable; traceability and rollback target.
- First-time only: confirm the two GHCR packages are linked to the `OJ-v2` repo and
  the beta VPS has read access (via the login token in §9).

---

## 7. VPS / Compose changes — `claoj-docker` repo

Edit `claoj/docker-compose.v2.yml`:

1. **Point images at GHCR** (was Docker-Hub-style implicit names):
   - `v2_backend`: `image: ghcr.io/claoj/claoj-go:latest`
   - `v2_web`: `image: ghcr.io/claoj/claoj-web:latest`
   The existing `build:` blocks **stay** (still useful for local dev); the VPS only
   ever pulls.
2. **Opt the two v2 app services into Watchtower** with labels:
   ```yaml
   labels:
     com.centurylinklabs.watchtower.enable: "true"
     com.centurylinklabs.watchtower.scope: "claoj-v2"
   ```
3. **Add a Watchtower service** to the overlay:
   ```yaml
   watchtower:
     container_name: claoj_v2_watchtower
     image: containrrr/watchtower
     restart: unless-stopped
     volumes:
       - /var/run/docker.sock:/var/run/docker.sock
       - /root/.docker/config.json:/config.json:ro   # ghcr read auth
     environment:
       WATCHTOWER_LABEL_ENABLE: "true"     # only manage labelled containers
       WATCHTOWER_SCOPE: "claoj-v2"        # ... and only this scope
       WATCHTOWER_CLEANUP: "true"          # prune replaced images
       WATCHTOWER_POLL_INTERVAL: "120"     # seconds
     labels:
       com.centurylinklabs.watchtower.scope: "claoj-v2"
     networks: [v2]
   ```

`WATCHTOWER_LABEL_ENABLE` + `WATCHTOWER_SCOPE` together guarantee Watchtower touches
**only** `claoj_v2_backend` and `claoj_v2_web`. The entire v1 Django stack, both
judges, and `v2_nginx` are never candidates for update.

Deploy invocation is unchanged (`docker compose -f docker-compose.yml -f
docker-compose.v2.yml up -d`); Watchtower simply becomes one more overlay service.

### 7.1 Keep `v2_nginx` re-resolving upstreams (required)

**Problem.** `local/v2-nginx/conf.d/v2.conf` currently does
`proxy_pass http://v2_backend:8081;` (and `http://v2_web:3000;`) with a hardcoded
name and **no `resolver`**. nginx resolves those names **once at worker start** and
caches the IP. When Watchtower recreates the backend/web container, Docker assigns
it a **new IP**; nginx keeps proxying the dead one → **502 until nginx is manually
restarted.** Without this fix, every auto-deploy is an outage.

**Fix.** Make nginx re-resolve at request time using Docker's embedded DNS. In
`v2.conf`, add the resolver once and switch each upstream to a variable so nginx
defers resolution (keep `$request_uri` so the path/query are passed unchanged):

```nginx
resolver 127.0.0.11 valid=10s ipv6=off;   # Docker embedded DNS

location /api/events {
    set $v2_backend v2_backend;
    proxy_pass http://$v2_backend:8081$request_uri;
    # ...existing WebSocket + proxy headers unchanged...
}
location /api/ {
    set $v2_backend v2_backend;
    proxy_pass http://$v2_backend:8081$request_uri;
    # ...existing proxy headers unchanged...
}
location / {
    set $v2_web v2_web;
    proxy_pass http://$v2_web:3000$request_uri;
    # ...existing proxy headers unchanged...
}
```

This makes nginx resilient to *any* container recreate, not just Watchtower's, and
is Docker-idiomatic. (Alternative — restarting `v2_nginx` after each update — is
rejected: it adds a moving part and a brief hard outage on every deploy.) The
implementation plan finalizes exact syntax and re-verifies the WebSocket path.

---

## 8. Out of scope / unchanged

- **Judges** (`v1_judge`/`v2_judge` = `claoj/judge-tiervnoj`) — built from
  `claoj-docker`, change rarely, not part of this pipeline; unlabelled, so Watchtower
  ignores them.
- **`claoj-judge/` (Go judge in OJ-v2)** — not the production judge; not built or
  deployed here. (Its Go tests may be added to CI later if desired.)
- **v1 Django stack** — entirely untouched.
- **Env / secrets** (`local/v2.prod.env`, `.env`) — stay on the box, gitignored,
  never read or written by CI/CD.
- **DB migrations** — v2 issues no DDL by design; `v2_runtime_tables.sql` stays a
  manual, out-of-band step. No migration runs in the deploy path.

---

## 9. Secrets & one-time setup

- **CI (GitHub Actions):** no new secret. GHCR push authenticates with the built-in
  `GITHUB_TOKEN` (`packages: write`).
- **VPS (once):** `docker login ghcr.io` with a read-only PAT (`read:packages`),
  which writes `/root/.docker/config.json`. Watchtower mounts that file to pull the
  private images.
- Document both in `OJ-v2/docs/deployment.md` (new "CI/CD" section).

---

## 10. Rollback & safety

- **Rollback:** repoint a service from `:latest` to a known-good `:sha-<short>` and
  `docker compose ... up -d <service>` (helper: `rollback.sh <backend|web> <sha>`).
  Watchtower tracks whichever tag a container runs, so a pinned `:sha-` tag freezes
  that service; repointing to `:latest` resumes auto-updates.
- **Failure window (known limitation):** a green-but-broken image *would* deploy.
  Mitigated by the CI gate, the existing `/health` (backend) and `/` (web)
  healthchecks, and fast `:sha-` rollback. Watchtower does **not** auto-rollback on
  an unhealthy container — accepted for a beta environment.
- **Notifications (optional):** Watchtower can post update summaries to
  Discord/Slack/email; included as a commented, off-by-default stub.

---

## 11. Files to create / change

**`OJ-v2` repo (branch `dev` → merge to `main`):**
- `.github/workflows/ci-cd.yml` — new (the pipeline).
- `docs/deployment.md` — new "CI/CD" section (§9 setup, §10 rollback).
- `docs/superpowers/specs/2026-07-21-oj-v2-cicd-design.md` — this doc.

**`claoj-docker` repo:**
- `claoj/docker-compose.v2.yml` — GHCR image refs, Watchtower labels, Watchtower
  service (§7).
- `claoj/local/v2-nginx/conf.d/v2.conf` — resolver + variable upstreams so nginx
  survives a backend/web recreate (§7.1). **Required for zero-downtime auto-deploy.**
- `claoj/scripts/rollback.sh` — new rollback helper (§10).
- `README.md` — note the login-once step and the auto-update behaviour.

**Git:** create/push the `dev` branch; open the first PR into `main` to exercise the
full path.

---

## 12. Verification plan

1. **Dev path:** push a trivial change to `dev` → CI runs tests + build-validate,
   pushes nothing to GHCR, beta unchanged.
2. **Main path:** merge to `main` → CI green → `ghcr.io/claoj/claoj-go:latest` and
   `:sha-<short>` appear in GHCR.
3. **Auto-deploy:** within ~2 min, Watchtower recreates `claoj_v2_backend` /
   `claoj_v2_web`; `docker logs claoj_v2_watchtower` shows the update.
4. **No-502 check (§7.1):** immediately after the recreate, `curl` beta `/api/...`
   and `/` — both must serve without a 502, proving nginx re-resolved the new
   container IP without a restart.
5. **Smoke test** per `deployment.md` §6 against `https://beta.claoj.edu.vn`
   (public reads, auth round-trip, a graded submission).
6. **Isolation:** confirm no v1 container was recreated (`docker ps` uptimes).
7. **Rollback:** pin `claoj-go` to the previous `:sha-`, confirm the box serves it,
   repoint to `:latest`, confirm auto-update resumes.

---

## 13. Future (explicitly deferred)

- A separate production environment + a tag/release-gated promotion when v2
  graduates past beta.
- DB-backed integration tests (`TestSchemaParity`) in CI via MySQL/Redis service
  containers.
- A `:dev` image + dedicated dev deployment box.
- Multi-arch images if the VPS ever runs on `arm64`.
