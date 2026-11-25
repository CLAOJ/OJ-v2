#!/usr/bin/env python3
"""Disguised-copy generator: plants known copies for the compare50 bench.

Usage: disguise.py --src data/<key> --out data/<key>-disguised [--per-problem 3] [--seed 1]

Reads <src>/<problem>/<lang>/<user>/main.<ext>, mirrors every submission into --out,
and adds <user>__copy/main.<ext> beside up to --per-problem originals per
problem x language. manifest.json at the top of --out lists the copies.
Tokenizing is done with Pygments; --src is never written.
"""

import argparse
import builtins
import json
import keyword
import random
import re
import shutil
import string
from pathlib import Path

from pygments.lexers import CppLexer, DelphiLexer, PythonLexer
from pygments.token import Token

LEXERS = {"cpp": CppLexer, "py": PythonLexer, "pas": DelphiLexer}

# Names that keep their spelling: std/library names, entry points, Python builtins.
_CPP_KEEP = {
    "std", "main", "cin", "cout", "cerr", "clog", "endl", "printf", "scanf", "puts", "gets",
    "getchar", "putchar", "string", "vector", "map", "set", "pair", "make_pair", "queue",
    "stack", "deque", "list", "priority_queue", "unordered_map", "unordered_set", "tuple",
    "sort", "reverse", "unique", "lower_bound", "upper_bound", "min", "max", "swap", "abs",
    "sqrt", "pow", "log", "exp", "ceil", "floor", "round", "fabs", "gcd", "lcm", "memset",
    "memcpy", "strlen", "strcpy", "strcmp", "malloc", "calloc", "free", "atoi", "atol",
    "atoll", "atof", "to_string", "stoi", "stol", "stoll", "getline", "fopen", "fclose",
    "fgets", "fprintf", "sprintf", "sscanf", "exit", "rand", "srand", "time", "assert", "NULL",
    "EOF", "FILE", "stdin", "stdout", "stderr", "freopen", "INT_MAX", "INT_MIN", "LLONG_MAX",
    "LLONG_MIN", "UINT_MAX", "RAND_MAX", "EXIT_SUCCESS", "EXIT_FAILURE", "bool", "size_t",
    "int8_t", "int16_t", "int32_t", "int64_t", "uint8_t", "uint16_t", "uint32_t", "uint64_t",
    "setprecision", "fixed", "iterator", "begin", "end", "numeric_limits", "ios", "ios_base",
    "sync_with_stdio", "isdigit", "isalpha", "isalnum", "isspace", "toupper", "tolower",
    "hash", "function", "count", "find", "accumulate",
}
_PY_KEEP = set(dir(builtins)) | set(keyword.kwlist) | {
    "main", "match", "case", "type", "_", "key", "reverse", "sep", "end", "file", "flush", "base",
    "default", "start", "stop", "step",
}
# Pascal is case-insensitive, so these are lowercase.
_PAS_KEEP = {
    "readln", "writeln", "read", "write", "length", "copy", "pos", "inc", "dec", "chr", "ord",
    "abs", "sqr", "sqrt", "round", "trunc", "int", "frac", "odd", "succ", "pred", "high", "low",
    "new", "dispose", "assign", "reset", "rewrite", "close", "eof", "eoln", "halt", "exit",
    "break", "continue", "sin", "cos", "ln", "exp", "arctan", "random", "randomize", "inttostr",
    "strtoint", "str", "val", "delete", "insert", "setlength", "concat", "upcase", "pi", "true",
    "false", "nil", "self", "result", "input", "output", "sizeof", "assigned", "format",
    "integer", "longint", "int64", "real", "double", "single", "string", "char", "boolean",
    "byte", "word", "cardinal", "pchar", "text", "file", "array", "program", "uses", "crt",
    "clrscr", "readkey", "main",
}
_KEEP = {"cpp": _CPP_KEEP, "py": _PY_KEEP, "pas": _PAS_KEEP}
# A name on a line with one of these keywords (import/uses/using) keeps its spelling.
_LINE_KW = {"import", "from", "uses", "using"}
# Never pick these for fresh names.
_BANNED_FRESH = set(keyword.kwlist) | {
    "auto", "case", "char", "const", "continue", "default", "delete", "double", "explicit",
    "export", "extern", "float", "friend", "goto", "inline", "long", "mutable", "namespace",
    "operator", "private", "protected", "public", "register", "return", "short", "signed",
    "sizeof", "static", "struct", "switch", "template", "this", "throw", "typedef", "typename",
    "union", "unsigned", "virtual", "volatile", "while", "class", "begin", "end", "array",
    "record", "downto", "nil", "procedure", "function", "program", "string", "integer",
    "boolean", "repeat", "until", "forward", "external", "unit", "uses",
}
_PAS_ROUTINE = {"procedure", "function", "constructor", "destructor"}
_PAS_OPEN = {"begin", "case", "try", "record", "asm"}
_PAS_ENTRY = {None, ";", "interface", "implementation"}  # tokens that may precede a routine
_IDENT = re.compile(r"[A-Za-z_]\w*")


