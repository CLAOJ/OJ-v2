// Exact DMOJ language keys (judge_language.key) → highlighter language id
// (Prism and Monaco share these ids). Keys are matched case-insensitively;
// anything not listed falls through to the prefix rules in resolveLanguage().
const LANGUAGE_MAP: Record<string, string> = {
    C: 'c',
    C11: 'c',
    CLANG: 'c',
    CPP03: 'cpp',
    CPP11: 'cpp',
    CPP14: 'cpp',
    CPP17: 'cpp',
    CPP20: 'cpp',
    CLANGX: 'cpp',
    PY2: 'python',
    PY3: 'python',
    PYPY2: 'python',
    PYPY3: 'python',
    JAVA: 'java',
    JAVA8: 'java',
    JAVA11: 'java',
    JAVA15: 'java',
    JAVA17: 'java',
    KOTLIN: 'kotlin',
    GO: 'go',
    RUST: 'rust',
    NODEJS: 'javascript',
    V8JS: 'javascript',
    JS: 'javascript',
    RUBY: 'ruby',
    PAS: 'pascal',
    PERL: 'perl',
    PHP: 'php',
    CS: 'csharp',
    MONOCS: 'csharp',
    HASK: 'haskell',
    LUA: 'lua',
    OCAML: 'ocaml',
    SCALA: 'scala',
    SWIFT: 'swift',
    D: 'd',
    DART: 'dart',
    GROOVY: 'groovy',
    SCM: 'scheme',
    TEXT: 'text',
};

// Map a raw DMOJ language key (e.g. "CPP17", "PY3", "JAVA8") to a
// highlighter language. The backend sends judge_language.key verbatim, which
// is uppercase and versioned, so a plain lowercase lookup never matches.
export function resolveLanguage(raw: string): string {
    if (!raw) return 'text';
    const key = raw.toUpperCase();
    if (LANGUAGE_MAP[key]) return LANGUAGE_MAP[key];
    if (key.startsWith('CPP') || key.startsWith('CLANGX')) return 'cpp';
    if (key.startsWith('PYPY') || key.startsWith('PY')) return 'python';
    if (key.startsWith('JAVA')) return 'java';
    if (key.startsWith('KOTLIN')) return 'kotlin';
    if (key === 'C' || key.startsWith('C1') || key === 'CLANG') return 'c';
    return raw.toLowerCase();
}

const TEMPLATES: Record<string, string> = {
    c: '#include <stdio.h>\n\nint main() {\n    return 0;\n}\n',
    cpp: '#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n    ios::sync_with_stdio(false);\n    cin.tie(nullptr);\n\n    return 0;\n}\n',
    python: 'def main():\n    pass\n\n\nif __name__ == "__main__":\n    main()\n',
    java: 'import java.util.*;\nimport java.io.*;\n\npublic class Main {\n    public static void main(String[] args) throws IOException {\n    }\n}\n',
    kotlin: 'fun main() {\n}\n',
    go: 'package main\n\nimport "fmt"\n\nfunc main() {\n    fmt.Println()\n}\n',
    rust: 'use std::io::{self, Read};\n\nfn main() {\n    let mut input = String::new();\n    io::stdin().read_to_string(&mut input).unwrap();\n}\n',
    javascript: "const input = require('fs').readFileSync(0, 'utf8');\n",
    pascal: 'program Main;\n\nbegin\nend.\n',
    csharp: 'using System;\n\nclass Program {\n    static void Main() {\n    }\n}\n',
};

/** Starter code for a DMOJ language key ('' when there is none). */
export function templateFor(key: string): string {
    return TEMPLATES[resolveLanguage(key)] ?? '';
}

/** True when the editor holds nothing the user wrote (blank or a stock template). */
export function isStockSource(source: string): boolean {
    return source.trim() === '' || Object.values(TEMPLATES).includes(source);
}
