'use client';

import { UseFormRegister } from 'react-hook-form';
import { cn } from '@/lib/utils';
import { Clock, Calendar } from 'lucide-react';

interface ScheduleInfo {
    start_time: string;
    end_time: string;
    time_limit?: number;
}

interface ScheduleErrors {
    start_time?: { message: string };
    end_time?: { message: string };
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

interface ScheduleSectionProps {
    formData: ScheduleInfo;
    errors: ScheduleErrors;
    register: UseFormRegister<ContestFormData>;
}

export function ScheduleSection({ formData, errors, register }: ScheduleSectionProps) {
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold flex items-center gap-2">
                <Clock size={20} className="text-primary" />
                Schedule
            </h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        <Calendar size={14} className="inline mr-1" />
                        Start Time *
                    </label>
                    <input
                        type="datetime-local"
                        className={cn(
                            "w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none",
                            errors.start_time && "border-destructive"
                        )}
                        {...register('start_time', { required: 'Start time is required' })}
                    />
                    {errors.start_time && (
                        <p className="text-destructive text-xs mt-1">{errors.start_time.message}</p>
                    )}
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        <Calendar size={14} className="inline mr-1" />
                        End Time *
                    </label>
                    <input
                        type="datetime-local"
                        className={cn(
                            "w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none",
                            errors.end_time && "border-destructive"
                        )}
                        {...register('end_time', { required: 'End time is required' })}
                    />
                    {errors.end_time && (
                        <p className="text-destructive text-xs mt-1">{errors.end_time.message}</p>
                    )}
                </div>
            </div>

            <div>
                <label className="text-sm font-medium text-muted-foreground block mb-2">
                    Time Limit per Problem (seconds, optional)
                </label>
                <input
                    type="number"
                    min="0.5"
                    step="0.5"
                    className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    placeholder="Default: Use problem time limits"
                    {...register('time_limit')}
                />
            </div>
        </div>
    );
}
