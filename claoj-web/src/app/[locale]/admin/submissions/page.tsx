'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { AdminSubmission } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import {
    Search,
    BrainCircuit,
    User,
    Hash,
    Clock,
    Filter,
    RefreshCw,
    ChevronLeft,
    ChevronRight
} from 'lucide-react';
import { getStatusColor, getStatusVariant } from '@/lib/utils';

export default function AdminSubmissionPage() {
    const [page, setPage] = useState(1);
    const [userFilter, setUserFilter] = useState('');
    const [problemFilter, setProblemFilter] = useState('');
    const [resultFilter, setResultFilter] = useState('');
    const [langFilter, setLangFilter] = useState('');

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-submissions', page, userFilter, problemFilter, resultFilter, langFilter],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminSubmission[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/submissions?page=${page}&page_size=50&user=${userFilter}&problem=${problemFilter}&result=${resultFilter}&language=${langFilter}`);
            return res.data;
        }
    });

    const submissions = data?.data || [];

    const results = ['AC', 'WA', 'TLE', 'MLE', 'OLE', 'RE', 'IE', 'CE', 'AB'];

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <BrainCircuit className="text-primary" size={32} />
                        Submissions
                    </h1>
                    <p className="text-muted-foreground mt-1">Review and rejudge all submissions</p>
                </div>

                <div className="flex items-center gap-2">
                    <button
                        onClick={() => {
                            setUserFilter('');
                            setProblemFilter('');
                            setResultFilter('');
                            setLangFilter('');
                            setPage(1);
                            refetch();
                        }}
                        className="px-3 py-2 rounded-xl bg-muted/50 hover:bg-muted transition-colors text-sm font-medium flex items-center gap-2"
                    >
                        <RefreshCw size={16} /> Reset Filters
                    </button>
                </div>
            </div>

            {/* Filters */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 p-6 rounded-2xl bg-card border">
                <div className="space-y-2">
                    <label className="text-xs font-bold uppercase text-muted-foreground ml-1">Username</label>
                    <div className="relative">
                        <User className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} />
                        <input
                            type="text"
                            placeholder="Search user..."
                            className="w-full h-10 pl-10 rounded-xl bg-muted/30 border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={userFilter}
                            onChange={(e) => { setUserFilter(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-xs font-bold uppercase text-muted-foreground ml-1">Problem Code</label>
                    <div className="relative">
                        <Hash className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} />
                        <input
                            type="text"
                            placeholder="P01, APB..."
                            className="w-full h-10 pl-10 rounded-xl bg-muted/30 border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={problemFilter}
                            onChange={(e) => { setProblemFilter(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-xs font-bold uppercase text-muted-foreground ml-1">Verdict</label>
                    <select
                        className="w-full h-10 px-3 rounded-xl bg-muted/30 border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={resultFilter}
                        onChange={(e) => { setResultFilter(e.target.value); setPage(1); }}
                    >
                        <option value="">All Results</option>
                        {results.map(r => <option key={r} value={r}>{r}</option>)}
                    </select>
                </div>

                <div className="space-y-2">
                    <label className="text-xs font-bold uppercase text-muted-foreground ml-1">Language</label>
                    <input
                        type="text"
                        placeholder="Filter language..."
                        className="w-full h-10 px-3 rounded-xl bg-muted/30 border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={langFilter}
                        onChange={(e) => { setLangFilter(e.target.value); setPage(1); }}
                    />
                </div>
            </div>

            {/* Submission Table */}
            <div className="bg-card rounded-2xl border overflow-hidden shadow-2xl shadow-primary/5">
                <div className="overflow-x-auto">
                    <table className="w-full text-left">
                        <thead>
                            <tr className="bg-muted/30 border-b">
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground w-40">User</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground">Problem</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground text-center w-32">Status</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground text-center">Score</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground text-center w-24">Time</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {isLoading ? (
                                Array.from({ length: 15 }).map((_, i) => (
                                    <tr key={i}>
                                        <td colSpan={6} className="px-6 py-6"><Skeleton className="h-10 w-full rounded-xl" /></td>
                                    </tr>
                                ))
                            ) : (
                                submissions.map((s) => (
                                    <tr key={s.id} className="hover:bg-muted/10 transition-colors">
                                        <td className="px-6 py-4">
                                            <Link href={`/user/${s.user}`} className="flex items-center gap-3 group outline-none">
                                                <div className="w-9 h-9 rounded-xl bg-muted flex items-center justify-center text-muted-foreground font-bold text-xs">
                                                    {s.user[0]?.toUpperCase()}
                                                </div>
                                                <span className="font-bold text-sm">{s.user}</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-4">
                                            <Link href={`/submissions/${s.id}`} className="inline-flex flex-col gap-0.5 outline-none hover:text-primary transition-colors">
                                                <span className="font-bold text-sm tracking-tight">{s.problem}</span>
                                                <span className="text-xs uppercase text-muted-foreground opacity-40">Problem Code</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-4 text-center">
                                            <Badge variant={getStatusVariant(s.status)} className="px-4 py-1.5 rounded-full text-xs font-bold uppercase shadow-sm">
                                                {s.status}
                                            </Badge>
                                        </td>
                                        <td className="px-6 py-4 text-center">
                                            <div className="inline-flex items-center justify-center w-12 h-8 rounded-xl font-bold text-xs border">
                                                {s.score !== null ? Math.round(s.score) : '-'}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 text-center">
                                            {s.time !== null ? (
                                                <span className="text-xs font-mono">{(s.time * 1000).toFixed(0)}ms</span>
                                            ) : (
                                                <span className="text-xs text-muted-foreground">-</span>
                                            )}
                                        </td>
                                        <td className="px-6 py-4 text-right">
                                            <button
                                                onClick={() => {
                                                    if (confirm('Are you sure you want to rejudge this submission?')) {
                                                        api.post(`/admin/submission/${s.id}/rejudge`);
                                                        refetch();
                                                    }
                                                }}
                                                className="px-3 py-1.5 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors text-xs font-bold"
                                            >
                                                Rejudge
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>

                {/* Pagination */}
                {submissions.length > 0 && (
                    <div className="flex items-center justify-between px-6 py-4 border-t bg-muted/30">
                        <div className="text-sm text-muted-foreground">
                            Showing {submissions.length} of {data?.total || 0} submissions
                        </div>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setPage(p => Math.max(1, p - 1))}
                                disabled={page === 1}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-50 transition-all"
                            >
                                <ChevronLeft size={18} />
                            </button>
                            <div className="h-10 px-4 rounded-xl bg-primary text-primary-foreground font-bold text-sm flex items-center shadow-lg shadow-primary/20">
                                {page}
                            </div>
                            <button
                                onClick={() => setPage(p => p + 1)}
                                disabled={submissions.length < 50}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-50 transition-all"
                            >
                                <ChevronRight size={18} />
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
