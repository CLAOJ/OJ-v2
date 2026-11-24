'use client';

import { UseFormRegister } from 'react-hook-form';
import { useTranslations } from 'next-intl';
import type { ScoreboardVisibility } from '@/types';
import type { ContestFormData } from '../ContestForm';
import { optionalInt } from './optionalInt';

interface ScoreboardSectionProps {
    register: UseFormRegister<ContestFormData>;
}

const VISIBILITIES: ScoreboardVisibility[] = ['V', 'C', 'P'];

export function ScoreboardSection({ register }: ScoreboardSectionProps) {
    const t = useTranslations('Admin');
    const labels: Record<ScoreboardVisibility, string> = {
        V: t('contestForm.scoreboardVisibleOption'),
        C: t('contestForm.scoreboardHiddenContestOption'),
        P: t('contestForm.scoreboardHiddenParticipationOption'),
    };
    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('contestForm.scoreboardTitle')}</h3>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.scoreboardVisibilityLabel')}
                    </label>
                    <select
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        {...register('scoreboard_visibility')}
                    >
                        {VISIBILITIES.map(code => (
                            <option key={code} value={code}>{labels[code]}</option>
                        ))}
                    </select>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.scoreboardCacheTimeoutLabel')}
                    </label>
                    <input
                        type="number"
                        min="0"
                        step="1"
                        inputMode="numeric"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={t('contestForm.scoreboardCacheTimeoutPlaceholder')}
                        {...register('scoreboard_cache_timeout', optionalInt)}
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                        {t('contestForm.scoreboardCacheTimeoutHint')}
                    </p>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.pointsPrecisionLabel')}
                    </label>
                    <input
                        type="number"
                        min="0"
                        max="10"
                        step="1"
                        inputMode="numeric"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={t('contestForm.pointsPrecisionPlaceholder')}
                        {...register('points_precision', optionalInt)}
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                        {t('contestForm.pointsPrecisionHint')}
                    </p>
                </div>
            </div>
        </div>
    );
}
