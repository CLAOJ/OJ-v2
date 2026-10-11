'use client';

import { useTranslations } from 'next-intl';
import { EntityPicker, EntityOption } from '../EntityPicker';
import { searchUsers } from '@/lib/adminSearch';

interface Language {
    id: number;
    name: string;
    key: string;
}

interface AuthorsSectionProps {
    languages?: { data: Language[] };
    selectedAuthors: number[];
    initialAuthors?: EntityOption[];
    selectedLangs: number[];
    onAuthorsChange: (ids: number[]) => void;
    onLangToggle: (langId: number, checked: boolean) => void;
}

export function AuthorsSection({
    languages,
    selectedAuthors,
    initialAuthors,
    selectedLangs,
    onAuthorsChange,
    onLangToggle
}: AuthorsSectionProps) {
    const t = useTranslations('Problems');

    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">{t('authorsAndLanguages')}</h3>

            <EntityPicker
                id="problem-authors"
                label={t('authors')}
                value={selectedAuthors}
                onChange={onAuthorsChange}
                search={searchUsers}
                queryKey="users"
                initialOptions={initialAuthors}
            />

            <div>
                <label htmlFor="allowed-languages" className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('allowedLanguages')}
                </label>
                <div id="allowed-languages" className="grid grid-cols-2 md:grid-cols-3 gap-2" role="group" aria-labelledby="allowed-languages">
                    {languages?.data.map(lang => (
                        <label key={lang.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedLangs.includes(lang.id)}
                                onChange={(e) => onLangToggle(lang.id, e.target.checked)}
                                className="rounded"
                            />
                            <span className="text-sm">{lang.name}</span>
                        </label>
                    ))}
                </div>
            </div>
        </div>
    );
}
