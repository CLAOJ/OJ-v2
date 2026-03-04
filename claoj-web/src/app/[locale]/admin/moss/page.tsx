'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { adminSubmissionApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import {
    Search,
    FileText,
    CheckCircle,
    Clock,
    XCircle,
    BarChart3,
    ExternalLink
} from 'lucide-react';

interface MossResult {
    id: number;
    submission_id: number;
    problem_code: string;
    problem_name: string;
    username: string;
    language: string;
    match_count: number;
    max_similarity: number;
    status: string;
    created_at: string;
    moss_url?: string;
}

export default function AdminMossPage() {
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');

    const queryClient = useQueryClient();

    const { data, isLoading } = useQuery({
        queryKey: ['admin-moss-results', page, search],
        queryFn: async () => {
            const res = await api.get<{
                data: MossResult[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/moss/results?page=${page}&page_size=50&search=${search}`);
            return res.data;
        }
    });

    const analyzeMutation = useMutation({
        mutationFn: ({ submissionId, language }: { submissionId: number; language: string }) =>
            adminSubmissionApi.mossAnalyze(submissionId, language),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-moss-results'] });
        }
    });

    const mossResults = data?.data || [];

    const filteredResults = mossResults.filter(r =>
        r.username.toLowerCase().includes(search.toLowerCase()) ||
        r.problem_code.toLowerCase().includes(search.toLowerCase())
    );

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'completed':
                return <CheckCircle size={18} className="text-success" />;
            case 'processing':
                return <Clock size={18} className="text-amber-500 animate-spin" />;
            case 'failed':
                return <XCircle size={18} className="text-destructive" />;
            default:
                return <Clock size={18} className="text-muted-foreground" />;
        }
    };

    const getSimilarityColor = (percentage: number) => {
        if (percentage >= 80) return 'text-destructive font-bold';
        if (percentage >= 50) return 'text-amber-500 font-medium';
        return 'text-success';
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <BarChart3 className="text-primary" size={32} />
                        MOSS Results
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        Plagiarism detection results for submissions
                    </p>
                </div>

                <div className="relative w-full md:w-80">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                    <input
                        type="text"
                        placeholder="Search by username or problem..."
                        className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                    />
                </div>
            </div>

            {/* Info Card */}
            <div className="bg-primary/5 border border-primary/20 rounded-2xl p-4">
                <div className="flex items-start gap-3">
                    <FileText className="text-primary mt-0.5" size={20} />
                    <div className="text-sm">
                        <p className="font-medium text-primary mb-1">About MOSS</p>
                        <p className="text-muted-foreground">
                            MOSS (Measure Of Software Similarity) is a system that automatically determines
                            the similarity of computer programs. It supports multiple programming languages
                            and is widely used for detecting plagiarism in programming assignments.
                        </p>
                    </div>
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-24 rounded-2xl" />)}
                </div>
            ) : (
                <div className="bg-card rounded-2xl border overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left">
                            <thead className="bg-muted/50 border-b">
                                <tr>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Submission</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">User</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Matches</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Max Similarity</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Status</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground text-right">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {filteredResults.length === 0 ? (
                                    <tr>
                                        <td colSpan={6} className="px-6 py-12 text-center text-muted-foreground">
                                            <div className="flex flex-col items-center gap-2">
                                                <FileText size={48} className="opacity-20" />
                                                <p>No MOSS results found</p>
                                            </div>
                                        </td>
                                    </tr>
                                ) : (
                                    filteredResults.map((result) => (
                                        <tr key={result.id} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col gap-1">
                                                    <Link
                                                        href={`/submissions/${result.submission_id}`}
                                                        className="font-medium text-sm hover:text-primary transition-colors"
                                                    >
                                                        {result.problem_code}
                                                    </Link>
                                                    <span className="text-xs text-muted-foreground">
                                                        {result.language}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <Link
                                                    href={`/user/${result.username}`}
                                                    className="text-sm hover:text-primary transition-colors"
                                                >
                                                    {result.username}
                                                </Link>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-2">
                                                    <FileText size={16} className="text-muted-foreground" />
                                                    <span className="text-sm">{result.match_count} matches</span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <span className={getSimilarityColor(result.max_similarity)}>
                                                    {result.max_similarity.toFixed(1)}%
                                                </span>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-2">
                                                    {getStatusIcon(result.status)}
                                                    <span className="text-sm capitalize">{result.status}</span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    {result.moss_url && (
                                                        <a
                                                            href={result.moss_url}
                                                            target="_blank"
                                                            rel="noopener noreferrer"
                                                            className="px-3 py-1.5 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 flex items-center gap-2 text-sm font-medium transition-colors"
                                                        >
                                                            <ExternalLink size={14} />
                                                            View Report
                                                        </a>
                                                    )}
                                                    {!result.moss_url && result.status === 'completed' && (
                                                        <span className="text-sm text-muted-foreground">
                                                            No report URL
                                                        </span>
                                                    )}
                                                    {result.status === 'pending' && (
                                                        <button
                                                            onClick={() => analyzeMutation.mutate({
                                                                submissionId: result.submission_id,
                                                                language: result.language
                                                            })}
                                                            disabled={analyzeMutation.isPending}
                                                            className="px-3 py-1.5 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 flex items-center gap-2 text-sm font-medium transition-colors disabled:opacity-50"
                                                        >
                                                            <BarChart3 size={14} />
                                                            Analyze
                                                        </button>
                                                    )}
                                                </div>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>

                    {/* Pagination */}
                    {filteredResults.length > 0 && (
                        <div className="flex items-center justify-between px-6 py-4 border-t bg-muted/30">
                            <div className="text-sm text-muted-foreground">
                                Showing {filteredResults.length} of {data?.total || 0} results
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
                                    disabled={filteredResults.length < 50}
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