def _normalize(text):
    return text.replace("\r\n", "\n").replace("\r", "\n")


def _lex(lang, text):
    lexer = LEXERS[lang](stripnl=False, ensurenl=False, stripall=False)
    return list(lexer.get_tokens(_normalize(text)))


def _is_ws(t, v):
    return t in Token.Text and not v.strip()


def _join(toks):
    return "".join(v for _, v in toks)


def _sig(toks):
    return [(t, v) for t, v in toks if not _is_ws(t, v)]


def _fresh_name(rng, taken):
    while True:
        name = "".join(rng.choice(string.ascii_lowercase) for _ in range(8))
        if name not in taken and name not in _BANNED_FRESH:
            return name


def _insert_after(toks, j, s):
    return _join(toks[: j + 1]) + s + _join(toks[j + 1:])


# --- identifier renaming -------------------------------------------------------

def rename_identifiers(text, lang, rng):
    toks = _lex(lang, text)
    fold = lang == "pas"

    def key(s):
        return s.lower() if fold else s

    keep = {key(k) for k in _KEEP[lang]}
    line, skip_lines, rows = 1, set(), []
    for t, v in toks:
        rows.append((t, v, line))
        if t in Token.Keyword and key(v) in _LINE_KW:
            skip_lines.add(line)
        line += v.count("\n")
    sig = [r for r in rows if not _is_ws(r[0], r[1])]

    def is_name(t):
        return t in Token.Name and t not in Token.Name.Builtin

    # A name after '.', '::' or '->' is an attribute or member we cannot resolve, so it
    # keeps its spelling everywhere (the same name is then never renamed elsewhere either).
    blocked = set(keep)
    names = set()
    for n, (t, v, ln) in enumerate(sig):
        if not is_name(t):
            continue
        names.add(key(v))
        prev1 = sig[n - 1][1] if n >= 1 else ""
        prev2 = sig[n - 2][1] if n >= 2 else ""
        if prev1 == "." or (prev1 == ":" and prev2 == ":") or (prev1 == ">" and prev2 == "-") \
                or ln in skip_lines:
            blocked.add(key(v))
    targets = sorted(n for n in names - blocked if not n.startswith("__"))

    taken = {v.lower() for _, v in toks}
    mapping = {}
    for old in targets:
        new = _fresh_name(rng, taken)
        taken.add(new)
        mapping[old] = new

    out = []
    for t, v in toks:
        if is_name(t) and key(v) in mapping:
            out.append(mapping[key(v)])
        else:
            out.append(v)
    return "".join(out)


# --- whitespace and keyword case ----------------------------------------------

def _respace(value, lang, rng, at_bol):
    out = []
    for i, part in enumerate(value.split("\n")):
        if i:
            out.append("\n")
        bol = at_bol if i == 0 else True
        if bol and lang == "py":
            out.append(part)  # indentation is syntax in Python
        elif bol:
            out.append(" " * rng.randint(0, 4))
        elif part:
            out.append(" " * rng.randint(1, 2))
    return "".join(out)


def reformat(text, lang, rng):
    """Changes existing whitespace only (and Pascal keyword case). Never adds a token boundary."""
    out, at_bol = [], True
    for t, v in _lex(lang, text):
        if _is_ws(t, v):
            out.append(_respace(v, lang, rng, at_bol))
        else:
            if lang == "pas" and t in Token.Keyword:
                v = v.upper() if rng.random() < 0.5 else v.lower()
            out.append(v)
        if "\n" in v:
            at_bol = v.rsplit("\n", 1)[1].strip() == ""
        elif v.strip():
            at_bol = False
    return "".join(out)


# --- function reordering -------------------------------------------------------
# Items are (movable, name, refs, text). Barrier items (movable False) never move, so
# globals, includes, prototypes and the main block keep their place. Movable items may
# only trade places within a run between barriers, and a function always follows the
# functions it refers to.

def _refs(text, lang):
    names = set(_IDENT.findall(text))
    return {n.lower() for n in names} if lang == "pas" else names


