'use client';

import { ChevronDown, ChevronUp } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { SortOrder } from '@/lib/listSort';

interface SortableHeaderProps {
    label: React.ReactNode;
    active: boolean;
    order: SortOrder;
    onClick: () => void;
    className?: string;
}

/** Table header cell that toggles server-side sorting on click. */
export function SortableHeader({ label, active, order, onClick, className }: SortableHeaderProps) {
    return (
        <th
            aria-sort={active ? (order === 'asc' ? 'ascending' : 'descending') : 'none'}
            className={cn('px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground', className)}
        >
            <button
                type="button"
                onClick={onClick}
                className={cn(
                    'inline-flex items-center gap-1 uppercase transition-colors hover:text-foreground',
                    active && 'text-foreground'
                )}
            >
                {label}
                {active && (order === 'asc' ? <ChevronUp size={12} /> : <ChevronDown size={12} />)}
            </button>
        </th>
    );
}