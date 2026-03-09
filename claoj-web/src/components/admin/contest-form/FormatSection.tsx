'use client';

import { UseFormRegister } from 'react-hook-form';
import { cn } from '@/lib/utils';

interface FormatSettings {
    max_submissions?: number | null;
    format_name?: string;
    access_code?: string;
    is_visible: boolean;
    is_rated: boolean;
    hide_problem_tags: boolean;
    run_pretests_only: boolean;
    is_organization_private: boolean;
}

interface ContestFormData {
    key: string;
    name: string;
    description: string;
    summary?: string;
    start_time: string;
    end_time: string;
    time_limit?: number;
    is_visible: boolean;
    is_rated: boolean;
    format_name?: string;
    format_config?: string;
    access_code?: string;
    hide_problem_tags: boolean;
    run_pretests_only: boolean;
    is_organization_private: boolean;
    max_submissions?: number | null;
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    problem_ids?: number[];
    tag_ids?: number[];
}

interface FormatSectionProps {
    formData: FormatSettings;
    selectedFormat: string;
    register: UseFormRegister<ContestFormData>;
    setValue: (field: keyof ContestFormData, value: any) => void;
}

const CONTEST_FORMATS = [
    { value: 'icpc', label: 'ICPC' },
    { value: 'ioi', label: 'IOI' },
    { value: 'atcoder', label: 'AtCoder' },
    { value: 'ecoo', label: 'ECOO' },
];

export function FormatSection({ formData, selectedFormat, register, setValue }: FormatSectionProps) {
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">Format & Settings</h3>

            <div>
                <label className="text-sm font-medium text-muted-foreground block mb-2">
                    Max Submissions (optional)
                </label>
                <input
                    type="number"
                    min="1"
                    className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    placeholder="No limit"
                    {...register('max_submissions', { valueAsNumber: true })}
                />
                <p className="text-xs text-muted-foreground mt-1">Limit total submissions per user in this contest. Leave empty for no limit.</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Contest Format
                    </label>
                    <select
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={selectedFormat || 'icpc'}
                        onChange={(e) => setValue('format_name', e.target.value)}
                    >
                        {CONTEST_FORMATS.map(f => (
                            <option key={f.value} value={f.value}>{f.label}</option>
                        ))}
                    </select>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Access Code (for private contests)
                    </label>
                    <input
                        type="text"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder="Leave empty for public contest"
                        {...register('access_code')}
                    />
                </div>
            </div>

            <div className="flex flex-wrap items-center gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('is_visible')}
                    />
                    <span className="text-sm font-medium">Visible (public)</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('is_rated')}
                    />
                    <span className="text-sm font-medium">Rated contest</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('hide_problem_tags')}
                    />
                    <span className="text-sm font-medium">Hide problem tags</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('run_pretests_only')}
                    />
                    <span className="text-sm font-medium">Pretests only</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('is_organization_private')}
                    />
                    <span className="text-sm font-medium">Private to organization</span>
                </label>
            </div>
        </div>
    );
}
