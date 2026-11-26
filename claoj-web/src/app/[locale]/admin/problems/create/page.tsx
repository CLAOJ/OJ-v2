'use client';

import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminProblemApi } from '@/lib/adminApi';
import { problemFormToCreate } from '@/lib/adminForms';
import { type ProblemFormData } from '@/components/admin/ProblemForm';
import ProblemForm from '@/components/admin/ProblemForm';
import { Code2, ArrowLeft } from 'lucide-react';
import { Link } from '@/navigation';
import { toast } from 'sonner';
import { apiErrorMessage } from '@/lib/api';
import { useAuth } from '@/components/providers/AuthProvider';

export default function CreateProblemPage() {
    const t = useTranslations('Admin');
    const router = useRouter();
    const { user } = useAuth();
    const meId = user?.profile?.id;

    const createMutation = useMutation({
        mutationFn: (data: ProblemFormData) => adminProblemApi.create(problemFormToCreate(data)),
        onSuccess: (response) => {
            if (response.data.problem?.code) {
                router.push(`/admin/problems`);
            }
        },
        onError: (err) => toast.error(apiErrorMessage(err, t('problems.createError'))),
    });

    const handleSubmit = async (data: ProblemFormData) => {
        await createMutation.mutateAsync(data);
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex flex-wrap items-center gap-4">
                <Link
                    href="/admin/problems"
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                >
                    <ArrowLeft size={20} />
                </Link>
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Code2 className="text-primary" size={32} />
                        {t('problems.createTitle')}
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {t('problems.createSubtitle')}
                    </p>
                </div>
            </div>

            {/* Form */}
            {/* The creator starts as an author; picking adds others. Remount once
                the session has loaded so the default applies. */}
            <ProblemForm
                key={meId ?? 0}
                initialAuthors={meId && user ? [{ id: meId, label: user.username }] : undefined}
                onSubmit={handleSubmit}
                isLoading={createMutation.isPending}
            />
        </div>
    );
}
