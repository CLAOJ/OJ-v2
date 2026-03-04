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

const LANGUAGE_MAP: Record<string, string> = {
    'cpp': 'cpp',
    'cpp11': 'cpp',
    'cpp14': 'cpp',
    'cpp17': 'cpp',
    'cpp20': 'cpp',
    'c': 'c',
    'python3': 'python',
    'python2': 'python',
    'py3': 'python',
    'py2': 'python',
    'java': 'java',
    'kotlin': 'kotlin',
    'go': 'go',
    'rust': 'rust',
    'javascript': 'javascript',
    'nodejs': 'javascript',
    'ruby': 'ruby',
    'pascal': 'pascal',
    'perl': 'perl',
    'php': 'php',
    'csharp': 'csharp',
    'haskell': 'haskell',
    'lua': 'lua',
    'ocaml': 'ocaml',
    'scala': 'scala',
    'swift': 'swift',
};

export default function SubmissionSource({ source, language, className }: SubmissionSourceProps) {
    const [copied, setCopied] = useState(false);

    const handleCopy = async () => {
        await navigator.clipboard.writeText(source);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    const syntaxLanguage = LANGUAGE_MAP[language] || 'text';

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
