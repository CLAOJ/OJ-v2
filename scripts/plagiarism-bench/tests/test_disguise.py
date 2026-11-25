import ast
import json
import random
import re
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

BENCH = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(BENCH))

import disguise  # noqa: E402

CPP_SRC = r'''#include <cstdio>
using namespace std;

int total(int n) {
    int acc = 0;
    for (int i = 0; i < n; i++) {
        acc += i;
    }
    return acc;
}

int main() {
    int n;
    long long big = 5;
    scanf("%d", &n);
    printf("%d\n", total(n) + (int)big);
    return 0;
}
'''

CPP_REORDER = r'''#include <cstdio>
int twice(int x) { return 2 * x; }
int add(int a, int b) { return a + b; }
int main() { printf("%d\n", add(twice(1), 3)); return 0; }
'''

CPP_LOOP = r'''#include <cstdio>
int main() {
    int n = 4, s = 0;
    for (int i = 0; i < n; ++i) {
        s += i;
    }
    printf("%d\n", s);
    return 0;
}
'''

CPP_CONTINUE = r'''#include <cstdio>
int main() {
    int n = 4, s = 0;
    for (int i = 0; i < n; ++i) {
        if (i == 2) continue;
        s += i;
    }
    printf("%d\n", s);
    return 0;
}
'''

PY_SRC = '''import math

def total(n):
    acc = 0
    for i in range(n):
        acc += i
    return acc

def main():
    n = int(input())
    print(total(n), math.sqrt(4))

if __name__ == "__main__":
    main()
'''

PY_REORDER = '''import math

def square(x):
    return x * x

def cube(x):
    return x * x * x

def main():
    print(square(3), cube(2), math.sqrt(4))

if __name__ == "__main__":
    main()
'''

PAS_SRC = '''program Demo;
var
  n: integer;
procedure Show(a: integer);
begin
  writeln(a)
end;
function Twice(a: integer): integer;
begin
  Twice := a * 2
end;
begin
  readln(n);
  Show(Twice(n))
end.
'''


def cpp_ok(text):
    """True/False from g++ -fsyntax-only, None when g++ is not installed."""
    if not shutil.which("g++"):
        return None
    with tempfile.TemporaryDirectory() as d:
        p = Path(d, "main.cpp")
        p.write_text(text, encoding="utf-8")
        r = subprocess.run(["g++", "-fsyntax-only", str(p)], capture_output=True, text=True)
    return r.returncode == 0


def py_ok(text):
    try:
        compile(text, "main.py", "exec")
    except SyntaxError:
        return False
    return True


def pas_ok(text):
    """True/False from fpc, None when fpc is not installed."""
    if not shutil.which("fpc"):
        return None
    with tempfile.TemporaryDirectory() as d:
        p = Path(d, "main.pas")
        p.write_text(text, encoding="utf-8")
        r = subprocess.run(["fpc", f"-FE{d}", f"-FU{d}", str(p)], capture_output=True, text=True)
    return r.returncode == 0


VALID = {"cpp": cpp_ok, "py": py_ok, "pas": pas_ok}


class Base(unittest.TestCase):
    def assert_valid(self, text, lang):
        ok = VALID[lang](text)
        if ok is not None:  # None = checker not installed, so only the diff assertions ran
            self.assertTrue(ok, text)

    def check_reorders(self, text, lang):
        outs = [disguise.reorder_functions(text, lang, random.Random(s)) for s in range(5)]
        self.assertTrue(any(o != text for o in outs), "no seed reordered anything")
        for o in outs:
            self.assert_valid(o, lang)


class RenameIdentifiers(Base):
    def test_cpp_keeps_std_and_main(self):
        out = disguise.rename_identifiers(CPP_SRC, "cpp", random.Random(1))
        self.assertNotEqual(out, CPP_SRC)
        for kept in ("int main", "printf", "scanf", "using namespace std"):
            self.assertIn(kept, out)
        self.assertNotIn("total", out)
        self.assertNotIn("acc", out)
        self.assert_valid(out, "cpp")

    def test_py_keeps_builtins_and_attributes(self):
        out = disguise.rename_identifiers(PY_SRC, "py", random.Random(1))
        self.assertNotEqual(out, PY_SRC)
        for kept in ("def main", "range(", "print(", "input(", "math.sqrt", "import math", "__name__"):
            self.assertIn(kept, out)
        self.assertNotIn("total", out)
        self.assertNotIn("acc", out)
        self.assert_valid(out, "py")

    def test_pas_keeps_standard_routines(self):
        out = disguise.rename_identifiers(PAS_SRC, "pas", random.Random(1))
        self.assertNotEqual(out, PAS_SRC)
        for kept in ("writeln(", "readln(", "integer"):
            self.assertIn(kept, out)
        self.assertNotIn("show", out.lower())
        self.assertNotIn("twice", out.lower())
        self.assert_valid(out, "pas")


class Reformat(Base):
    def test_cpp(self):
        out = disguise.reformat(CPP_SRC, "cpp", random.Random(1))
        self.assertNotEqual(out, CPP_SRC)
        self.assert_valid(out, "cpp")

    def test_py_same_ast(self):
        out = disguise.reformat(PY_SRC, "py", random.Random(1))
        self.assertNotEqual(out, PY_SRC)
        self.assertEqual(ast.dump(ast.parse(out)), ast.dump(ast.parse(PY_SRC)))

    def test_pas_same_tokens_ignoring_case_and_space(self):
        out = disguise.reformat(PAS_SRC, "pas", random.Random(1))
        self.assertNotEqual(out, PAS_SRC)

        def squash(s):
            return re.sub(r"\s+", "", s).lower()

        self.assertEqual(squash(out), squash(PAS_SRC))


