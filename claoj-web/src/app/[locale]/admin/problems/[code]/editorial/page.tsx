'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation } from '@tanstack/react-query';
import api from '@/lib/api';
import { adminSolutionApi } from '@/lib/adminApi';
import type { AdminSolutionDetail, SolutionCreateRequest, SolutionUpdateRequest } from '@/types';
import Link from 'next/link';
import CodeEditor from '@/components/ui/CodeEditor';
import LoadingSpinner from '@/components/common/LoadingSpinner';

export default function AdminSolutionEditorPage() {
    const params = useParams();
    const router = useRouter();
    const id = params.id as string;
    const isEdit = id !== 'new';
    const t = useTranslations();

    const [content, setContent] = useState('');
    const [summary, setSummary] = useState('');
    const [isPublic, setIsPublic] = useState(false);
    const [isOfficial, setIsOfficial] = useState(false);
    const [publishOn, setPublishOn] = useState('');
    const [validUntil, setValidUntil] = useState('');
    const [language, setLanguage] = useState('en');
    const [problemCode, setProblemCode] = useState('');

    // Fetch solution if editing
    const { data: solution, isLoading: loadingSolution } = useQuery({
        queryKey: ['admin-solution', id],
        queryFn: () => adminSolutionApi.detail(parseInt(id)),
        enabled: isEdit
    });

    useEffect(() => {
        if (solution?.data) {
            setContent(solution.data.content);
            setSummary(solution.data.summary);
            setIsPublic(solution.data.is_public);
            setIsOfficial(solution.data.is_official);
            setPublishOn(solution.data.publish_on ? new Date(solution.data.publish_on).toISOString().slice(0, 16) : '');
            setValidUntil(solution.data.valid_until ? new Date(solution.data.valid_until).toISOString().slice(0, 16) : '');
            setLanguage(solution.data.language);
            setProblemCode(solution.data.problem_code);
        }
    }, [solution]);

    const createMutation = useMutation({
        mutationFn: (data: SolutionCreateRequest) =>
            adminSolutionApi.create(data),
        onSuccess: () => {
            alert(t('admin.solution_created'));
            router.push('/admin/problems');
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || t('admin.error_creating_solution'));
        }
    });

    const updateMutation = useMutation({
        mutationFn: (data: SolutionUpdateRequest) =>
            adminSolutionApi.update(parseInt(id), data),
        onSuccess: () => {
            alert(t('admin.solution_updated'));
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || t('admin.error_updating_solution'));
        }
    });

    const deleteMutation = useMutation({
        mutationFn: () => adminSolutionApi.delete(parseInt(id)),
        onSuccess: () => {
            if (confirm(t('admin.confirm_delete'))) {
                router.push('/admin/problems');
            }
        }
    });

    const handleSubmit = () => {
        const problemId = solution?.data?.problem_id || 0;
        if (!problemId && !isEdit) {
            alert(t('admin.problem_required'));
            return;
        }

        const data = {
            content,
            summary,
            is_public: isPublic,
            is_official: isOfficial,
            publish_on: publishOn ? new Date(publishOn).toISOString() : undefined,
            valid_until: validUntil ? new Date(validUntil).toISOString() : undefined,
            language
        };

        if (isEdit) {
            updateMutation.mutate(data);
        } else {
            createMutation.mutate({ ...data, problem_id: problemId } as SolutionCreateRequest);
        }
    };

    if (loadingSolution) {
        return (
            <div className="container mx-auto py-8">
                <LoadingSpinner />
            </div>
        );
    }

    return (
        <div className="container mx-auto py-8">
            <div className="mb-6">
                <Link href="/admin/problems" className="text-blue-600 dark:text-blue-400 hover:underline">
                    &larr; {t('admin.back_to_problems')}
                </Link>
                <h1 className="text-3xl font-bold mt-2">
                    {isEdit ? t('admin.edit_solution') : t('admin.create_solution')}
                </h1>
            </div>

            <div className="space-y-6">
                {/* Problem Info */}
                {solution?.data && (
                    <div className="bg-card border rounded-lg p-4">
                        <h3 className="font-semibold mb-2">{t('admin.problem')}</h3>
                        <p className="text-lg">
                            <Link href={`/problems/${solution.data.problem_code}`} className="text-blue-600 dark:text-blue-400 hover:underline">
                                {solution.data.problem_code} - {solution.data.problem_name}
                            </Link>
                        </p>
                    </div>
                )}

                {/* Summary */}
                <div>
                    <label className="block text-sm font-medium mb-2">{t('admin.summary')}</label>
                    <textarea
                        value={summary}
                        onChange={(e) => setSummary(e.target.value)}
                        className="w-full p-3 border rounded-lg bg-card"
                        rows={2}
                        placeholder={t('admin.solution_summary_placeholder')}
                    />
                </div>

                {/* Content */}
                <div>
                    <label className="block text-sm font-medium mb-2">{t('admin.content')}</label>
                    <div className="h-[60vh] border rounded-lg overflow-hidden">
                        <CodeEditor
                            value={content}
                            onChange={(val) => setContent(val || '')}
                            language="markdown"
                        />
                    </div>
                    <p className="text-sm text-muted-foreground mt-2">
                        {t('admin.solution_content_help')}
                    </p>
                </div>

                {/* Settings */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                        <label className="flex items-center gap-2">
                            <input
                                type="checkbox"
                                checked={isPublic}
                                onChange={(e) => setIsPublic(e.target.checked)}
                                className="w-4 h-4"
                            />
                            <span>{t('admin.is_public')}</span>
                        </label>
                    </div>
                    <div>
                        <label className="flex items-center gap-2">
                            <input
                                type="checkbox"
                                checked={isOfficial}
                                onChange={(e) => setIsOfficial(e.target.checked)}
                                className="w-4 h-4"
                            />
                            <span>{t('admin.is_official')}</span>
                        </label>
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1">{t('admin.publish_on')}</label>
                        <input
                            type="datetime-local"
                            value={publishOn}
                            onChange={(e) => setPublishOn(e.target.value)}
                            className="w-full p-2 border rounded-lg bg-card"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1">{t('admin.valid_until')}</label>
                        <input
                            type="datetime-local"
                            value={validUntil}
                            onChange={(e) => setValidUntil(e.target.value)}
                            className="w-full p-2 border rounded-lg bg-card"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1">{t('admin.language')}</label>
                        <select
                            value={language}
                            onChange={(e) => setLanguage(e.target.value)}
                            className="w-full p-2 border rounded-lg bg-card"
                        >
                            <option value="en">English</option>
                            <option value="vi">Tiếng Việt</option>
                        </select>
                    </div>
                </div>

                {/* Actions */}
                <div className="flex gap-4 pt-4 border-t">
                    <button
                        onClick={handleSubmit}
                        disabled={createMutation.isPending || updateMutation.isPending}
                        className="px-6 py-2 bg-primary text-primary-foreground rounded-lg hover:opacity-90 disabled:opacity-50"
                    >
                        {isEdit ? t('admin.save') : t('admin.create')}
                    </button>
                    {isEdit && (
                        <button
                            onClick={() => deleteMutation.mutate()}
                            className="px-6 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
                        >
                            {t('admin.delete')}
                        </button>
                    )}
                </div>
            </div>
        </div>
    );
}
