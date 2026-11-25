import sqlite3
import sys
import tempfile
import unittest
from pathlib import Path

BENCH = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(BENCH))

import export  # noqa: E402

SCHEMA = """
CREATE TABLE judge_language (id INTEGER PRIMARY KEY, key TEXT, name TEXT, common_name TEXT);
CREATE TABLE judge_problem (id INTEGER PRIMARY KEY, code TEXT, name TEXT);
CREATE TABLE judge_contest (id INTEGER PRIMARY KEY, key TEXT, name TEXT);
CREATE TABLE judge_contestproblem (id INTEGER PRIMARY KEY, contest_id INTEGER, problem_id INTEGER, points REAL, "order" INTEGER);
CREATE TABLE judge_contestparticipation (id INTEGER PRIMARY KEY, contest_id INTEGER, user_id INTEGER, virtual INTEGER);
CREATE TABLE judge_contestsubmission (id INTEGER PRIMARY KEY, submission_id INTEGER, problem_id INTEGER, participation_id INTEGER);
CREATE TABLE judge_profile (id INTEGER PRIMARY KEY, user_id INTEGER);
CREATE TABLE auth_user (id INTEGER PRIMARY KEY, username TEXT);
CREATE TABLE judge_submission (id INTEGER PRIMARY KEY, user_id INTEGER, problem_id INTEGER, language_id INTEGER, contest_object_id INTEGER, points REAL);
CREATE TABLE judge_submissionsource (id INTEGER PRIMARY KEY, submission_id INTEGER, source TEXT);
"""


def make_db():
    conn = sqlite3.connect(":memory:")
    conn.executescript(SCHEMA)
    conn.executemany("INSERT INTO judge_language VALUES (?,?,?,?)", [
        (1, "CPP17", "C++17", "C++"), (2, "PY3", "Python 3", "Python")])
    conn.executemany("INSERT INTO judge_problem VALUES (?,?,?)", [
        (1, "knap", "Knapsack"), (2, "sum", "Sum"), (3, "hidden", "Not In Contest")])
    conn.executemany("INSERT INTO judge_contest VALUES (?,?,?)", [(1, "tik25", "TiK25"), (2, "other", "Other")])
    conn.executemany("INSERT INTO judge_contestproblem VALUES (?,?,?,?,?)", [(1, 1, 1, 100, 1), (2, 1, 2, 100, 2)])
    # profile id == auth_user id for simplicity
    conn.executemany("INSERT INTO auth_user VALUES (?,?)", [
        (1, "alice"), (2, "bob"), (3, "carol"), (4, "dave"), (5, "con"), (6, "erin")])
    conn.executemany("INSERT INTO judge_profile VALUES (?,?)", [(i, i) for i in range(1, 7)])
    # participation: id, contest_id, user_id (profile), virtual
    conn.executemany("INSERT INTO judge_contestparticipation VALUES (?,?,?,?)", [
        (1, 1, 1, 0),   # alice LIVE
        (2, 1, 2, -1),  # bob SPECTATE
        (3, 1, 3, 1),   # carol virtual -> excluded
        (4, 1, 4, 0),   # dave LIVE
        (5, 1, 5, 0),   # con LIVE
        (6, 1, 6, 0),   # erin LIVE
        (7, 2, 1, 0),   # alice in the other contest
    ])
    # submission: id, user_id (profile), problem_id, language_id, contest_object_id, points
    subs = [
        (1, 1, 1, 1, 1, 100),   # alice knap cpp best
        (2, 1, 1, 1, 1, 40),    # alice knap cpp worse
        (3, 2, 1, 1, 1, 70),    # bob knap cpp (spectate)
        (4, 3, 1, 1, 1, 100),   # carol knap cpp (virtual)
        (5, 4, 1, 1, 1, 50),    # dave knap cpp
        (6, 1, 1, 2, 1, 10),    # alice knap python
        (7, 6, 1, 1, 1, 30),    # erin knap cpp tie, lower id
        (8, 6, 1, 1, 1, 30),    # erin knap cpp tie, higher id
        (9, 1, 2, 1, 1, 5),     # alice sum cpp
        (10, 1, 1, 1, 2, 999),  # alice knap cpp in the other contest
        (11, 5, 1, 1, 1, 20),   # con knap cpp (reserved name)
        (12, 1, 3, 1, 1, 60),   # alice, problem 3 not in contest 1
    ]
    conn.executemany("INSERT INTO judge_submission VALUES (?,?,?,?,?,?)", subs)
    # contestsubmission: submission id, problem_id (unused), participation id
    links = [(1, 1, 1), (2, 1, 1), (3, 1, 2), (4, 1, 3), (5, 1, 4), (6, 1, 1),
             (7, 1, 6), (8, 1, 6), (9, 2, 1), (10, 1, 7), (11, 1, 5), (12, 3, 1)]
    conn.executemany("INSERT INTO judge_contestsubmission VALUES (?,?,?,?)",
                     [(sid, sid, pid, part) for sid, pid, part in links])
    sources = {1: "best alice", 2: "worse alice", 3: "bob code", 4: "virtual carol",
               5: "dave code", 6: "print(1)", 7: "erin first", 8: "erin second",
               9: "sum alice", 10: "other contest", 11: "con code", 12: "hidden problem"}
    conn.executemany("INSERT INTO judge_submissionsource VALUES (?,?,?)",
                     [(sid, sid, src) for sid, src in sources.items()])
    return conn


