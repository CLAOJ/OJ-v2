'use client';

import { UseFormRegister } from 'react-hook-form';
import { useTranslations } from 'next-intl';
import { EntityPicker, EntityOption } from '../EntityPicker';
import { searchOrganizations, searchUsers } from '@/lib/adminSearch';
import type { ProblemFormData } from '../ProblemForm';

interface AccessSectionProps {
    isOrganizationPrivate: boolean;
    organizationIds: number[];
    bannedUserIds: number[];
    initialOptions?: {
        organization_ids?: EntityOption[];
        banned_user_ids?: EntityOption[];
    };
    register: UseFormRegister<ProblemFormData>;
    onOrganizationsChange: (ids: number[]) => void;
    onBannedUsersChange: (ids: number[]) => void;
}

export function AccessSection({
    isOrganizationPrivate,
    organizationIds,
    bannedUserIds,
    initialOptions = {},
    register,
    onOrganizationsChange,
    onBannedUsersChange
}: AccessSectionProps) {
    const t = useTranslations('Admin.problemForm');

    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('accessTitle')}</h3>

            <label className="flex items-center gap-2 cursor-pointer">
                <input
                    type="checkbox"
                    className="rounded w-5 h-5"
                    {...register('is_organization_private')}
                    checked={isOrganizationPrivate}
                />
                <span className="text-sm font-medium">{t('privateToOrgLabel')}</span>
            </label>

            <EntityPicker
                id="problem-organizations"
                label={t('organizationsLabel')}
                value={organizationIds}
                onChange={onOrganizationsChange}
                search={searchOrganizations}
                queryKey="organizations"
                initialOptions={initialOptions.organization_ids}
            />

            <EntityPicker
                id="problem-banned-users"
                label={t('bannedUsersLabel')}
                value={bannedUserIds}
                onChange={onBannedUsersChange}
                search={searchUsers}
                queryKey="users"
                initialOptions={initialOptions.banned_user_ids}
            />
        </div>
    );
}