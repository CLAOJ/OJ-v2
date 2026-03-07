'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import Link from 'next/link';
import {
    Clock,
    HardDrive,
    FileCode,
    CheckCircle2,
    XCircle,
    AlertCircle,
    Timer,
    MemoryStick,
    ArrowUpRight,
    ExternalLink,
    User,
    Calendar,
    Hash,
    Cpu
} from 'lucide-react';
import { cn, getStatusColor, getStatusVariant } from '@/lib/utils';
import { useState } from 'react';

interface Submission {
    id: number;
    problem: string;
    problem_name?: string;
    user: string;
    date: string;
    language: string;
    language_name?: string;
    status: string;
    result: string | null;
    points: number | null;
    time: number | null;
    memory: number | null;
    case_points: number;
    case_total: number;
}

interface SingleSubmissionWidgetProps {
    submissionId: number;
    compact?: boolean;
    showProblem?: boolean;
    showUser?: boolean;
    onClick?: () => void;
}

const STATUS_INFO: Record<string, { label: string; color: string; icon: React.ReactNode }> = {
    'AC': { label: 'Accepted', color: 'text-emerald-500', icon: <CheckCircle2 size={16} /> },
    'WA': { label: 'Wrong Answer', color: 'text-red-500', icon: <XCircle size={16} /> },
    'TLE': { label: 'Time Limit Exceeded', color: 'text-amber-500', icon: <Timer size={16} /> },
    'MLE': { label: 'Memory Limit Exceeded', color: 'text-amber-500', icon: <MemoryStick size={16} /> },
    'OLE': { label: 'Output Limit Exceeded', color: 'text-amber-500', icon: <AlertCircle size={16} /> },
    'RTE': { label: 'Runtime Error', color: 'text-purple-500', icon: <AlertCircle size={16} /> },
    'CE': { label: 'Compilation Error', color: 'text-blue-500', icon: <FileCode size={16} /> },
    'IE': { label: 'Internal Error', color: 'text-zinc-500', icon: <AlertCircle size={16} /> },
    'QU': { label: 'Queued', color: 'text-zinc-400', icon: <Clock size={16} /> },
    'P': { label: 'Processing', color: 'text-blue-400', icon: <AlertCircle size={16} /> },
    'G': { label: 'Grading', color: 'text-blue-400', icon: <AlertCircle size={16} /> },
    'D': { label: 'Completed', color: 'text-emerald-500', icon: <CheckCircle2 size={16} /> },
};

