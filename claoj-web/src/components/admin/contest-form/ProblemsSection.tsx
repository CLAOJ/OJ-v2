'use client';

import { useTranslations } from 'next-intl';
import { EntityPicker, EntityOption } from '../EntityPicker';
import { searchProblems } from '@/lib/adminSearch';

interface ProblemsSectionProps {
    selectedProblems: number[];
    initialOptions?: EntityOption[];
    onChange: (ids: number[]) => void;
}

/** Selection order is contest problem order. */
export function ProblemsSection({ selectedProblems, initialOptions, onChange }: ProblemsSectionProps) {
    const t = useTranslations('Admin');
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">{t('contestForm.problemsTitle')}</h3>
            <p className="text-sm text-muted-foreground">{t('contestForm.problemsDesc')}</p>
            <EntityPicker
                id="contest-problems"
                label={t('contestForm.problemsTitle')}
                value={selectedProblems}
                onChange={onChange}
                search={searchProblems}
                queryKey="problems"
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
