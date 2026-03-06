'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api, { contestStatsApi } from '@/lib/api';
import { ContestStats, ContestDetail } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { use, useMemo } from 'react';
import {
    Trophy,
    Target,
    Clock,
    BarChart3,
    TrendingUp,
    CheckCircle2,
    XCircle,
    Activity,
    Award,
    Users,
    Timer,
    ArrowUpRight,
    ArrowDownRight
} from 'lucide-react';
import { cn, getRankBadgeColor } from '@/lib/utils';
import dayjs from 'dayjs';
import { Link } from '@/navigation';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Progress } from '@/components/ui/Progress';

export default function ContestStatsPage({ params }: { params: Promise<{ key: string }> }) {
    const { key } = use(params);
    const t = useTranslations('Contest');

    const { data: stats, isLoading } = useQuery({
        queryKey: ['contest-stats', key],
        queryFn: async () => {
            const res = await contestStatsApi.getContestStats(key);
            return res.data;
        }
    });

    if (isLoading) {
        return (
            <div className="max-w-7xl mx-auto p-8 space-y-8">
                <Skeleton className="h-32 w-full rounded-[2rem]" />
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    <Skeleton className="h-40 rounded-[2rem]" />
                    <Skeleton className="h-40 rounded-[2rem]" />
                    <Skeleton className="h-40 rounded-[2rem]" />
                </div>
                <Skeleton className="h-96 w-full rounded-[2rem]" />
            </div>
        );
    }

    if (!stats) {
        return (
            <div className="max-w-2xl mx-auto p-8 text-center">
                <div className="p-12 rounded-[3rem] border bg-card">
                    <Target size={64} className="mx-auto mb-4 text-muted-foreground opacity-50" />
                    <h2 className="text-2xl font-black mb-2">No Statistics Available</h2>
                    <p className="text-muted-foreground">You haven&apos;t participated in this contest yet.</p>
                </div>
            </div>
        );
    }

    const solveRate = stats.total_problems > 0
        ? (stats.solved_count / stats.total_problems) * 100
        : 0;

    const isAboveAverage = stats.score >= stats.average_score;

    return (
        <div className="max-w-7xl mx-auto p-4 md:p-8 space-y-8 animate-in fade-in duration-700">
            {/* Header */}
            <div className="relative overflow-hidden rounded-[2rem] border bg-card shadow-lg">
                <div className="absolute top-0 right-0 p-32 opacity-5 pointer-events-none">
                    <BarChart3 size={320} className="text-primary" />
                </div>
                <div className="p-8 md:p-10 space-y-4 relative">
                    <div className="flex items-center gap-3">
                        <Link
                            href={`/contests/${key}`}
                            className="text-xs font-black uppercase tracking-widest text-primary hover:underline flex items-center gap-1"
                        >
                            <ArrowUpRight size={14} />
                            Back to Contest
                        </Link>
                    </div>
                    <h1 className="text-3xl md:text-4xl font-black tracking-tight">{stats.contest_name}</h1>
                    <div className="flex flex-wrap items-center gap-3">
                        <Badge variant="outline" className="px-4 py-1.5 rounded-full border-primary/20 bg-primary/5 text-primary text-[10px] font-black uppercase tracking-widest">
                            {t('statistics')}
                        </Badge>
                        <span className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                            <Trophy size={14} className="text-amber-500" />
                            Rank #{stats.rank} of {stats.total_participants}
                        </span>
                    </div>
                </div>
            </div>

            {/* Summary Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                {/* Rank Card */}
                <Card className="rounded-[2rem] border-2 shadow-lg">
                    <CardHeader className="pb-3">
                        <CardTitle className="text-xs font-black uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                            <Award size={16} />
                            {t('rank')}
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-3">
                            <div className="text-4xl font-black tracking-tighter">#{stats.rank}</div>
                            <div className="flex items-center gap-2 text-sm">
                                <Users size={14} className="text-muted-foreground" />
                                <span className="font-medium text-muted-foreground">
                                    {stats.total_participants} participants
                                </span>
                            </div>
                            <div className={cn(
                                "flex items-center gap-1 text-xs font-black px-2 py-1 rounded-lg inline-flex",
                                stats.percentile >= 90 ? "bg-emerald-500/10 text-emerald-500" :
                                stats.percentile >= 50 ? "bg-blue-500/10 text-blue-500" :
                                "bg-muted text-muted-foreground"
                            )}>
                                {stats.percentile >= 50 ? <TrendingUp size={12} /> : <ArrowDownRight size={12} />}
                                Top {stats.percentile.toFixed(1)}%
                            </div>
                        </div>
                    </CardContent>
                </Card>

                {/* Score Card */}
                <Card className="rounded-[2rem] border-2 shadow-lg">
                    <CardHeader className="pb-3">
                        <CardTitle className="text-xs font-black uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                            <Activity size={16} />
                            {t('score')}
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-3">
                            <div className="text-4xl font-black tracking-tighter">
                                {Math.round(stats.score)}
                                <span className="text-lg text-muted-foreground font-medium"> pts</span>
                            </div>
                            <div className="flex items-center gap-2 text-sm">
                                <span className={cn(
                                    "font-black px-2 py-0.5 rounded-md",
                                    isAboveAverage ? "bg-emerald-500/10 text-emerald-500" : "bg-rose-500/10 text-rose-500"
                                )}>
                                    {isAboveAverage ? '+' : ''}{Math.round(stats.score - stats.average_score)}
                                </span>
                                <span className="font-medium text-muted-foreground">
                                    vs avg ({Math.round(stats.average_score)})
                                </span>
                            </div>
                        </div>
                    </CardContent>
                </Card>

                {/* Solved Problems Card */}
                <Card className="rounded-[2rem] border-2 shadow-lg">
                    <CardHeader className="pb-3">
                        <CardTitle className="text-xs font-black uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                            <CheckCircle2 size={16} />
                            {t('solved')}
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-3">
                            <div className="text-4xl font-black tracking-tighter">
                                {stats.solved_count}
                                <span className="text-lg text-muted-foreground font-medium">/{stats.total_problems}</span>
                            </div>
                            <div className="space-y-1.5">
                                <div className="flex justify-between text-xs">
                                    <span className="font-medium text-muted-foreground">Solve Rate</span>
                                    <span className="font-black">{solveRate.toFixed(0)}%</span>
                                </div>
                                <Progress value={solveRate} className="h-2" />
                            </div>
                        </div>
                    </CardContent>
                </Card>

                {/* Time Stats Card */}
                <Card className="rounded-[2rem] border-2 shadow-lg">
                    <CardHeader className="pb-3">
                        <CardTitle className="text-xs font-black uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                            <Timer size={16} />
                            Time Stats
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-3">
                            <div className="text-4xl font-black tracking-tighter">
                                {Math.floor(stats.average_solve_time / 60)}
                                <span className="text-lg text-muted-foreground font-medium"> min</span>
                            </div>
                            <div className="flex items-center gap-2 text-sm">
                                <Clock size={14} className="text-muted-foreground" />
                                <span className="font-medium text-muted-foreground">
                                    avg solve time
                                </span>
                            </div>
                            <div className="text-xs font-medium text-muted-foreground">
                                {stats.total_attempts} total submissions
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* Problem Breakdown */}
            <Card className="rounded-[2rem] border-2 shadow-lg overflow-hidden">
                <CardHeader className="pb-4 border-b">
                    <CardTitle className="text-lg font-black flex items-center gap-3">
                        <Target size={20} className="text-primary" />
                        Problem-by-Problem Breakdown
                    </CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                    <div className="divide-y">
                        {stats.problems.map((problem, idx) => {
                            const avgTime = stats.average_solve_times_by_problem[problem.problem_code];
                            const isFasterThanAvg = avgTime && problem.solve_time && problem.solve_time < avgTime;

                            return (
                                <div
                                    key={problem.problem_code}
                                    className={cn(
                                        "p-6 transition-colors",
                                        problem.is_solved ? "bg-emerald-500/5" : "bg-muted/20"
                                    )}
                                >
                                    <div className="flex items-center gap-6 flex-wrap">
                                        {/* Problem Label */}
                                        <div className="w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center flex-shrink-0">
                                            <span className="text-2xl font-black text-primary">
                                                {problem.problem_label}
                                            </span>
                                        </div>

                                        {/* Problem Info */}
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-3 mb-2">
                                                <Link
                                                    href={`/problems/${problem.problem_code}`}
                                                    className="text-lg font-black hover:text-primary transition-colors truncate"
                                                >
                                                    {problem.problem_code}
                                                </Link>
                                                {problem.is_solved ? (
                                                    <Badge className="bg-emerald-500 text-white">
                                                        <CheckCircle2 size={12} className="mr-1" />
                                                        Solved
                                                    </Badge>
                                                ) : (
                                                    <Badge variant="outline" className="text-muted-foreground">
                                                        <XCircle size={12} className="mr-1" />
                                                        Not Solved
                                                    </Badge>
                                                )}
                                            </div>
                                            <div className="flex items-center gap-4 text-sm text-muted-foreground">
                                                <span className="font-medium">
                                                    {problem.max_score}/{problem.points} points
                                                </span>
                                                <span className="font-medium">
                                                    {problem.attempt_count} {problem.attempt_count === 1 ? 'attempt' : 'attempts'}
                                                </span>
                                            </div>
                                        </div>

                                        {/* Time Stats */}
                                        <div className="text-right space-y-2">
                                            {problem.solve_time ? (
                                                <>
                                                    <div className="text-sm font-black text-emerald-500 flex items-center gap-1 justify-end">
                                                        <Clock size={14} />
                                                        {formatTime(problem.solve_time)}
                                                    </div>
                                                    {avgTime && (
                                                        <div className={cn(
                                                            "text-xs font-medium flex items-center gap-1 justify-end",
                                                            isFasterThanAvg ? "text-emerald-500" : "text-amber-500"
                                                        )}>
                                                            {isFasterThanAvg ? (
                                                                <><ArrowDownRight size={12} /> Faster than avg</>
                                                            ) : (
                                                                <><ArrowUpRight size={12} /> Slower than avg</>
                                                            )}
                                                        </div>
                                                    )}
                                                </>
                                            ) : (
                                                <div className="text-sm font-medium text-muted-foreground">
                                                    Not solved
                                                </div>
                                            )}
                                        </div>

                                        {/* First Submit Time */}
                                        {problem.first_submit_time && (
                                            <div className="text-xs text-muted-foreground min-w-[120px] text-right">
                                                <div className="font-medium">First submit</div>
                                                <div className="font-mono">
                                                    {dayjs(problem.first_submit_time).format('HH:mm:ss')}
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                </CardContent>
            </Card>

            {/* Performance Comparison Chart */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Attempts Distribution */}
                <Card className="rounded-[2rem] border-2 shadow-lg">
                    <CardHeader className="pb-4 border-b">
                        <CardTitle className="text-lg font-black flex items-center gap-3">
                            <Activity size={20} className="text-primary" />
                            Submission Distribution
                        </CardTitle>
                    </CardHeader>
                    <CardContent className="p-6 space-y-4">
                        {stats.problems.map((problem) => {
                            const maxAttempts = Math.max(...stats.problems.map(p => p.attempt_count), 1);
                            const percentage = (problem.attempt_count / maxAttempts) * 100;

                            return (
                                <div key={problem.problem_code} className="space-y-2">
                                    <div className="flex justify-between text-xs font-medium">
                                        <span className="font-black text-primary">{problem.problem_label}</span>
                                        <span className="text-muted-foreground">{problem.attempt_count} submissions</span>
                                    </div>
                                    <div className="h-3 bg-muted rounded-full overflow-hidden">
                                        <div
                                            className={cn(
                                                "h-full rounded-full transition-all",
                                                problem.is_solved ? "bg-gradient-to-r from-emerald-500 to-emerald-400" : "bg-gradient-to-r from-rose-500 to-rose-400"
                                            )}
                                            style={{ width: `${percentage}%` }}
                                        />
                                    </div>
                                </div>
                            );
                        })}
                    </CardContent>
                </Card>

                {/* Solve Time Comparison */}
                <Card className="rounded-[2rem] border-2 shadow-lg">
                    <CardHeader className="pb-4 border-b">
                        <CardTitle className="text-lg font-black flex items-center gap-3">
                            <Clock size={20} className="text-primary" />
                            Solve Time vs Average
                        </CardTitle>
                    </CardHeader>
                    <CardContent className="p-6 space-y-4">
                        {stats.problems.filter(p => p.is_solved).map((problem) => {
                            const avgTime = stats.average_solve_times_by_problem[problem.problem_code];
                            if (!avgTime || !problem.solve_time) return null;

                            const ratio = problem.solve_time / avgTime;
                            const percentage = Math.min(ratio * 50, 100);

                            return (
                                <div key={problem.problem_code} className="space-y-2">
                                    <div className="flex justify-between text-xs font-medium">
                                        <span className="font-black text-primary">{problem.problem_label}</span>
                                        <span className={cn(
                                            "font-medium",
                                            ratio < 1 ? "text-emerald-500" : "text-amber-500"
                                        )}>
                                            {ratio < 1 ? '-' : '+'}{((ratio - 1) * 100).toFixed(0)}% vs avg
                                        </span>
                                    </div>
                                    <div className="h-3 bg-muted rounded-full overflow-hidden relative">
                                        {/* Average marker */}
                                        <div
                                            className="absolute top-0 bottom-0 w-0.5 bg-amber-500 z-10"
                                            style={{ left: '50%' }}
                                        />
                                        {/* User time bar */}
                                        <div
                                            className={cn(
                                                "h-full rounded-full transition-all absolute",
                                                ratio < 1 ? "bg-emerald-500" : "bg-amber-500"
                                            )}
                                            style={{ width: `${percentage}%` }}
                                        />
                                    </div>
                                    <div className="flex justify-between text-[10px] text-muted-foreground">
                                        <span>{formatTime(problem.solve_time!)} (you)</span>
                                        <span>{formatTime(Math.round(avgTime))} (avg)</span>
                                    </div>
                                </div>
                            );
                        })}
                        {stats.problems.filter(p => p.is_solved).length === 0 && (
                            <div className="text-center py-8 text-muted-foreground">
                                No solved problems to compare
                            </div>
                        )}
                    </CardContent>
                </Card>
            </div>
        </div>
    );
}

function formatTime(seconds: number): string {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hrs > 0) {
        return `${hrs}h ${mins}m ${secs}s`;
    } else if (mins > 0) {
        return `${mins}m ${secs}s`;
    } else {
        return `${secs}s`;
    }
}
