#!/usr/bin/env bash
# USER-RUN ONLY. Copies a few production tables into a local MariaDB for the
# plagiarism benchmark. Prod is only read: mariadb-dump and SELECT inside the
# prod DB container. Nothing is written to prod.
#
# Overrides (env): CLAOJ_SSH_HOST (root@claoj), CLAOJ_DB_CONTAINER (claoj_db),
# CLAOJ_DB_NAME (claoj). Assumes the prod container's root can connect over
# its unix socket (the stock mariadb image default).
set -euo pipefail

HOST="${CLAOJ_SSH_HOST:-root@claoj}"
PROD_CONTAINER="${CLAOJ_DB_CONTAINER:-claoj_db}"
PROD_DB="${CLAOJ_DB_NAME:-claoj}"

LOCAL_CONTAINER=claoj-bench-db
LOCAL_VOLUME=claoj-bench-db
LOCAL_PORT=3307
LOCAL_DB=claoj_bench

HERE="$(cd "$(dirname "$0")" && pwd)"
DUMPS="$HERE/dumps"
mkdir -p "$DUMPS"

CORE_TABLES="judge_language judge_problem judge_contest judge_contestproblem judge_contestparticipation judge_contestsubmission"

# Run a command inside the prod DB container over ssh. $1 is the command line
# for the container's sh; printf %q keeps quoting intact across both shells.
prod() {
  local inner="$1"
  ssh "$HOST" "docker exec $PROD_CONTAINER sh -c $(printf '%q' "$inner")"
}

echo "dump core tables"
prod "mariadb-dump -uroot --single-transaction --skip-lock-tables $PROD_DB $CORE_TABLES" > "$DUMPS/core.sql"

echo "dump contest submissions"
prod "mariadb-dump -uroot --single-transaction --skip-lock-tables $PROD_DB judge_submission --where=$(printf '%q' 'contest_object_id IS NOT NULL')" > "$DUMPS/submissions.sql"

echo "dump contest submission sources"
prod "mariadb-dump -uroot --single-transaction --skip-lock-tables $PROD_DB judge_submissionsource --where=$(printf '%q' 'submission_id IN (SELECT id FROM judge_submission WHERE contest_object_id IS NOT NULL)')" > "$DUMPS/sources.sql"

echo "export reduced user tables as TSV"
prod "mariadb -uroot -N -B $PROD_DB -e $(printf '%q' 'SELECT id, username FROM auth_user')" > "$DUMPS/auth_user.tsv"
prod "mariadb -uroot -N -B $PROD_DB -e $(printf '%q' 'SELECT id, user_id FROM judge_profile')" > "$DUMPS/judge_profile.tsv"

echo "start local MariaDB on 127.0.0.1:$LOCAL_PORT"
docker volume inspect "$LOCAL_VOLUME" >/dev/null 2>&1 || docker volume create "$LOCAL_VOLUME" >/dev/null
if docker ps -a --format '{{.Names}}' | grep -qx "$LOCAL_CONTAINER"; then
  docker start "$LOCAL_CONTAINER" >/dev/null
else
  docker run -d --name "$LOCAL_CONTAINER" -p "127.0.0.1:$LOCAL_PORT:3306" \
    -v "$LOCAL_VOLUME:/var/lib/mysql" \
    -e MARIADB_ALLOW_EMPTY_ROOT_PASSWORD=1 \
    mariadb:11 >/dev/null
fi
for _ in $(seq 1 60); do
  docker exec "$LOCAL_CONTAINER" mariadb -uroot -e 'SELECT 1' >/dev/null 2>&1 && break
  sleep 2
done

local_sql() { docker exec -i "$LOCAL_CONTAINER" mariadb -uroot "$@"; }

echo "load into $LOCAL_DB (drops and recreates it)"
local_sql -e "DROP DATABASE IF EXISTS $LOCAL_DB; CREATE DATABASE $LOCAL_DB"
local_sql "$LOCAL_DB" < "$DUMPS/core.sql"
local_sql "$LOCAL_DB" < "$DUMPS/submissions.sql"
local_sql "$LOCAL_DB" < "$DUMPS/sources.sql"
local_sql "$LOCAL_DB" -e "
  CREATE TABLE auth_user (id INT PRIMARY KEY, username VARCHAR(150) NOT NULL);
  CREATE TABLE judge_profile (id INT PRIMARY KEY, user_id INT NOT NULL);"

# Load the reduced TSVs through a file inside the container (server-side LOAD DATA).
load_tsv() { # $1 table, $2 tsv file name, $3 column list
  docker exec -i "$LOCAL_CONTAINER" sh -c "cat > /tmp/$2" < "$DUMPS/$2"
  local_sql "$LOCAL_DB" -e "LOAD DATA INFILE '/tmp/$2' INTO TABLE $1 FIELDS TERMINATED BY '\t' ($3)"
  docker exec "$LOCAL_CONTAINER" rm -f "/tmp/$2"
}
load_tsv auth_user auth_user.tsv "id, username"
load_tsv judge_profile judge_profile.tsv "id, user_id"

echo "done. Local DB: mysql://root@127.0.0.1:$LOCAL_PORT/$LOCAL_DB"
echo "dumps kept in $DUMPS (gitignored; they contain contestants' code)"
