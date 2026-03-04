'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { useAuth } from '@/components/providers/AuthProvider';
import SubmissionSource from '@/components/submission/SubmissionSource';
import TestCaseResults from '@/components/submission/TestCaseResults';
import { use, useState, useEffect } from 'react';
import Link from 'next/link';
import { motion } from 'framer-motion';
import {
    ChevronRight,
    Clock,
    HardDrive,
    Hash,
    User,
    Calendar,
    Cpu,
    FileCode,
    AlertCircle,
    CheckCircle2,
    XCircle,
    Timer,
    MemoryStick,
    Award,
    RefreshCw,
    Loader2,
    ArrowLeft,
    ExternalLink
} from 'lucide-react';
import { cn, getStatusColor, getStatusVariant } from '@/lib/utils';
import dayjs from 'dayjs';
import { useWebSocketContext } from '@/contexts/WebSocketContext';

interface TestCase {
    case: number;
    status: string;
    time: number | null;
    memory: number | null;
    points: number | null;
    total: number | null;
    feedback: string;
}

interface SubmissionDetail {
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
    source: string;
    case_points: number;
    case_total: number;
    test_cases: TestCase[];
    error?: string;
}

const STATUS_INFO: Record<string, { label: string; color: string; icon: React.ReactNode }> = {
    'AC': {
        label: 'Accepted',
        color: 'text-emerald-500',
        icon: <CheckCircle2 size={20} />
    },
    'WA': {
        label: 'Wrong Answer',
        color: 'text-red-500',
        icon: <XCircle size={20} />
    },
    'TLE': {
        label: 'Time Limit Exceeded',
        color: 'text-amber-500',
        icon: <Timer size={20} />
    },
    'MLE': {
        label: 'Memory Limit Exceeded',
        color: 'text-amber-500',
        icon: <MemoryStick size={20} />
    },
    'OLE': {
        label: 'Output Limit Exceeded',
        color: 'text-amber-500',
        icon: <AlertCircle size={20} />
    },
    'RTE': {
        label: 'Runtime Error',
        color: 'text-purple-500',
        icon: <AlertCircle size={20} />
    },
    'CE': {
        label: 'Compilation Error',
        color: 'text-blue-500',
        icon: <FileCode size={20} />
    },
    'IE': {
        label: 'Internal Error',
        color: 'text-zinc-500',
        icon: <AlertCircle size={20} />
    },
    'QU': {
        label: 'Queued',
        color: 'text-zinc-400',
        icon: <Clock size={20} />
    },
    'P': {
        label: 'Processing',
        color: 'text-blue-400',
        icon: <Loader2 size={20} className="animate-spin" />
    },
    'G': {
        label: 'Grading',
        color: 'text-blue-400',
        icon: <Loader2 size={20} className="animate-spin" />
    },
    'D': {
        label: 'Completed',
        color: 'text-emerald-500',
        icon: <CheckCircle2 size={20} />
    },
};

