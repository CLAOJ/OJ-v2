'use client';

import { cn } from '@/lib/utils';

interface DiffLineProps {
    line: number;
    content: string;
    type: string;
    side: 'left' | 'right';
}

export function DiffLine({ line, content, type, side }: DiffLineProps) {
    if (type === 'empty') {
        return (
            <div className="p-1 bg-muted/30 min-h-[1.5rem]">
                <span className="inline-block w-12 text-muted-foreground text-xs select-none">
                    &nbsp;
                </span>
            </div>
        );
    }

    const bgColor = type === 'delete'
        ? 'bg-red-500/10'
        : type === 'add'
            ? 'bg-emerald-500/10'
            : 'bg-transparent';

    const textColor = type === 'delete'
        ? 'text-red-500'
        : type === 'add'
            ? 'text-emerald-500'
            : 'text-foreground';

    const prefix = type === 'delete' ? '-' : type === 'add' ? '+' : ' ';

    return (
        <div className={cn("flex p-1 hover:bg-muted/50", bgColor)}>
            <span className={cn(
                "inline-block w-12 text-muted-foreground text-xs select-none shrink-0",
                textColor
            )}>
                {line > 0 ? line : ''}
            </span>
            <span className={cn("text-xs whitespace-pre break-all", textColor)}>
                {content || '\u00A0'}
            </span>
        </div>
    );
}
