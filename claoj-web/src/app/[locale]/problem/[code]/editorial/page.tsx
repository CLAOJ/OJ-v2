'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { solutionApi } from '@/lib/api';
import api from '@/lib/api';
import type { Solution } from '@/types';
import Link from 'next/link';
import LoadingSpinner from '@/components/common/LoadingSpinner';
import Markdown from '@/components/common/Markdown';
import Comments from '@/components/common/Comments';
import dayjs from 'dayjs';

export default function ProblemEditorialPage() {
    const params = useParams();
    const t = useTranslations();
    const code = params.code as string;

    const [solution, setSolution] = useState<Solution | null>(null);
    const [problemName, setProblemName] = useState<string>('');
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    // 404/403 are ordinary states (no editorial yet, or not published), not failures.
    const [unavailable, setUnavailable] = useState(false);

    useEffect(() => {
        async function fetchSolution() {
            try {
                const res = await solutionApi.getSolution(code);
                setSolution(res.data);

                // Fetch problem detail to get problem name
                const problemRes = await api.get<{ name: string }>(`/problem/${code}`);
                setProblemName(problemRes.data.name);
            } catch (err: unknown) {
                if (err && typeof err === 'object' && 'response' in err) {
                    const axiosError = err as { response?: { status?: number } };
                    if (axiosError.response?.status === 404) {
                        setUnavailable(true);
                        setError(t('problem.editorial_not_found'));
                    } else if (axiosError.response?.status === 403) {
                        setUnavailable(true);
                        setError(t('problem.editorial_not_public'));
                    } else {
                        setError(t('Common.error_loading_data'));
                    }
                } else {
                    setError(t('Common.error_loading_data'));
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
                <div className={unavailable || !error
                    ? "rounded-lg border bg-card p-6 text-center"
                    : "rounded-lg border border-destructive/30 bg-destructive/5 p-6 text-center"}>
                    <h2 className={unavailable || !error ? "text-lg font-semibold mb-2" : "text-lg font-semibold text-destructive mb-2"}>
                        {error || t('problem.editorial_not_found')}
                    </h2>
                    <Link
                        href={`/problem/${code}`}
                        className="text-primary hover:underline underline-offset-4"
                    >
                        {t('Common.back_to_problem')}
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
                    href={`/problem/${code}`}
                    className="text-primary hover:underline underline-offset-4 mb-4 inline-block"
                >
                    &larr; {t('Common.back_to_problem')}
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
                        {t('problem.published_on')}: {dayjs(solution.publish_on).format('L')}
                    </p>
                )}
            </div>

            {/* Comments Section */}
            <div className="mt-12">
                <Comments page={`s:${code}`} problemName={problemName || code} contextType="editorial" />
            </div>
        </div>
    );
}