def read_tree(root):
    root = Path(root)
    return {p.relative_to(root).as_posix(): p.read_text(encoding="utf-8")
            for p in root.rglob("*") if p.is_file()}


class SelectionAndLayoutTest(unittest.TestCase):
    def test_export_layout_and_selection(self):
        conn = make_db()
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp) / "tik25"
            n = export.export_contest(conn, "tik25", out, ph="?")
            self.assertEqual(n, 7)
            self.assertEqual(read_tree(out), {
                "knap/cpp/alice/main.cpp": "best alice",
                "knap/cpp/bob/main.cpp": "bob code",
                "knap/cpp/dave/main.cpp": "dave code",
                "knap/cpp/erin/main.cpp": "erin first",
                "knap/cpp/~con~/main.cpp": "con code",
                "knap/python/alice/main.py": "print(1)",
                "sum/cpp/alice/main.cpp": "sum alice",
            })

    def test_select_rows_best_per_user_and_virtual_excluded(self):
        rows = export.select_rows(make_db(), "tik25", ph="?")
        got = {(code, lang, user): src for code, lang, user, src in rows}
        self.assertEqual(got[("knap", "C++", "alice")], "best alice")
        self.assertNotIn(("knap", "C++", "carol"), got)
        self.assertEqual(len(rows), 7)
        self.assertNotIn("hidden", [code for code, _, _, _ in rows])  # problem not in contest

    def test_unknown_contest_is_empty(self):
        self.assertEqual(export.select_rows(make_db(), "nope", ph="?"), [])


class ExtAndSlugAndFolderTest(unittest.TestCase):
    def test_language_ext_and_slug(self):
        exts = [("C", "c"), ("C++", "cpp"), ("Python", "py"), ("Pascal", "pas"), ("Java", "java"),
                ("Golang", "go"), ("Rust", "rs"), ("Perl", "pl"), ("Ruby", "rb"), ("Kotlin", "kt"),
                ("C#", "cs"), ("PHP", "php"), ("JavaScript", "js"), ("Haskell", "hs"), ("Scala", "scala"),
                ("Swift", "swift"), ("Lua", "lua"), ("D", "d"), ("Brainfuck", "txt")]
        for name, want in exts:
            self.assertEqual(export.language_ext(name), want, name)
        slugs = [("C++", "cpp"), ("C#", "csharp"), ("Python", "python"), ("Golang", "golang"),
                 ("Visual Basic .NET", "visual_basic__net")]
        for name, want in slugs:
            self.assertEqual(export.language_slug(name), want, name)

    def test_submission_folder(self):
        cases = [("alice", "alice"), ("tik25_lêthanh", "tik25_lêthanh"), ("a.b", "a.b"),
                 ("bob@x+y-z", "bob@x+y-z"), ("console", "console"), ("com10", "com10"),
                 ("aux2", "aux2"), (".", "~.~"), ("..", "~..~"), (".hidden", "~.hidden~"),
                 ("a.", "~a.~"), ("con", "~con~"), ("CON", "~CON~"), ("nul.txt", "~nul.txt~"),
                 ("com1", "~com1~"), ("lpt9.x", "~lpt9.x~")]
        for name, want in cases:
            self.assertEqual(export.submission_folder(name), want, name)


class DsnTest(unittest.TestCase):
    def test_default_dsn(self):
        kw = export.dsn_to_kwargs("root@127.0.0.1:3307/claoj_bench")
        self.assertEqual(kw["host"], "127.0.0.1")
        self.assertEqual(kw["port"], 3307)
        self.assertEqual(kw["user"], "root")
        self.assertEqual(kw["database"], "claoj_bench")


if __name__ == "__main__":
    unittest.main()
