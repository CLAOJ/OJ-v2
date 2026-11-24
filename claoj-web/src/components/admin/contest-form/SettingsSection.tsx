'use client';

import { UseFormRegister } from 'react-hook-form';
import { useTranslations } from 'next-intl';
import type { ContestFormData } from '../ContestForm';

interface Settings {
    push_announcements: boolean;
    hide_problem_authors: boolean;
    show_short_display: boolean;
    use_clarifications: boolean;
}

interface SettingsSectionProps {
    settings: Settings;
    register: UseFormRegister<ContestFormData>;
}

export function SettingsSection({ settings, register }: SettingsSectionProps) {
    const t = useTranslations('Admin');
    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('contestForm.settingsTitle')}</h3>

            <div className="flex flex-wrap items-center gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('push_announcements')}
                        checked={settings.push_announcements}
                    />
                    <span className="text-sm font-medium">{t('contestForm.pushAnnouncementsLabel')}</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('hide_problem_authors')}
                        checked={settings.hide_problem_authors}
                    />
                    <span className="text-sm font-medium">{t('contestForm.hideProblemAuthorsLabel')}</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('show_short_display')}
                        checked={settings.show_short_display}
                    />
                    <span className="text-sm font-medium">{t('contestForm.showShortDisplayLabel')}</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('use_clarifications')}
                        checked={settings.use_clarifications}
                    />
                    <span className="text-sm font-medium">{t('contestForm.noCommentsLabel')}</span>
                </label>
            </div>
        </div>
    );
}