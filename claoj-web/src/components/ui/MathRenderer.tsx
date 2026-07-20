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
    fullMarkup?: boolean;
}

export default function MathRenderer({ content, className, fullMarkup }: MathRendererProps) {
    return (
        <div className={cn("prose dark:prose-invert max-w-none prose-headings:font-bold prose-a:text-primary text-foreground", className)}>
            <ReactMarkdown
                remarkPlugins={[[remarkGfm, { singleTilde: false }], remarkMath]}
                rehypePlugins={[...(fullMarkup ? [rehypeRaw] : []), rehypeKatex]}
            >
                {normalizeDmojMarkdown(content)}
            </ReactMarkdown>
        </div>
    );
}
