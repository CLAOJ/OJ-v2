'use client';

import { useTranslations } from 'next-intl';
import { TagSelect } from '@/components/ui/TagSelect';

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
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('contestForm.tagsTitle')}</h3>
            <p className="text-sm text-muted-foreground">
                {t('contestForm.tagsDesc')}
            </p>
            <TagSelect
                id="contest-tags"
                options={(tags?.data ?? []).map(tag => ({ id: tag.id, label: tag.name, color: tag.color }))}
                value={selectedTags}
                onChange={(ids) => {
                    ids.filter(id => !selectedTags.includes(id)).forEach(id => onTagToggle(id, true));
                    selectedTags.filter(id => !ids.includes(id)).forEach(id => onTagToggle(id, false));
                }}
                placeholder={t('contestForm.addTagPlaceholder')}
                removeLabel={(name) => t('contestForm.removeTag', { name })}
            />
        </div>
    );
}
