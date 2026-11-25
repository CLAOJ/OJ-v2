'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { BasicInfoSection } from './contest-form/BasicInfoSection';
import { ScheduleSection } from './contest-form/ScheduleSection';
import { FormatSection } from './contest-form/FormatSection';
import { RatingSection } from './contest-form/RatingSection';
import { ScoreboardSection } from './contest-form/ScoreboardSection';
import { AccessSection } from './contest-form/AccessSection';
import { DetailsSection } from './contest-form/DetailsSection';
import { SettingsSection } from './contest-form/SettingsSection';
import { ProblemsSection } from './contest-form/ProblemsSection';
import { TagsSection } from './contest-form/TagsSection';
import { StaffSection } from './contest-form/StaffSection';
import type { EntityOption } from './EntityPicker';
import type { ScoreboardVisibility } from '@/types';

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
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    problem_ids?: number[];
    tag_ids?: number[];
    rate_all?: boolean;
    rating_floor?: number;
    rating_ceiling?: number;
    performance_ceiling_override?: number;
    scoreboard_visibility?: ScoreboardVisibility;
    scoreboard_cache_timeout?: number;
    points_precision?: number;
    is_private?: boolean;
    hide_problem_authors?: boolean;
    show_short_display?: boolean;
    use_clarifications?: boolean;
    push_announcements?: boolean;
    limit_join_organizations?: boolean;
    og_image?: string;
    logo_override_image?: string;
    problem_label_script?: string;
    organization_ids?: number[];
    banned_user_ids?: number[];
    private_contestant_ids?: number[];
    view_scoreboard_user_ids?: number[];
    join_organization_ids?: number[];
    class_ids?: number[];
}

/** Chips for the relation pickers, keyed by the form field they belong to. */
export interface ContestFormOptions {
    problems?: EntityOption[];
    author_ids?: EntityOption[];
    curator_ids?: EntityOption[];
    tester_ids?: EntityOption[];
    organization_ids?: EntityOption[];
    banned_user_ids?: EntityOption[];
    private_contestant_ids?: EntityOption[];
    view_scoreboard_user_ids?: EntityOption[];
    join_organization_ids?: EntityOption[];
    class_ids?: EntityOption[];
}

export interface ContestFormProps {
    initialData?: ContestFormData;
    initialOptions?: ContestFormOptions;
    /**
     * Search sources for the relation pickers. Omitted on the admin pages
     * (the staff endpoints); the organization pages pass public/owned sources
     * so non-staff org admins can fill them in.
     */
    search?: {
        problems?: (term: string) => Promise<EntityOption[]>;
        users?: (term: string) => Promise<EntityOption[]>;
        /** Cache namespace for the pickers, e.g. `org-7-` (separate sources). */
        key?: string;
    };
    onSubmit: (data: ContestFormData) => Promise<void>;
    isLoading?: boolean;
}

interface ContestTag {
    id: number;
    name: string;
    color: string;
    description: string;
}

