'use client';

import { useParams, useRouter } from 'next/navigation';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { isAxiosError } from 'axios';
import { ArrowLeft, Code2, Loader2, Shield } from 'lucide-react';
import { toast } from 'sonner';
import api from '@/lib/api';
import { Link } from '@/navigation';
import ProblemForm, { type ProblemFormData } from '@/components/admin/ProblemForm';
import { problemFormToCreate } from '@/lib/adminForms';
import { parseLeadingId } from '@/utils/route';
import { useAuth } from '@/components/providers/AuthProvider';
import { hasPerm } from '@/lib/permissions';
import type { OrganizationDetail } from '@/types';

/** v1 ProblemCreateOrganization: org admins create a problem inside the org. */
export default function OrganizationProblemCreatePage() {
    const { id: idParam } = useParams<{ id: string }>();
    const id = parseLeadingId(idParam);
    const t = useTranslations('Organization');
    const tCommon = useTranslations('Common');
    const router = useRouter();
    const { user } = useAuth();

    const { data: org, isLoading } = useQuery({
        queryKey: ['organization', id],
        queryFn: async () => {
            const res = await api.get<OrganizationDetail>(`/organization/${id}`);
            return res.data;
        }
    });

    const createMutation = useMutation({
        mutationFn: async (data: ProblemFormData) => {
            const res = await api.post(`/organization/${id}/problems`, problemFormToCreate(data));
            return res.data as { problem: { code: string } };
        },
        onSuccess: (data) => {
            toast.success(t('created'));
            router.push(`/problem/${data.problem.code}`);
        },
        onError: (error) => {
            toast.error(isAxiosError(error) ? error.response?.data?.error : tCommon('error'));
        }
    });

    const allowed = hasPerm(user, 'judge.create_organization_problem')
        && (!!org?.viewer_is_admin || hasPerm(user, 'judge.edit_all_problem'));

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-20">
                <Loader2 size={32} className="animate-spin text-muted-foreground" />
            </div>
        );
    }

    if (!org) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <h2 className="text-2xl font-semibold mb-2">{t('notFound')}</h2>
                <Link href="/organizations" className="text-primary hover:underline">{t('backToOrganizations')}</Link>
            </div>
        );
    }

    if (!allowed) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <Shield size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-semibold mb-2">{t('accessDenied')}</h2>
                <p className="text-muted-foreground">{t('createDenied')}</p>
            </div>
        );
    }

    const codePrefix = org.slug.toLowerCase().replace(/[^a-z]/g, '') + '_';

    return (
        <div className="max-w-5xl mx-auto space-y-6 py-8">
            <div className="flex items-center gap-4">
                <Link
                    href={`/organization/${org.id}-${org.slug}`}
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                >
                    <ArrowLeft size={20} />
                </Link>
                <div>
                    <h1 className="text-3xl font-semibold flex items-center gap-3">
                        <Code2 className="text-primary" size={32} />
                        {t('newProblem')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{org.name}</p>
                </div>
            </div>

            <ProblemForm
                initialData={{
                    code: codePrefix,
                    name: '',
                    description: '',
                    points: 100,
                    partial: true,
                    is_public: false,
                    time_limit: 1,
                    memory_limit: 256,
                    is_organization_private: true,
                    organization_ids: [org.id],
                    author_ids: [org.user_id],
                }}
                initialAuthors={user ? [{ id: org.user_id, label: user.username }] : []}
                initialOptions={{ organization_ids: [{ id: org.id, label: org.name }] }}
                onSubmit={async data => { await createMutation.mutateAsync(data); }}
                isLoading={createMutation.isPending}
            />
        </div>
    );
}