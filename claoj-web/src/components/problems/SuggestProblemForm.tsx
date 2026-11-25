'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useRouter, Link } from '@/navigation';
import api from '@/lib/api';
import { type ProblemFormData } from '@/components/admin/ProblemForm';
import { BasicInfoSection } from '@/components/admin/problem-form/BasicInfoSection';
import { ClassificationSection } from '@/components/admin/problem-form/ClassificationSection';
import { useAuth } from '@/components/providers/AuthProvider';
import { hasPerm } from '@/lib/permissions';
import { Skeleton } from '@/components/ui/Skeleton';

const TAXONOMY_PAGE_SIZE = 500;
const TESTCASE_MODES = ['A', 'C', 'O'] as const;

interface ProblemGroup {
    id: number;
    name: string;
}

interface ProblemType {
    id: number;
    full_name: string;
}

/** v1 ProblemSuggest: the create form minus authors and the public toggle. */
export default function SuggestProblemForm() {
    const t = useTranslations('Problems.suggest.form');
    const tQueue = useTranslations('Problems.suggest');
    const tForm = useTranslations('Admin.problemForm');
    const { user, loading } = useAuth();
    const router = useRouter();
    const [submitError, setSubmitError] = useState<string | null>(null);
    const [description, setDescription] = useState('');

    const { register, handleSubmit, watch, setValue, setError, formState: { errors } } = useForm<ProblemFormData>({
        defaultValues: {
            code: '',
            name: '',
            description: '',
            points: 100,
            partial: true,
            time_limit: 1,
            memory_limit: 512,
            source: '',
            type_ids: [],
            testcase_visibility_mode: 'C',
        },
    });

    const { data: groups } = useQuery<{ data: ProblemGroup[] }>({
        queryKey: ['problem-groups'],
        queryFn: async () => (await api.get(`/problem-groups?page_size=${TAXONOMY_PAGE_SIZE}`)).data,
    });
    const { data: types } = useQuery<{ data: ProblemType[] }>({
        queryKey: ['problem-types'],
        queryFn: async () => (await api.get(`/problem-types?page_size=${TAXONOMY_PAGE_SIZE}`)).data,
    });

    const createMutation = useMutation({
        mutationFn: async (data: ProblemFormData) => {
            // memory_limit is edited in MB and stored in KB.
            return api.post('/problems/suggest', {
                ...data,
                memory_limit: Math.round((Number(data.memory_limit) || 0) * 1024),
            });
        },
        onSuccess: () => router.push('/problems/suggest'),
        onError: (err: unknown) => {
            const message = (err as { response?: { data?: { error?: string } } }).response?.data?.error;
            setSubmitError(message || t('submitFailed'));
        },
    });

    const submit = (data: ProblemFormData) => {
        setSubmitError(null);
        if (!data.group_id) {
            setError('group_id', { message: t('groupRequired') });
            return;
        }
        createMutation.mutate(data);
    };

    const selectedTypes = watch('type_ids') || [];
    const handleTypeToggle = (typeId: number, checked: boolean) => {
        setValue('type_ids', checked ? [...selectedTypes, typeId] : selectedTypes.filter(id => id !== typeId));
    };

    if (loading) {
        return <Skeleton className="h-96 w-full max-w-4xl mx-auto mt-4" />;
    }
    // v1 ProblemSuggest is permission-gated at the page level as well.
    if (!hasPerm(user, 'judge.suggest_new_problem')) {
        return <div className="max-w-4xl mx-auto mt-4 p-10 rounded-md border border-dashed text-center text-sm">{tQueue('forbidden')}</div>;
    }

    return (
        <form onSubmit={handleSubmit(submit)} className="space-y-6">
            <div className="flex flex-col gap-1">
                <Link href="/problems/suggest" className="text-xs font-medium text-muted-foreground hover:text-primary w-fit">
                    {t('backToQueue')}
                </Link>
                <h1 className="text-2xl font-semibold">{t('title')}</h1>
                <p className="text-sm text-muted-foreground">{t('subtitle')}</p>
            </div>

            <BasicInfoSection
                formData={{
                    code: watch('code'),
                    name: watch('name'),
                    description,
                    points: watch('points'),
                    time_limit: watch('time_limit'),
                    memory_limit: watch('memory_limit'),
                }}
                errors={errors as unknown as { code?: { message: string }; name?: { message: string }; description?: { message: string } }}
                register={register}
                onDescriptionChange={(value) => { setDescription(value); setValue('description', value); }}
                isEditMode={false}
            />

            <ClassificationSection
                groups={groups as { data: ProblemGroup[] } | undefined}
                types={types as { data: ProblemType[] } | undefined}
                selectedGroup={watch('group_id')}
                selectedTypes={selectedTypes}
                onGroupChange={(groupId) => setValue('group_id', groupId)}
                onTypeToggle={handleTypeToggle}
            />
            {errors.group_id && <p className="text-destructive text-xs">{errors.group_id.message}</p>}

            <div className="bg-card rounded-md border p-6 space-y-4">
                <div>
                    <label htmlFor="suggest-source" className="text-sm font-medium text-muted-foreground block mb-2">
                        {tForm('sourceLabel')}
                    </label>
                    <input
                        id="suggest-source"
                        type="text"
                        maxLength={200}
                        className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={tForm('sourcePlaceholder')}
                        {...register('source')}
                    />
                </div>

                <div>
                    <label htmlFor="suggest-testcase-visibility" className="text-sm font-medium text-muted-foreground block mb-2">
                        {tForm('testcaseVisibilityLabel')}
                    </label>
                    <select
                        id="suggest-testcase-visibility"
                        className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        {...register('testcase_visibility_mode')}
                    >
                        {TESTCASE_MODES.map(mode => (
                            <option key={mode} value={mode}>{tForm(`testcaseVisibility.${mode}`)}</option>
                        ))}
                    </select>
                </div>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" className="rounded w-5 h-5" {...register('partial')} />
                    <span className="text-sm font-medium">{tForm('partialLabel')}</span>
                </label>
            </div>

            {submitError && <p role="alert" className="text-sm text-destructive">{submitError}</p>}

            <div className="flex justify-end gap-3">
                <button
                    type="submit"
                    disabled={createMutation.isPending}
                    className="px-6 py-2.5 rounded-lg bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                    {createMutation.isPending ? t('submitting') : t('submit')}
                </button>
            </div>
        </form>
    );
}