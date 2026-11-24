'use client';

import { UseFormRegister } from 'react-hook-form';
import { useTranslations } from 'next-intl';
import type { ContestFormData } from '../ContestForm';

interface RatingSectionProps {
    register: UseFormRegister<ContestFormData>;
}

/** Empty number input → undefined, so the mapper can tell "cleared" from zero. */
export const optionalInt = { setValueAs: (v: string) => (v === '' || v == null ? undefined : Number(v)) };

export function RatingSection({ register }: RatingSectionProps) {
    const t = useTranslations('Admin');
    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('contestForm.ratingTitle')}</h3>

            <div className="flex flex-wrap items-center gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('is_rated')}
                    />
                    <span className="text-sm font-medium">{t('contestForm.ratedContestLabel')}</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('rate_all')}
                    />
                    <span className="text-sm font-medium">{t('contestForm.rateAllLabel')}</span>
                </label>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.ratingFloorLabel')}
                    </label>
                    <input
                        type="number"
                        step="1"
                        inputMode="numeric"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={t('contestForm.ratingFloorPlaceholder')}
                        {...register('rating_floor', optionalInt)}
                    />
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.ratingCeilingLabel')}
                    </label>
                    <input
                        type="number"
                        step="1"
                        inputMode="numeric"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={t('contestForm.ratingCeilingPlaceholder')}
                        {...register('rating_ceiling', optionalInt)}
                    />
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.performanceCeilingOverrideLabel')}
                    </label>
                    <input
                        type="number"
                        step="1"
                        inputMode="numeric"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={t('contestForm.performanceCeilingOverridePlaceholder')}
                        {...register('performance_ceiling_override', optionalInt)}
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                        {t('contestForm.performanceCeilingOverrideHint')}
                    </p>
                </div>
            </div>
        </div>
    );
}