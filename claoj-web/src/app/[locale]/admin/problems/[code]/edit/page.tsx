'use client';

import { useParams } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { adminOrganizationApi, adminProblemApi, adminUserApi } from '@/lib/adminApi';
import { problemFormToUpdate, problemToFormData } from '@/lib/adminForms';
import { type ProblemFormData } from '@/components/admin/ProblemForm';
import ProblemForm from '@/components/admin/ProblemForm';
import { CloneProblemButton } from '@/components/admin/CloneProblemModal';
import { Code2, ArrowLeft, Database, MessageSquare, BookOpen, SlidersHorizontal, Languages } from 'lucide-react';
import { Link } from '@/navigation';

export default function EditProblemPage() {
    const t = useTranslations('Admin');
    const params = useParams();
    const code = params.code as string;

    const { data: problemData, isLoading: isLoadingProblem } = useQuery({
        queryKey: ['admin-problem-detail', code],
        queryFn: async () => {
            const res = await adminProblemApi.detail(code);
            return res.data;
        }
    });

    // The profile only carries relation ids for these two; fetch labels for the chips.
    const suggesterId = problemData?.suggester_id;
    const { data: suggesterUser } = useQuery({
        queryKey: ['admin-user', suggesterId],
        queryFn: async () => (await adminUserApi.detail(suggesterId ?? 0)).data,
        enabled: !!suggesterId,
    });

    const { data: organizations } = useQuery({
        queryKey: ['admin-organizations-for-problem', problemData?.organization_ids],
        queryFn: async () => (await adminOrganizationApi.list(1, 500)).data.data,
        enabled: !!problemData?.organization_ids?.length,
    });

    const queryClient = useQueryClient();
    const updateMutation = useMutation({
        mutationFn: ({ code, data }: { code: string; data: ProblemFormData }) =>
            adminProblemApi.update(code, problemFormToUpdate(data, problemData)),
        onSuccess: () => {
            toast.success(t('problems.saved'));
            queryClient.invalidateQueries({ queryKey: ['admin-problem-detail', code] });
        },
        onError: (err: unknown) => {
            const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? String(err);
            toast.error(t('problems.saveFailed', { error: msg }));
        },
    });

    const handleSubmit = async (data: ProblemFormData) => {
        await updateMutation.mutateAsync({ code, data });
    };

    if (isLoadingProblem || !problemData) {
        return (
            <div className="flex items-center justify-center py-12">
                <div className="text-muted-foreground">{t('problems.loadingProblem')}</div>
            </div>
        );
    }

    const initialData = problemData ? problemToFormData(problemData) : undefined;
    const initialAuthors = problemData.authors.map(a => ({ id: a.id, label: a.username }));
    const initialOptions = {
        suggester_id: suggesterUser ? [{ id: suggesterUser.id, label: suggesterUser.username }] : undefined,
        curator_ids: problemData.curators.map(u => ({ id: u.id, label: u.username })),
        tester_ids: problemData.testers.map(u => ({ id: u.id, label: u.username })),
        banned_user_ids: problemData.banned_users.map(u => ({ id: u.id, label: u.username })),
        organization_ids: (organizations ?? [])
            .filter(o => problemData.organization_ids.includes(o.id))
            .map(o => ({ id: o.id, label: o.short_name || o.name, detail: o.name })),
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Link
                    href="/admin/problems"
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                >
                    <ArrowLeft size={20} />
                </Link>
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Code2 className="text-primary" size={32} />
                        {t('problems.editProblemTitle')}
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {problemData?.name} ({code})
                    </p>
                </div>
                <Link
                    href={`/admin/problems/${code}/data`}
                    className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                >
                    <Database size={18} />
                    {t('problems.manageDataLink')}
                </Link>
                <Link
                    href={`/admin/problems/${code}/clarifications`}
                    className="px-4 py-2 rounded-xl bg-muted hover:bg-muted/80 transition-colors flex items-center gap-2 font-medium"
                >
                    <MessageSquare size={18} />
                    {t('problems.clarificationsLink')}
                </Link>
                <Link
                    href={`/admin/problems/${code}/editorial`}
                    className="px-4 py-2 rounded-xl bg-muted hover:bg-muted/80 transition-colors flex items-center gap-2 font-medium"
                >
                    <BookOpen size={18} />
                    {t('problems.editorialLink')}
                </Link>
                <Link
                    href={`/admin/problems/${code}/translations`}
                    className="px-4 py-2 rounded-xl bg-muted hover:bg-muted/80 transition-colors flex items-center gap-2 font-medium"
                >
                    <Languages size={18} />
                    {t('problems.translationsLink')}
                </Link>
                <Link
                    href={`/admin/language-limits?problem=${encodeURIComponent(code)}`}
                    className="px-4 py-2 rounded-xl bg-muted hover:bg-muted/80 transition-colors flex items-center gap-2 font-medium"
                >
                    <SlidersHorizontal size={18} />
                    {t('problems.languageLimitsLink')}
                </Link>
                <CloneProblemButton code={code} name={problemData?.name ?? code} />
            </div>

            {/* Form */}
            <ProblemForm
                initialData={initialData}
                initialAuthors={initialAuthors}
                initialOptions={initialOptions}
                onSubmit={handleSubmit}
                isLoading={updateMutation.isPending}
            />
        </div>
    );
}
