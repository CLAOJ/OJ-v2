"""Export one contest's best submissions into the compare50 folder layout.

Layout: <out>/<problem_code>/<lang_slug>/<folder>/main.<ext>
Mirrors claoj/jobs/compare50.go (LanguageExt, LanguageSlug, SubmissionFolder)
and the selection in claoj/jobs/moss.go. Stdlib only at import time, so the
tests do not need pymysql.
"""

import argparse
import os
import re
from pathlib import Path
from urllib.parse import unquote, urlsplit

DEFAULT_DSN = "root@127.0.0.1:3307/claoj_bench"

LANGUAGE_EXTS = {
    "C": "c", "C++": "cpp", "Python": "py", "Pascal": "pas", "Java": "java",
    "Golang": "go", "Rust": "rs", "Perl": "pl", "Ruby": "rb", "Kotlin": "kt",
    "C#": "cs", "PHP": "php", "JavaScript": "js", "Haskell": "hs", "Scala": "scala",
    "Swift": "swift", "Lua": "lua", "D": "d",
}

_RESERVED_STEM = re.compile(r"(con|prn|aux|nul|com[1-9]|lpt[1-9])", re.IGNORECASE)

# Same rules as the Go query: contest submissions, LIVE (0) or SPECTATE (-1)
# participations only. Best points first; id breaks ties (Go has no tiebreak).
# Identifiers are backtick-quoted so the same SQL runs on MariaDB and SQLite.
SELECT_SQL = """
SELECT p.`code`, l.`common_name`, u.`username`, src.`source`
FROM judge_submission s
JOIN judge_submissionsource src ON src.submission_id = s.id
JOIN judge_language l ON l.id = s.language_id
JOIN judge_problem p ON p.id = s.problem_id
JOIN judge_profile pr ON pr.id = s.user_id
JOIN auth_user u ON u.id = pr.user_id
JOIN judge_contestsubmission cs ON cs.submission_id = s.id
JOIN judge_contestparticipation part ON part.id = cs.participation_id
JOIN judge_contest c ON c.id = s.contest_object_id
JOIN judge_contestproblem cp ON cp.contest_id = c.id AND cp.problem_id = s.problem_id
WHERE c.`key` = {ph} AND part.virtual IN (0, -1)
ORDER BY s.points DESC, s.id ASC
"""


def language_ext(common_name):
    return LANGUAGE_EXTS.get(common_name, "txt")


def language_slug(common_name):
    out = []
    for ch in common_name.lower():
        if ch == "+":
            out.append("p")
        elif ch == "#":
            out.append("sharp")
        elif ("a" <= ch <= "z") or ("0" <= ch <= "9"):
            out.append(ch)
        else:
            out.append("_")
    return "".join(out)


def submission_folder(username):
    stem = username.split(".", 1)[0]
    if username in (".", "..") or username.startswith(".") or username.endswith(".") \
            or _RESERVED_STEM.fullmatch(stem):
        return "~" + username + "~"
    return username


def select_rows(conn, contest_key, ph="%s"):
    """Best submission per (problem, language, user): [(code, language, username, source)]."""
    cur = conn.cursor()
    cur.execute(SELECT_SQL.format(ph=ph), (contest_key,))
    seen = set()
    out = []
    for code, lang, user, source in cur.fetchall():
        key = (code, lang, user)
        if key in seen:  # rows come best-first, so the first one wins
            continue
        seen.add(key)
        out.append((code, lang, user, source))
    return out


def export_contest(conn, contest_key, out_dir, ph="%s"):
    count = 0
    for code, lang, user, source in select_rows(conn, contest_key, ph=ph):
        d = Path(out_dir) / code / language_slug(lang) / submission_folder(user)
        d.mkdir(parents=True, exist_ok=True)
        (d / ("main." + language_ext(lang))).write_bytes(source.encode("utf-8"))
        count += 1
    return count


def dsn_to_kwargs(dsn):
    """Parse [user[:password]@]host[:port][/database] (a mysql:// prefix is optional)."""
    parts = urlsplit(dsn if "://" in dsn else "mysql://" + dsn)
    return {
        "host": parts.hostname or "127.0.0.1",
        "port": parts.port or 3306,
        "user": unquote(parts.username or "root"),
        "password": unquote(parts.password or ""),
        "database": parts.path.lstrip("/"),
        "charset": "utf8mb4",
    }


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--contest", required=True, help="contest key, e.g. tik25")
    ap.add_argument("--out", required=True, help="output dir, e.g. data/tik25")
    args = ap.parse_args(argv)

    import pymysql  # lazy: only the CLI needs it

    dsn = os.environ.get("BENCH_DB_DSN", DEFAULT_DSN)
    conn = pymysql.connect(**dsn_to_kwargs(dsn))
    try:
        n = export_contest(conn, args.contest, args.out, ph="%s")
    finally:
        conn.close()
    print(f"wrote {n} submission files to {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
