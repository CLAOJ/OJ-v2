'use client';

import { useState } from 'react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { Copy, Check, FileCode } from 'lucide-react';
import { cn } from '@/lib/utils';

interface SubmissionSourceProps {
    source: string;
    language: string;
    className?: string;
}

// Exact DMOJ language keys (judge_language.key) → Prism language.
// Keys are matched case-insensitively; anything not listed falls through
// to the prefix rules / lowercase fallback in resolvePrismLanguage().
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

// Map a raw DMOJ language key (e.g. "CPP17", "PY3", "JAVA8") to a Prism
// language. The backend sends judge_language.key verbatim, which is uppercase
// and versioned, so a plain lowercase lookup never matched (everything fell
// back to "text" → no highlighting).
function resolvePrismLanguage(raw: string): string {
    if (!raw) return 'text';
    const key = raw.toUpperCase();
    if (LANGUAGE_MAP[key]) return LANGUAGE_MAP[key];
    // Prefix rules cover versioned variants not listed explicitly above.
    if (key.startsWith('CPP') || key.startsWith('CLANGX')) return 'cpp';
    if (key.startsWith('PYPY') || key.startsWith('PY')) return 'python';
    if (key.startsWith('JAVA')) return 'java';
    if (key.startsWith('KOTLIN')) return 'kotlin';
    if (key === 'C' || key.startsWith('C1') || key === 'CLANG') return 'c';
    // Last resort: Prism language ids are lowercase.
    return raw.toLowerCase();
}

export default function SubmissionSource({ source, language, className }: SubmissionSourceProps) {
    const [copied, setCopied] = useState(false);

    const handleCopy = async () => {
        await navigator.clipboard.writeText(source);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    const syntaxLanguage = resolvePrismLanguage(language);

    return (
        <div className={cn("relative rounded-2xl overflow-hidden border bg-[#1e1e1e]", className)}>
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-3 bg-[#252526] border-b border-[#333]">
                <div className="flex items-center gap-2 text-sm text-gray-400">
                    <FileCode size={16} />
                    <span className="font-mono uppercase">{language}</span>
                </div>
                <button
                    onClick={handleCopy}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-[#333] hover:bg-[#444] text-gray-300 text-sm font-medium transition-colors"
                >
                    {copied ? (
                        <>
                            <Check size={14} className="text-emerald-500" />
                            <span className="text-emerald-500">Copied!</span>
                        </>
                    ) : (
                        <>
                            <Copy size={14} />
                            <span>Copy</span>
                        </>
                    )}
                </button>
            </div>

            {/* Source Code */}
            <div className="max-h-[60vh] overflow-auto">
                <SyntaxHighlighter
                    language={syntaxLanguage}
                    style={vscDarkPlus}
                    showLineNumbers
                    lineNumberStyle={{
                        minWidth: '3em',
                        paddingRight: '1em',
                        color: '#6e7681',
                        backgroundColor: '#1e1e1e',
                    }}
                    customStyle={{
                        margin: 0,
                        padding: '1.5rem',
                        fontSize: '14px',
                        lineHeight: '1.6',
                        backgroundColor: '#1e1e1e',
                    }}
                >
                    {source || '// No source code available'}
                </SyntaxHighlighter>
            </div>
        </div>
    );
}
