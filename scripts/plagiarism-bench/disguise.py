#!/usr/bin/env python3
"""Disguised-copy generator: plants known copies for the compare50 bench.

Usage: disguise.py --src data/<key> --out data/<key>-disguised [--per-problem 3] [--seed 1]

Reads <src>/<problem>/<lang>/<user>/main.<ext>, mirrors every submission into --out,
and adds <user>__copy/main.<ext> beside up to --per-problem originals per
problem x language. manifest.json at the top of --out lists the copies.
Tokenizing is done with Pygments; --src is never written.
"""

import argparse
import ast
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
# Only names the file itself declares are renamed (under-renaming is safe, over-renaming
# breaks the copy). Detection is heuristic per language.

_CPP_QUAL = {"const", "static", "constexpr", "inline", "extern", "register", "volatile",
             "typedef", "struct", "class", "enum", "union", "mutable"}
_CPP_TYPEKW = {"auto", "unsigned", "signed"}
_CPP_FOLLOW = {"=", ";", "(", ")", "[", ",", "{", ":"}
_TPL_OK = {",", "*", "&", "<", ">", ">>", ":"}


def _is_code_tok(t):
    return t not in Token.Comment or t in Token.Comment.Preproc or t in Token.Comment.PreprocFile


def _cpp_declared(sig):
    vals = [v for _, v in sig]
    n = len(sig)
    vals += [""] * 4

    def is_name(i):
        return i < n and sig[i][0] in Token.Name and sig[i][0] not in Token.Name.Builtin

    def skip_tpl(k):
        depth, m = 0, k
        while m < n:
            t, v = sig[m]
            if v == "<":
                depth += 1
            elif v in (">", ">>"):
                depth -= len(v)
            elif not (t in Token.Name or t in Token.Keyword or t in Token.Literal.Number
                      or v in _TPL_OK):
                return None
            m += 1
            if depth <= 0:
                return m
        return None

    def skip_type(i):
        k, has = i, False
        while k < n and (vals[k] in _CPP_QUAL or sig[k][0] in Token.Keyword.Type
                         or vals[k] in _CPP_TYPEKW):
            has = has or vals[k] not in _CPP_QUAL
            k += 1
        if not has:
            if k >= n or sig[k][0] not in Token.Name:
                return None
            k += 1
            while vals[k] == ":" and vals[k + 1] == ":" and k + 2 < n and sig[k + 2][0] in Token.Name:
                k += 3
            if vals[k] == "<":
                k = skip_tpl(k)
                if k is None:
                    return None
        while k < n and vals[k] in ("*", "&", "&&", "const"):
            k += 1
        return k

    def decl_name(i):
        k = skip_type(i)
        return k if k is not None and is_name(k) and vals[k + 1] in _CPP_FOLLOW else None

    brace, depth_at = 0, []
    for v in vals[:n]:
        depth_at.append(brace)
        brace += v == "{"
        brace -= v == "}"

    declared = set()

    def params(m):
        depth, j = 0, m
        while j < n:
            v = vals[j]
            if v in ("(", "[", "{"):
                depth += 1
                if depth == 1 and v == "(":
                    k = decl_name(j + 1)
                    if k is not None:
                        declared.add(vals[k])
            elif v in (")", "]", "}"):
                depth -= 1
                if depth == 0:
                    return
            elif v == "," and depth == 1:
                k = decl_name(j + 1)
                if k is not None:
                    declared.add(vals[k])
            j += 1

    for i in range(n):
        v = vals[i]
        if v in ("struct", "class", "union", "enum") and is_name(i + 1) and vals[i + 2] in ("{", ";", ":"):
            declared.add(vals[i + 1])
        if v == "using" and is_name(i + 1) and vals[i + 2] == "=":
            declared.add(vals[i + 1])
        start = (i == 0 or vals[i - 1] in (";", "{", "}") or sig[i - 1][0] in Token.Comment.Preproc
                 or sig[i - 1][0] in Token.Comment.PreprocFile
                 or (vals[i - 1] == "(" and i >= 2 and vals[i - 2] == "for"))
        if not start:
            continue
        k = decl_name(i)
        if k is None:
            continue
        declared.add(vals[k])
        if vals[k + 1] == "(" and depth_at[k] == 0:
            params(k + 1)
        m, depth = k + 1, 0
        while m < n:
            w = vals[m]
            if w in ("(", "[", "{"):
                if w == "{" and depth == 0 and vals[m - 1] == ")":
                    break  # function body
                depth += 1
            elif w in (")", "]", "}"):
                if depth == 0:
                    break
                depth -= 1
            elif depth == 0 and w in (";", ":"):
                break
            elif depth == 0 and w == ",":
                j = m + 1
                while vals[j] in ("*", "&"):
                    j += 1
                if is_name(j) and vals[j + 1] in _CPP_FOLLOW:
                    declared.add(vals[j])
            m += 1
    # names inside #define/#if etc. (macro names and what macro bodies use) must stay
    blocked = set()
    for t, v in sig:
        if t in Token.Comment.Preproc:
            blocked.update(_IDENT.findall(v))
    return declared - blocked


