'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { solutionApi } from '@/lib/api';
import type { Solution } from '@/types';
import Link from 'next/link';
import LoadingSpinner from '@/components/common/LoadingSpinner';
import Markdown from '@/components/common/Markdown';

export default function ProblemEditorialPage() {
    const params = useParams();
    const t = useTranslations();
    const code = params.code as string;

    const [solution, setSolution] = useState<Solution | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        async function fetchSolution() {
            try {
                const res = await solutionApi.getSolution(code);
                setSolution(res.data);
            } catch (err: unknown) {
                if (err && typeof err === 'object' && 'response' in err) {
                    const axiosError = err as { response?: { status?: number } };
                    if (axiosError.response?.status === 404) {
                        setError(t('problem.editorial_not_found'));
                    } else if (axiosError.response?.status === 403) {
                        setError(t('problem.editorial_not_public'));
                    } else {
                        setError(t('common.error_loading_data'));
                    }
                } else {
                    setError(t('common.error_loading_data'));
                }
            } finally {
                setLoading(false);
            }
        }

        fetchSolution();
    }, [code, t]);

    if (loading) {
        return (
            <div className="container mx-auto py-8">
                <LoadingSpinner />
            </div>
        );
    }

    if (error || !solution) {
        return (
            <div className="container mx-auto py-8">
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6 text-center">
                    <h2 className="text-xl font-semibold text-red-800 dark:text-red-200 mb-2">
                        {error || t('problem.editorial_not_found')}
                    </h2>
                    <Link
                        href={`/problems/${code}`}
                        className="text-blue-600 dark:text-blue-400 hover:underline"
                    >
                        {t('common.back_to_problem')}
                    </Link>
                </div>
            </div>
        );
    }

    return (
        <div className="container mx-auto py-8">
            {/* Header */}
            <div className="mb-6">
                <Link
                    href={`/problems/${code}`}
                    className="text-blue-600 dark:text-blue-400 hover:underline mb-4 inline-block"
                >
                    &larr; {t('common.back_to_problem')}
                </Link>
                <h1 className="text-3xl font-bold mb-2">
                    {t('problem.editorial')}
                </h1>
                {solution.summary && (
                    <p className="text-gray-600 dark:text-gray-400">
                        {solution.summary}
                    </p>
                )}
            </div>

            {/* Authors */}
            {solution.authors && solution.authors.length > 0 && (
                <div className="mb-6 p-4 bg-gray-100 dark:bg-gray-800 rounded-lg">
                    <span className="font-semibold mr-2">{t('problem.authors')}:</span>
                    {solution.authors.map((author, index) => (
                        <span key={author.id}>
                            {index > 0 && ', '}
                            <Link
                                href={`/user/${author.username}`}
                                className="text-blue-600 dark:text-blue-400 hover:underline"
                            >
                                {author.username}
                            </Link>
                        </span>
                    ))}
                    {solution.is_official && (
                        <span className="ml-4 px-2 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 text-sm rounded">
                            {t('problem.official_solution')}
                        </span>
                    )}
                </div>
            )}

            {/* Content */}
            <div className="prose prose-lg dark:prose-invert max-w-none">
                <Markdown content={solution.content} />
            </div>

            {/* Footer */}
            <div className="mt-8 pt-6 border-t border-gray-200 dark:border-gray-700">
                {solution.publish_on && (
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                        {t('problem.published_on')}: {new Date(solution.publish_on).toLocaleDateString()}
                    </p>
                )}
            </div>
        </div>
    );
}
