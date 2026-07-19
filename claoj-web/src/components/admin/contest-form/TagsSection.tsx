'use client';

import { useTranslations } from 'next-intl';

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
    const t = useTranslations('Admin');
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">{t('contestForm.tagsTitle')}</h3>
            <p className="text-sm text-muted-foreground">
                {t('contestForm.tagsDesc')}
            </p>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-48 overflow-y-auto">
                {tags?.data.map(tag => (
                    <label key={tag.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                        <input
                            type="checkbox"
                            checked={selectedTags.includes(tag.id)}
                            onChange={(e) => onTagToggle(tag.id, e.target.checked)}
                            className="rounded"
                        />
                        <div className="flex-1 min-w-0">
                            <div className="font-medium text-sm truncate" style={{ color: tag.color }}>{tag.name}</div>
                        </div>
                    </label>
                ))}
            </div>
            {selectedTags.length > 0 && (
                <div className="text-sm text-muted-foreground">
                    {t('contestForm.selectedTagsCount', { count: selectedTags.length })}
                </div>
            )}
        </div>
    );
}
