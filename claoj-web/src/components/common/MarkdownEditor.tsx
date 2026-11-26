'use client';

import { useRef, useState, type ClipboardEvent, type DragEvent, type KeyboardEvent } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
    Bold, Italic, Heading, Link2, Code, SquareCode, Quote, List, ListOrdered, Sigma, ImagePlus, Loader2,
    type LucideIcon,
} from 'lucide-react';
import api, { apiErrorMessage } from '@/lib/api';
import { uploadDirect } from '@/lib/directUpload';
import { insertText, prefixLines, wrapSelection, type Edit } from '@/lib/markdownEdit';
import { cn } from '@/lib/utils';
import Markdown from '@/components/common/Markdown';

const MAX_BYTES = 2 * 1024 * 1024;
const IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'];

async function uploadImage(file: File): Promise<string> {
    const direct = await uploadDirect('/user/uploads/image/sign', file,
        key => api.post<{ url: string }>('/user/markdown/image/commit', { key }));
    if (direct) return direct.data.url;
    const form = new FormData();
    form.append('image', file);
    return (await api.post<{ url: string }>('/user/markdown/image', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
    })).data.url;
}

interface MarkdownEditorProps {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    rows?: number;
    autoFocus?: boolean;
    /** Ctrl/Cmd+Enter. */
    onSubmit?: () => void;
}

/** Markdown textarea with a formatting toolbar, preview and image upload
 * (button, paste or drop): what v1's martor widget offered for comments. */
