'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { adminLicenseApi, adminProblemGroupApi, adminProblemTypeApi } from '@/lib/adminApi';
import type { AdminLicense, SubmissionSourceVisibilityMode, TestcaseVisibilityMode } from '@/types';
import { BasicInfoSection } from './problem-form/BasicInfoSection';
import type { EntityOption } from './EntityPicker';

// The taxonomy selects need every group/type, not a page of them.
const TAXONOMY_PAGE_SIZE = 500;

import { ClassificationSection } from './problem-form/ClassificationSection';
import { AuthorsSection } from './problem-form/AuthorsSection';
import { PeopleSection } from './problem-form/PeopleSection';
import { PublishingSection } from './problem-form/PublishingSection';
import { AccessSection } from './problem-form/AccessSection';
import { SettingsSection } from './problem-form/SettingsSection';

export interface ProblemFormData {
    code: string;
    name: string;
    description: string;
    points: number;
    partial: boolean;
    is_public: boolean;
    time_limit: number;
    memory_limit: number;
    group_id?: number;
    type_ids?: number[];
    author_ids?: number[];
    allowed_lang_ids?: number[];
    is_manually_managed?: boolean;
    pdf_url?: string;
    source?: string;
    /** `datetime-local` value, converted to ISO on submit. */
    date?: string;
    license_id?: number;
    og_image?: string;
    summary?: string;
    is_full_markup?: boolean;
    submission_source_visibility_mode?: SubmissionSourceVisibilityMode;
    testcase_visibility_mode?: TestcaseVisibilityMode;
    is_organization_private?: boolean;
    organization_ids?: number[];
    banned_user_ids?: number[];
    suggester_id?: number;
    curator_ids?: number[];
    tester_ids?: number[];
}

export interface ProblemFormProps {
    initialData?: ProblemFormData;
    initialAuthors?: EntityOption[];
    /** Chips for users/organizations already linked to the problem. */
    initialOptions?: {
        curator_ids?: EntityOption[];
        tester_ids?: EntityOption[];
        suggester_id?: EntityOption[];
        organization_ids?: EntityOption[];
        banned_user_ids?: EntityOption[];
    };
    onSubmit: (data: ProblemFormData) => Promise<void>;
    isLoading?: boolean;
}

interface ProblemGroup {
    id: number;
    name: string;
}

interface ProblemType {
    id: number;
    full_name: string;
}

interface Language {
    id: number;
    name: string;
    key: string;
}

