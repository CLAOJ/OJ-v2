import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import ai_bench  # noqa: E402


def row(pct=None, rewrites=0, above=None, inv=False):
    return {"ai_match": None if pct is None else {"pct": pct}, "rewrites": rewrites,
            "above_history": above, "speed_inversion": inv}


class FpTest(unittest.TestCase):
    def test_fp_rate(self):
        ai = {"rows": [row(pct=80)],
              "problems": [{"problem_code": "a", "contestants": 10, "trivial": False},
                           {"problem_code": "b", "contestants": 10, "trivial": False}]}
        with tempfile.TemporaryDirectory() as d:
            (Path(d) / "7").mkdir()
            (Path(d) / "7" / "ai.json").write_text(json.dumps(ai))
            r = ai_bench.fp_rates(d, ["7"])
        self.assertEqual(r["total"], 20)
        self.assertEqual(r["combined"], 1)
        self.assertEqual(r["s1"], 1)
        self.assertAlmostEqual(r["combined"] / r["total"], 0.05)


class RecallTest(unittest.TestCase):
    def test_share_at_threshold(self):
        res = {"pairs": [{"a": "x", "pct_a": 90, "archive_b": True}, {"a": "y", "pct_a": 50, "archive_b": True}, {"a": "z", "pct_a": 10, "archive_b": True}]}
        self.assertAlmostEqual(ai_bench.recall_share(res, 3), 2 / 3)

    def test_best_pct_per_submission(self):
        res = {"pairs": [{"a": "s1", "pct_a": 20, "archive_b": True}, {"a": "s1", "pct_a": 70, "archive_b": True}, {"a": "s2", "pct_a": 5, "archive_b": True}]}
        self.assertEqual(ai_bench.best_pct(res), {"s1": 70, "s2": 5})

    def test_archive_on_side_a_and_archive_pairs(self):
        res = {"pairs": [{"a": "r1", "b": "s1", "archive_a": True, "archive_b": False, "pct_a": 5, "pct_b": 80},
                         {"a": "r1", "b": "r2", "archive_a": True, "archive_b": True, "pct_a": 99, "pct_b": 99}]}
        self.assertEqual(ai_bench.best_pct(res), {"s1": 80})


if __name__ == "__main__":
    unittest.main()
