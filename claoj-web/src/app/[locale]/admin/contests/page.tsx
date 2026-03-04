'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import api from '@/lib/api';
import { AdminContest } from '@/types';
import { adminContestApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import {
    Search,
    Trophy,
    Globe,
    Ban,
    Edit,
    Trash2,
    Calendar,
    Clock,
    RefreshCw,
    Play
} from 'lucide-react';

export default function AdminContestPage() {
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');

    const queryClient = useQueryClient();
    const router = useRouter();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-contests', page, search],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminContest[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/contests?page=${page}&page_size=50&search=${search}`);
            return res.data;
        }
    });

    const deleteMutation = useMutation({
        mutationFn: (key: string) => adminContestApi.delete(key),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-contests'] });
        }
    });

    const contests = data?.data || [];

    const filteredContests = contests.filter(c =>
        c.name.toLowerCase().includes(search.toLowerCase()) ||
        c.key.toLowerCase().includes(search.toLowerCase())
    );

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Trophy className="text-primary" size={32} />
                        Contests
                    </h1>
                    <p className="text-muted-foreground mt-1">Manage contests and their settings</p>
                </div>

                <div className="flex items-center gap-3">
                    <div className="relative w-full md:w-80">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder="Search contests..."
                            className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                    <Link
                        href="/admin/contests/create"
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                    >
                        <Play size={18} />
                        Create
                    </Link>
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-20 rounded-2xl" />)}
                </div>
            ) : (
                <div className="bg-card rounded-2xl border overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left">
                            <thead className="bg-muted/50 border-b">
                                <tr>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Contest</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Time</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Settings</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground text-right">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {filteredContests.length === 0 ? (
                                    <tr>
                                        <td colSpan={4} className="px-6 py-12 text-center text-muted-foreground">
                                            No contests found
                                        </td>
                                    </tr>
                                ) : (
                                    filteredContests.map((contest) => (
                                        <tr key={contest.key} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-6 py-4">
                                                <div className="font-bold text-sm mb-1">{contest.name}</div>
                                                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                                    <span className="font-mono">{contest.key}</span>
                                                    {contest.is_rated && (
                                                        <Badge variant="warning" className="text-[10px]">Rated</Badge>
                                                    )}
                                                    {contest.is_organization_private && (
                                                        <Badge variant="secondary" className="text-[10px]">Private</Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col gap-1 text-sm">
                                                    <span className="text-muted-foreground">
                                                        {contest.start_time && new Date(contest.start_time).toLocaleDateString()}
                                                    </span>
                                                    <span className="text-xs text-muted-foreground flex items-center gap-1">
                                                        <Clock size={12} />
                                                        {contest.end_time && new Date(contest.end_time).toLocaleDateString()}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-2">
                                                    {contest.is_visible ? (
                                                        <Badge variant="success" className="flex items-center gap-1 text-xs">
                                                            Visible
                                                        </Badge>
                                                    ) : (
                                                        <Badge variant="destructive" className="flex items-center text-xs">
                                                            Hidden
                                                        </Badge>
                                                    )}
                                                    <span className="text-xs text-muted-foreground">
                                                        {contest.user_count} users
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    <Link
                                                        href={`/admin/contests/${contest.key}/edit`}
                                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title="Edit contest"
                                                    >
                                                        <Edit size={18} />
                                                    </Link>
                                                    <button
                                                        onClick={() => deleteMutation.mutate(contest.key)}
                                                        disabled={deleteMutation.isPending}
                                                        className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                        title="Delete contest"
                                                    >
                                                        <Trash2 size={18} />
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>

                    {/* Pagination */}
                    {filteredContests.length > 0 && (
                        <div className="flex items-center justify-between px-6 py-4 border-t bg-muted/30">
                            <div className="text-sm text-muted-foreground">
                                Showing {filteredContests.length} of {data?.total || 0} contests
                            </div>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => setPage(p => Math.max(1, p - 1))}
                                    disabled={page === 1}
                                    className="px-3 py-1.5 rounded-lg bg-card border disabled:opacity-50 hover:bg-muted transition-colors"
                                >
                                    Previous
                                </button>
                                <div className="px-3 py-1.5 rounded-lg bg-primary text-primary-foreground font-bold">
                                    {page}
                                </div>
                                <button
                                    onClick={() => setPage(p => p + 1)}
                                    disabled={filteredContests.length < 50}
                                    className="px-3 py-1.5 rounded-lg bg-card border disabled:opacity-50 hover:bg-muted transition-colors"
                                >
                                    Next
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
