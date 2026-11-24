'use client';

import { useTranslations } from 'next-intl';
import { EntityPicker, EntityOption } from '../EntityPicker';
import { searchUsers } from '@/lib/adminSearch';

interface PeopleSectionProps {
    suggesterId?: number;
    curatorIds: number[];
    testerIds: number[];
    initialOptions?: {
        suggester_id?: EntityOption[];
        curator_ids?: EntityOption[];
        tester_ids?: EntityOption[];
    };
    onSuggesterChange: (id?: number) => void;
    onCuratorsChange: (ids: number[]) => void;
    onTestersChange: (ids: number[]) => void;
}

export function PeopleSection({
    suggesterId,
    curatorIds,
    testerIds,
    initialOptions = {},
    onSuggesterChange,
    onCuratorsChange,
    onTestersChange
}: PeopleSectionProps) {
    const t = useTranslations('Admin.problemForm');

    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('peopleTitle')}</h3>

            <EntityPicker
                id="problem-suggester"
                label={t('suggesterLabel')}
                value={suggesterId ? [suggesterId] : []}
                // Single select: the newest pick wins, removing the chip clears it.
                onChange={ids => onSuggesterChange(ids[ids.length - 1])}
                search={searchUsers}
                queryKey="users"
                initialOptions={initialOptions.suggester_id}
            />

            <EntityPicker
                id="problem-curators"
                label={t('curatorsLabel')}
                value={curatorIds}
                onChange={onCuratorsChange}
                search={searchUsers}
                queryKey="users"
                initialOptions={initialOptions.curator_ids}
            />

            <EntityPicker
                id="problem-testers"
                label={t('testersLabel')}
                value={testerIds}
                onChange={onTestersChange}
                search={searchUsers}
                queryKey="users"
                initialOptions={initialOptions.tester_ids}
            />
        </div>
    );
}