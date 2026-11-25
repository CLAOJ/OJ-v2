'use client';

import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import rehypeRaw from 'rehype-raw';
import 'katex/dist/katex.min.css';
import { useTranslations } from 'next-intl';
import { Check, Copy } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { cn } from '@/lib/utils';
import { normalizeDmojMarkdown } from '@/lib/markdown';

interface MathRendererProps {
    content: string;
    className?: string;
    /**
     * Show a Copy button on every code block. Problem statements carry their
     * samples as plain code blocks (v2 has no sample model), so this is what
     * gives each sample its copy affordance. Off by default: blogs and
     * comments don't need it.
     */
    copyable?: boolean;
}

function CopyablePre({ children }: { children?: React.ReactNode }) {
    const t = useTranslations('Problems');
    const preRef = useRef<HTMLPreElement>(null);
    const [copied, setCopied] = useState(false);
    const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

    useEffect(() => () => {
        if (timer.current) clearTimeout(timer.current);
    }, []);

    const copy = async () => {
        const text = preRef.current?.innerText ?? '';
        try {
            await navigator.clipboard.writeText(text);
            setCopied(true);
            if (timer.current) clearTimeout(timer.current);
            timer.current = setTimeout(() => setCopied(false), 1500);
        } catch {
            // Clipboard API unavailable (insecure context): leave the label as-is.
        }
    };

    const label = copied ? t('copied') : t('copy');

    return (
        <div className="group relative">
            <button
                type="button"
                onClick={copy}
                aria-label={label}
                title={label}
                className="absolute right-2 top-2 z-10 rounded border bg-card/90 p-1.5 text-muted-foreground opacity-0 transition-opacity hover:text-primary focus:opacity-100 group-hover:opacity-100 print:hidden"
            >
                {copied ? <Check size={14} /> : <Copy size={14} />}
            </button>
            <pre ref={preRef}>{children}</pre>
        </div>
    );
}

export default function MathRenderer({ content, className, copyable = false }: MathRendererProps) {
    return (
        <div className={cn("prose dark:prose-invert max-w-none prose-headings:font-bold prose-a:text-primary text-foreground", className)}>
            <ReactMarkdown
                remarkPlugins={[[remarkGfm, { singleTilde: false }], remarkMath]}
                // rehype-raw always runs: v1 rendered statements with Python-Markdown,
                // which passes inline HTML through, so plain statements legitimately
                // contain `<i>`, `<sub>`, etc. It must run before rehype-katex so math
                // inside raw HTML is processed. Content is authored by trusted problem
                // setters/admins – the same trust boundary as v1.
                rehypePlugins={[rehypeRaw, rehypeKatex]}
                components={copyable ? { pre: ({ children }) => <CopyablePre>{children}</CopyablePre> } : undefined}
            >
                {normalizeDmojMarkdown(content)}
            </ReactMarkdown>
        </div>
    );
}