export default function SingleSubmissionWidget({
    submissionId,
    compact = false,
    showProblem = true,
    showUser = true,
    onClick
}: SingleSubmissionWidgetProps) {
    const t = useTranslations('Submissions');

    const { data: sub, isLoading } = useQuery({
        queryKey: ['submission-widget', submissionId],
        queryFn: async () => {
            const res = await api.get<Submission>(`/submission/${submissionId}`);
            return res.data;
        },
        // Poll for updates when submission is being graded
        refetchInterval: (query) => {
            const submission = query.state.data;
            if (!submission) return 1000;
            if (['QU', 'P', 'G'].includes(submission.status)) {
                return 2000;
            }
            return false;
        }
    });

    if (isLoading) {
        return (
            <div className={cn(
                "bg-card border rounded-2xl overflow-hidden",
                compact ? "p-3" : "p-6"
            )}>
                <Skeleton className={cn("w-full", compact ? "h-20" : "h-32")} />
            </div>
        );
    }

    if (!sub) {
        return (
            <div className={cn(
                "bg-card border rounded-2xl overflow-hidden p-6 text-center"
            )}>
                <AlertCircle size={32} className="mx-auto text-muted-foreground mb-2" />
                <p className="text-sm font-medium text-muted-foreground">Submission not found</p>
            </div>
        );
    }

    const resultInfo = sub.result ? STATUS_INFO[sub.result] : null;
    const isGrading = ['QU', 'P', 'G'].includes(sub.status);

    if (compact) {
        return (
            <Link
                href={`/submissions/${sub.id}`}
                className={cn(
                    "block bg-card border rounded-xl p-3 hover:border-primary/50 transition-all group outline-none",
                    isGrading && "animate-pulse"
                )}
                onClick={onClick}
            >
                <div className="flex items-center justify-between gap-3">
                    <div className="flex items-center gap-2 min-w-0">
                        <Badge variant={getStatusVariant(sub.result || '')} className="shrink-0">
                            {resultInfo?.icon}
                            <span className="ml-1">{sub.result}</span>
                        </Badge>
                        {showProblem && (
                            <span className="text-sm font-bold truncate group-hover:text-primary transition-colors">
                                {sub.problem}
                            </span>
                        )}
                    </div>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground shrink-0">
                        <span className="font-medium">{sub.points !== null ? sub.points.toFixed(0) : '-'}</span>
                        <span className="w-1 h-1 rounded-full bg-muted-foreground/30" />
                        <span className="font-mono">{sub.time !== null ? `${sub.time.toFixed(2)}s` : '-'}</span>
                    </div>
                </div>
            </Link>
        );
    }

    return (
        <Link
            href={`/submissions/${sub.id}`}
            className={cn(
                "block bg-card border rounded-2xl p-6 hover:border-primary/50 hover:shadow-md transition-all group outline-none",
                isGrading && "animate-pulse border-primary/30"
            )}
            onClick={onClick}
        >
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                    <h3 className="text-lg font-black">Submission #{sub.id}</h3>
                    {sub.result && (
                        <Badge variant={getStatusVariant(sub.result)} className="font-bold">
                            {resultInfo?.icon}
                            <span className="ml-1">{resultInfo?.label}</span>
                        </Badge>
                    )}
                    {isGrading && (
                        <span className="flex items-center gap-1 text-xs font-bold text-primary">
                            <span className="w-2 h-2 rounded-full bg-primary animate-pulse" />
                            Live
                        </span>
                    )}
                </div>
                <ExternalLink size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
            </div>

            {/* Meta info */}
            {(showProblem || showUser) && (
                <div className="flex flex-wrap gap-4 mb-4 text-sm">
                    {showProblem && (
                        <div className="flex items-center gap-2 text-muted-foreground hover:text-primary transition-colors">
                            <Hash size={14} />
                            <span className="font-medium">{sub.problem}</span>
                        </div>
                    )}
                    {showUser && (
                        <div className="flex items-center gap-2 text-muted-foreground hover:text-primary transition-colors">
                            <User size={14} />
                            <span className="font-medium">@{sub.user}</span>
                        </div>
                    )}
                    <div className="flex items-center gap-2 text-muted-foreground">
                        <Calendar size={14} />
                        <span className="font-medium">
                            {new Date(sub.date).toLocaleDateString()} {new Date(sub.date).toLocaleTimeString()}
                        </span>
                    </div>
                </div>
            )}

            {/* Stats grid */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                <StatBox
                    icon={<Cpu size={18} />}
                    label="Score"
                    value={sub.points !== null ? `${sub.points.toFixed(0)}/${sub.case_total}` : '-'}
                    highlight
                />
                <StatBox
                    icon={<Timer size={18} />}
                    label="Time"
                    value={sub.time !== null ? `${sub.time.toFixed(2)}s` : '-'}
                />
                <StatBox
                    icon={<MemoryStick size={18} />}
                    label="Memory"
                    value={sub.memory !== null ? `${sub.memory.toFixed(1)}MB` : '-'}
                />
                <StatBox
                    icon={<FileCode size={18} />}
                    label="Language"
                    value={sub.language.toUpperCase()}
                    mono
                />
            </div>

            {/* Test case progress */}
            {sub.case_total > 0 && (
                <div className="mt-4 pt-4 border-t">
                    <div className="flex items-center justify-between text-sm mb-2">
                        <span className="text-muted-foreground font-medium">Test Cases</span>
                        <span className="font-bold">
                            {sub.case_points}/{sub.case_total} passed
                        </span>
                    </div>
                    <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                        <div
                            className="h-full bg-primary transition-all duration-500"
                            style={{ width: `${(sub.case_points / sub.case_total) * 100}%` }}
                            role="progressbar"
                            aria-valuenow={(sub.case_points / sub.case_total) * 100}
                            aria-valuemin={0}
                            aria-valuemax={100}
                            aria-label="Test cases passed"
                        />
                    </div>
                </div>
            )}
        </Link>
    );
}

interface StatBoxProps {
    icon: React.ReactNode;
    label: string;
    value: string;
    highlight?: boolean;
    mono?: boolean;
}

function StatBox({ icon, label, value, highlight = false, mono = false }: StatBoxProps) {
    return (
        <div className={cn(
            "p-3 rounded-xl border text-center",
            highlight ? "bg-primary/5 border-primary/20" : "bg-muted/30 border-muted-foreground/10"
        )}>
            <div className={cn(
                "flex items-center justify-center mb-1",
                highlight ? "text-primary" : "text-muted-foreground"
            )}>
                {icon}
            </div>
            <div className={cn(
                "text-xs text-muted-foreground mb-0.5",
                highlight && "text-primary/70"
            )}>
                {label}
            </div>
            <div className={cn(
                "text-lg font-black",
                highlight ? "text-primary" : "text-foreground",
                mono && "font-mono"
            )}>
                {value}
            </div>
        </div>
    );
}