def _cpp_item(toks):
    text = _join(toks)
    sig = _sig(toks)
    code = [(t, v) for t, v in sig if t not in Token.Comment]
    pre = any(t in Token.Comment.Preproc or t in Token.Comment.PreprocFile for t, _ in sig)
    paren = next((i for i, (t, v) in enumerate(code) if t in Token.Punctuation and v == "("), None)
    brace = next((i for i, (t, v) in enumerate(code) if t in Token.Punctuation and v == "{"), None)
    movable = (
        not pre and paren is not None and brace is not None and 0 < paren < brace
        and code[-1][0] in Token.Punctuation and code[-1][1] == "}"
        and code[paren - 1][0] in Token.Name
    )
    name = code[paren - 1][1] if movable else None
    return (movable, name, _refs(text, "cpp"), text)


def _cpp_items(text):
    items, cur, depth, pending, pre = [], [], 0, False, False
    for t, v in _lex("cpp", text):
        if pre:  # rest of a preprocessor line ends at its newline (a Comment.Preproc token)
            cur.append((t, v))
            if "\n" in v:
                items.append(_cpp_item(cur))
                cur, pre = [], False
            continue
        if _is_ws(t, v):
            cur.append((t, v))
            continue
        if t in Token.Comment.Preproc or t in Token.Comment.PreprocFile:
            if _sig(cur):
                items.append(_cpp_item(cur))
                cur = []
            pre, pending = True, False
            cur.append((t, v))
            continue
        if pending and v != ";":
            items.append(_cpp_item(cur))
            cur = []
        pending = False
        cur.append((t, v))
        if t in Token.Punctuation:
            if v == "{":
                depth += 1
            elif v == "}":
                depth -= 1
                pending = depth == 0
            elif v == ";" and depth == 0:
                items.append(_cpp_item(cur))
                cur = []
    if cur:
        items.append(_cpp_item(cur))
    return items


def _py_items(text):
    blocks = []
    for line in text.splitlines(keepends=True):
        if not line[:1].isspace() or not blocks:
            blocks.append([line])
        else:
            blocks[-1].append(line)
    items, prev_decorated = [], False
    for lines in blocks:
        body = "".join(lines)
        m = re.match(r"def\s+(\w+)", lines[0])
        movable = bool(m) and not prev_decorated
        items.append((movable, m.group(1) if movable else None, _refs(body, "py"), body))
        prev_decorated = lines[0].startswith("@")
    return items


def _pas_items(text):
    """Top-level routines only. Returns None (no reorder) for nested routines or odd layouts."""
    items, buf, routine = [], [], []
    state, depth, paren, name, prev, begun = None, 0, 0, None, None, False
    for t, v in _lex("pas", text):
        if _is_ws(t, v):
            (routine if state else buf).append((t, v))
            continue
        low = v.lower()
        if state is None:
            if t in Token.Keyword and low in _PAS_ROUTINE and prev in _PAS_ENTRY:
                lead = buf  # whitespace-only gap stays attached to the routine, not a barrier
                if any(not _is_ws(*x) for x in buf):
                    items.append((False, None, set(), _join(buf)))
                    lead = []
                buf = []
                state, paren, name, begun = "header", 0, None, False
                routine = lead + [(t, v)]
            else:
                buf.append((t, v))
        else:
            routine.append((t, v))
            if state == "header":
                if name is None and t in Token.Name:
                    name = low
                if v == "(":
                    paren += 1
                elif v == ")":
                    paren -= 1
                elif v == ";" and paren == 0:
                    if any(w.lower() in ("forward", "external") for _, w in routine):
                        items.append((False, None, set(), _join(routine)))
                        state, routine = None, []
                    else:
                        state = "body"
            elif state == "body":
                if t in Token.Keyword and low in _PAS_ROUTINE:
                    return None  # nested routine
                if not begun:
                    if t in Token.Keyword and low == "begin":
                        begun, depth = True, 1
                elif t in Token.Keyword and low in _PAS_OPEN:
                    depth += 1
                elif t in Token.Keyword and low == "end":
                    depth -= 1
                    if depth == 0:
                        state = "tail"
            else:  # tail: the ';' after the routine's final end
                if v != ";":
                    return None
                items.append((name is not None, name, _refs(_join(routine), "pas"), _join(routine)))
                routine, state = [], None
        prev = low
    if state is not None:
        return None
    if buf:
        items.append((False, None, set(), _join(buf)))
    return items


_CHUNKERS = {"cpp": _cpp_items, "py": _py_items, "pas": _pas_items}


