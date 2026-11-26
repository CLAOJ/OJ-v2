"""AI detection benchmark: false-positive rate on human contests, recall on fresh LLM solutions.

  ai_bench.py fp --out <report dir> --contests 12,13      (contest ids = report subdirs)
  ai_bench.py recall --dsn <dsn> --contest <key> --per-problem 3 --python <compare50 python>

recall calls the LLM (COMPARE50_LLM_URL/_KEY/_MODEL, same as the Go job). Stdlib only
except disguise (pygments) and pymysql, both imported lazily.
"""

import argparse
import json
import os
import random
import re
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

AI_MATCH_PCT = 50  # keep in sync with aiMatchPct in claoj/jobs/ai_detect.go
REWRITES_TO_FLAG = 2  # rewritesToFlag
SEED = 1234
SYSTEM_PROMPT = ("You solve competitive programming problems. Reply with one complete program that reads "
                 "standard input and writes standard output. Reply with code only, no explanation.")


def fp_rates(out_dir, contests):
    """Flagged rows per signal and combined, over contestant x problem rows."""
    t = {"total": 0, "combined": 0, "s1": 0, "s3": 0, "s4": 0}
    for c in contests:
        ai = json.loads((Path(out_dir) / str(c) / "ai.json").read_text(encoding="utf-8"))
        trivial = {(p["problem_code"], p["language"]) for p in ai["problems"] if p.get("trivial")}
        t["total"] += sum(p["contestants"] for p in ai["problems"])
        for r in ai["rows"]:  # ai.json lists only rows that were flagged
            t["combined"] += 1
            m = r.get("ai_match")
            t["s1"] += bool(m and m["pct"] >= AI_MATCH_PCT
                            and (r.get("problem_code"), r.get("language")) not in trivial)
            t["s3"] += r.get("rewrites", 0) >= REWRITES_TO_FLAG
            t["s4"] += bool(r.get("above_history")) or bool(r.get("speed_inversion"))
    return t


def best_pct(results):
    """Highest pct_a per submission (a = submission side, b = archive)."""
    best = {}
    for p in results.get("pairs", []):
        best[p["a"]] = max(best.get(p["a"], 0), p["pct_a"])
    return best


def recall_share(results, n_subs):
    return sum(v >= AI_MATCH_PCT for v in best_pct(results).values()) / n_subs if n_subs else 0.0


def cmd_fp(args):
    t = fp_rates(args.out, args.contests.split(","))
    for k, name in (("s1", "S1 AI match"), ("s3", "S3 rewrites"), ("s4", "S4 history/speed"),
                    ("combined", "combined")):
        print(f"{name}: {t[k]}/{t['total']} = {100 * t[k] / max(t['total'], 1):.2f}%")
    return 0


def llm_generate(statement, language, limits):
    url, key, model = (os.environ.get(k) for k in ("COMPARE50_LLM_URL", "COMPARE50_LLM_KEY", "COMPARE50_LLM_MODEL"))
    if not (url and key and model):
        sys.exit("set COMPARE50_LLM_URL, COMPARE50_LLM_KEY and COMPARE50_LLM_MODEL")
    body = json.dumps({"model": model, "temperature": 0.8, "messages": [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": f"Language: {language}\n{limits}\n\n{statement}"}]}).encode()
    req = urllib.request.Request(url.rstrip("/") + "/chat/completions", body, {
        "Content-Type": "application/json", "Authorization": "Bearer " + key})
    for wait in (5, 15, 30, None):
        try:
            with urllib.request.urlopen(req, timeout=90) as r:
                text = json.load(r)["choices"][0]["message"]["content"]
            break
        except urllib.error.HTTPError as e:
            if wait is None or (e.code != 429 and e.code < 500):
                raise
            time.sleep(wait)
    m = re.search(r"```[^\n]*\n(.*?)```", text, re.S)
    return (m.group(1) if m else text).strip() + "\n"


def cmd_recall(args):
    import pymysql
    import disguise
    from export import dsn_to_kwargs, language_ext, language_slug

    conn = pymysql.connect(**dsn_to_kwargs(args.dsn))
    cur = conn.cursor()
    cur.execute("SELECT p.id, p.code, p.description, p.time_limit, p.memory_limit, l.common_name "
                "FROM judge_contest c JOIN judge_contestproblem cp ON cp.contest_id = c.id "
                "JOIN judge_problem p ON p.id = cp.problem_id "
                "JOIN judge_submission s ON s.problem_id = p.id AND s.contest_object_id = c.id "
                "JOIN judge_language l ON l.id = s.language_id WHERE c.`key` = %s "
                "GROUP BY p.id, p.code, p.description, p.time_limit, p.memory_limit, l.common_name",
                (args.contest,))
    combos = cur.fetchall()
    rng = random.Random(SEED)
    shares = {"undisguised": [0, 0], "disguised": [0, 0]}  # [hits, total]
    with tempfile.TemporaryDirectory() as tmp:
        for pid, code, desc, tl, ml, lang in combos:
            cur.execute("SELECT source FROM plagiarism_ai_reference WHERE problem_id = %s AND language = %s",
                        (pid, lang))
            refs = [r[0] for r in cur.fetchall()]
            if not refs:
                continue
            ext, slug = language_ext(lang), language_slug(lang)
            base = Path(tmp, code, slug)
            arch = base / "arch"
            for i, src in enumerate(refs):
                (arch / f"r{i}").mkdir(parents=True, exist_ok=True)
                (arch / f"r{i}" / f"main.{ext}").write_text(src, encoding="utf-8")
            # ponytail: disguise only handles cpp/py/pas; other languages are copied as-is
            dlang = {"py": "py", "pas": "pas", "cpp": "cpp", "c": "cpp"}.get(ext)
            for i in range(args.per_problem):
                src = llm_generate(desc, lang, f"Time limit: {tl} seconds\nMemory limit: {ml} KB")
                dis = disguise.reformat(disguise.rename_identifiers(src, dlang, rng), dlang, rng) if dlang else src
                for kind, text in (("undisguised", src), ("disguised", dis)):
                    d = base / kind / f"g{i}"
                    d.mkdir(parents=True, exist_ok=True)
                    (d / f"main.{ext}").write_text(text, encoding="utf-8")
            for kind in shares:
                subs = sorted(str(p) for p in (base / kind).iterdir())
                out = base / ("out_" + kind)
                subprocess.run([args.python, "-m", "compare50", *subs, "-a", str(arch), "--archive-only",
                                "-n", "999", "-o", str(out)], check=True)
                rf = out / "results.json"
                res = json.loads(rf.read_text()) if rf.exists() else {"pairs": []}
                shares[kind][0] += sum(v >= AI_MATCH_PCT for v in best_pct(res).values())
                shares[kind][1] += len(subs)
    for kind, (h, n) in shares.items():
        print(f"{kind}: {h}/{n} = {100 * h / max(n, 1):.1f}% with pct >= {AI_MATCH_PCT}")
    return 0


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    f = sub.add_parser("fp")
    f.add_argument("--out", required=True)
    f.add_argument("--contests", required=True, help="comma-separated contest ids (report subdirs)")
    r = sub.add_parser("recall")
    r.add_argument("--dsn", default=os.environ.get("BENCH_DB_DSN", "root@127.0.0.1:3307/claoj_bench"))
    r.add_argument("--contest", required=True)
    r.add_argument("--per-problem", type=int, default=3)
    r.add_argument("--python", required=True, help="compare50 venv python")
    args = ap.parse_args(argv)
    return {"fp": cmd_fp, "recall": cmd_recall}[args.cmd](args)


if __name__ == "__main__":
    raise SystemExit(main())