def _py_declared(text):
    try:
        tree = ast.parse(text)
    except (SyntaxError, ValueError):
        return set()
    names, blocked = set(), set()
    for node in ast.walk(tree):
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
            names.add(node.name)
        elif isinstance(node, ast.arg):
            names.add(node.arg)
        elif isinstance(node, ast.Name) and isinstance(node.ctx, ast.Store):
            names.add(node.id)
        elif isinstance(node, ast.ExceptHandler) and node.name:
            names.add(node.name)
        elif isinstance(node, ast.keyword) and node.arg:
            blocked.add(node.arg)  # f(x=1): the callee may be a library
        elif isinstance(node, ast.ImportFrom) and any(a.name == "*" for a in node.names):
            return set()
        elif isinstance(node, ast.Constant) and isinstance(node.value, str) and node.value.isidentifier():
            blocked.add(node.value)  # getattr(K, "val") and friends
        elif isinstance(node, ast.JoinedStr):
            blocked.update(s.id for s in ast.walk(node) if isinstance(s, ast.Name))
    return names - blocked


def _pas_declared(sig):
    vals = [v.lower() for _, v in sig] + [""] * 4
    n = len(sig)
    declared = set()

    def is_name(i):
        return i < n and sig[i][0] in Token.Name

    def name_list(j):
        names = []
        while is_name(j):
            names.append(j)
            j += 1
            if vals[j] != ",":
                break
            j += 1
        return names, j

    def skip_entry(j):
        depth = 0
        while j < n:
            v = vals[j]
            if v in ("(", "["):
                depth += 1
            elif v in (")", "]"):
                depth -= 1
            elif v == "record" and depth == 0:
                return j + 1  # fields follow as entries
            elif v == ";" and depth == 0:
                return j + 1
            j += 1
        return j

    def section(i):
        while i < n:
            if vals[i] == "end" and sig[i][0] in Token.Keyword:
                i = skip_entry(i)
                continue
            names, j = name_list(i)
            if names and vals[j] in (":", "="):
                declared.update(vals[k] for k in names)
                i = skip_entry(j)
            else:
                return i
        return i

    def routine(i):
        if is_name(i) and vals[i + 1] != ".":
            declared.add(vals[i])
        i += 1
        if vals[i] != "(":
            return i
        j = i + 1
        while j < n and vals[j] != ")":
            if sig[j][0] in Token.Keyword and vals[j] in ("var", "const", "out"):
                j += 1
            names, j = name_list(j)
            if names and vals[j] == ":":
                declared.update(vals[k] for k in names)
            depth = 0
            while j < n:
                v = vals[j]
                if v in ("(", "["):
                    depth += 1
                elif v in (")", "]"):
                    if depth == 0:
                        break
                    depth -= 1
                elif v == ";" and depth == 0:
                    j += 1
                    break
                j += 1
        return j

    i = 0
    while i < n:
        if sig[i][0] in Token.Keyword and vals[i] in ("var", "const", "type", "label"):
            i = section(i + 1)
        elif sig[i][0] in Token.Keyword and vals[i] in _PAS_ROUTINE:
            i = routine(i + 1)
        else:
            i += 1
    return declared


