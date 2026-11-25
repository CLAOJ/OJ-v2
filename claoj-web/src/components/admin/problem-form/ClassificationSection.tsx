'use client';

import { useTranslations } from 'next-intl';
import { TagSelect } from '@/components/ui/TagSelect';

interface ProblemGroup {
    id: number;
    name: string;
}

interface ProblemType {
    id: number;
    full_name: string;
}

interface ClassificationSectionProps {
    groups?: { data: ProblemGroup[] };
    types?: { data: ProblemType[] };
    selectedGroup?: number;
    selectedTypes: number[];
    onGroupChange: (groupId: number | undefined) => void;
    onTypeToggle: (typeId: number, checked: boolean) => void;
}

export function ClassificationSection({
    groups,
    types,
    selectedGroup,
    selectedTypes,
    onGroupChange,
    onTypeToggle
}: ClassificationSectionProps) {
    const tProblems = useTranslations('Problems');

    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{tProblems('classification')}</h3>

            <div>
                <label htmlFor="problem-group" className="text-sm font-medium text-muted-foreground block mb-2">
                    {tProblems('problemGroup')}
                </label>
                <select
                    id="problem-group"
                    className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    value={selectedGroup || ''}
                    onChange={(e) => onGroupChange(e.target.value ? Number(e.target.value) : undefined)}
                >
                    <option value="">{tProblems('selectGroupPlaceholder')}</option>
                    {groups?.data.map(g => (
                        <option key={g.id} value={g.id}>{g.name}</option>
                    ))}
                </select>
            </div>

            {/* v1 multi-select shape: chosen types as removable tags above one
                closed "add" select, instead of a wall of ~90 checkboxes. */}
            <div>
                <label htmlFor="problem-types" className="text-sm font-medium text-muted-foreground block mb-2">
                    {tProblems('problemTypes')}
                </label>
                <TagSelect
                    id="problem-types"
                    options={(types?.data ?? []).map(pt => ({ id: pt.id, label: pt.full_name }))}
                    value={selectedTypes}
                    onChange={(ids) => {
                        ids.filter(id => !selectedTypes.includes(id)).forEach(id => onTypeToggle(id, true));
                        selectedTypes.filter(id => !ids.includes(id)).forEach(id => onTypeToggle(id, false));
                    }}
                    placeholder={tProblems('addTypePlaceholder')}
                    removeLabel={(name) => tProblems('removeType', { name })}
                />
            </div>
        </div>
    );
}
