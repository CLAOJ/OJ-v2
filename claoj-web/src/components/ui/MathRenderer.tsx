'use client';

import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import rehypeRaw from 'rehype-raw';
import 'katex/dist/katex.min.css';
import { cn } from '@/lib/utils';
import { normalizeDmojMarkdown } from '@/lib/markdown';

interface MathRendererProps {
    content: string;
    className?: string;
}

export default function MathRenderer({ content, className }: MathRendererProps) {
    return (
        <div className={cn("prose dark:prose-invert max-w-none prose-headings:font-bold prose-a:text-primary text-foreground", className)}>
            <ReactMarkdown
                remarkPlugins={[[remarkGfm, { singleTilde: false }], remarkMath]}
                // rehype-raw always runs: v1 rendered statements with Python-Markdown,
                // which passes inline HTML through, so plain statements legitimately
                // contain `<i>`, `<sub>`, etc. It must run before rehype-katex so math
                // inside raw HTML is processed. Content is authored by trusted problem
                // setters/admins — the same trust boundary as v1.
                rehypePlugins={[rehypeRaw, rehypeKatex]}
            >
                {normalizeDmojMarkdown(content)}
            </ReactMarkdown>
        </div>
    );
}