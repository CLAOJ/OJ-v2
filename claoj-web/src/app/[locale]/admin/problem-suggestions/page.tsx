'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { problemSuggestionApi } from '@/lib/api';
import { toast } from 'sonner';
import type { ProblemSuggestionAdmin } from '@/types';

export default function AdminProblemSuggestionsPage() {
    const router = useRouter();
    const [suggestions, setSuggestions] = useState<ProblemSuggestionAdmin[]>([]);
    const [loading, setLoading] = useState(true);
    const [statusFilter, setStatusFilter] = useState<string>('pending');
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);

    const fetchSuggestions = async () => {
        setLoading(true);
        try {
            const response = await problemSuggestionApi.listSuggestions(page, 20, statusFilter || undefined);
            setSuggestions(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error('Failed to fetch suggestions');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchSuggestions();
    }, [page, statusFilter]);

    const handleApprove = async (id: number) => {
        const code = prompt('Enter the final problem code (e.g., PROB123):');
        if (!code) return;

        try {
            await problemSuggestionApi.approveSuggestion(id, {
                code,
                is_public: true,
            });
            alert('Problem suggestion approved!');
            fetchSuggestions();
        } catch (error: any) {
            alert('Failed to approve: ' + (error.response?.data?.message || 'Unknown error'));
        }
    };

    const handleReject = async (id: number) => {
        const reason = prompt('Enter rejection reason:');
        if (!reason) return;

        try {
            await problemSuggestionApi.rejectSuggestion(id, { reason });
            alert('Problem suggestion rejected.');
            fetchSuggestions();
        } catch (error: any) {
            alert('Failed to reject: ' + (error.response?.data?.message || 'Unknown error'));
        }
    };

    const handleDelete = async (id: number) => {
        if (!confirm('Are you sure you want to delete this suggestion?')) return;

        try {
            await problemSuggestionApi.deleteSuggestion(id);
            alert('Problem suggestion deleted.');
            fetchSuggestions();
        } catch (error: any) {
            alert('Failed to delete: ' + (error.response?.data?.message || 'Unknown error'));
        }
    };

    const getStatusBadgeClass = (status: string) => {
        switch (status) {
            case 'pending':
                return 'bg-yellow-100 text-yellow-800';
            case 'approved':
                return 'bg-green-100 text-green-800';
            case 'rejected':
                return 'bg-red-100 text-red-800';
            default:
                return 'bg-gray-100 text-gray-800';
        }
    };

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-8">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                <div className="mb-8">
                    <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
                        Problem Suggestions
                    </h1>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">
                        Review and manage problem suggestions from users
                    </p>
                </div>

                {/* Filters */}
                <div className="mb-6 flex items-center gap-4">
                    <select
                        value={statusFilter}
                        onChange={(e) => {
                            setStatusFilter(e.target.value);
                            setPage(1);
                        }}
                        className="px-4 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-700"
                    >
                        <option value="pending">Pending</option>
                        <option value="approved">Approved</option>
                        <option value="rejected">Rejected</option>
                        <option value="">All</option>
                    </select>
                </div>

                {/* Table */}
                {loading ? (
                    <div className="flex justify-center py-12">
                        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
                    </div>
                ) : suggestions.length === 0 ? (
                    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-8 text-center">
                        <p className="text-gray-500">No suggestions found.</p>
                    </div>
                ) : (
                    <>
                        <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                                <thead className="bg-gray-50 dark:bg-gray-700">
                                    <tr>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                            ID
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                            Name
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                            Code
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                            Suggester
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                            Status
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                            Date
                                        </th>
                                        <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                            Actions
                                        </th>
                                    </tr>
                                </thead>
                                <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                    {suggestions.map((suggestion) => (
                                        <tr key={suggestion.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                                                #{suggestion.id}
                                            </td>
                                            <td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-100">
                                                {suggestion.name}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-600 dark:text-gray-400">
                                                {suggestion.code}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                                                {suggestion.suggester_username || 'Unknown'}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusBadgeClass(suggestion.suggestion_status)}`}>
                                                    {suggestion.suggestion_status}
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                                                {suggestion.date ? new Date(suggestion.date).toLocaleDateString() : '-'}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                                <button
                                                    onClick={() => router.push(`/admin/problem-suggestions/${suggestion.id}`)}
                                                    className="text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300 mr-3"
                                                >
                                                    View
                                                </button>
                                                {suggestion.suggestion_status === 'pending' && (
                                                    <>
                                                        <button
                                                            onClick={() => handleApprove(suggestion.id)}
                                                            className="text-green-600 hover:text-green-900 dark:text-green-400 dark:hover:text-green-300 mr-3"
                                                        >
                                                            Approve
                                                        </button>
                                                        <button
                                                            onClick={() => handleReject(suggestion.id)}
                                                            className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300 mr-3"
                                                        >
                                                            Reject
                                                        </button>
                                                    </>
                                                )}
                                                {suggestion.suggestion_status !== 'approved' && (
                                                    <button
                                                        onClick={() => handleDelete(suggestion.id)}
                                                        className="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-300"
                                                    >
                                                        Delete
                                                    </button>
                                                )}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>

                        {/* Pagination */}
                        <div className="mt-6 flex items-center justify-between">
                            <p className="text-sm text-gray-600 dark:text-gray-400">
                                Total: {total} suggestions
                            </p>
                            <div className="flex gap-2">
                                <button
                                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                                    disabled={page === 1}
                                    className="px-4 py-2 border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed dark:border-gray-700"
                                >
                                    Previous
                                </button>
                                <span className="px-4 py-2 text-gray-600 dark:text-gray-400">
                                    Page {page}
                                </span>
                                <button
                                    onClick={() => setPage((p) => p + 1)}
                                    disabled={suggestions.length < 20}
                                    className="px-4 py-2 border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed dark:border-gray-700"
                                >
                                    Next
                                </button>
                            </div>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}
