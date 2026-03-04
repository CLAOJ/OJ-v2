'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { AdminProblem } from '@/types';
import { adminProblemApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import {
    Search,
    Code2,
    Ban,
    Edit,
    Trash2,
    Database,
    Clock,
    Plus
} from 'lucide-react';

export default function AdminProblemPage() {
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');

    const queryClient = useQueryClient();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-problems', page, search],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminProblem[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/problems?page=${page}&page_size=50&search=${search}`);
            return res.data;
        }
    });

    const deleteMutation = useMutation({
        mutationFn: (code: string) => adminProblemApi.delete(code),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-problems'] });
        }
    });

    const problems = data?.data || [];

    const filteredProblems = problems.filter(p =>
        p.name.toLowerCase().includes(search.toLowerCase()) ||
        p.code.toLowerCase().includes(search.toLowerCase())
    );

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Code2 className="text-primary" size={32} />
                        Problems
                    </h1>
                    <p className="text-muted-foreground mt-1">Manage problems and their settings</p>
                </div>

                <div className="flex items-center gap-3">
                    <div className="relative w-full md:w-80">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder="Search problems..."
                            className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                    <Link
                        href="/admin/problems/create"
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                    >
                        <Plus size={18} />
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
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Problem</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Stats</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Group</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground text-right">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {filteredProblems.length === 0 ? (
                                    <tr>
                                        <td colSpan={4} className="px-6 py-12 text-center text-muted-foreground">
                                            No problems found
                                        </td>
                                    </tr>
                                ) : (
                                    filteredProblems.map((problem) => (
                                        <tr key={problem.code} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-3 mb-1">
                                                    <Link
                                                        href={`/problems/${problem.code}`}
                                                        className="font-bold text-sm hover:text-primary transition-colors"
                                                    >
                                                        {problem.name}
                                                    </Link>
                                                    {problem.partial && (
                                                        <Badge variant="secondary" className="text-[10px]">Partial</Badge>
                                                    )}
                                                </div>
                                                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                                    <span className="font-mono">{problem.code}</span>
                                                    {problem.is_public ? (
                                                        <Badge variant="success" className="text-[10px]">Public</Badge>
                                                    ) : (
                                                        <Badge variant="destructive" className="text-[10px]">Private</Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col gap-1 text-sm">
                                                    <span className="text-muted-foreground font-bold">
                                                        {problem.points.toFixed(1)} Points
                                                    </span>
                                                    <span className="text-xs text-muted-foreground">
                                                        {Math.round(problem.ac_rate * 100)}% AC Rate
                                                    </span>
                                                    <span className="text-xs text-muted-foreground">
                                                        {problem.user_count} Users
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="text-sm text-muted-foreground">
                                                    {problem.group_name || 'Uncategorized'}
                                                </div>
                                                <div className="text-xs text-muted-foreground mt-1">
                                                    {problem.is_manually_managed ? 'Manually Managed' : 'Auto'}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    <Link
                                                        href={`/admin/problems/${problem.code}/edit`}
                                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title="Edit problem"
                                                    >
                                                        <Edit size={18} />
                                                    </Link>
                                                    <button
                                                        onClick={() => deleteMutation.mutate(problem.code)}
                                                        disabled={deleteMutation.isPending}
                                                        className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                        title="Delete problem"
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
                    {filteredProblems.length > 0 && (
                        <div className="flex items-center justify-between px-6 py-4 border-t bg-muted/30">
                            <div className="text-sm text-muted-foreground">
                                Showing {filteredProblems.length} of {data?.total || 0} problems
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
                                    disabled={filteredProblems.length < 50}
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