class ReorderFunctions(Base):
    def test_cpp(self):
        self.check_reorders(CPP_REORDER, "cpp")

    def test_py(self):
        self.check_reorders(PY_REORDER, "py")

    def test_pas(self):
        self.check_reorders(PAS_SRC, "pas")


class SwapIntTypes(Base):
    def test_cpp(self):
        out = disguise.swap_int_types(CPP_SRC, "cpp", random.Random(1))
        self.assertNotEqual(out, CPP_SRC)
        self.assertIn("long long n;", out)
        self.assertRegex(out, r"\bint\s+big")  # long long -> int leaves a double space
        self.assertIn("int main", out)
        self.assert_valid(out, "cpp")

    def test_pas(self):
        out = disguise.swap_int_types(PAS_SRC, "pas", random.Random(1))
        self.assertNotEqual(out, PAS_SRC)
        self.assertIn("longint", out.lower())
        self.assert_valid(out, "pas")

    def test_py_has_no_int_types(self):
        self.assertEqual(disguise.swap_int_types(PY_SRC, "py", random.Random(1)), PY_SRC)


class InsertDeadCode(Base):
    def test_cpp(self):
        out = disguise.insert_dead_code(CPP_SRC, "cpp", random.Random(1))
        self.assertNotEqual(out, CPP_SRC)
        self.assertIn("if (0)", out)
        self.assert_valid(out, "cpp")

    def test_py(self):
        out = disguise.insert_dead_code(PY_SRC, "py", random.Random(1))
        self.assertNotEqual(out, PY_SRC)
        self.assertIn("return 0", out)
        self.assert_valid(out, "py")

    def test_pas(self):
        out = disguise.insert_dead_code(PAS_SRC, "pas", random.Random(1))
        self.assertNotEqual(out, PAS_SRC)
        self.assertIn("if false then", out.lower())
        self.assert_valid(out, "pas")


class ForToWhile(Base):
    def test_cpp_counted_loop(self):
        out = disguise.for_to_while(CPP_LOOP, "cpp", random.Random(1))
        self.assertNotEqual(out, CPP_LOOP)
        self.assertIn("while (i < n)", out)
        self.assertNotIn("for (", out)
        self.assert_valid(out, "cpp")

    def test_cpp_continue_loop_left_alone(self):
        self.assertEqual(disguise.for_to_while(CPP_CONTINUE, "cpp", random.Random(1)), CPP_CONTINUE)

    def test_other_languages_untouched(self):
        self.assertEqual(disguise.for_to_while(PY_SRC, "py", random.Random(1)), PY_SRC)
        self.assertEqual(disguise.for_to_while(PAS_SRC, "pas", random.Random(1)), PAS_SRC)


class Pipeline(Base):
    def test_each_language(self):
        for lang, src in (("cpp", CPP_SRC), ("py", PY_SRC), ("pas", PAS_SRC)):
            with self.subTest(lang=lang):
                out, applied = disguise.disguise(src, lang, random.Random(1))
                self.assertNotEqual(out, src)
                self.assertTrue(applied)
                self.assert_valid(out, lang)

    def test_deterministic_for_seed(self):
        a = disguise.disguise(CPP_SRC, "cpp", random.Random(7))
        b = disguise.disguise(CPP_SRC, "cpp", random.Random(7))
        self.assertEqual(a, b)


def snapshot(root):
    return {p.relative_to(root).as_posix(): p.read_bytes() for p in root.rglob("*") if p.is_file()}


class DisguiseTree(unittest.TestCase):
    def make_src(self, root):
        src = root / "data" / "tik25"
        files = {
            "p1/cpp/alice/main.cpp": CPP_SRC,
            "p1/cpp/bob/main.cpp": CPP_LOOP,
            "p1/python/alice/main.py": PY_SRC,
            "p2/pascal/carol/main.pas": PAS_SRC,
            "p2/c/dave/main.c": "int main(void) { return 0; }\n",
        }
        for rel, text in files.items():
            f = src / rel
            f.parent.mkdir(parents=True, exist_ok=True)
            f.write_text(text, encoding="utf-8")
        return src

    def test_tree_layout_manifest_and_src_untouched(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            src = self.make_src(root)
            before = snapshot(src)
            out = root / "out"
            manifest = disguise.disguise_tree(src, out, per_problem=1, seed=1)

            self.assertEqual(snapshot(src), before)
            # one copy for p1/cpp, p1/python, p2/pascal; the C file gets no copy
            self.assertEqual(len(manifest), 3)
            for e in manifest:
                self.assertEqual(set(e), {"problem", "lang", "original", "copy", "transforms"})
                self.assertTrue((out / e["original"]).is_file())
                self.assertTrue((out / e["copy"]).is_file())
                self.assertIn("__copy/main.", e["copy"])
                self.assertTrue(e["transforms"])
            self.assertTrue((out / "p2" / "c" / "dave" / "main.c").is_file())
            self.assertEqual(json.loads((out / "manifest.json").read_text(encoding="utf-8")), manifest)

    def test_tree_is_deterministic(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            src = self.make_src(root)
            m1 = disguise.disguise_tree(src, root / "a", per_problem=1, seed=3)
            m2 = disguise.disguise_tree(src, root / "b", per_problem=1, seed=3)
            self.assertEqual(m1, m2)
            self.assertEqual(snapshot(root / "a"), snapshot(root / "b"))


class Cli(unittest.TestCase):
    def test_rejects_out_equal_to_src(self):
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(SystemExit):
                disguise.main(["--src", tmp, "--out", tmp])


if __name__ == "__main__":
    unittest.main()
