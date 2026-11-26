'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useConfirm } from '@/components/providers/ConfirmProvider';
import { useQuery, useMutation } from '@tanstack/react-query';
import api from '@/lib/api';
import { toast } from 'sonner';
import { adminSolutionApi } from '@/lib/adminApi';
import type { AdminProblemDetail, SolutionCreateRequest, SolutionUpdateRequest } from '@/types';
import Link from 'next/link';
import CodeEditor from '@/components/ui/CodeEditor';
import LoadingSpinner from '@/components/common/LoadingSpinner';
import { toLocalInput } from '@/lib/adminForms';

export default function AdminSolutionEditorPage() {
    const params = useParams();
    const router = useRouter();
    // The route is /admin/problems/<code>/editorial: the page edits that
    // problem's editorial, or creates it when the problem has none yet.
    const code = params.code as string;
    const t = useTranslations();
    const confirm = useConfirm();

    const [content, setContent] = useState('');
    const [summary, setSummary] = useState('');
    const [isPublic, setIsPublic] = useState(false);
    const [isOfficial, setIsOfficial] = useState(false);
    const [publishOn, setPublishOn] = useState('');
    const [validUntil, setValidUntil] = useState('');
    const [language, setLanguage] = useState('en');

    const { data: problem, isLoading: loadingProblem } = useQuery({
        queryKey: ['admin-problem', code],
        queryFn: async () => (await api.get<AdminProblemDetail>(`/admin/problem/${code}`)).data,
    });
    const { data: existingId, isLoading: loadingLookup } = useQuery({
        queryKey: ['admin-solution-for', code],
        queryFn: async () => (await adminSolutionApi.forProblem(code)).data.data[0]?.id ?? null,
    });
    const id = existingId ?? null;
    const isEdit = id !== null;
    const { data: solution, isLoading: loadingDetail } = useQuery({
        queryKey: ['admin-solution', id],
        queryFn: () => adminSolutionApi.detail(id as number),
        enabled: isEdit,
    });
    const loadingSolution = loadingProblem || loadingLookup || (isEdit && loadingDetail);

    // Load the fetched editorial into the form once per fetch (adjusted during render).
    const [loadedSolution, setLoadedSolution] = useState<typeof solution>(undefined);
    if (solution !== loadedSolution) {
        setLoadedSolution(solution);
        if (solution?.data) {
            setContent(solution.data.content);
            setSummary(solution.data.summary);
            setIsPublic(solution.data.is_public);
            setIsOfficial(solution.data.is_official);
            setPublishOn(solution.data.publish_on ? toLocalInput(solution.data.publish_on) : '');
            setValidUntil(solution.data.valid_until ? toLocalInput(solution.data.valid_until) : '');
            setLanguage(solution.data.language);
        }
    }

    const createMutation = useMutation({
        mutationFn: (data: SolutionCreateRequest) =>
            adminSolutionApi.create(data),
        onSuccess: () => {
            toast.success(t('admin.solution_created'));
            router.push('/admin/problems');
        },
        onError: (err: { response?: { data?: { error?: string } } }) => {
            toast.error(err.response?.data?.error || t('admin.error_creating_solution'));
        }
    });

    const updateMutation = useMutation({
        mutationFn: (data: SolutionUpdateRequest) =>
            adminSolutionApi.update(id as number, data),
        onSuccess: () => {
            toast.success(t('admin.solution_updated'));
        },
        onError: (err: { response?: { data?: { error?: string } } }) => {
            toast.error(err.response?.data?.error || t('admin.error_updating_solution'));
        }
    });

    const deleteMutation = useMutation({
        mutationFn: () => adminSolutionApi.delete(id as number),
        // The question is asked before deleting (below); it used to be asked
        // only after the editorial was already gone.
        onSuccess: () => router.push('/admin/problems'),
    });

    const handleSubmit = () => {
        const problemId = solution?.data?.problem_id || problem?.id || 0;
        if (!problemId && !isEdit) {
            toast.error(t('admin.problem_required'));
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
                <Link href="/admin/problems" className="text-sm text-muted-foreground hover:text-foreground hover:underline underline-offset-4">
                    &larr; {t('admin.back_to_problems')}
                </Link>
                <h1 className="text-3xl font-bold mt-2">
                    {isEdit ? t('admin.edit_solution') : t('admin.create_solution')}
                </h1>
            </div>

            <div className="space-y-6">
                {/* Problem Info */}
                {problem && (
                    <p className="text-sm text-muted-foreground">
                        {t('admin.problem')}:{' '}
                        <Link href={`/problem/${problem.code}`} className="font-medium text-foreground hover:text-primary hover:underline underline-offset-4">
                            {problem.code} – {problem.name}
                        </Link>
                    </p>
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
                            onClick={async () => {
                                if (await confirm(t('admin.confirm_delete'), { variant: 'danger' })) deleteMutation.mutate();
                            }}
                            className="px-6 py-2 bg-destructive text-white rounded-lg hover:bg-destructive"
                        >
                            {t('admin.delete')}
                        </button>
                    )}
                </div>
            </div>
        </div>
    );
}