def _shuffle_segment(seg, rng):
    by_name = {}
    for i, it in enumerate(seg):
        by_name.setdefault(it[1], []).append(i)
    deps = [{j for r in it[2] for j in by_name.get(r, []) if j != i} for i, it in enumerate(seg)]
    order, placed = [], set()
    while len(order) < len(seg):
        ready = [i for i in range(len(seg)) if i not in placed and deps[i] <= placed]
        if not ready:  # cycle (cannot happen for code that compiles): keep source order
            return seg
        i = rng.choice(ready)
        order.append(i)
        placed.add(i)
    return [seg[i] for i in order]


def _shuffle_items(items, rng):
    out, seg = [], []
    for item in items + [None]:
        if item is not None and item[0]:
            seg.append(item)
            continue
        out.extend(_shuffle_segment(seg, rng))
        seg = []
        if item is not None:
            out.append(item)
    return [it[3] for it in out]


def reorder_functions(text, lang, rng):
    text = _normalize(text)
    items = _CHUNKERS[lang](text)
    if items is None:
        return text
    return "".join(_shuffle_items(items, rng))


# --- integer type swap --------------------------------------------------------

def _swap_cpp(text):
    toks = _lex("cpp", text)
    sig_idx = [i for i, (t, v) in enumerate(toks) if not _is_ws(t, v)]
    repl = {}
    for k, i in enumerate(sig_idx):
        if i in repl:
            continue
        t, v = toks[i]
        if t not in Token.Keyword.Type:
            continue
        prev = toks[sig_idx[k - 1]][1] if k >= 1 else ""
        nxt = toks[sig_idx[k + 1]][1] if k + 1 < len(sig_idx) else ""
        if v == "int":
            if prev in ("long", "short") or nxt == "main":
                continue
            repl[i] = "long long"
        elif v == "long" and nxt == "long" and prev not in ("long", "short"):
            after = toks[sig_idx[k + 2]][1] if k + 2 < len(sig_idx) else ""
            if after == "int":
                continue
            repl[i] = "int"
            repl[sig_idx[k + 1]] = ""
    return "".join(repl.get(i, v) for i, (t, v) in enumerate(toks))


def _swap_pas(text):
    out = []
    for t, v in _lex("pas", text):
        low = v.lower()
        if not _is_ws(t, v) and (t in Token.Keyword.Type or t in Token.Name) and low in ("integer", "longint"):
            out.append("longint" if low == "integer" else "integer")
        else:
            out.append(v)
    return "".join(out)


def swap_int_types(text, lang, rng):
    text = _normalize(text)
    if lang == "cpp":
        return _swap_cpp(text)
    if lang == "pas":
        return _swap_pas(text)
    return text


# --- dead code ----------------------------------------------------------------

def insert_dead_code(text, lang, rng):
    text = _normalize(text)
    toks = _lex(lang, text)
    taken = {v.lower() for _, v in toks}
    if lang == "cpp":
        i = next((k for k, (t, v) in enumerate(toks) if t in Token.Name and v == "main"), None)
        if i is None:
            return text
        j = next((k for k in range(i + 1, len(toks))
                  if toks[k][0] in Token.Punctuation and toks[k][1] == "{"), None)
        if j is None:
            return text
        var = _fresh_name(rng, taken)
        return _insert_after(toks, j, f" if (0) {{ int {var} = 0; {var}++; }}")
    if lang == "py":
        var = _fresh_name(rng, taken)
        sep = "\n" if text and not text.endswith("\n") else ""
        return text + sep + f"\ndef {var}():\n    return 0\n"
    # pas: the main block is the last begin at nesting depth 0
    j, depth = None, 0
    for k, (t, v) in enumerate(toks):
        if t not in Token.Keyword:
            continue
        low = v.lower()
        if low == "begin" and depth == 0:
            j = k
        if low in _PAS_OPEN:
            depth += 1
        elif low == "end":
            depth -= 1
    if j is None:
        return text
    return _insert_after(toks, j, " if false then begin end;")


# --- counted for loop to while ------------------------------------------------

def _next_sig(toks, i):
    while i < len(toks):
        if not _is_ws(*toks[i]):
            return i
        i += 1
    return None


