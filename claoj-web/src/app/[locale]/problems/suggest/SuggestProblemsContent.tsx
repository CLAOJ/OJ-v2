'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { isAxiosError } from 'axios';
import api from '@/lib/api';
import { PaginatedList, SuggestedProblem } from '@/types';
import { Link } from '@/navigation';
import { Skeleton } from '@/components/ui/Skeleton';
import { PaginationBar, PAGE_SIZE_OPTIONS } from '@/components/ui/PaginationBar';
import { formatDate } from '@/lib/date';
import { ArrowLeft, ClipboardList, Lock } from 'lucide-react';
import { useState } from 'react';

export default function SuggestProblemsContent() {
    const t = useTranslations('Problems.suggest');
    const tProblems = useTranslations('Problems');
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState<number>(PAGE_SIZE_OPTIONS[1]);

    const { data, isLoading, error } = useQuery({
        queryKey: ['suggested-problems', page, pageSize],
        // A 403 is the answer, not a transient failure: don't retry it.
        retry: false,
        queryFn: async () => {
            const res = await api.get<PaginatedList<SuggestedProblem>>(
                `/problems/suggest?page=${page}&page_size=${pageSize}`
            );
            return res.data;
        }
    });

    const forbidden = isAxiosError(error) && error.response?.status === 403;
    const failed = !!error && !forbidden;
    const rows = data?.data ?? [];

    return (
        <div className="flex flex-col gap-6 mt-4 min-h-[calc(100vh-12rem)]">
            <div className="flex flex-col gap-1">
                <Link
                    href="/problems"
                    className="text-xs font-medium text-muted-foreground hover:text-primary flex items-center gap-2 w-fit"
                >
                    <ArrowLeft size={14} /> {tProblems('backToProblems')}
                </Link>
                <h1 className="text-2xl font-semibold flex items-center gap-3">
                    <ClipboardList size={24} className="text-primary" /> {t('title')}
                </h1>
                <p className="text-sm text-muted-foreground">{t('subtitle')}</p>
            </div>

            {forbidden ? (
                <div className="p-10 rounded-md bg-card border border-dashed text-center space-y-2">
                    <Lock size={28} className="mx-auto text-muted-foreground/50" />
                    <p className="text-sm font-medium">{t('forbidden')}</p>
                </div>
            ) : failed ? (
                <div role="alert" className="bg-card rounded-md border p-12 text-center text-destructive">
                    {t('error')}
                </div>
            ) : (
                <>
                    <div className="bg-card border rounded-md overflow-hidden shadow-sm">
                        <div className="overflow-x-auto">
                            <table className="w-full text-left border-collapse">
                                <thead>
                                    <tr className="bg-muted/30 border-b">
                                        <th className="px-6 py-6 text-xs font-medium text-muted-foreground w-32">{tProblems('code')}</th>
                                        <th className="px-6 py-6 text-xs font-medium text-muted-foreground">{tProblems('name')}</th>
                                        <th className="px-6 py-6 text-xs font-medium text-muted-foreground w-48">{t('suggester')}</th>
                                        <th className="px-6 py-6 text-xs font-medium text-muted-foreground w-24 text-center">{tProblems('points')}</th>
                                        <th className="px-6 py-6 text-xs font-medium text-muted-foreground w-40 text-center">{t('date')}</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-muted/50">
                                    {isLoading ? (
                                        Array.from({ length: Math.min(pageSize, 10) }).map((_, i) => (
                                            <tr key={i}><td colSpan={5} className="px-6 py-4"><Skeleton className="h-8 w-full rounded-md" /></td></tr>
                                        ))
                                    ) : rows.length === 0 ? (
                                        <tr>
                                            <td colSpan={5} className="px-6 py-12 text-center text-sm text-muted-foreground">
                                                {t('empty')}
                                            </td>
                                        </tr>
                                    ) : (
                                        rows.map(p => (
                                            <tr key={p.code} className="hover:bg-muted/5 transition-colors">
                                                <td className="px-6 py-5 font-medium font-mono text-sm text-primary tracking-tighter">{p.code}</td>
                                                <td className="px-6 py-5">
                                                    <Link
                                                        href={`/problem/${p.code}`}
                                                        className="font-semibold hover:text-primary transition-colors underline-offset-4 decoration-primary/20 hover:underline"
                                                    >
                                                        {p.name}
                                                    </Link>
                                                </td>
                                                <td className="px-6 py-5 text-sm text-muted-foreground">{p.suggester || '—'}</td>
                                                <td className="px-6 py-5 text-center text-sm font-medium">{p.points}</td>
                                                <td className="px-6 py-5 text-center text-sm text-muted-foreground">
                                                    {p.date ? formatDate(p.date, 'short') : '—'}
                                                </td>
                                            </tr>
                                        ))
                                    )}
                                </tbody>
                            </table>
                        </div>
                    </div>

                    <PaginationBar
                        page={page}
                        onPageChange={setPage}
                        total={data?.total}
                        pageSize={pageSize}
                        onPageSizeChange={size => { setPageSize(size); setPage(1); }}
                    />
                </>
            )}
            <div className="pb-8" />
        </div>
    );
}
