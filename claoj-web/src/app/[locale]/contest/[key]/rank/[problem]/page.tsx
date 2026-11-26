'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import { useState } from 'react';
import api from '@/lib/api';
import { ContestProblemRankResponse } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { PaginationBar } from '@/components/ui/PaginationBar';
import { Link } from '@/navigation';
import { ArrowLeft, BarChart3, Trophy } from 'lucide-react';
import { cn, getStatusVariant } from '@/lib/utils';
import { formatMemoryMb } from '@/lib/submissionDetail';
import dayjs from 'dayjs';

// v1 ContestRankedSubmission: the best submission per user for one contest
// problem, ordered by contest points then runtime (only points > 0).
const PAGE_SIZE = 50;

export default function ContestProblemRankPage() {
    const t = useTranslations('Contest');
    const tSub = useTranslations('Submissions');
    const params = useParams();
    const key = params.key as string;
    const problem = params.problem as string;
    const [page, setPage] = useState(1);

    const { data, isLoading } = useQuery({
        queryKey: ['contest-problem-rank', key, problem, page],
        queryFn: async () => {
            const res = await api.get<ContestProblemRankResponse>(`/contest/${key}/rank/${problem}`, {
                params: { page, page_size: PAGE_SIZE },
            });
            return res.data;
        },
    });

    if (isLoading) {
        return (
            <div className="max-w-7xl mx-auto space-y-4 mt-4 pb-20">
                <Skeleton className="h-28 w-full rounded-md" />
                <Skeleton className="h-[50vh] w-full rounded-md" />
            </div>
        );
    }

    if (!data) {
        return <div className="p-8 text-center text-muted-foreground">{t('notFound')}</div>;
    }

    const problemName = data.problem.name || `${t('problem')} ${data.problem.label}`;

    return (
        <div className="max-w-7xl mx-auto space-y-4 mt-4 pb-20">
            <Link
                href={`/contest/${key}?tab=scoreboard`}
                className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-primary transition-colors"
            >
                <ArrowLeft size={16} />
                {data.contest_name || key}
            </Link>

            <header className="flex flex-wrap items-end justify-between gap-4">
                <div className="space-y-2">
                    <h1 className="text-3xl md:text-4xl font-semibold tracking-tighter flex items-center gap-3">
                        <Trophy className="text-muted-foreground" size={36} />
                        {t('bestSolutionsTitle', { problem: problemName, contest: data.contest_name || key })}
                    </h1>
                    <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                        <Badge variant="outline" className="font-mono">{data.problem.label}</Badge>
                        {data.problem.name && (
                            <Link href={`/problem/${data.problem.code}`} className="hover:text-primary transition-colors">
                                {data.problem.code}
                            </Link>
                        )}
                        <span>{t('points')}: {data.problem.points}</span>
                    </div>
                </div>
            </header>

            <div className="bg-card border rounded-md overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                        <thead>
                            <tr className="bg-muted/30 border-b">
                                <th className="px-8 py-6 text-xs font-medium text-muted-foreground w-20 text-center">{t('rank')}</th>
                                <th className="px-8 py-6 text-xs font-medium text-muted-foreground">{t('competitor')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-24">{t('score')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-28">{t('time')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-28">{tSub('memory')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-32">{tSub('language')}</th>
                                <th className="px-8 py-6 text-xs font-medium text-muted-foreground text-right w-40">{tSub('date')}</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {data.data.map((row) => (
                                <tr key={row.id} className="hover:bg-muted/5 transition-colors">
                                    <td className="px-8 py-6 text-center">
                                        <span className={cn(
                                            "inline-flex items-center justify-center w-9 h-9 rounded-xl font-medium text-sm",
                                            row.rank === 1 ? "bg-amber-700 text-white" :
                                                row.rank === 2 ? "bg-zinc-400 text-white" :
                                                    row.rank === 3 ? "bg-orange-600 text-white" :
                                                        "bg-muted text-muted-foreground"
                                        )}>
                                            {row.rank}
                                        </span>
                                    </td>
                                    <td className="px-8 py-6">
                                        <div className="flex items-center gap-2">
                                            <Link href={`/user/${row.user}`} className="font-medium hover:text-primary transition-colors">
                                                {row.user}
                                            </Link>
                                            {row.display_name && row.display_name !== row.user && (
                                                <span className="text-xs text-muted-foreground">{row.display_name}</span>
                                            )}
                                        </div>
                                    </td>
                                    <td className="px-6 py-6 text-center">
                                        <Link href={`/submission/${row.id}`} className="inline-block">
                                            <span className={cn(
                                                "text-sm font-medium text-white px-2 py-1 rounded-lg inline-block min-w-[50px] shadow-sm",
                                                row.points > 0 ? "bg-emerald-500 hover:bg-emerald-600" : "bg-zinc-200 text-zinc-500 dark:bg-zinc-800"
                                            )}>
                                                {row.points}
                                            </span>
                                        </Link>
                                    </td>
                                    <td className="px-6 py-6 text-center font-mono text-xs text-muted-foreground">
                                        {row.time != null ? `${(row.time * 1000).toFixed(0)}ms` : '-'}
                                    </td>
                                    <td className="px-6 py-6 text-center font-mono text-xs text-muted-foreground">
                                        {formatMemoryMb(row.memory)}
                                    </td>
                                    <td className="px-6 py-6 text-center">
                                        <div className="flex flex-col items-center gap-1">
                                            <Link
                                                href={`/submission/${row.id}`}
                                                className="text-xs font-medium text-muted-foreground opacity-60 hover:text-primary hover:opacity-100"
                                            >
                                                {row.language}
                                            </Link>
                                            {row.result && (
                                                <Badge variant={getStatusVariant(row.result)} className="text-xs px-2">
                                                    {row.result}
                                                </Badge>
                                            )}
                                        </div>
                                    </td>
                                    <td className="px-8 py-6 text-right text-xs text-muted-foreground">
                                        {dayjs(row.date).format('L HH:mm')}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
                {data.scoreboard_hidden && (
                    <p className="p-12 text-center text-sm text-muted-foreground">{t('scoreboardHidden')}</p>
                )}
                {!data.scoreboard_hidden && data.data.length === 0 && (
                    <p className="p-12 text-center text-sm text-muted-foreground flex items-center justify-center gap-2">
                        <BarChart3 size={16} />
                        {t('noSolutions')}
                    </p>
                )}
            </div>

            <div className="text-xs text-muted-foreground">
                {t('bestSolutionsHint')}
            </div>

            <PaginationBar
                page={page}
                onPageChange={setPage}
                total={data.total}
                pageSize={PAGE_SIZE}
            />
        </div>
    );
}