export default function SubmissionPage({ params }: { params: Promise<{ id: string }> }) {
    const resolvedParams = use(params);
    const id = resolvedParams.id;
    const t = useTranslations('Submissions');
    const { user } = useAuth();
    const queryClient = useQueryClient();
    const { subscribe, unsubscribe } = useWebSocketContext();
    const [isRejudging, setIsRejudging] = useState(false);
    const [rejudgeError, setRejudgeError] = useState<string | null>(null);
    const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

    const { data: sub, isLoading } = useQuery({
        queryKey: ['submission', id],
        queryFn: async () => {
            const res = await api.get<SubmissionDetail>(`/submission/${id}`);
            return res.data;
        },
        // Poll for updates when submission is being graded
        refetchInterval: (query) => {
            const submission = query.state.data;
            if (!submission) return 1000;
            // Poll every 2 seconds if processing/grading
            if (['QU', 'P', 'G'].includes(submission.status)) {
                return 2000;
            }
            return false;
        }
    });

    // Subscribe to submission-specific WebSocket channel for live updates
    useEffect(() => {
        const submissionChannel = `sub_${id}`;
        subscribe(submissionChannel);

        return () => {
            unsubscribe(submissionChannel);
        };
    }, [id, subscribe, unsubscribe]);

    // Track last update time
    useEffect(() => {
        if (sub) {
            setLastUpdated(new Date());
        }
    }, [sub]);

    const handleRejudge = async () => {
        if (!confirm('Are you sure you want to rejudge this submission?')) return;

        setIsRejudging(true);
        setRejudgeError(null);
        try {
            await api.post(`/admin/submission/${id}/rejudge`);
            // Refresh submission data
            queryClient.invalidateQueries({ queryKey: ['submission', id] });
        } catch (err: any) {
            setRejudgeError(err.response?.data?.error || 'Failed to rejudge submission');
        } finally {
            setIsRejudging(false);
        }
    };

    const isAdmin = user?.is_admin || user?.is_staff;
    const resultInfo = sub?.result ? STATUS_INFO[sub.result] : null;
    const statusInfo = sub?.status ? STATUS_INFO[sub.status] || { label: sub.status, color: 'text-zinc-500', icon: <AlertCircle size={20} /> } : null;

    if (isLoading) {
        return (
            <div className="max-w-7xl mx-auto space-y-8 p-4">
                <Skeleton className="h-32 w-full" />
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    <Skeleton className="h-[60vh] lg:col-span-2" />
                    <Skeleton className="h-[60vh]" />
                </div>
            </div>
        );
    }

    if (!sub) return (
        <div className="flex items-center justify-center min-h-[60vh]">
            <div className="text-center space-y-4">
                <AlertCircle size={48} className="mx-auto text-muted-foreground" />
                <h2 className="text-2xl font-bold">Submission not found</h2>
                <p className="text-muted-foreground">The submission you&apos;re looking for doesn&apos;t exist.</p>
                <Link
                    href="/submissions"
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:opacity-90 transition-opacity"
                >
                    <ArrowLeft size={18} />
                    Back to Submissions
                </Link>
            </div>
        </div>
    );

    return (
        <div className="max-w-7xl mx-auto space-y-8 animate-in fade-in duration-500 pb-20 p-4">
            {/* Breadcrumb */}
            <nav className="flex items-center gap-2 text-sm text-muted-foreground">
                <Link href="/submissions" className="hover:text-foreground transition-colors">
                    Submissions
                </Link>
                <ChevronRight size={14} />
                <span className="font-medium text-foreground">#{id}</span>
            </nav>

            {/* Header Card */}
            <motion.header
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                className="bg-card border rounded-3xl p-6 md:p-8 shadow-sm"
            >
                <div className="flex flex-col lg:flex-row justify-between gap-6">
                    {/* Left: Title & Info */}
                    <div className="space-y-4">
                        <div className="flex items-center gap-3 flex-wrap">
                            <h1 className="text-3xl md:text-4xl font-black tracking-tight">
                                Submission #{id}
                            </h1>
                            {sub.result && (
                                <Badge
                                    variant={getStatusVariant(sub.result)}
                                    className="text-sm px-4 py-1.5 font-black tracking-wider uppercase"
                                >
                                    <span className="flex items-center gap-1.5">
                                        {resultInfo?.icon}
                                        {sub.result}
                                    </span>
                                </Badge>
                            )}
                            {/* Live indicator for grading submissions */}
                            {['QU', 'P', 'G'].includes(sub.status || '') && (
                                <span className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 text-emerald-500 text-xs font-bold animate-pulse">
                                    <span className="w-2 h-2 rounded-full bg-emerald-500" />
                                    Live Updates
                                </span>
                            )}
                        </div>

                        <div className="flex flex-wrap gap-x-6 gap-y-2 text-sm">
                            <Link
                                href={`/user/${sub.user}`}
                                className="flex items-center gap-1.5 text-muted-foreground hover:text-primary transition-colors"
                            >
                                <User size={16} />
                                <span className="font-medium">@{sub.user}</span>
                            </Link>
                            <Link
                                href={`/problems/${sub.problem}`}
                                className="flex items-center gap-1.5 text-muted-foreground hover:text-primary transition-colors"
                            >
                                <Hash size={16} />
                                <span className="font-medium">{sub.problem}</span>
                                <ExternalLink size={12} />
                            </Link>
                            <span className="flex items-center gap-1.5 text-muted-foreground">
                                <Calendar size={16} />
                                {dayjs(sub.date).format('MMM D, YYYY HH:mm:ss')}
                            </span>
                            <span className="flex items-center gap-1.5 text-muted-foreground">
                                <FileCode size={16} />
                                <span className="font-mono uppercase">{sub.language}</span>
                            </span>
                        </div>

                        {sub.error && (
                            <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">
                                <div className="flex items-start gap-2">
                                    <AlertCircle size={16} className="mt-0.5 shrink-0" />
                                    <pre className="whitespace-pre-wrap font-mono text-xs">{sub.error}</pre>
                                </div>
                            </div>
                        )}

                        {rejudgeError && (
                            <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">
                                {rejudgeError}
                            </div>
                        )}
                    </div>

                    {/* Right: Stats */}
                    <div className="flex gap-3 flex-wrap">
                        <div className="px-6 py-4 rounded-2xl bg-primary/5 border border-primary/10 text-center min-w-[100px]">
                            <span className="text-[10px] uppercase tracking-wider text-muted-foreground block mb-1">Score</span>
                            <span className="text-2xl font-black text-primary">
                                {sub.points !== null ? sub.points.toFixed(0) : '-'}
                                {sub.case_total > 0 && (
                                    <span className="text-sm text-muted-foreground font-medium">/{sub.case_total}</span>
                                )}
                            </span>
                        </div>
                        <div className="px-6 py-4 rounded-2xl bg-card border text-center min-w-[100px]">
                            <span className="text-[10px] uppercase tracking-wider text-muted-foreground block mb-1">Time</span>
                            <span className="text-2xl font-black">
                                {sub.time !== null ? `${sub.time.toFixed(2)}s` : '-'}
                            </span>
                        </div>
                        <div className="px-6 py-4 rounded-2xl bg-card border text-center min-w-[100px]">
                            <span className="text-[10px] uppercase tracking-wider text-muted-foreground block mb-1">Memory</span>
                            <span className="text-2xl font-black">
                                {sub.memory !== null ? `${sub.memory.toFixed(1)}MB` : '-'}
                            </span>
                        </div>
                    </div>
                </div>

                {/* Admin Actions */}
                {isAdmin && (
                    <div className="mt-6 pt-6 border-t flex items-center gap-3">
                        <span className="text-sm text-muted-foreground font-medium">Admin Actions:</span>
                        <button
                            onClick={handleRejudge}
                            disabled={isRejudging}
                            className="flex items-center gap-2 px-4 py-2 rounded-xl bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 font-bold text-sm transition-colors disabled:opacity-50"
                        >
                            {isRejudging ? (
                                <Loader2 size={16} className="animate-spin" />
                            ) : (
                                <RefreshCw size={16} />
                            )}
                            Rejudge
                        </button>
                    </div>
                )}
            </motion.header>

            {/* Main Content */}
            <div className="grid grid-cols-1 xl:grid-cols-5 gap-8">
                {/* Source Code */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.1 }}
                    className="xl:col-span-3 space-y-4"
                >
                    <div className="flex items-center justify-between">
                        <h2 className="text-xl font-bold flex items-center gap-2">
                            <FileCode size={20} className="text-primary" />
                            Source Code
                        </h2>
                    </div>
                    <SubmissionSource
                        source={sub.source}
                        language={sub.language}
                    />
                </motion.div>

                {/* Test Cases */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.2 }}
                    className="xl:col-span-2 space-y-4"
                >
                    <div className="flex items-center justify-between">
                        <h2 className="text-xl font-bold flex items-center gap-2">
                            <Cpu size={20} className="text-primary" />
                            Test Case Results
                        </h2>
                        {sub.test_cases.length > 0 && (
                            <span className="text-sm text-muted-foreground">
                                {sub.test_cases.filter(tc => tc.status === 'AC').length}/{sub.test_cases.length} passed
                            </span>
                        )}
                    </div>

                    {sub.test_cases.length > 0 ? (
                        <TestCaseResults testCases={sub.test_cases} />
                    ) : (
                        <div className="p-8 rounded-2xl border bg-card text-center">
                            <Cpu size={40} className="mx-auto text-muted-foreground mb-4" />
                            <p className="text-muted-foreground font-medium">No test case results available</p>
                            <p className="text-sm text-muted-foreground/70 mt-1">
                                {sub.status === 'CE'
                                    ? 'Compilation failed before test cases could run'
                                    : 'Test cases will appear once judging is complete'}
                            </p>
                        </div>
                    )}
                </motion.div>
            </div>
        </div>
    );
}
