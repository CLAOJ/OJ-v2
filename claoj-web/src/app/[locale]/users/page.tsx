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
    Hash,
    ChevronLeft,
    ChevronRight,
    RefreshCw,
    User as UserIcon
} from 'lucide-react';
import { cn, getRankColor } from '@/lib/utils';

export default function UsersListPage() {
    const t = useTranslations('Users');
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');
    const [sortBy, setSortBy] = useState<'points' | 'rating' | 'problem_count'>('points');
    const [order, setOrder] = useState<'asc' | 'desc'>('desc');

    const { data, isLoading } = useQuery({
        queryKey: ['users', page, search, sortBy, order],
        queryFn: async () => {
            const params = new URLSearchParams({
                page: page.toString(),
                page_size: '50',
                search,
                sort: sortBy,
                order,
            });
            const res = await api.get<PaginatedList<UserListItem>>(`/users?${params.toString()}`);
            return res.data;
        }
    });

    const users = data?.data || [];

    const toggleSort = (field: 'points' | 'rating' | 'problem_count') => {
        if (sortBy === field) {
            setOrder(order === 'asc' ? 'desc' : 'asc');
        } else {
            setSortBy(field);
            setOrder('desc');
        }
    };

    return (
        <div className="max-w-7xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            <div className="flex flex-col md:flex-row justify-between items-end gap-6">
                <header className="space-y-2">
                    <h1 className="text-4xl md:text-5xl font-black tracking-tighter flex items-center gap-4">
                        <Users className="text-primary" size={48} />
                        {t('title') || 'Users'}
                    </h1>
                    <p className="text-muted-foreground font-black opacity-80">Competitive programmers from around the world.</p>
                </header>

                <div className="flex flex-wrap items-center gap-3 bg-muted/30 p-4 rounded-[2.5rem] border border-dashed">
                    <div className="flex flex-col gap-1">
                        <span className="text-[10px] font-black uppercase text-muted-foreground ml-1">Page</span>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setPage(p => Math.max(1, p - 1))}
                                disabled={page === 1}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-20 transition-all"
                            >
                                <ChevronLeft size={18} />
                            </button>
                            <div className="h-10 px-4 rounded-xl bg-primary text-primary-foreground font-black text-xs flex items-center shadow-lg shadow-primary/20">
                                {page}
                            </div>
                            <button
                                onClick={() => setPage(p => p + 1)}
                                disabled={users.length < 50}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-20 transition-all"
                            >
                                <ChevronRight size={18} />
                            </button>
                        </div>
                    </div>

                    <button
                        onClick={() => {
                            setSearch('');
                            setPage(1);
                        }}
                        className="h-10 px-6 rounded-xl bg-muted/50 hover:bg-muted font-black text-[10px] uppercase tracking-widest flex items-center gap-2 mt-auto"
                    >
                        <RefreshCw size={14} /> Reset
                    </button>
                </div>
            </div>

            {/* Search and Sort Bar */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 p-6 rounded-[2.5rem] bg-card border shadow-sm">
                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Search</label>
                    <div className="relative">
                        <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <input
                            type="text"
                            placeholder="Username or display name..."
                            className="w-full h-12 bg-muted/30 border border-transparent rounded-2xl pl-12 pr-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none"
                            value={search}
                            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Sort By</label>
                    <div className="flex gap-2">
                        <button
                            onClick={() => toggleSort('points')}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all flex items-center justify-center gap-2",
                                sortBy === 'points'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            <Trophy size={14} /> Points
                        </button>
                        <button
                            onClick={() => toggleSort('rating')}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all flex items-center justify-center gap-2",
                                sortBy === 'rating'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            <TrendingUp size={14} /> Rating
                        </button>
                        <button
                            onClick={() => toggleSort('problem_count')}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all flex items-center justify-center gap-2",
                                sortBy === 'problem_count'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            <Hash size={14} /> Solved
                        </button>
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Order</label>
                    <div className="flex gap-2">
                        <button
                            onClick={() => setOrder('desc')}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                order === 'desc'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            High to Low
                        </button>
                        <button
                            onClick={() => setOrder('asc')}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                order === 'asc'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            Low to High
                        </button>
                    </div>
                </div>
            </div>

            <div className="bg-card border rounded-[3rem] overflow-hidden shadow-2xl shadow-primary/5">
                <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                        <thead>
                            <tr className="bg-muted/30 border-b">
                                <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground w-20 text-center">Rank</th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">User</th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center cursor-pointer" onClick={() => toggleSort('points')}>
                                    <div className="flex items-center justify-center gap-2">
                                        Points {sortBy === 'points' && (order === 'asc' ? '↑' : '↓')}
                                    </div>
                                </th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center cursor-pointer" onClick={() => toggleSort('rating')}>
                                    <div className="flex items-center justify-center gap-2">
                                        Rating {sortBy === 'rating' && (order === 'asc' ? '↑' : '↓')}
                                    </div>
                                </th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center cursor-pointer" onClick={() => toggleSort('problem_count')}>
                                    <div className="flex items-center justify-center gap-2">
                                        Solved {sortBy === 'problem_count' && (order === 'asc' ? '↑' : '↓')}
                                    </div>
                                </th>
                                <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Organization</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {isLoading ? (
                                Array.from({ length: 15 }).map((_, i) => (
                                    <tr key={i}>
                                        <td colSpan={6} className="px-10 py-6"><Skeleton className="h-16 w-full rounded-2xl" /></td>
                                    </tr>
                                ))
                            ) : (
                                users.map((user, index) => (
                                    <tr key={user.id} className="hover:bg-muted/10 transition-colors group">
                                        <td className="px-10 py-8 text-center">
                                            <div className="inline-flex items-center justify-center w-10 h-10 rounded-xl bg-muted font-black text-sm text-muted-foreground">
                                                {(page - 1) * 50 + index + 1}
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
                                        <td className="px-6 py-8 text-center">
                                            <div className="inline-flex items-center justify-center px-4 py-2 rounded-xl bg-amber-500/10 text-amber-500 border border-amber-500/20 shadow-sm">
                                                <Trophy size={14} className="mr-2" />
                                                <span className="font-black">{Math.round(user.points)}</span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            {user.rating ? (
                                                <Badge className={cn("px-4 py-2 rounded-xl text-[10px] font-black uppercase tracking-widest", getRankColor(user.rating))}>
                                                    {Math.round(user.rating)}
                                                </Badge>
                                            ) : (
                                                <span className="text-muted-foreground text-[10px] font-black">-</span>
                                            )}
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            <div className="inline-flex items-center gap-2 justify-center px-4 py-2 rounded-xl bg-primary/5 text-primary border border-primary/10">
                                                <Hash size={14} />
                                                <span className="font-black">{user.problem_count}</span>
                                            </div>
                                        </td>
                                        <td className="px-10 py-8">
                                            {user.organizations && user.organizations.length > 0 ? (
                                                <div className="flex flex-col gap-1">
                                                    {user.organizations.slice(0, 2).map(org => (
                                                        <Link
                                                            key={org.id}
                                                            href={`/organization/${org.id}`}
                                                            className="text-sm font-bold text-muted-foreground hover:text-primary transition-colors"
                                                        >
                                                            {org.name}
                                                        </Link>
                                                    ))}
                                                    {user.organizations.length > 2 && (
                                                        <span className="text-[10px] text-muted-foreground">+{user.organizations.length - 2} more</span>
                                                    )}
                                                </div>
                                            ) : (
                                                <span className="text-muted-foreground text-[10px] font-black uppercase opacity-50">None</span>
                                            )}
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            {users.length > 0 && (
                <div className="text-center text-sm text-muted-foreground font-bold">
                    Showing {users.length} users
                </div>
            )}
        </div>
    );
}
