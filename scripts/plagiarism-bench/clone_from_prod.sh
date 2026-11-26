#!/usr/bin/env bash
# USER-RUN ONLY. Copies a few production tables into a local MariaDB for the
# plagiarism benchmark. Prod is only read: mariadb-dump and SELECT inside the
# prod DB container. Nothing is written to prod.
#
# Overrides (env): CLAOJ_SSH_HOST (root@claoj), CLAOJ_DB_CONTAINER (claoj_db).
# Prod DB user, password and name come from the container's own env
# (MYSQL_USER, MYSQL_PASSWORD, MYSQL_DATABASE), expanded by the container's sh.
set -euo pipefail

HOST="${CLAOJ_SSH_HOST:-root@claoj}"
PROD_CONTAINER="${CLAOJ_DB_CONTAINER:-claoj_db}"

LOCAL_CONTAINER=claoj-bench-db
LOCAL_VOLUME=claoj-bench-db
LOCAL_PORT=3307
LOCAL_DB=claoj_bench

HERE="$(cd "$(dirname "$0")" && pwd)"
DUMPS="$HERE/dumps"
mkdir -p "$DUMPS"

CORE_TABLES="judge_language judge_problem judge_contest judge_contestproblem judge_contestparticipation judge_contestsubmission judge_rating"

# Single quotes keep $MYSQL_* literal here; the container's sh expands them.
MYSQL_CREDS='-u"$MYSQL_USER" -p"$MYSQL_PASSWORD"'
MYSQL_DB='"$MYSQL_DATABASE"'

# Run a command inside the prod DB container over ssh. $1 is the command line
# for the container's sh; printf %q keeps quoting intact across both shells.
prod() {
  local inner="$1"
  ssh "$HOST" "docker exec $PROD_CONTAINER sh -c $(printf '%q' "$inner")"
}

echo "dump core tables"
prod "mariadb-dump $MYSQL_CREDS --no-tablespaces --single-transaction --skip-lock-tables --default-character-set=utf8mb4 $MYSQL_DB $CORE_TABLES" > "$DUMPS/core.sql"

echo "dump contest submissions"
prod "mariadb-dump $MYSQL_CREDS --no-tablespaces --single-transaction --skip-lock-tables --default-character-set=utf8mb4 $MYSQL_DB judge_submission --where=$(printf '%q' 'contest_object_id IS NOT NULL')" > "$DUMPS/submissions.sql"

echo "dump contest submission sources"
prod "mariadb-dump $MYSQL_CREDS --no-tablespaces --single-transaction --skip-lock-tables --default-character-set=utf8mb4 $MYSQL_DB judge_submissionsource --where=$(printf '%q' 'submission_id IN (SELECT id FROM judge_submission WHERE contest_object_id IS NOT NULL)')" > "$DUMPS/sources.sql"

echo "export reduced user tables as TSV"
prod "mariadb $MYSQL_CREDS --default-character-set=utf8mb4 -N -B $MYSQL_DB -e $(printf '%q' 'SELECT id, username FROM auth_user')" > "$DUMPS/auth_user.tsv"
prod "mariadb $MYSQL_CREDS --default-character-set=utf8mb4 -N -B $MYSQL_DB -e $(printf '%q' 'SELECT id, user_id, rating FROM judge_profile')" > "$DUMPS/judge_profile.tsv"
echo "export non-contest submissions (history signals) as TSV"
prod "mariadb $MYSQL_CREDS --default-character-set=utf8mb4 -N -B $MYSQL_DB -e $(printf '%q' 'SELECT id, user_id, problem_id, language_id, date, result, points, status FROM judge_submission WHERE contest_object_id IS NULL')" > "$DUMPS/submissions_noncontest.tsv"

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

ready=0
for _ in $(seq 1 60); do
  if docker exec "$LOCAL_CONTAINER" mariadb -h127.0.0.1 -uroot -e 'SELECT 1' >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done
if [ "$ready" -ne 1 ]; then
  echo "local MariaDB did not accept TCP connections on 127.0.0.1 within 120s" >&2
  exit 1
fi

local_sql() { docker exec -i "$LOCAL_CONTAINER" mariadb -uroot --default-character-set=utf8mb4 "$@"; }

echo "load into $LOCAL_DB (drops and recreates it)"
local_sql -e "DROP DATABASE IF EXISTS $LOCAL_DB; CREATE DATABASE $LOCAL_DB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
local_sql "$LOCAL_DB" < "$DUMPS/core.sql"
local_sql "$LOCAL_DB" < "$DUMPS/submissions.sql"
local_sql "$LOCAL_DB" < "$DUMPS/sources.sql"
local_sql "$LOCAL_DB" -e "
  CREATE TABLE auth_user (id INT PRIMARY KEY, username VARCHAR(150) NOT NULL) CHARACTER SET utf8mb4;
  CREATE TABLE judge_profile (id INT PRIMARY KEY, user_id INT NOT NULL, rating INT NULL) CHARACTER SET utf8mb4;"

# Load the reduced TSVs through a file inside the container (server-side LOAD DATA).
load_tsv() { # $1 table, $2 tsv file name, $3 column list
  docker exec -i "$LOCAL_CONTAINER" sh -c "cat > /tmp/$2" < "$DUMPS/$2"
  local_sql "$LOCAL_DB" -e "LOAD DATA INFILE '/tmp/$2' INTO TABLE $1 CHARACTER SET utf8mb4 FIELDS TERMINATED BY '\t' ($3)"
  docker exec "$LOCAL_CONTAINER" rm -f "/tmp/$2"
}
load_tsv auth_user auth_user.tsv "id, username"
load_tsv judge_profile judge_profile.tsv "id, user_id, rating"

# Non-contest submissions: only the history columns. sql_mode='' lets the other
# NOT NULL columns take their implicit defaults; INSERT IGNORE skips existing ids.
docker exec -i "$LOCAL_CONTAINER" sh -c "cat > /tmp/sub_nc.tsv" < "$DUMPS/submissions_noncontest.tsv"
local_sql "$LOCAL_DB" -e "SET SESSION sql_mode=''; CREATE TEMPORARY TABLE sub_nc (id INT, user_id INT, problem_id INT, language_id INT, date DATETIME(6), result VARCHAR(3), points DOUBLE, status VARCHAR(2));
  LOAD DATA INFILE '/tmp/sub_nc.tsv' INTO TABLE sub_nc CHARACTER SET utf8mb4 FIELDS TERMINATED BY '\t';
  INSERT IGNORE INTO judge_submission (id, user_id, problem_id, language_id, date, result, points, status) SELECT id, user_id, problem_id, language_id, date, result, points, status FROM sub_nc;"
docker exec "$LOCAL_CONTAINER" rm -f /tmp/sub_nc.tsv

echo "apply v2 runtime tables"
local_sql "$LOCAL_DB" < "$HERE/../v2_runtime_tables.sql"

echo "done. Local DB: mysql://root@127.0.0.1:$LOCAL_PORT/$LOCAL_DB"
echo "dumps kept in $DUMPS (gitignored; they contain contestants' code)"
