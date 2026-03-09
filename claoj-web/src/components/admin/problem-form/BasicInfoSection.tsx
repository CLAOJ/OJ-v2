'use client';

import { UseFormRegister } from 'react-hook-form';
import { cn } from '@/lib/utils';

interface BasicInfo {
    code: string;
    name: string;
    description: string;
    points: number;
    time_limit: number;
    memory_limit: number;
}

interface BasicInfoErrors {
    code?: { message: string };
    name?: { message: string };
    description?: { message: string };
}

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

interface BasicInfoSectionProps {
    formData: BasicInfo;
    errors: BasicInfoErrors;
    register: UseFormRegister<ProblemFormData>;
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
            <h3 className="text-lg font-bold">Basic Information</h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label htmlFor="problem-code" className="text-sm font-medium text-muted-foreground block mb-2">
                        Problem Code *
                    </label>
                    <input
                        id="problem-code"
                        type="text"
                        className={cn(
                            "w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none",
                            errors.code && "border-destructive"
                        )}
                        placeholder="e.g., SAMPLE"
                        {...register('code', { required: 'Problem code is required' })}
                        disabled={isEditMode}
                    />
                    {errors.code && (
                        <p className="text-destructive text-xs mt-1">{errors.code.message}</p>
                    )}
                </div>

                <div>
                    <label htmlFor="problem-name" className="text-sm font-medium text-muted-foreground block mb-2">
                        Problem Name *
                    </label>
                    <input
                        id="problem-name"
                        type="text"
                        className={cn(
                            "w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none",
                            errors.name && "border-destructive"
                        )}
                        placeholder="e.g., Sample Problem"
                        {...register('name', { required: 'Problem name is required' })}
                    />
                    {errors.name && (
                        <p className="text-destructive text-xs mt-1">{errors.name.message}</p>
                    )}
                </div>
            </div>

            <div>
                <label htmlFor="problem-description" className="text-sm font-medium text-muted-foreground block mb-2">
                    Description *
                </label>
                <textarea
                    id="problem-description"
                    className={cn(
                        "w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none min-h-[300px] font-mono text-sm",
                        errors.description && "border-destructive"
                    )}
                    placeholder="Problem description in Markdown..."
                    value={formData.description}
                    onChange={(e) => onDescriptionChange(e.target.value)}
                    required
                />
                {errors.description && (
                    <p className="text-destructive text-xs mt-1">{errors.description.message}</p>
                )}
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label htmlFor="problem-points" className="text-sm font-medium text-muted-foreground block mb-2">
                        Points *
                    </label>
                    <input
                        id="problem-points"
                        type="number"
                        step="0.01"
                        className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        {...register('points', { required: true, min: 0 })}
                    />
                </div>

                <div>
                    <label htmlFor="problem-time-limit" className="text-sm font-medium text-muted-foreground block mb-2">
                        Time Limit (seconds) *
                    </label>
                    <input
                        id="problem-time-limit"
                        type="number"
                        step="0.1"
                        min="0.1"
                        className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        {...register('time_limit', { required: true, min: 0.1 })}
                    />
                </div>
            </div>

            <div>
                <label htmlFor="problem-memory-limit" className="text-sm font-medium text-muted-foreground block mb-2">
                    Memory Limit (MB) *
                </label>
                <input
                    id="problem-memory-limit"
                    type="number"
                    min="1"
                    className="w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    {...register('memory_limit', { required: true, min: 1 })}
                />
            </div>
        </div>
    );
}
