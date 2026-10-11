'use client';

import { useParams } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { adminProblemApi } from '@/lib/adminApi';
import { problemToFormData } from '@/lib/adminForms';
import { type ProblemFormData } from '@/components/admin/ProblemForm';
import ProblemForm from '@/components/admin/ProblemForm';
import { Code2, ArrowLeft, Database, MessageSquare } from 'lucide-react';
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

    const queryClient = useQueryClient();
    const updateMutation = useMutation({
        mutationFn: ({ code, data }: { code: string; data: ProblemFormData }) =>
            adminProblemApi.update(code, data),
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
            </div>

            {/* Form */}
            <ProblemForm
                initialData={initialData}
                initialAuthors={initialAuthors}
                onSubmit={handleSubmit}
                isLoading={updateMutation.isPending}
            />
        </div>
    );
}
