'use client';

interface ContestTag {
    id: number;
    name: string;
    color: string;
    description: string;
}

interface TagsSectionProps {
    tags?: { data: ContestTag[] };
    selectedTags: number[];
    onTagToggle: (tagId: number, checked: boolean) => void;
}

export function TagsSection({ tags, selectedTags, onTagToggle }: TagsSectionProps) {
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">Contest Tags</h3>
            <p className="text-sm text-muted-foreground">
                Select tags to categorize this contest.
            </p>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-48 overflow-y-auto">
                {tags?.data.map(t => (
                    <label key={t.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                        <input
                            type="checkbox"
                            checked={selectedTags.includes(t.id)}
                            onChange={(e) => onTagToggle(t.id, e.target.checked)}
                            className="rounded"
                        />
                        <div className="flex-1 min-w-0">
                            <div className="font-medium text-sm truncate" style={{ color: t.color }}>{t.name}</div>
                        </div>
                    </label>
                ))}
            </div>
            {selectedTags.length > 0 && (
                <div className="text-sm text-muted-foreground">
                    Selected: {selectedTags.length} tag(s)
                </div>
            )}
        </div>
    );
}