export function MarkdownEditor({ value, onChange, placeholder, rows = 6, autoFocus, onSubmit }: MarkdownEditorProps) {
    const t = useTranslations('MarkdownEditor');
    const ref = useRef<HTMLTextAreaElement>(null);
    const fileRef = useRef<HTMLInputElement>(null);
    const valueRef = useRef(value);
    valueRef.current = value;
    const [tab, setTab] = useState<'write' | 'preview'>('write');
    const [uploading, setUploading] = useState(0);

    const apply = (make: (v: string, start: number, end: number) => Edit) => {
        const el = ref.current;
        const v = valueRef.current;
        const e = make(v, el?.selectionStart ?? v.length, el?.selectionEnd ?? v.length);
        onChange(e.value);
        requestAnimationFrame(() => {
            ref.current?.focus();
            ref.current?.setSelectionRange(e.start, e.end);
        });
    };

    const upload = async (files: File[]) => {
        for (const file of files) {
            if (!IMAGE_TYPES.includes(file.type)) { toast.error(t('invalidType')); continue; }
            if (file.size > MAX_BYTES) { toast.error(t('tooLarge')); continue; }
            setUploading(n => n + 1);
            try {
                const url = await uploadImage(file);
                const alt = file.name.replace(/\.[^.]+$/, '').replace(/[[\]]/g, '') || 'image';
                apply((v, s, e) => insertText(v, s, e, `![${alt}](${url})`));
            } catch (err) {
                toast.error(apiErrorMessage(err, t('uploadFailed')));
            } finally {
                setUploading(n => n - 1);
            }
        }
    };

    const tools: [LucideIcon, string, (v: string, s: number, e: number) => Edit][] = [
        [Bold, 'bold', (v, s, e) => wrapSelection(v, s, e, '**', '**', t('sampleText'))],
        [Italic, 'italic', (v, s, e) => wrapSelection(v, s, e, '*', '*', t('sampleText'))],
        [Heading, 'heading', (v, s, e) => prefixLines(v, s, e, '### ')],
        [Link2, 'link', (v, s, e) => wrapSelection(v, s, e, '[', '](https://)', t('sampleText'))],
        [Code, 'code', (v, s, e) => wrapSelection(v, s, e, '`', '`', 'code')],
        [SquareCode, 'codeBlock', (v, s, e) => wrapSelection(v, s, e, '```cpp\n', '\n```', 'code')],
        [Quote, 'quote', (v, s, e) => prefixLines(v, s, e, '> ')],
        [List, 'bulletList', (v, s, e) => prefixLines(v, s, e, '- ')],
        [ListOrdered, 'numberedList', (v, s, e) => prefixLines(v, s, e, i => `${i + 1}. `)],
        [Sigma, 'math', (v, s, e) => wrapSelection(v, s, e, '$', '$', 'x^2')],
    ];

    const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
        if (!(e.ctrlKey || e.metaKey)) return;
        const key = e.key.toLowerCase();
        if (key === 'enter' && onSubmit) { e.preventDefault(); onSubmit(); }
        else if (key === 'b') { e.preventDefault(); apply(tools[0][2]); }
        else if (key === 'i') { e.preventDefault(); apply(tools[1][2]); }
    };

    const imagesOf = (items: DataTransferItemList | FileList) =>
        Array.from(items as ArrayLike<DataTransferItem | File>)
            .map(i => (i instanceof File ? i : i.kind === 'file' ? i.getAsFile() : null))
            .filter((f): f is File => !!f && f.type.startsWith('image/'));

    const onPaste = (e: ClipboardEvent<HTMLTextAreaElement>) => {
        const files = imagesOf(e.clipboardData.items);
        if (files.length) { e.preventDefault(); upload(files); }
    };
    const onDrop = (e: DragEvent<HTMLTextAreaElement>) => {
        const files = imagesOf(e.dataTransfer.files);
        if (files.length) { e.preventDefault(); upload(files); }
    };

    const tabClass = (on: boolean) => cn(
        'h-8 px-3 rounded text-xs font-medium transition-colors',
        on ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground',
    );

    return (
        <div className="rounded-md border bg-background focus-within:ring-2 focus-within:ring-primary/20 transition-shadow">
            <div className="flex flex-wrap items-center gap-1 border-b bg-muted/40 px-2 py-1.5">
                <div role="tablist" className="flex gap-1 rounded-md bg-muted p-0.5">
                    <button type="button" role="tab" aria-selected={tab === 'write'} className={tabClass(tab === 'write')} onClick={() => setTab('write')}>
                        {t('write')}
                    </button>
                    <button type="button" role="tab" aria-selected={tab === 'preview'} className={tabClass(tab === 'preview')} onClick={() => setTab('preview')}>
                        {t('preview')}
                    </button>
                </div>
                {tab === 'write' && (
                    <div className="ml-auto flex flex-wrap items-center">
                        {tools.map(([Icon, key, make]) => (
                            <button
                                key={key}
                                type="button"
                                title={t(key)}
                                aria-label={t(key)}
                                onClick={() => apply(make)}
                                className="h-8 w-8 inline-flex items-center justify-center rounded text-muted-foreground hover:bg-background hover:text-foreground transition-colors"
                            >
                                <Icon size={16} />
                            </button>
                        ))}
                        <button
                            type="button"
                            title={t('image')}
                            aria-label={t('image')}
                            disabled={uploading > 0}
                            onClick={() => fileRef.current?.click()}
                            className="h-8 w-8 inline-flex items-center justify-center rounded text-muted-foreground hover:bg-background hover:text-foreground transition-colors disabled:opacity-50"
                        >
                            {uploading > 0 ? <Loader2 size={16} className="animate-spin" /> : <ImagePlus size={16} />}
                        </button>
                        <input
                            ref={fileRef}
                            type="file"
                            accept={IMAGE_TYPES.join(',')}
                            multiple
                            className="hidden"
                            onChange={(e) => {
                                const files = Array.from(e.target.files ?? []);
                                e.target.value = '';
                                upload(files);
                            }}
                        />
                    </div>
                )}
            </div>

            {tab === 'write' ? (
                <textarea
                    ref={ref}
                    value={value}
                    onChange={(e) => onChange(e.target.value)}
                    onKeyDown={onKeyDown}
                    onPaste={onPaste}
                    onDrop={onDrop}
                    rows={rows}
                    autoFocus={autoFocus}
                    placeholder={placeholder}
                    className="block w-full resize-y bg-transparent px-4 py-3 text-sm leading-relaxed outline-none"
                />
            ) : (
                <div className="px-4 py-3 overflow-x-auto" style={{ minHeight: `${rows * 1.6 + 1.5}rem` }}>
                    {value.trim()
                        ? <Markdown content={value} safe className="prose-sm break-words" />
                        : <p className="text-sm text-muted-foreground">{t('nothingToPreview')}</p>}
                </div>
            )}

            <p className="border-t px-4 py-1.5 text-xs text-muted-foreground">
                {uploading > 0 ? t('uploading') : t('hint')}
            </p>
        </div>
    );
}
