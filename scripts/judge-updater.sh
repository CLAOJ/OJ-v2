#!/usr/bin/env bash
#
# Auto-update the CLAOJ v2 judge agent on the box, but recreate the container
# ONLY when the asynq "judge" queue is empty, so a rebuild never interrupts a
# grade. Run from claoj-judge-updater.timer (see docs/deployment.md §10 and
# /root/claoj-v2/RUNBOOK.md).
#
# Watchtower is deliberately NOT used for the judge: its lifecycle hooks cannot
# veto an update (a failing pre-check only logs), and the agent image is built
# on the box rather than pulled from a registry.
#
# The judge source is a partial + sparse checkout of the public OJ-v2 repo, so
# only claoj-judge/ is downloaded — not the whole monorepo. This script itself
# is installed at a stable path (/root/claoj-v2/judge-updater.sh); re-copy it
# from scripts/ in the repo if it changes.
set -euo pipefail

GIT_REMOTE="${GIT_REMOTE:-https://github.com/CLAOJ/OJ-v2.git}"
SRC="${SRC:-/root/OJ-v2-src}"
COMPOSE="${COMPOSE:-/root/claoj-v2/docker-compose.yml}"
STATE="${STATE:-/root/claoj-v2/judge-agent.built-sha}"
IMAGE="${IMAGE:-claoj/claoj-judge-agent}"
TIERVNOJ_IMAGE="${TIERVNOJ_IMAGE:-judge-judge1}"
REDIS_CONTAINER="${REDIS_CONTAINER:-claoj_redis}"
REDIS_DB="${REDIS_DB:-2}"

log() { printf '%s judge-updater: %s\n' "$(date -Is)" "$*"; }

# 1. Latest judge source (sparse: only what the build needs).
if [ ! -d "$SRC/.git" ]; then
    log "cloning $GIT_REMOTE (sparse: claoj-judge) into $SRC"
    git clone --depth 1 --filter=blob:none --sparse "$GIT_REMOTE" "$SRC"
    git -C "$SRC" sparse-checkout set claoj-judge
else
    git -C "$SRC" fetch --depth 1 origin main
    git -C "$SRC" reset --hard FETCH_HEAD
fi

sha=$(git -C "$SRC" rev-parse HEAD:claoj-judge)
if [ -f "$STATE" ] && [ "$(cat "$STATE")" = "$sha" ]; then
    exit 0  # already running this source
fi

# 2. Build the new agent image. This never touches the running container.
log "building $IMAGE from judge source $sha"
docker build --build-arg "TIERVNOJ_IMAGE=$TIERVNOJ_IMAGE" -t "$IMAGE" "$SRC/claoj-judge"

# 3. Gate: recreate only when nothing is pending, active, scheduled or retrying.
# An unreadable queue is treated as busy — never guess.
busy=0
for q in pending active scheduled retry; do
    n=$(docker exec "$REDIS_CONTAINER" redis-cli -n "$REDIS_DB" LLEN "asynq:{judge}:$q") || {
        log "cannot read asynq:{judge}:$q (redis unavailable); deferring recreate"
        exit 0
    }
    busy=$((busy + n))
done
if [ "$busy" -gt 0 ]; then
    log "judge queue not empty ($busy tasks); image built, deferring recreate"
    exit 0
fi

# 4. Recreate and remember what we deployed.
log "judge queue empty; recreating v2_judge"
docker compose -f "$COMPOSE" up -d v2_judge
printf '%s\n' "$sha" > "$STATE"
log "v2_judge updated to judge source $sha"