'use client';

import { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { problemSuggestionApi } from '@/lib/api';
import type { ProblemSuggestionDetail } from '@/types';

export default function AdminProblemSuggestionDetailPage() {
    const router = useRouter();
    const params = useParams();
    const id = params.id as string;

    const [suggestion, setSuggestion] = useState<ProblemSuggestionDetail | null>(null);
    const [loading, setLoading] = useState(true);
    const [actionLoading, setActionLoading] = useState(false);

    useEffect(() => {
        const fetchSuggestion = async () => {
            try {
                const response = await problemSuggestionApi.getSuggestion(parseInt(id));
                setSuggestion(response.data);
            } catch (error) {
                // Failed to fetch suggestion - will show error state
            } finally {
                setLoading(false);
            }
        };

        fetchSuggestion();
    }, [id]);

    const handleApprove = async () => {
        const code = prompt('Enter the final problem code (e.g., PROB123):');
        if (!code) return;

        const adminNotes = prompt('Enter admin notes (optional):') || '';
        const isPublic = confirm('Make the problem public immediately?');

        setActionLoading(true);
        try {
            await problemSuggestionApi.approveSuggestion(parseInt(id), {
                code,
                admin_notes: adminNotes,
                is_public: isPublic,
            });
            alert('Problem suggestion approved!');
            router.push('/admin/problem-suggestions');
        } catch (error: any) {
            alert('Failed to approve: ' + (error.response?.data?.message || 'Unknown error'));
        } finally {
            setActionLoading(false);
        }
    };

    const handleReject = async () => {
        const reason = prompt('Enter rejection reason:');
        if (!reason) return;

        const adminNotes = prompt('Enter admin notes (optional):') || '';

        setActionLoading(true);
        try {
            await problemSuggestionApi.rejectSuggestion(parseInt(id), {
                reason,
                admin_notes: adminNotes,
            });
            alert('Problem suggestion rejected.');
            router.push('/admin/problem-suggestions');
        } catch (error: any) {
            alert('Failed to reject: ' + (error.response?.data?.message || 'Unknown error'));
        } finally {
            setActionLoading(false);
        }
    };

    const handleDelete = async () => {
        if (!confirm('Are you sure you want to delete this suggestion?')) return;

        setActionLoading(true);
        try {
            await problemSuggestionApi.deleteSuggestion(parseInt(id));
            alert('Problem suggestion deleted.');
            router.push('/admin/problem-suggestions');
        } catch (error: any) {
            alert('Failed to delete: ' + (error.response?.data?.message || 'Unknown error'));
        } finally {
            setActionLoading(false);
        }
    };

    if (loading) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        );
    }

    if (!suggestion) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <p className="text-gray-500">Suggestion not found</p>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-8">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                {/* Header */}
                <div className="mb-8">
                    <div className="flex items-center justify-between">
                        <div>
                            <button
                                onClick={() => router.push('/admin/problem-suggestions')}
                                className="text-blue-600 hover:text-blue-900 dark:text-blue-400 mb-2"
                            >
                                &larr; Back to Suggestions
                            </button>
                            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
                                {suggestion.name}
                            </h1>
                            <p className="text-gray-500 dark:text-gray-400">
                                Code: <span className="font-mono">{suggestion.code}</span>
                            </p>
                        </div>
                        <div className="flex gap-2">
                            {suggestion.suggestion_status === 'pending' && (
                                <>
                                    <button
                                        onClick={handleApprove}
                                        disabled={actionLoading}
                                        className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50"
                                    >
                                        Approve
                                    </button>
                                    <button
                                        onClick={handleReject}
                                        disabled={actionLoading}
                                        className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
                                    >
                                        Reject
                                    </button>
                                </>
                            )}
                            {suggestion.suggestion_status !== 'approved' && (
                                <button
                                    onClick={handleDelete}
                                    disabled={actionLoading}
                                    className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 disabled:opacity-50"
                                >
                                    Delete
                                </button>
                            )}
                        </div>
                    </div>

                    <div className="mt-4 flex gap-4">
                        <span className={`px-3 py-1 text-sm font-medium rounded-full ${
                            suggestion.suggestion_status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                            suggestion.suggestion_status === 'approved' ? 'bg-green-100 text-green-800' :
                            'bg-red-100 text-red-800'
                        }`}>
                            {suggestion.suggestion_status.toUpperCase()}
                        </span>
                        {suggestion.is_public && (
                            <span className="px-3 py-1 text-sm font-medium rounded-full bg-blue-100 text-blue-800">
                                PUBLIC
                            </span>
                        )}
                    </div>
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    {/* Main Content */}
                    <div className="lg:col-span-2 space-y-6">
                        {/* Problem Details */}
                        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                            <h2 className="text-lg font-semibold mb-4">Problem Details</h2>

                            <dl className="grid grid-cols-2 gap-4">
                                <div>
                                    <dt className="text-sm font-medium text-gray-500">Points</dt>
                                    <dd className="text-gray-900 dark:text-gray-100">{suggestion.points}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-gray-500">Time Limit</dt>
                                    <dd className="text-gray-900 dark:text-gray-100">{suggestion.time_limit}s</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-gray-500">Memory Limit</dt>
                                    <dd className="text-gray-900 dark:text-gray-100">{suggestion.memory_limit} MB</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-gray-500">Group</dt>
                                    <dd className="text-gray-900 dark:text-gray-100">{suggestion.group}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-gray-500">Partial Scoring</dt>
                                    <dd className="text-gray-900 dark:text-gray-100">{suggestion.partial ? 'Yes' : 'No'}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-gray-500">Full Markup</dt>
                                    <dd className="text-gray-900 dark:text-gray-100">{suggestion.is_full_markup ? 'Yes' : 'No'}</dd>
                                </div>
                            </dl>

                            {suggestion.types && suggestion.types.length > 0 && (
                                <div className="mt-4">
                                    <dt className="text-sm font-medium text-gray-500">Types</dt>
                                    <dd className="flex flex-wrap gap-2 mt-1">
                                        {suggestion.types.map((type, i) => (
                                            <span key={i} className="px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded text-sm">
                                                {type.name}
                                            </span>
                                        ))}
                                    </dd>
                                </div>
                            )}
                        </div>

                        {/* Description */}
                        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                            <h2 className="text-lg font-semibold mb-4">Description</h2>
                            <div className="prose dark:prose-invert max-w-none" dangerouslySetInnerHTML={{ __html: suggestion.description }} />
                        </div>
                    </div>

                    {/* Sidebar */}
                    <div className="space-y-6">
                        {/* Suggester Info */}
                        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                            <h2 className="text-lg font-semibold mb-4">Suggester Information</h2>

                            <dl className="space-y-3">
                                <div>
                                    <dt className="text-sm font-medium text-gray-500">Username</dt>
                                    <dd className="text-gray-900 dark:text-gray-100">{suggestion.suggester_username || 'Unknown'}</dd>
                                </div>
                                <div>
                                    <dt className="text-sm font-medium text-gray-500">Email</dt>
                                    <dd className="text-gray-900 dark:text-gray-100">{suggestion.suggester_email || 'Hidden'}</dd>
                                </div>
                            </dl>
                        </div>

                        {/* Review Info */}
                        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                            <h2 className="text-lg font-semibold mb-4">Review Information</h2>

                            <dl className="space-y-3">
                                <div>
                                    <dt className="text-sm font-medium text-gray-500">Status</dt>
                                    <dd className="text-gray-900 dark:text-gray-100 capitalize">{suggestion.suggestion_status}</dd>
                                </div>
                                {suggestion.suggestion_reviewed_at && (
                                    <div>
                                        <dt className="text-sm font-medium text-gray-500">Reviewed At</dt>
                                        <dd className="text-gray-900 dark:text-gray-100">
                                            {new Date(suggestion.suggestion_reviewed_at).toLocaleString()}
                                        </dd>
                                    </div>
                                )}
                                {suggestion.suggestion_notes && (
                                    <div>
                                        <dt className="text-sm font-medium text-gray-500">Notes</dt>
                                        <dd className="text-gray-900 dark:text-gray-100 text-sm whitespace-pre-wrap">{suggestion.suggestion_notes}</dd>
                                    </div>
                                )}
                            </dl>
                        </div>

                        {/* Source & Summary */}
                        {suggestion.source && (
                            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                                <h2 className="text-lg font-semibold mb-4">Source</h2>
                                <p className="text-gray-900 dark:text-gray-100">{suggestion.source}</p>
                            </div>
                        )}

                        {suggestion.summary && (
                            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                                <h2 className="text-lg font-semibold mb-4">Summary</h2>
                                <p className="text-gray-900 dark:text-gray-100">{suggestion.summary}</p>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