def _counted_for(toks, k):
    """(replacement, index after the loop) for `for (init; cond; ++i) { ... }`, else None."""
    i = _next_sig(toks, k + 1)
    if i is None or toks[i][1] != "(":
        return None
    parts, cur, depth, j = [], [], 0, i + 1
    while True:
        if j >= len(toks):
            return None
        t, v = toks[j]
        if t in Token.Punctuation and v in ("(", "[", "{"):
            depth += 1
        elif t in Token.Punctuation and v in (")", "]", "}"):
            if depth == 0:
                break
            depth -= 1
        if depth == 0 and t in Token.Punctuation and v == ";":
            parts.append(cur)
            cur = []
        else:
            cur.append((t, v))
        j += 1
    parts.append(cur)
    if len(parts) != 3:
        return None
    init, cond, step = (_join(p).strip() for p in parts)
    m = re.fullmatch(r"\+\+(\w+)|(\w+)\+\+", "".join(v for _, v in _sig(parts[2])))
    if not m:
        return None
    var = m.group(1) or m.group(2)
    if not any(t in Token.Name and v == var for t, v in parts[0]):
        return None
    b = _next_sig(toks, j + 1)
    if b is None or toks[b][1] != "{":
        return None
    depth, close = 0, None
    for c in range(b, len(toks)):
        t, v = toks[c]
        if t in Token.Punctuation and v == "{":
            depth += 1
        elif t in Token.Punctuation and v == "}":
            depth -= 1
            if depth == 0:
                close = c
                break
    if close is None:
        return None
    body = toks[b + 1:close]
    if any(t in Token.Keyword and v == "continue" for t, v in body):
        return None  # `continue` would skip the step in the while form
    repl = f"{{ {init}; while ({cond}) {{{_join(body)}{step}; }} }}"
    return repl, close + 1


def for_to_while(text, lang, rng):
    if lang != "cpp":
        return text
    toks = _lex("cpp", text)
    out, k = [], 0
    while k < len(toks):
        t, v = toks[k]
        got = _counted_for(toks, k) if t in Token.Keyword and v == "for" else None
        if got:
            out.append(got[0])
            k = got[1]
        else:
            out.append(v)
            k += 1
    return "".join(out)


TRANSFORMS = [
    ("rename_identifiers", rename_identifiers),
    ("reformat", reformat),
    ("reorder_functions", reorder_functions),
    ("swap_int_types", swap_int_types),
    ("insert_dead_code", insert_dead_code),
    ("for_to_while", for_to_while),
]


def disguise(text, lang, rng):
    """Apply every transform in order. Returns (new_text, names of transforms that changed it)."""
    applied = []
    text = _normalize(text)
    for name, fn in TRANSFORMS:
        new = fn(text, lang, rng)
        if new != text:
            applied.append(name)
            text = new
    return text, applied


# --- tree driver --------------------------------------------------------------

def disguise_tree(src, out, per_problem=3, seed=1):
    src, out = Path(src), Path(out)
    out.mkdir(parents=True, exist_ok=True)
    rng = random.Random(seed)
    groups = {}
    for f in sorted(src.glob("*/*/*/main.*")):
        problem, lang, user = f.relative_to(src).parts[:3]
        dest = out / f.relative_to(src)
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(f, dest)
        ext = f.suffix[1:]
        if ext in LEXERS:
            groups.setdefault((problem, lang), []).append((user, f, ext))

    manifest = []
    for (problem, lang), entries in sorted(groups.items()):
        for user, f, ext in rng.sample(entries, min(per_problem, len(entries))):
            text = f.read_bytes().decode("utf-8", errors="surrogateescape")
            new, applied = disguise(text, ext, rng)
            copy_rel = Path(problem, lang, f"{user}__copy", f"main.{ext}")
            (out / copy_rel).parent.mkdir(parents=True, exist_ok=True)
            (out / copy_rel).write_bytes(new.encode("utf-8", errors="surrogateescape"))
            manifest.append({
                "problem": problem,
                "lang": lang,
                "original": Path(problem, lang, user, f"main.{ext}").as_posix(),
                "copy": copy_rel.as_posix(),
                "transforms": applied,
            })
    (out / "manifest.json").write_text(json.dumps(manifest, indent=2), encoding="utf-8")
    return manifest


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--src", required=True, help="input dir, e.g. data/tik25")
    ap.add_argument("--out", required=True, help="output dir, e.g. data/tik25-disguised")
    ap.add_argument("--per-problem", type=int, default=3, help="copies per problem x language")
    ap.add_argument("--seed", type=int, default=1)
    args = ap.parse_args(argv)
    if Path(args.src).resolve() == Path(args.out).resolve():
        ap.error("--out must differ from --src")
    manifest = disguise_tree(args.src, args.out, args.per_problem, args.seed)
    print(f"wrote {len(manifest)} disguised copies to {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
