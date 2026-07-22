'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { UserListItem, PaginatedList } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link } from '@/navigation';
import { useState } from 'react';
import {
    Users,
    Search,
    Trophy,
    TrendingUp,
    ThumbsUp,
    RefreshCw
} from 'lucide-react';
import { cn, getRankColor } from '@/lib/utils';
import { PaginationBar, PAGE_SIZE_OPTIONS } from '@/components/ui/PaginationBar';

type RankBy = 'points' | 'rating' | 'contribution';

const RANKINGS: { key: RankBy; icon: typeof Trophy; labelKey: 'points' | 'rating' | 'contributors' }[] = [
    { key: 'points', icon: Trophy, labelKey: 'points' },
    { key: 'rating', icon: TrendingUp, labelKey: 'rating' },
    { key: 'contribution', icon: ThumbsUp, labelKey: 'contributors' },
];

export default function UsersListPage() {
    const t = useTranslations('Users');
    const tCommon = useTranslations('Common');
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(PAGE_SIZE_OPTIONS[1]);
    const [search, setSearch] = useState('');
    const [rankBy, setRankBy] = useState<RankBy>('points');

    // Rankings are always highest-first server-side; only `search` is applied
    // client-side, over the current page.
    const { data, isLoading } = useQuery({
        queryKey: ['users', page, pageSize, rankBy],
        queryFn: async () => {
            const params = new URLSearchParams({
                page: page.toString(),
                page_size: pageSize.toString(),
                sort: rankBy,
            });
            const res = await api.get<PaginatedList<UserListItem>>(`/users?${params.toString()}`);
            return res.data;
        }
    });

    const fetchedUsers = data?.data || [];
    // Keep the server-assigned rank even when the search box narrows the list.
    const users = fetchedUsers
        .map((user, index) => ({ user, rank: (page - 1) * pageSize + index + 1 }))
        .filter(({ user }) =>
            !search ||
            user.username.toLowerCase().includes(search.toLowerCase()) ||
            (user.display_name || '').toLowerCase().includes(search.toLowerCase())
        );

    const changeRanking = (key: RankBy) => {
        setRankBy(key);
        setPage(1);
    };

    return (
        <div className="max-w-7xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            <div className="flex flex-col md:flex-row justify-between items-end gap-6">
                <header className="space-y-2">
                    <h1 className="text-4xl md:text-5xl font-black tracking-tighter flex items-center gap-4">
                        <Users className="text-primary" size={48} />
                        {t('title') || 'Users'}
                    </h1>
                    <p className="text-muted-foreground font-black opacity-80">{t('subtitle')}</p>
                </header>

                <div className="flex flex-wrap items-center gap-3">
                    <button
                        onClick={() => {
                            setSearch('');
                            setRankBy('points');
                            setPage(1);
                        }}
                        className="h-12 px-6 rounded-2xl bg-muted/30 border hover:bg-muted font-black text-[10px] uppercase tracking-widest flex items-center gap-2"
                    >
                        <RefreshCw size={14} /> {tCommon('reset')}
                    </button>
                </div>
            </div>

            {/* Search and Ranking Bar */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 p-6 rounded-[2.5rem] bg-card border shadow-sm">
                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">{tCommon('search')}</label>
                    <div className="relative">
                        <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <input
                            type="text"
                            placeholder={t('searchPlaceholder')}
                            className="w-full h-12 bg-muted/30 border border-transparent rounded-2xl pl-12 pr-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none"
                            value={search}
                            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">{t('rankBy')}</label>
                    <div className="flex gap-2">
                        {RANKINGS.map(({ key, icon: Icon, labelKey }) => (
                            <button
                                key={key}
                                onClick={() => changeRanking(key)}
                                className={cn(
                                    "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all flex items-center justify-center gap-2",
                                    rankBy === key
                                        ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                        : "bg-muted/30 hover:bg-muted border-transparent"
                                )}
                            >
                                <Icon size={14} /> {t(labelKey)}
                            </button>
                        ))}
                    </div>
                </div>
            </div>

            <div className="bg-card border rounded-[3rem] overflow-hidden shadow-2xl shadow-primary/5">
                <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                        <thead>
                            <tr className="bg-muted/30 border-b">
                                <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground w-20 text-center">{t('rank')}</th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">{t('user')}</th>
                                <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center">
                                    {rankBy === 'points' ? t('points') : rankBy === 'rating' ? t('rating') : t('contribution')}
                                </th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {isLoading ? (
                                Array.from({ length: 15 }).map((_, i) => (
                                    <tr key={i}>
                                        <td colSpan={3} className="px-10 py-6"><Skeleton className="h-16 w-full rounded-2xl" /></td>
                                    </tr>
                                ))
                            ) : (
                                users.map(({ user, rank }) => (
                                    <tr key={user.username} className="hover:bg-muted/10 transition-colors group">
                                        <td className="px-10 py-8 text-center">
                                            <div className="inline-flex items-center justify-center w-10 h-10 rounded-xl bg-muted font-black text-sm text-muted-foreground">
                                                {rank}
                                            </div>
                                        </td>
                                        <td className="px-6 py-8">
                                            <Link href={`/user/${user.username}`} className="flex items-center gap-3 group/user outline-none">
                                                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center text-primary font-black text-lg group-hover/user:scale-110 transition-all">
                                                    {user.display_name?.[0]?.toUpperCase() || user.username[0]?.toUpperCase()}
                                                </div>
                                                <div className="flex flex-col gap-1">
                                                    <span className="font-black text-lg group-hover/user:text-primary transition-colors">
                                                        {user.display_name || user.username}
                                                    </span>
                                                    <span className="text-[10px] font-mono text-muted-foreground">@{user.username}</span>
                                                </div>
                                            </Link>
                                        </td>
                                        <td className="px-10 py-8 text-center">
                                            {rankBy === 'points' && (
                                                <div className="inline-flex items-center justify-center px-4 py-2 rounded-xl bg-amber-500/10 text-amber-500 border border-amber-500/20 shadow-sm">
                                                    <Trophy size={14} className="mr-2" />
                                                    <span className="font-black">{Math.round(user.performance_points)}</span>
                                                </div>
                                            )}
                                            {rankBy === 'rating' && (
                                                user.rating ? (
                                                    <Badge className={cn("px-4 py-2 rounded-xl text-[10px] font-black uppercase tracking-widest", getRankColor(user.rating))}>
                                                        {Math.round(user.rating)}
                                                    </Badge>
                                                ) : (
                                                    <span className="text-muted-foreground text-[10px] font-black">-</span>
                                                )
                                            )}
                                            {rankBy === 'contribution' && (
                                                <div className="inline-flex items-center gap-2 justify-center px-4 py-2 rounded-xl bg-primary/5 text-primary border border-primary/10">
                                                    <ThumbsUp size={14} />
                                                    <span className="font-black">{user.contribution_points}</span>
                                                </div>
                                            )}
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
        </div>
    );
}
