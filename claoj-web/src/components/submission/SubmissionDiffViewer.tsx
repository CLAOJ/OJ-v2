'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Skeleton } from '@/components/ui/Skeleton';
import { GitCompare, AlertCircle, XCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { SubmissionInfoCard } from './diff/SubmissionInfoCard';
import { SideBySideDiff } from './diff/SideBySideDiff';
import { UnifiedDiff } from './diff/UnifiedDiff';

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
                <SubmissionInfoCard
                    title={`Submission #${data.submission1.id}`}
                    submission={data.submission1}
                />
                <SubmissionInfoCard
                    title={`Submission #${data.submission2.id}`}
                    submission={data.submission2}
                />
            </div>

            {/* Diff stats */}
            <div className="flex gap-4 items-center">
                <div className="flex items-center gap-2 px-4 py-2 rounded-xl bg-red-500/10 border border-red-500/20">
                    <span className="text-sm font-bold text-red-500">
                        {data.stats.deletions} deletions
                    </span>
                </div>
                <div className="flex items-center gap-2 px-4 py-2 rounded-xl bg-emerald-500/10 border border-emerald-500/20">
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
