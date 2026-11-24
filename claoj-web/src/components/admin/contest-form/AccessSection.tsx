'use client';

import { UseFormRegister } from 'react-hook-form';
import { useTranslations } from 'next-intl';
import { EntityPicker } from '../EntityPicker';
import { searchClasses, searchOrganizations, searchUsers } from '@/lib/adminSearch';
import type { ContestFormOptions, ContestFormData } from '../ContestForm';

type PickerField =
    | 'organization_ids'
    | 'private_contestant_ids'
    | 'banned_user_ids'
    | 'view_scoreboard_user_ids'
    | 'join_organization_ids'
    | 'class_ids';

interface AccessSectionProps {
    selected: Record<PickerField, number[]>;
    initialOptions?: ContestFormOptions;
    register: UseFormRegister<ContestFormData>;
    onChange: (field: PickerField, ids: number[]) => void;
}

export function AccessSection({ selected, initialOptions = {}, register, onChange }: AccessSectionProps) {
    const t = useTranslations('Admin');
    const pickers: {
        field: PickerField;
        label: string;
        search: (term: string) => Promise<{ id: number; label: string; detail?: string }[]>;
        queryKey: string;
    }[] = [
        { field: 'private_contestant_ids', label: t('contestForm.privateContestantsLabel'), search: searchUsers, queryKey: 'users' },
        { field: 'banned_user_ids', label: t('contestForm.bannedUsersLabel'), search: searchUsers, queryKey: 'users' },
        { field: 'view_scoreboard_user_ids', label: t('contestForm.viewScoreboardUsersLabel'), search: searchUsers, queryKey: 'users' },
        { field: 'join_organization_ids', label: t('contestForm.joinOrganizationsLabel'), search: searchOrganizations, queryKey: 'organizations' },
        { field: 'class_ids', label: t('contestForm.classesLabel'), search: searchClasses, queryKey: 'classes' },
    ];

    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('contestForm.accessTitle')}</h3>

            <div className="flex flex-wrap items-center gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('is_private')}
                    />
                    <span className="text-sm font-medium">{t('contestForm.privateContestLabel')}</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('is_organization_private')}
                    />
                    <span className="text-sm font-medium">{t('contestForm.privateToOrgLabel')}</span>
                </label>
            </div>

            <EntityPicker
                id="contest-organizations"
                label={t('contestForm.organizationsLabel')}
                value={selected.organization_ids}
                onChange={ids => onChange('organization_ids', ids)}
                search={searchOrganizations}
                queryKey="organizations"
                initialOptions={initialOptions.organization_ids}
            />

            <label className="flex items-center gap-2 cursor-pointer">
                <input
                    type="checkbox"
                    className="rounded w-5 h-5"
                    {...register('limit_join_organizations')}
                />
                <span className="text-sm font-medium">{t('contestForm.limitJoinOrganizationsLabel')}</span>
            </label>

            {pickers.map(({ field, label, search, queryKey }) => (
                <EntityPicker
                    key={field}
                    id={`contest-${field}`}
                    label={label}
                    value={selected[field]}
                    onChange={ids => onChange(field, ids)}
                    search={search}
                    queryKey={queryKey}
                    initialOptions={initialOptions[field]}
                />
            ))}
        </div>
    );
}