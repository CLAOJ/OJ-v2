'use client';

import { UseFormRegister } from 'react-hook-form';

interface ProblemFormData {
    code: string;
    name: string;
    description: string;
    points: number;
    partial: boolean;
    is_public: boolean;
    time_limit: number;
    memory_limit: number;
    group_id?: number;
    type_ids?: number[];
    author_ids?: number[];
    allowed_lang_ids?: number[];
    is_manually_managed?: boolean;
    pdf_url?: string;
}

interface Settings {
    is_public: boolean;
    partial: boolean;
    is_manually_managed: boolean;
    pdf_url?: string;
}

interface SettingsSectionProps {
    settings: Settings;
    register: UseFormRegister<ProblemFormData>;
}

export function SettingsSection({ settings, register }: SettingsSectionProps) {
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">Settings</h3>

            <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('is_public')}
                        checked={settings.is_public}
                    />
                    <span className="text-sm font-medium">Public (visible to users)</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('partial')}
                        checked={settings.partial}
                    />
                    <span className="text-sm font-medium">Partial scoring</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('is_manually_managed')}
                        checked={settings.is_manually_managed}
                    />
                    <span className="text-sm font-medium">Manually managed</span>
                </label>
            </div>

            <div>
                <label htmlFor="pdf-url" className="text-sm font-medium text-muted-foreground block mb-2">
                    PDF URL (optional)
                </label>
                <input
                    id="pdf-url"
                    type="url"
                    className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    placeholder="https://example.com/problem.pdf"
                    {...register('pdf_url')}
                    value={settings.pdf_url || ''}
                />
            </div>
        </div>
    );
}
