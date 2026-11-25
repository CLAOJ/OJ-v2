'use client';

import { useTranslations } from 'next-intl';
import { X } from 'lucide-react';

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
                {selectedTypes.length > 0 && (
                    <div className="flex flex-wrap gap-1.5 mb-2">
                        {selectedTypes.map(id => {
                            const pt = types?.data.find(x => x.id === id);
                            if (!pt) return null;
                            return (
                                <span key={id} className="inline-flex items-center gap-1 pl-2.5 pr-1 h-7 rounded-md bg-primary/15 text-primary text-xs font-medium">
                                    {pt.full_name}
                                    <button
                                        type="button"
                                        aria-label={tProblems('removeType', { name: pt.full_name })}
                                        onClick={() => onTypeToggle(id, false)}
                                        className="p-0.5 rounded hover:bg-primary/20 transition-colors"
                                    >
                                        <X size={12} />
                                    </button>
                                </span>
                            );
                        })}
                    </div>
                )}
                <select
                    id="problem-types"
                    value=""
                    onChange={(e) => { const id = Number(e.target.value); if (id) onTypeToggle(id, true); }}
                    className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                >
                    <option value="">{tProblems('addTypePlaceholder')}</option>
                    {types?.data.filter(pt => !selectedTypes.includes(pt.id)).map(pt => (
                        <option key={pt.id} value={pt.id}>{pt.full_name}</option>
                    ))}
                </select>
            </div>
        </div>
    );
}
