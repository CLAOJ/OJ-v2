'use client';

import ReactMarkdown, { type Components } from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import rehypeRaw from 'rehype-raw';
import 'katex/dist/katex.min.css';
import { normalizeDmojMarkdown } from '@/lib/markdown';
import Spoiler from '@/components/common/Spoiler';

interface MarkdownProps {
    content: string;
}

function hasSpoilerClass(className: unknown): boolean {
    if (Array.isArray(className)) return className.includes('spoiler');
    return typeof className === 'string' && /\bspoiler\b/.test(className);
}

const components: Components = {
    // DMOJ marks reference-solution spoilers with `<blockquote class="spoiler">`.
    // normalizeDmojMarkdown keeps that wrapper and rehype-raw turns it into a real
    // element; render those as a collapsible Spoiler, everything else as a quote.
    blockquote({ node, children, ...props }) {
        const className = node?.properties?.className;
        if (hasSpoilerClass(className)) return <Spoiler>{children}</Spoiler>;
        return <blockquote {...props}>{children}</blockquote>;
    },
};

export default function Markdown({ content }: MarkdownProps) {
    return (
        <div className="prose dark:prose-invert max-w-none text-foreground">
            <ReactMarkdown
                remarkPlugins={[[remarkGfm, { singleTilde: false }], remarkMath]}
                // rehype-raw must run before rehype-katex: it parses the raw
                // `<blockquote class="spoiler">` HTML into the tree, then katex
                // processes any math inside. Editorials are authored by trusted
                // problem setters (same trust tier as full-markup statements).
                rehypePlugins={[rehypeRaw, rehypeKatex]}
                components={components}
            >
                {normalizeDmojMarkdown(content)}
            </ReactMarkdown>
        </div>
    );
}
