'use client';

import { cn } from '@/lib/utils';

interface UnifiedDiffProps {
    diff: string;
}

export function UnifiedDiff({ diff }: UnifiedDiffProps) {
    const lines = diff.split('\n');

    return (
        <div className="bg-card border rounded-2xl overflow-hidden">
            <div className="p-3 bg-muted/50 border-b">
                <span className="text-sm font-bold text-muted-foreground">Unified Diff</span>
            </div>
            <div className="max-h-[60vh] overflow-auto font-mono text-sm">
                {lines.map((line, idx) => {
                    let bgColor = 'bg-transparent';
                    let textColor = 'text-foreground';

                    if (line.startsWith('+') && !line.startsWith('+++')) {
                        bgColor = 'bg-emerald-500/10';
                        textColor = 'text-emerald-500';
                    } else if (line.startsWith('-') && !line.startsWith('---')) {
                        bgColor = 'bg-red-500/10';
                        textColor = 'text-red-500';
                    } else if (line.startsWith('@@')) {
                        bgColor = 'bg-blue-500/10';
                        textColor = 'text-blue-500';
                    } else if (line.startsWith('---') || line.startsWith('+++')) {
                        bgColor = 'bg-muted/50';
                        textColor = 'text-muted-foreground';
                    }

                    return (
                        <div
                            key={idx}
                            className={cn("px-4 py-1 hover:bg-muted/30 whitespace-pre", bgColor)}
                        >
                            <span className={cn("text-xs", textColor)}>{line}</span>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
