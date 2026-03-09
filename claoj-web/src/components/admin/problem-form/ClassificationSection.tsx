'use client';

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
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">Classification</h3>

            <div>
                <label htmlFor="problem-group" className="text-sm font-medium text-muted-foreground block mb-2">
                    Problem Group
                </label>
                <select
                    id="problem-group"
                    className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    value={selectedGroup || ''}
                    onChange={(e) => onGroupChange(e.target.value ? Number(e.target.value) : undefined)}
                >
                    <option value="">Select a group...</option>
                    {groups?.data.map(g => (
                        <option key={g.id} value={g.id}>{g.name}</option>
                    ))}
                </select>
            </div>

            <div>
                <label htmlFor="problem-types" className="text-sm font-medium text-muted-foreground block mb-2">
                    Problem Types
                </label>
                <div id="problem-types" className="grid grid-cols-2 md:grid-cols-3 gap-2" role="group" aria-labelledby="problem-types">
                    {types?.data.map(t => (
                        <label key={t.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedTypes.includes(t.id)}
                                onChange={(e) => onTypeToggle(t.id, e.target.checked)}
                                className="rounded"
                            />
                            <span className="text-sm">{t.full_name}</span>
                        </label>
                    ))}
                </div>
            </div>
        </div>
    );
}
