'use client';

import { useTranslations } from 'next-intl';
import { EntityPicker, EntityOption } from '../EntityPicker';
import { searchUsers } from '@/lib/adminSearch';

type StaffField = 'author_ids' | 'curator_ids' | 'tester_ids';

interface StaffSectionProps {
    selected: Record<StaffField, number[]>;
    initialOptions?: Partial<Record<StaffField, EntityOption[]>>;
    onChange: (field: StaffField, ids: number[]) => void;
}

export function StaffSection({ selected, initialOptions = {}, onChange }: StaffSectionProps) {
    const t = useTranslations('Admin');
    const fields: { field: StaffField; label: string }[] = [
        { field: 'author_ids', label: t('contestForm.authorsLabel') },
        { field: 'curator_ids', label: t('contestForm.curatorsLabel') },
        { field: 'tester_ids', label: t('contestForm.testersLabel') },
    ];
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">{t('contestForm.staffTitle')}</h3>
            {fields.map(({ field, label }) => (
                <EntityPicker
                    key={field}
                    id={`contest-${field}`}
                    label={label}
                    value={selected[field]}
                    onChange={ids => onChange(field, ids)}
                    search={searchUsers}
                    queryKey="users"
                    initialOptions={initialOptions[field]}
                />
            ))}
        </div>
    );
}
