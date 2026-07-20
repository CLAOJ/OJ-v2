'use client';

import { useState, type ReactNode } from 'react';
import { ChevronRight } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';

/**
 * Collapsible spoiler, the v2 equivalent of DMOJ's `blockquote.spoiler`.
 * Editorials hide reference solutions behind these so the answer isn't given
 * away on load. Collapsed by default; click to reveal.
 */
export default function Spoiler({ children }: { children: ReactNode }) {
    const t = useTranslations('problem');
    const [open, setOpen] = useState(false);

    return (
        <div className="not-prose my-4 overflow-hidden rounded-xl border border-border bg-muted/30">
            <button
                type="button"
                onClick={() => setOpen((o) => !o)}
                aria-expanded={open}
                className="flex w-full items-center gap-2 px-4 py-3 text-sm font-bold text-muted-foreground transition-colors hover:text-foreground"
            >
                <ChevronRight
                    size={16}
                    className={cn('transition-transform duration-200', open && 'rotate-90')}
                />
                {open ? t('spoiler_hide') : t('spoiler_show')}
            </button>
            {open && <div className="border-t border-border px-4 py-3">{children}</div>}
        </div>
    );
}
