'use client';

import { useTranslations } from 'next-intl';
import { EntityPicker, EntityOption } from '../EntityPicker';
import { searchProblems } from '@/lib/adminSearch';

interface ProblemsSectionProps {
    selectedProblems: number[];
    initialOptions?: EntityOption[];
    onChange: (ids: number[]) => void;
    /** Override the staff problem search (organization pages). */
    search?: (term: string) => Promise<EntityOption[]>;
    /** Cache namespace for the picker results. */
    queryKey?: string;
}

/** Selection order is contest problem order. */
export function ProblemsSection({ selectedProblems, initialOptions, onChange, search, queryKey = 'problems' }: ProblemsSectionProps) {
    const t = useTranslations('Admin');
    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-bold">{t('contestForm.problemsTitle')}</h3>
            <p className="text-sm text-muted-foreground">{t('contestForm.problemsDesc')}</p>
            <EntityPicker
                id="contest-problems"
                label={t('contestForm.problemsTitle')}
                value={selectedProblems}
                onChange={onChange}
                search={search ?? searchProblems}
                queryKey={queryKey}
                initialOptions={initialOptions}
            />
            {selectedProblems.length > 0 && (
                <div className="text-sm text-muted-foreground">
                    {t('contestForm.selectedProblemsCount', { count: selectedProblems.length })}
                </div>
            )}
        </div>
    );
}