export default function ProblemForm({ initialData, initialAuthors, initialOptions, onSubmit, isLoading }: ProblemFormProps) {
    const { register, handleSubmit, formState: { errors }, setValue, watch } = useForm<ProblemFormData>({
        defaultValues: {
            code: initialData?.code || '',
            name: initialData?.name || '',
            description: initialData?.description || '',
            points: initialData?.points || 100,
            partial: initialData?.partial ?? true,
            is_public: initialData?.is_public ?? false,
            time_limit: initialData?.time_limit || 1,
            // memory_limit is stored in KB but presented/edited in MB (see submitInMbToKb).
            memory_limit: initialData?.memory_limit ? Math.round(initialData.memory_limit / 1024) : 256,
            is_manually_managed: initialData?.is_manually_managed ?? false,
            pdf_url: initialData?.pdf_url || '',
            group_id: initialData?.group_id,
            type_ids: initialData?.type_ids ?? [],
            author_ids: initialData?.author_ids ?? [],
            allowed_lang_ids: initialData?.allowed_lang_ids ?? [],
            source: initialData?.source || '',
            date: initialData?.date || '',
            license_id: initialData?.license_id ?? undefined,
            og_image: initialData?.og_image || '',
            summary: initialData?.summary || '',
            is_full_markup: initialData?.is_full_markup ?? false,
            submission_source_visibility_mode: initialData?.submission_source_visibility_mode || 'F',
            testcase_visibility_mode: initialData?.testcase_visibility_mode || 'C',
            is_organization_private: initialData?.is_organization_private ?? false,
            organization_ids: initialData?.organization_ids ?? [],
            banned_user_ids: initialData?.banned_user_ids ?? [],
            suggester_id: initialData?.suggester_id,
            curator_ids: initialData?.curator_ids ?? [],
            tester_ids: initialData?.tester_ids ?? [],
        }
    });

    const [description, setDescription] = useState(initialData?.description || '');

    // Problem taxonomy is only exposed under the admin routes
    // (/admin/problem-groups, /admin/problem-types). The bare /problem/groups
    // and /problem/types paths this used to call don't exist — worse, they were
    // swallowed by the /problem/:code route and came back as a "problem not
    // found" 404, so both selects silently rendered empty on every problem
    // create/edit page load.
    const { data: groups } = useQuery<{ data: ProblemGroup[] }>({
        queryKey: ['problem-groups'],
        queryFn: async () => {
            const res = await adminProblemGroupApi.list(1, TAXONOMY_PAGE_SIZE);
            return res.data;
        }
    });

    const { data: types } = useQuery<{ data: ProblemType[] }>({
        queryKey: ['problem-types'],
        queryFn: async () => {
            const res = await adminProblemTypeApi.list(1, TAXONOMY_PAGE_SIZE);
            return res.data;
        }
    });

    const { data: languages } = useQuery<{ data: Language[] }>({
        queryKey: ['languages'],
        queryFn: async () => {
            const res = await api.get('/languages');
            return res.data;
        }
    });

    const { data: licenses } = useQuery<{ data: AdminLicense[] }>({
        queryKey: ['admin-licenses'],
        queryFn: async () => {
            const res = await adminLicenseApi.list(1, TAXONOMY_PAGE_SIZE);
            return res.data;
        }
    });

    const selectedGroup = watch('group_id');
    const selectedTypes = watch('type_ids') || [];
    const selectedAuthors = watch('author_ids') || [];
    const selectedLangs = watch('allowed_lang_ids') || [];
    const selectedPeople = {
        suggester_id: watch('suggester_id'),
        curator_ids: watch('curator_ids') || [],
        tester_ids: watch('tester_ids') || [],
    };

    const handleMultiSelect = (
        field: 'type_ids' | 'author_ids' | 'allowed_lang_ids',
        id: number,
        checked: boolean
    ) => {
        const current = (watch(field) as number[] | undefined) || [];
        if (checked) {
            setValue(field, [...current, id]);
        } else {
            setValue(field, current.filter(i => i !== id));
        }
    };

    // The form field is in MB; storage/judge expect KB. Convert on the way out.
    const submitInMbToKb = (data: ProblemFormData) =>
        onSubmit({ ...data, memory_limit: Math.round((Number(data.memory_limit) || 0) * 1024) });

    return (
        <form onSubmit={handleSubmit(submitInMbToKb)} className="space-y-6">
            <BasicInfoSection
                formData={{
                    code: watch('code'),
                    name: watch('name'),
                    description: watch('description'),
                    points: watch('points'),
                    time_limit: watch('time_limit'),
                    memory_limit: watch('memory_limit')
                }}
                errors={errors as any}
                register={register}
                onDescriptionChange={(value) => {
                    setDescription(value);
                    setValue('description', value);
                }}
                isEditMode={!!initialData?.code}
            />

            <ClassificationSection
                groups={groups}
                types={types}
                selectedGroup={selectedGroup}
                selectedTypes={selectedTypes}
                onGroupChange={(groupId) => setValue('group_id', groupId)}
                onTypeToggle={(typeId, checked) => handleMultiSelect('type_ids', typeId, checked)}
            />

            <AuthorsSection
                languages={languages}
                selectedAuthors={selectedAuthors}
                initialAuthors={initialAuthors}
                selectedLangs={selectedLangs}
                onAuthorsChange={(ids) => setValue('author_ids', ids)}
                onLangToggle={(langId, checked) => handleMultiSelect('allowed_lang_ids', langId, checked)}
            />

            <PeopleSection
                suggesterId={selectedPeople.suggester_id}
                curatorIds={selectedPeople.curator_ids}
                testerIds={selectedPeople.tester_ids}
                initialOptions={initialOptions}
                onSuggesterChange={(id) => setValue('suggester_id', id)}
                onCuratorsChange={(ids) => setValue('curator_ids', ids)}
                onTestersChange={(ids) => setValue('tester_ids', ids)}
            />

            <PublishingSection
                licenses={licenses?.data}
                values={{
                    source: watch('source') || '',
                    date: watch('date') || '',
                    license_id: watch('license_id'),
                    submissionSourceMode: watch('submission_source_visibility_mode') || 'F',
                    testcaseMode: watch('testcase_visibility_mode') || 'C',
                    is_full_markup: watch('is_full_markup') || false,
                    og_image: watch('og_image') || '',
                    summary: watch('summary') || '',
                }}
                register={register}
                onDateChange={(value) => setValue('date', value)}
                onLicenseChange={(id) => setValue('license_id', id)}
                onSubmissionSourceModeChange={(mode) => setValue('submission_source_visibility_mode', mode as SubmissionSourceVisibilityMode)}
                onTestcaseModeChange={(mode) => setValue('testcase_visibility_mode', mode as TestcaseVisibilityMode)}
            />

            <AccessSection
                isOrganizationPrivate={watch('is_organization_private') || false}
                organizationIds={watch('organization_ids') || []}
                bannedUserIds={watch('banned_user_ids') || []}
                initialOptions={initialOptions}
                register={register}
                onOrganizationsChange={(ids) => setValue('organization_ids', ids)}
                onBannedUsersChange={(ids) => setValue('banned_user_ids', ids)}
            />

            <SettingsSection
                settings={{
                    is_public: watch('is_public') || false,
                    partial: watch('partial') || false,
                    is_manually_managed: watch('is_manually_managed') || false,
                    pdf_url: watch('pdf_url')
                }}
                register={register}
            />

            <div className="flex justify-end gap-3">
                <button
                    type="submit"
                    disabled={isLoading}
                    className="px-6 py-2.5 rounded-lg bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                    {isLoading ? 'Saving...' : (initialData?.code ? 'Update Problem' : 'Create Problem')}
                </button>
            </div>
        </form>
    );
}
