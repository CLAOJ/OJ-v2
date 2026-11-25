'use client';

import { X } from 'lucide-react';

export interface TagOption {
    id: number;
    label: string;
    color?: string;
}

interface TagSelectProps {
    id: string;
    options: TagOption[];
    value: number[];
    onChange: (ids: number[]) => void;
    /** First (empty) entry of the "add" select. */
    placeholder: string;
    removeLabel: (label: string) => string;
}

/**
 * Multi-choice from a short fixed list (v1 multi-select shape): chosen items
 * as removable tags above one closed "add" select. For long or server-side
 * lists use admin/EntityPicker, which searches as you type.
 */
export function TagSelect({ id, options, value, onChange, placeholder, removeLabel }: TagSelectProps) {
    const byId = new Map(options.map(o => [o.id, o]));
    return (
        <div>
            {value.length > 0 && (
                <div className="flex flex-wrap gap-1.5 mb-2">
                    {value.map(v => {
                        const o = byId.get(v);
                        if (!o) return null;
                        return (
                            <span
                                key={v}
                                className="inline-flex items-center gap-1 pl-2.5 pr-1 h-7 rounded-md bg-primary/15 text-primary text-xs font-medium"
                                style={o.color ? { color: o.color } : undefined}
                            >
                                {o.label}
                                <button
                                    type="button"
                                    aria-label={removeLabel(o.label)}
                                    onClick={() => onChange(value.filter(x => x !== v))}
                                    className="p-0.5 rounded hover:bg-primary/20 transition-colors"
                                >
                                    <X size={12} />
                                </button>
                            </span>
                        );
                    })}
                </div>
            )}
            <select
                id={id}
                value=""
                onChange={(e) => { const n = Number(e.target.value); if (n) onChange([...value, n]); }}
                className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none text-sm"
            >
                <option value="">{placeholder}</option>
                {options.filter(o => !value.includes(o.id)).map(o => (
                    <option key={o.id} value={o.id}>{o.label}</option>
                ))}
            </select>
        </div>
    );
}
