'use client';

import { UseFormRegister } from 'react-hook-form';
import { useTranslations } from 'next-intl';
import type { ContestFormData } from '../ContestForm';

interface DetailsSectionProps {
    register: UseFormRegister<ContestFormData>;
}

export function DetailsSection({ register }: DetailsSectionProps) {
    const t = useTranslations('Admin');
    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('contestForm.detailsTitle')}</h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.ogImageLabel')}
                    </label>
                    <input
                        type="text"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={t('contestForm.ogImagePlaceholder')}
                        {...register('og_image')}
                    />
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.logoOverrideLabel')}
                    </label>
                    <input
                        type="text"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={t('contestForm.logoOverridePlaceholder')}
                        {...register('logo_override_image')}
                    />
                </div>
            </div>

            <div>
                <label className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('contestForm.problemLabelScriptLabel')}
                </label>
                <textarea
                    rows={4}
                    className="w-full px-3 py-2 rounded-xl bg-card border font-mono text-sm focus:ring-2 focus:ring-primary/20 outline-none"
                    placeholder={t('contestForm.problemLabelScriptPlaceholder')}
                    {...register('problem_label_script')}
                />
                <p className="text-xs text-muted-foreground mt-1">
                    {t('contestForm.problemLabelScriptHint')}
                </p>
            </div>
        </div>
    );
}