def rename_identifiers(text, lang, rng):
    text = _normalize(text)
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
    code = [(t, v) for t, v, _ in sig if _is_code_tok(t)]
    if lang == "cpp":
        declared = _cpp_declared(code)
    elif lang == "py":
        declared = _py_declared(text)
    else:
        declared = _pas_declared(code)
    targets = sorted(n for n in (names & {key(d) for d in declared}) - blocked
                     if not n.startswith("__"))

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
    """Top-level statements by ast line ranges (never splits strings or brackets). None = no reorder."""
    try:
        tree = ast.parse(text)
    except (SyntaxError, ValueError):
        return None
    lines = text.split("\n")
    items, prev_end = [], 0
    for node in tree.body:
        start = min([node.lineno] + [d.lineno for d in getattr(node, "decorator_list", [])])
        if start <= prev_end:
            return None  # two statements share a line
        body = "\n".join(lines[prev_end:node.end_lineno]) + "\n"  # gap text rides with the next node
        movable = isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)) and not node.decorator_list
        items.append((movable, node.name if movable else None, _refs(body, "py"), body))
        prev_end = node.end_lineno
    tail = "\n".join(lines[prev_end:])
    if tail:
        items.append((False, None, set(), tail if tail.endswith("\n") else tail + "\n"))
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
        if v != "int" or t not in Token.Keyword.Type:
            continue
        prev = toks[sig_idx[k - 1]][1] if k >= 1 else ""
        nxt = toks[sig_idx[k + 1]][1] if k + 1 < len(sig_idx) else ""
        if prev in ("long", "short") or nxt == "main":
            continue
        if nxt == "(":  # `int(x)` cast style stays
            continue
        repl[i] = "long long"
    # never touch main's parameter list
    for k, i in enumerate(sig_idx):
        if toks[i][1] == "main" and k + 1 < len(sig_idx) and toks[sig_idx[k + 1]][1] == "(":
            depth = 0
            for j in sig_idx[k + 1:]:
                depth += toks[j][1] == "("
                depth -= toks[j][1] == ")"
                repl.pop(j, None)
                if depth == 0:
                    break
    return "".join(repl.get(i, v) for i, (t, v) in enumerate(toks))


# Any of these makes a wider int change meaning or break overload/format matching.
_SWAP_SKIP = ("printf", "scanf", "max(", "min(", "swap(", "abs(", "gcd(", "lcm(", "%d", "sizeof(",
              "qsort", "bsearch", "function<")
_FN_PTR = re.compile(r"\bint\s*\(\s*\*")


def _swap_pas(text):
    out = []
    for t, v in _lex("pas", text):
        low = v.lower()
        if not _is_ws(t, v) and (t in Token.Keyword.Type or t in Token.Name) and low in ("integer", "longint"):
            out.append("longint" if low == "integer" else v)
        else:
            out.append(v)
    return "".join(out)


def swap_int_types(text, lang, rng):
    text = _normalize(text)
    if lang == "cpp":
        if any(s in text for s in _SWAP_SKIP) or _FN_PTR.search(text):
            return text
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
        sig_idx = [k for k, (t, v) in enumerate(toks) if not _is_ws(t, v) and t not in Token.Comment]
        j = None  # first `main ( ... ) {` is the definition; `main();` is a prototype
        for q, i in enumerate(sig_idx):
            if toks[i][0] in Token.Name and toks[i][1] == "main" and q + 1 < len(sig_idx) \
                    and toks[sig_idx[q + 1]][1] == "(":
                depth = 0
                for r in range(q + 1, len(sig_idx)):
                    depth += toks[sig_idx[r]][1] == "("
                    depth -= toks[sig_idx[r]][1] == ")"
                    if depth == 0:
                        if r + 1 < len(sig_idx) and toks[sig_idx[r + 1]][1] == "{":
                            j = sig_idx[r + 1]
                        break
            if j is not None:
                break
        if j is None:
            return text
        var = _fresh_name(rng, taken)
        return _insert_after(toks, j, f" if (0) {{ int {var} = 0; {var}++; }}")
    if lang == "py":
        var = _fresh_name(rng, taken)
        sep = "\n" if text and not text.endswith("\n") else ""
        return text + sep + f"\ndef {var}():\n    return 0\n"
    # pas: the main block is the last begin at nesting depth 0. A `case` directly inside a
    # `record` has no `end` of its own; any unbalanced `end` means we are unsure, so skip.
    j, stack = None, []
    for k, (t, v) in enumerate(toks):
        if t not in Token.Keyword:
            continue
        low = v.lower()
        if low == "begin" and not stack:
            j = k
        if low == "case" and stack and stack[-1] == "record":
            continue
        if low in _PAS_OPEN:
            stack.append(low)
        elif low == "end":
            if not stack:
                return text
            stack.pop()
    if j is None or stack:
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
    if not cond:
        return None  # `while ()` is invalid; leave the loop alone
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
    if out.exists() and any(out.iterdir()):
        raise SystemExit(f"refusing: --out {out} exists and is not empty")
    if out.resolve().is_relative_to(src.resolve()):
        raise SystemExit(f"refusing: --out {out} is inside --src {src}")
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
            if not applied:
                continue  # nothing changed: a byte-identical "copy" would be a free true positive
            # ponytail: review item 11 (rare case) not handled; add when it shows up in real data
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