export default function ContestForm({ initialData, initialOptions, search, onSubmit, isLoading }: ContestFormProps) {
    const t = useTranslations('Admin');
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
            format_name: initialData?.format_name || 'default',
            format_config: initialData?.format_config || '',
            access_code: initialData?.access_code || '',
            hide_problem_tags: initialData?.hide_problem_tags ?? false,
            run_pretests_only: initialData?.run_pretests_only ?? false,
            is_organization_private: initialData?.is_organization_private ?? false,
            author_ids: initialData?.author_ids ?? [],
            curator_ids: initialData?.curator_ids ?? [],
            tester_ids: initialData?.tester_ids ?? [],
            problem_ids: initialData?.problem_ids ?? [],
            tag_ids: initialData?.tag_ids ?? [],
            rate_all: initialData?.rate_all ?? false,
            rating_floor: initialData?.rating_floor,
            rating_ceiling: initialData?.rating_ceiling,
            performance_ceiling_override: initialData?.performance_ceiling_override,
            scoreboard_visibility: initialData?.scoreboard_visibility ?? 'V',
            scoreboard_cache_timeout: initialData?.scoreboard_cache_timeout ?? 0,
            points_precision: initialData?.points_precision,
            is_private: initialData?.is_private ?? false,
            hide_problem_authors: initialData?.hide_problem_authors ?? false,
            show_short_display: initialData?.show_short_display ?? false,
            use_clarifications: initialData?.use_clarifications ?? true,
            push_announcements: initialData?.push_announcements ?? false,
            limit_join_organizations: initialData?.limit_join_organizations ?? false,
            og_image: initialData?.og_image || '',
            logo_override_image: initialData?.logo_override_image || '',
            problem_label_script: initialData?.problem_label_script || '',
            organization_ids: initialData?.organization_ids ?? [],
            banned_user_ids: initialData?.banned_user_ids ?? [],
            private_contestant_ids: initialData?.private_contestant_ids ?? [],
            view_scoreboard_user_ids: initialData?.view_scoreboard_user_ids ?? [],
            join_organization_ids: initialData?.join_organization_ids ?? [],
            class_ids: initialData?.class_ids ?? [],
        }
    });

    const [description, setDescription] = useState(initialData?.description || '');

    const { data: tags } = useQuery<{ data: ContestTag[] }>({
        queryKey: ['contest-tags-list'],
        queryFn: async () => {
            const res = await api.get('/contest-tags?page=1&page_size=1000');
            return res.data;
        }
    });

    const selectedFormat = watch('format_name');
    // The org pages pass their own picker sources; keep their cached results
    // out of the staff pickers' namespace.
    const pickerScope = search?.key ? `${search.key}-` : '';
    const selectedProblems = watch('problem_ids') || [];
    const selectedAuthors = watch('author_ids') || [];
    const selectedCurators = watch('curator_ids') || [];
    const selectedTesters = watch('tester_ids') || [];
    const selectedTags = watch('tag_ids') || [];

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
                errors={errors}
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
                errors={errors}
                register={register}
            />

            <FormatSection
                formData={{
                    format_name: watch('format_name'),
                    access_code: watch('access_code'),
                    is_visible: watch('is_visible'),
                    hide_problem_tags: watch('hide_problem_tags'),
                    run_pretests_only: watch('run_pretests_only')
                }}
                selectedFormat={selectedFormat || 'default'}
                register={register}
                setValue={setValue}
                formatConfigError={(errors.format_config?.message as string | undefined)}
            />

            <RatingSection register={register} />

            <ScoreboardSection register={register} />

            <AccessSection
                selected={{
                    organization_ids: watch('organization_ids') || [],
                    banned_user_ids: watch('banned_user_ids') || [],
                    private_contestant_ids: watch('private_contestant_ids') || [],
                    view_scoreboard_user_ids: watch('view_scoreboard_user_ids') || [],
                    join_organization_ids: watch('join_organization_ids') || [],
                    class_ids: watch('class_ids') || []
                }}
                initialOptions={initialOptions}
                register={register}
                onChange={(field, ids) => setValue(field, ids)}
            />

            <DetailsSection register={register} />

            <SettingsSection
                settings={{
                    push_announcements: watch('push_announcements') || false,
                    hide_problem_authors: watch('hide_problem_authors') || false,
                    show_short_display: watch('show_short_display') || false,
                    use_clarifications: watch('use_clarifications') ?? true
                }}
                register={register}
            />

            <ProblemsSection
                selectedProblems={selectedProblems}
                initialOptions={initialOptions?.problems}
                onChange={(ids) => setValue('problem_ids', ids)}
                search={search?.problems}
                queryKey={`${pickerScope}problems`}
            />

            <TagsSection
                tags={tags}
                selectedTags={selectedTags}
                onTagToggle={handleTagSelect}
            />

            <StaffSection
                selected={{ author_ids: selectedAuthors, curator_ids: selectedCurators, tester_ids: selectedTesters }}
                initialOptions={initialOptions}
                onChange={(field, ids) => setValue(field, ids)}
                search={search?.users}
                queryKey={`${pickerScope}users`}
            />

            <div className="flex justify-end gap-3">
                <button
                    type="submit"
                    disabled={isLoading}
                    className="px-6 py-2.5 rounded-xl bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                    {isLoading ? t('contestForm.savingButton') : (initialData?.key ? t('contestForm.updateContestButton') : t('contestForm.createContestButton'))}
                </button>
            </div>
        </form>
    );
}
