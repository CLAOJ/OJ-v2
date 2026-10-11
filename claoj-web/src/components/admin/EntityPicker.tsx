'use client';

import { useMemo, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { X } from 'lucide-react';
import { useDebounce } from '@/hooks/useDebounce';
import { useOnClickOutside } from '@/hooks/useOnClickOutside';
import { cn } from '@/lib/utils';

export interface EntityOption {
    id: number;
    label: string;
    detail?: string;
}

interface EntityPickerProps {
    id: string;
    label: string;
    value: number[];
    onChange: (ids: number[]) => void;
    search: (term: string) => Promise<EntityOption[]>;
    /** Cache namespace for results, e.g. "users". */
    queryKey: string;
    /** Labels for ids already selected when the form opens. */
    initialOptions?: EntityOption[];
    placeholder?: string;
}

/**
 * Multi-select that searches the server as you type, so it works for
 * thousands of users/problems (list endpoints cap a page at 100). Selected
 * items render as removable chips in selection order.
 */
export function EntityPicker({ id, label, value, onChange, search, queryKey, initialOptions = [], placeholder }: EntityPickerProps) {
    const t = useTranslations('Admin');
    const [term, setTerm] = useState('');
    const [open, setOpen] = useState(false);
    const [learned, setLearned] = useState<Record<number, EntityOption>>({});
    const debounced = useDebounce(term.trim(), 250);
    const boxRef = useRef<HTMLDivElement>(null);
    useOnClickOutside(boxRef, () => setOpen(false));

    const known = useMemo(() => {
        const map: Record<number, EntityOption> = {};
        for (const o of initialOptions) map[o.id] = o;
        return { ...map, ...learned };
    }, [initialOptions, learned]);

    const { data: results = [], isFetching } = useQuery({
        queryKey: ['entity-picker', queryKey, debounced],
        queryFn: () => search(debounced),
        enabled: open && debounced.length > 0,
        staleTime: 30_000,
    });
    const visible = results.filter(o => !value.includes(o.id));

    const add = (o: EntityOption) => {
        setLearned(prev => ({ ...prev, [o.id]: o }));
        onChange([...value, o.id]);
        setTerm('');
    };

    return (
        <div ref={boxRef} className="space-y-2">
            <label htmlFor={id} className="text-sm font-medium text-muted-foreground block">{label}</label>
            {value.length > 0 && (
                <ul className="flex flex-wrap gap-1.5">
                    {value.map(v => {
                        const o = known[v];
                        const text = o?.label ?? `#${v}`;
                        return (
                            <li key={v} className="inline-flex items-center gap-1 rounded-md border bg-muted/50 pl-2 pr-1 py-0.5 text-sm">
                                <span>{text}</span>
                                {o?.detail && <span className="text-xs text-muted-foreground">{o.detail}</span>}
                                <button
                                    type="button"
                                    aria-label={t('picker.remove', { label: text })}
                                    onClick={() => onChange(value.filter(x => x !== v))}
                                    className="p-0.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
                                >
                                    <X size={14} />
                                </button>
                            </li>
                        );
                    })}
                </ul>
            )}
            <div className="relative">
                <input
                    id={id}
                    role="combobox"
                    aria-expanded={open && debounced.length > 0}
                    aria-controls={`${id}-listbox`}
                    aria-autocomplete="list"
                    autoComplete="off"
                    value={term}
                    placeholder={placeholder ?? t('picker.searchPlaceholder')}
                    onChange={e => { setTerm(e.target.value); setOpen(true); }}
                    onFocus={() => setOpen(true)}
                    onKeyDown={e => {
                        if (e.key === 'Escape') setOpen(false);
                        if (e.key === 'Enter') {
                            e.preventDefault();
                            if (visible[0]) add(visible[0]);
                        }
                    }}
                    className="w-full h-9 px-3 rounded-md border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-ring/40"
                />
                {open && debounced.length > 0 && (
                    <ul
                        id={`${id}-listbox`}
                        role="listbox"
                        className="absolute z-20 mt-1 w-full max-h-60 overflow-y-auto rounded-md border bg-popover shadow-lg"
                    >
                        {isFetching && visible.length === 0 ? (
                            <li className="px-3 py-2 text-sm text-muted-foreground">{t('picker.searching')}</li>
                        ) : visible.length === 0 ? (
                            <li className="px-3 py-2 text-sm text-muted-foreground">{t('picker.noResults')}</li>
                        ) : visible.map(o => (
                            <li
                                key={o.id}
                                role="option"
                                aria-selected={false}
                                onMouseDown={e => e.preventDefault()}
                                onClick={() => add(o)}
                                className={cn('px-3 py-2 text-sm cursor-pointer hover:bg-muted flex items-baseline gap-2')}
                            >
                                <span className="font-medium">{o.label}</span>
                                {o.detail && <span className="text-xs text-muted-foreground truncate">{o.detail}</span>}
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </div>
    );
}
