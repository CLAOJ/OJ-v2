'use client';

import { cn } from '@/lib/utils';
import { Plus, Minus } from 'lucide-react';
import { DiffLine } from './DiffLine';

interface DiffLineData {
    type: 'add' | 'delete' | 'context';
    line: number;
    content: string;
}

interface SideBySideDiffProps {
    diffLines: DiffLineData[];
}

export function SideBySideDiff({ diffLines }: SideBySideDiffProps) {
    const leftLines: { line: number; content: string; type: string }[] = [];
    const rightLines: { line: number; content: string; type: string }[] = [];

    let currentLeftLine = 1;
    let currentRightLine = 1;

    diffLines.forEach((line) => {
        if (line.type === 'delete') {
            leftLines.push({
                line: currentLeftLine++,
                content: line.content,
                type: 'delete'
            });
            rightLines.push({ line: 0, content: '', type: 'empty' });
        } else if (line.type === 'add') {
            leftLines.push({ line: 0, content: '', type: 'empty' });
            rightLines.push({
                line: currentRightLine++,
                content: line.content,
                type: 'add'
            });
        }
    });

    const maxLines = Math.max(leftLines.length, rightLines.length);

    return (
        <div className="bg-card border rounded-2xl overflow-hidden">
            <div className="grid grid-cols-2 border-b">
                <div className="p-3 bg-red-500/10 text-center text-sm font-bold text-red-500">
                    <Minus size={16} className="inline mr-1" />
                    Removed
                </div>
                <div className="p-3 bg-emerald-500/10 text-center text-sm font-bold text-emerald-500">
                    <Plus size={16} className="inline mr-1" />
                    Added
                </div>
            </div>
            <div className="max-h-[60vh] overflow-auto font-mono text-sm">
                {Array.from({ length: maxLines }).map((_, idx) => (
                    <div key={idx} className="grid grid-cols-2 border-b last:border-0">
                        <DiffLine
                            line={leftLines[idx]?.line || 0}
                            content={leftLines[idx]?.content || ''}
                            type={leftLines[idx]?.type || 'empty'}
                            side="left"
                        />
                        <DiffLine
                            line={rightLines[idx]?.line || 0}
                            content={rightLines[idx]?.content || ''}
                            type={rightLines[idx]?.type || 'empty'}
                            side="right"
                        />
                    </div>
                ))}
            </div>
        </div>
    );
}
