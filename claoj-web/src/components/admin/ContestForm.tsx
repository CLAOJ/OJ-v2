'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { BasicInfoSection } from './contest-form/BasicInfoSection';
import { ScheduleSection } from './contest-form/ScheduleSection';
import { FormatSection } from './contest-form/FormatSection';
import { ProblemsSection } from './contest-form/ProblemsSection';
import { TagsSection } from './contest-form/TagsSection';
import { StaffSection } from './contest-form/StaffSection';

export interface ContestFormData {
    key: string;
    name: string;
    description: string;
    summary?: string;
    start_time: string;
    end_time: string;
    time_limit?: number;
    is_visible: boolean;
    is_rated: boolean;
    format_name?: string;
    format_config?: string;
    access_code?: string;
    hide_problem_tags: boolean;
    run_pretests_only: boolean;
    is_organization_private: boolean;
    max_submissions?: number | null;
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    problem_ids?: number[];
    tag_ids?: number[];
}

export interface ContestFormProps {
    initialData?: ContestFormData;
    onSubmit: (data: ContestFormData) => Promise<void>;
    isLoading?: boolean;
}

interface Problem {
    id: number;
    code: string;
    name: string;
    points: number;
}

interface UserProfile {
    id: number;
    username: string;
}

interface ContestTag {
    id: number;
    name: string;
    color: string;
    description: string;
}

export default function ContestForm({ initialData, onSubmit, isLoading }: ContestFormProps) {
    const { register, handleSubmit, formState: { errors }, setValue, watch } = useForm<ContestFormData>({
        defaultValues: {
            key: initialData?.key || '',
            name: initialData?.name || '',
            description: initialData?.description || '',
            summary: initialData?.summary || '',
            start_time: initialData?.start_time || '',
            end_time: initialData?.end_time || '',
            time_limit: initialData?.time_limit,
            is_visible: initialData?.is_visible ?? false,
            is_rated: initialData?.is_rated ?? false,
            format_name: initialData?.format_name || 'icpc',
            format_config: initialData?.format_config || '',
            access_code: initialData?.access_code || '',
            hide_problem_tags: initialData?.hide_problem_tags ?? false,
            run_pretests_only: initialData?.run_pretests_only ?? false,
            is_organization_private: initialData?.is_organization_private ?? false,
            max_submissions: initialData?.max_submissions,
        }
    });

    const [description, setDescription] = useState(initialData?.description || '');

    const { data: problems } = useQuery<{ data: Problem[] }>({
        queryKey: ['admin-problems-list'],
        queryFn: async () => {
            const res = await api.get('/admin/problems?page=1&page_size=1000');
            return res.data;
        }
    });

    const { data: users } = useQuery<{ data: UserProfile[] }>({
        queryKey: ['admin-users-list'],
        queryFn: async () => {
            const res = await api.get('/admin/users?page=1&page_size=1000');
            return res.data;
        }
    });

    const { data: tags } = useQuery<{ data: ContestTag[] }>({
        queryKey: ['admin-contest-tags-list'],
        queryFn: async () => {
            const res = await api.get('/admin/contest-tags?page=1&page_size=1000');
            return res.data;
        }
    });

    const selectedFormat = watch('format_name');
    const selectedProblems = watch('problem_ids') || [];
    const selectedAuthors = watch('author_ids') || [];
    const selectedCurators = watch('curator_ids') || [];
    const selectedTesters = watch('tester_ids') || [];
    const selectedTags = watch('tag_ids') || [];

    const handleMultiSelect = (
        field: 'problem_ids' | 'author_ids' | 'curator_ids' | 'tester_ids',
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

    const handleProblemSelect = (problemId: number, checked: boolean) => {
        const current = (watch('problem_ids') as number[] | undefined) || [];
        if (checked) {
            setValue('problem_ids', [...current, problemId]);
        } else {
            setValue('problem_ids', current.filter(i => i !== problemId));
        }
    };

    const handleTagSelect = (tagId: number, checked: boolean) => {
        const current = (watch('tag_ids') as number[] | undefined) || [];
        if (checked) {
            setValue('tag_ids', [...current, tagId]);
        } else {
            setValue('tag_ids', current.filter(i => i !== tagId));
        }
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            <BasicInfoSection
                formData={{
                    key: watch('key'),
                    name: watch('name'),
                    description: watch('description'),
                    summary: watch('summary')
                }}
                errors={errors as any}
                register={register}
                onDescriptionChange={(value) => {
                    setDescription(value);
                    setValue('description', value);
                }}
                isEditMode={!!initialData?.key}
            />

            <ScheduleSection
                formData={{
                    start_time: watch('start_time'),
                    end_time: watch('end_time'),
                    time_limit: watch('time_limit')
                }}
                errors={errors as any}
                register={register}
            />

            <FormatSection
                formData={{
                    max_submissions: watch('max_submissions'),
                    format_name: watch('format_name'),
                    access_code: watch('access_code'),
                    is_visible: watch('is_visible'),
                    is_rated: watch('is_rated'),
                    hide_problem_tags: watch('hide_problem_tags'),
                    run_pretests_only: watch('run_pretests_only'),
                    is_organization_private: watch('is_organization_private')
                }}
                selectedFormat={selectedFormat || 'icpc'}
                register={register}
                setValue={setValue}
            />

            <ProblemsSection
                problems={problems}
                selectedProblems={selectedProblems}
                onProblemToggle={handleProblemSelect}
            />

            <TagsSection
                tags={tags}
                selectedTags={selectedTags}
                onTagToggle={handleTagSelect}
            />

            <StaffSection
                users={users}
                selectedAuthors={selectedAuthors}
                selectedCurators={selectedCurators}
                selectedTesters={selectedTesters}
                onUserToggle={handleMultiSelect}
            />

            <div className="flex justify-end gap-3">
                <button
                    type="submit"
                    disabled={isLoading}
                    className="px-6 py-2.5 rounded-xl bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                    {isLoading ? 'Saving...' : (initialData?.key ? 'Update Contest' : 'Create Contest')}
                </button>
            </div>
        </form>
    );
}
