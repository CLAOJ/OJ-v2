'use client';

import { UseFormRegister, RegisterOptions } from 'react-hook-form';
import { cn } from '@/lib/utils';
import { Trophy } from 'lucide-react';

interface BasicInfo {
    key: string;
    name: string;
    description: string;
    summary?: string;
}

interface BasicInfoErrors {
    key?: { message: string };
    name?: { message: string };
    description?: { message: string };
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

interface BasicInfoSectionProps {
    formData: BasicInfo;
    errors: BasicInfoErrors;
    register: UseFormRegister<ContestFormData>;
    onDescriptionChange: (value: string) => void;
    isEditMode: boolean;
}

export function BasicInfoSection({
    formData,
    errors,
    register,
    onDescriptionChange,
    isEditMode
}: BasicInfoSectionProps) {
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold flex items-center gap-2">
                <Trophy size={20} className="text-primary" />
                Basic Information
            </h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Contest Key *
                    </label>
                    <input
                        type="text"
                        className={cn(
                            "w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none font-mono",
                            errors.key && "border-destructive"
                        )}
                        placeholder="e.g., SAMPLE2026"
                        {...register('key', { required: 'Contest key is required' })}
                        disabled={isEditMode}
                    />
                    {errors.key && (
                        <p className="text-destructive text-xs mt-1">{errors.key.message}</p>
                    )}
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Contest Name *
                    </label>
                    <input
                        type="text"
                        className={cn(
                            "w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none",
                            errors.name && "border-destructive"
                        )}
                        placeholder="e.g., Sample Contest 2026"
                        {...register('name', { required: 'Contest name is required' })}
                    />
                    {errors.name && (
                        <p className="text-destructive text-xs mt-1">{errors.name.message}</p>
                    )}
                </div>
            </div>

            <div>
                <label className="text-sm font-medium text-muted-foreground block mb-2">
                    Description *
                </label>
                <textarea
                    className={cn(
                        "w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none min-h-[200px] font-mono text-sm",
                        errors.description && "border-destructive"
                    )}
                    placeholder="Contest description in Markdown..."
                    value={formData.description}
                    onChange={(e) => onDescriptionChange(e.target.value)}
                    required
                />
                {errors.description && (
                    <p className="text-destructive text-xs mt-1">{errors.description.message}</p>
                )}
            </div>

            <div>
                <label className="text-sm font-medium text-muted-foreground block mb-2">
                    Summary (optional)
                </label>
                <textarea
                    className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none min-h-[100px] text-sm"
                    placeholder="Short summary for contest list..."
                    {...register('summary')}
                />
            </div>
        </div>
    );
}
