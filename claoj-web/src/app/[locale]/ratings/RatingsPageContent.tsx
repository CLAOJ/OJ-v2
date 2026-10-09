'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { RatingLeaderboardResponse } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { useState } from 'react';
import { Trophy, Search, TrendingUp, Users } from 'lucide-react';
import { cn } from '@/lib/utils';
import { getRankBadgeColor, getRankTitle } from '@/lib/utils';
import { PaginationBar, PAGE_SIZE_OPTIONS } from '@/components/ui/PaginationBar';
import { Link } from '@/navigation';

export default function RatingsPageContent() {
    const t = useTranslations('Ratings');
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');
    const [limit, setLimit] = useState(PAGE_SIZE_OPTIONS[1]);

    const { data, isLoading } = useQuery({
        queryKey: ['ratings-leaderboard', page, limit, search],
        queryFn: async () => {
            const params = new URLSearchParams({
                page: page.toString(),
                limit: limit.toString(),
                ...(search && { search })
            });
            const res = await api.get<RatingLeaderboardResponse>(`/ratings/leaderboard?${params}`);
            return res.data;
        }
    });


    return (
        <div className="max-w-7xl mx-auto space-y-4 pb-20 mt-4">
            {/* Header */}
            <div className="relative overflow-hidden rounded-md border bg-card">
                <div className="absolute top-0 right-0 p-16 opacity-5 pointer-events-none rotate-12">
                    <Trophy size={240} className="text-primary" />
                </div>

                <div className="p-4 md:p-6 space-y-6 relative">
                    <div className="space-y-4">
                        <h1 className="text-4xl md:text-6xl font-semibold tracking-tighter leading-none">
                            {t('title')}
                        </h1>
                        <p className="text-lg font-medium text-muted-foreground max-w-2xl">
                            {t('description')}
                        </p>
                    </div>

                    {/* Stats */}
                    <div className="flex flex-wrap gap-4 pt-4">
                        <div className="flex items-center gap-3 px-6 py-3 rounded-md bg-primary/5 border border-primary/20">
                            <Users size={20} className="text-primary" />
                            <span className="text-sm font-medium">
                                {data?.total ?? 0} {t('ratedUsers')}
                            </span>
                        </div>
                    </div>
                </div>
            </div>

            {/* Search Bar */}
            <div className="flex gap-4">
                <div className="relative flex-1">
                    <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground" size={20} />
                    <input
                        type="text"
                        placeholder={t('searchPlaceholder')}
                        value={search}
                        onChange={(e) => {
                            setSearch(e.target.value);
                            setPage(1);
                        }}
                        className="w-full pl-12 pr-4 py-4 rounded-md border bg-card focus:outline-none focus:ring-2 focus:ring-primary/50 font-medium"
                    />
                </div>
            </div>

            {/* Leaderboard Table */}
            <div className="rounded-md border bg-card overflow-hidden">
                <div className="overflow-x-auto custom-scrollbar">
                    <table className="w-full text-left border-collapse">
                        <thead>
                            <tr className="bg-muted/30 border-b">
                                <th className="px-10 py-6 text-xs font-medium text-muted-foreground w-20 text-center">
                                    {t('rank')}
                                </th>
                                <th className="px-8 py-6 text-xs font-medium text-muted-foreground">
                                    {t('user')}
                                </th>
                                <th className="px-8 py-6 text-xs font-medium text-muted-foreground text-center">
                                    {t('rating')}
                                </th>
                                <th className="px-8 py-6 text-xs font-medium text-muted-foreground text-center w-32">
                                    {t('contests')}
                                </th>
                                <th className="px-8 py-6 text-xs font-medium text-muted-foreground text-center w-32">
                                    {t('highest')}
                                </th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {data?.data.map((entry) => (
                                <tr key={entry.username} className="hover:bg-muted/5 transition-colors">
                                    <td className="px-10 py-6 text-center">
                                        <div className="flex items-center justify-center gap-2">
                                            {entry.rank === 1 && <Trophy size={16} className="text-amber-500" />}
                                            <span className={cn(
                                                "inline-flex items-center justify-center w-8 h-8 rounded-xl font-medium text-sm",
                                                entry.rank === 1 ? "bg-amber-500 text-white" :
                                                    entry.rank === 2 ? "bg-zinc-400 text-white" :
                                                        entry.rank === 3 ? "bg-orange-600 text-white" :
                                                            "bg-muted text-muted-foreground"
                                            )}>
                                                {entry.rank}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="px-8 py-6">
                                        <Link href={`/user/${entry.username}`} className="flex items-center gap-3">
                                            <img
                                                src={entry.avatar_url}
                                                alt={entry.username}
                                                className="w-10 h-10 rounded-full border-2 border-muted"
                                            />
                                            <span className="font-medium text-base hover:text-primary transition-colors">
                                                {entry.username}
                                            </span>
                                        </Link>
                                    </td>
                                    <td className="px-8 py-6 text-center">
                                        <div className="flex flex-col items-center gap-1">
                                            <span className={cn(
                                                "text-xs font-medium px-3 py-1 rounded-md",
                                                getRankBadgeColor(entry.rating),
                                                "text-white"
                                            )}>
                                                {entry.rating}
                                            </span>
                                            <span className="text-xs text-muted-foreground font-medium ">
                                                {getRankTitle(entry.rating)}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="px-8 py-6 text-center">
                                        <div className="flex items-center justify-center gap-2 text-sm font-medium">
                                            <TrendingUp size={14} className="text-muted-foreground" />
                                            {entry.contests_attended}
                                        </div>
                                    </td>
                                    <td className="px-8 py-6 text-center">
                                        <span className={cn(
                                            "text-xs font-medium px-2 py-0.5 rounded-md",
                                            getRankBadgeColor(entry.highest_rating),
                                            "text-white"
                                        )}>
                                            {entry.highest_rating}
                                        </span>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>

                {isLoading && (
                    <div className="p-24 space-y-4">
                        <Skeleton className="h-20 w-full rounded-md" />
                        <Skeleton className="h-20 w-full rounded-md opacity-60" />
                        <Skeleton className="h-20 w-full rounded-md opacity-30" />
                    </div>
                )}

                {!isLoading && data?.data.length === 0 && (
                    <div className="p-24 text-center text-muted-foreground">
                        <p className="text-lg font-semibold">{t('noResults')}</p>
                    </div>
                )}
            </div>

            <PaginationBar
                page={page}
                onPageChange={setPage}
                total={data?.total}
                pageSize={limit}
                onPageSizeChange={size => { setLimit(size); setPage(1); }}
            />
        </div>
    );
}
