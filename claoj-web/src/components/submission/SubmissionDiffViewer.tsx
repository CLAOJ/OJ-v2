'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import {
    Diff,
    ArrowRight,
    GitCompare,
    Plus,
    Minus,
    FileCode,
    User,
    Calendar,
    CheckCircle2,
    XCircle,
    AlertCircle
} from 'lucide-react';
import { cn, getStatusColor, getStatusVariant } from '@/lib/utils';
import Link from 'next/link';
import { useAuth } from '@/components/providers/AuthProvider';

interface SubmissionInfo {
    id: number;
    problem: string;
    user: string;
    date: string;
    language: string;
    result: string | null;
    points: number | null;
}

interface DiffLine {
    type: 'add' | 'delete' | 'context';
    line: number;
    content: string;
}

interface SubmissionDiffData {
    submission1: SubmissionInfo;
    submission2: SubmissionInfo;
    unified_diff: string;
    diff_lines: DiffLine[];
    stats: {
        additions: number;
        deletions: number;
    };
}

interface SubmissionDiffViewerProps {
    submission1Id: number;
    submission2Id: number;
    onClose?: () => void;
}

export default function SubmissionDiffViewer({
    submission1Id,
    submission2Id,
    onClose
}: SubmissionDiffViewerProps) {
    const t = useTranslations('Submissions');
    const { user } = useAuth();
    const [viewMode, setViewMode] = useState<'side-by-side' | 'unified'>('side-by-side');

    const { data, isLoading, error } = useQuery({
        queryKey: ['submission-diff', submission1Id, submission2Id],
        queryFn: async () => {
            const res = await api.get<SubmissionDiffData>(
                `/submissions/${submission1Id}/diff/${submission2Id}`
            );
            return res.data;
        },
    });

    if (isLoading) {
        return (
            <div className="space-y-4 p-4">
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-64 w-full" />
                <Skeleton className="h-64 w-full" />
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-8 text-center space-y-4">
                <AlertCircle size={48} className="mx-auto text-red-500" />
                <h3 className="text-lg font-bold">Failed to load diff</h3>
                <p className="text-muted-foreground">
                    {(error as Error).message || 'Unable to compare submissions'}
                </p>
            </div>
        );
    }

    if (!data) {
        return null;
    }

    return (
        <div className="space-y-6">
            {/* Header with submission info */}
            <div className="flex flex-col lg:flex-row gap-4 items-center justify-between bg-card border rounded-2xl p-6">
                <div className="flex items-center gap-4">
                    <GitCompare size={24} className="text-primary" />
                    <h2 className="text-xl font-bold">Submission Comparison</h2>
                </div>

                <div className="flex items-center gap-4">
                    <span className="text-sm text-muted-foreground">View:</span>
                    <div className="flex gap-2">
                        <button
                            onClick={() => setViewMode('side-by-side')}
                            className={cn(
                                "px-4 py-2 rounded-lg text-sm font-bold transition-colors",
                                viewMode === 'side-by-side'
                                    ? "bg-primary text-primary-foreground"
                                    : "bg-muted text-muted-foreground hover:bg-primary/10"
                            )}
                        >
                            Side by Side
                        </button>
                        <button
                            onClick={() => setViewMode('unified')}
                            className={cn(
                                "px-4 py-2 rounded-lg text-sm font-bold transition-colors",
                                viewMode === 'unified'
                                    ? "bg-primary text-primary-foreground"
                                    : "bg-muted text-muted-foreground hover:bg-primary/10"
                            )}
                        >
                            Unified
                        </button>
                    </div>
                    {onClose && (
                        <button
                            onClick={onClose}
                            className="p-2 hover:bg-muted rounded-lg transition-colors"
                            aria-label="Close diff viewer"
                        >
                            <XCircle size={20} />
                        </button>
                    )}
                </div>
            </div>

            {/* Submission info cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Submission 1 */}
                <div className="bg-card border rounded-2xl p-6 space-y-3">
                    <div className="flex items-center justify-between">
                        <h3 className="text-sm font-black uppercase tracking-widest text-muted-foreground">
                            Submission #{data.submission1.id}
                        </h3>
                        <Badge variant={getStatusVariant(data.submission1.result || '')}>
                            {data.submission1.result || 'Unknown'}
                        </Badge>
                    </div>
                    <div className="space-y-2">
                        <Link
                            href={`/problems/${data.submission1.problem}`}
                            className="flex items-center gap-2 text-sm hover:text-primary transition-colors"
                        >
                            <FileCode size={16} />
                            <span className="font-bold">{data.submission1.problem}</span>
                        </Link>
                        <Link
                            href={`/user/${data.submission1.user}`}
                            className="flex items-center gap-2 text-sm hover:text-primary transition-colors"
                        >
                            <User size={16} />
                            <span className="font-medium">@{data.submission1.user}</span>
                        </Link>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <Calendar size={16} />
                            {new Date(data.submission1.date).toLocaleString()}
                        </div>
                        <div className="text-sm font-mono text-muted-foreground">
                            {data.submission1.language}
                        </div>
                    </div>
                </div>

                {/* Submission 2 */}
                <div className="bg-card border rounded-2xl p-6 space-y-3">
                    <div className="flex items-center justify-between">
                        <h3 className="text-sm font-black uppercase tracking-widest text-muted-foreground">
                            Submission #{data.submission2.id}
                        </h3>
                        <Badge variant={getStatusVariant(data.submission2.result || '')}>
                            {data.submission2.result || 'Unknown'}
                        </Badge>
                    </div>
                    <div className="space-y-2">
                        <Link
                            href={`/problems/${data.submission2.problem}`}
                            className="flex items-center gap-2 text-sm hover:text-primary transition-colors"
                        >
                            <FileCode size={16} />
                            <span className="font-bold">{data.submission2.problem}</span>
                        </Link>
                        <Link
                            href={`/user/${data.submission2.user}`}
                            className="flex items-center gap-2 text-sm hover:text-primary transition-colors"
                        >
                            <User size={16} />
                            <span className="font-medium">@{data.submission2.user}</span>
                        </Link>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <Calendar size={16} />
                            {new Date(data.submission2.date).toLocaleString()}
                        </div>
                        <div className="text-sm font-mono text-muted-foreground">
                            {data.submission2.language}
                        </div>
                    </div>
                </div>
            </div>

            {/* Diff stats */}
            <div className="flex gap-4 items-center">
                <div className="flex items-center gap-2 px-4 py-2 rounded-xl bg-red-500/10 border border-red-500/20">
                    <Minus size={16} className="text-red-500" />
                    <span className="text-sm font-bold text-red-500">
                        {data.stats.deletions} deletions
                    </span>
                </div>
                <div className="flex items-center gap-2 px-4 py-2 rounded-xl bg-emerald-500/10 border border-emerald-500/20">
                    <Plus size={16} className="text-emerald-500" />
                    <span className="text-sm font-bold text-emerald-500">
                        {data.stats.additions} additions
                    </span>
                </div>
            </div>

            {/* Diff viewer */}
            {viewMode === 'side-by-side' ? (
                <SideBySideDiff diffLines={data.diff_lines} />
            ) : (
                <UnifiedDiff diff={data.unified_diff} />
            )}
        </div>
    );
}

interface SideBySideDiffProps {
    diffLines: DiffLine[];
}

function SideBySideDiff({ diffLines }: SideBySideDiffProps) {
    // Group diff lines into pairs for side-by-side view
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
                    Removed
                </div>
                <div className="p-3 bg-emerald-500/10 text-center text-sm font-bold text-emerald-500">
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

interface DiffLineProps {
    line: number;
    content: string;
    type: string;
    side: 'left' | 'right';
}

function DiffLine({ line, content, type, side }: DiffLineProps) {
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

interface UnifiedDiffProps {
    diff: string;
}

function UnifiedDiff({ diff }: UnifiedDiffProps) {
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
