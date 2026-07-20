'use client';

import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import { normalizeDmojMarkdown } from '@/lib/markdown';

interface MarkdownProps {
    content: string;
}

export default function Markdown({ content }: MarkdownProps) {
    return (
        <div className="prose dark:prose-invert max-w-none text-foreground">
            <ReactMarkdown
                remarkPlugins={[[remarkGfm, { singleTilde: false }], remarkMath]}
                rehypePlugins={[rehypeKatex]}
            >
                {normalizeDmojMarkdown(content)}
            </ReactMarkdown>
        </div>
    );
}
