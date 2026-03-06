'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { cn } from '@/lib/utils';
import { Trophy, Clock, Calendar, Tags } from 'lucide-react';

export interface ContestFormData {
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

export interface ContestFormProps {
    initialData?: ContestFormData;
    onSubmit: (data: ContestFormData) => Promise<void>;
    isLoading?: boolean;
}

interface Problem {
    id: number;
    code: string;
    name: string;
    points: number;
}

interface UserProfile {
    id: number;
    username: string;
}

interface ContestTag {
    id: number;
    name: string;
    color: string;
    description: string;
}

const CONTEST_FORMATS = [
    { value: 'icpc', label: 'ICPC' },
    { value: 'ioi', label: 'IOI' },
    { value: 'atcoder', label: 'AtCoder' },
    { value: 'ecoo', label: 'ECOO' },
];

export default function ContestForm({ initialData, onSubmit, isLoading }: ContestFormProps) {
    const { register, handleSubmit, formState: { errors }, setValue, watch } = useForm<ContestFormData>({
        defaultValues: {
            key: initialData?.key || '',
            name: initialData?.name || '',
            description: initialData?.description || '',
            summary: initialData?.summary || '',
            start_time: initialData?.start_time || '',
            end_time: initialData?.end_time || '',
            time_limit: initialData?.time_limit,
            is_visible: initialData?.is_visible ?? false,
            is_rated: initialData?.is_rated ?? false,
            format_name: initialData?.format_name || 'icpc',
            format_config: initialData?.format_config || '',
            access_code: initialData?.access_code || '',
            hide_problem_tags: initialData?.hide_problem_tags ?? false,
            run_pretests_only: initialData?.run_pretests_only ?? false,
            is_organization_private: initialData?.is_organization_private ?? false,
            max_submissions: initialData?.max_submissions,
        }
    });

    const [description, setDescription] = useState(initialData?.description || '');

    // Fetch problems and users for selects
    const { data: problems } = useQuery<{ data: Problem[] }>({
        queryKey: ['admin-problems-list'],
        queryFn: async () => {
            const res = await api.get('/admin/problems?page=1&page_size=1000');
            return res.data;
        }
    });

    const { data: users } = useQuery<{ data: UserProfile[] }>({
        queryKey: ['admin-users-list'],
        queryFn: async () => {
            const res = await api.get('/admin/users?page=1&page_size=1000');
            return res.data;
        }
    });

    const { data: tags } = useQuery<{ data: ContestTag[] }>({
        queryKey: ['admin-contest-tags-list'],
        queryFn: async () => {
            const res = await api.get('/admin/contest-tags?page=1&page_size=1000');
            return res.data;
        }
    });

    const selectedFormat = watch('format_name');
    const selectedProblems = watch('problem_ids') || [];
    const selectedAuthors = watch('author_ids') || [];
    const selectedCurators = watch('curator_ids') || [];
    const selectedTesters = watch('tester_ids') || [];
    const selectedTags = watch('tag_ids') || [];

    const handleMultiSelect = (
        field: 'problem_ids' | 'author_ids' | 'curator_ids' | 'tester_ids',
        id: number,
        checked: boolean
    ) => {
        const current = watch(field) || [];
        if (checked) {
            setValue(field, [...current, id] as any);
        } else {
            setValue(field, current.filter(i => i !== id) as any);
        }
    };

    const handleProblemSelect = (problemId: number, checked: boolean) => {
        const current = watch('problem_ids') || [];
        if (checked) {
            setValue('problem_ids', [...current, problemId] as any);
        } else {
            setValue('problem_ids', current.filter(i => i !== problemId) as any);
        }
    };

    const handleTagSelect = (tagId: number, checked: boolean) => {
        const current = watch('tag_ids') || [];
        if (checked) {
            setValue('tag_ids', [...current, tagId] as any);
        } else {
            setValue('tag_ids', current.filter(i => i !== tagId) as any);
        }
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            {/* Basic Info */}
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
                            disabled={!!initialData?.key}
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
                        value={description}
                        onChange={(e) => {
                            setDescription(e.target.value);
                            setValue('description', e.target.value);
                        }}
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

            {/* Schedule */}
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

            {/* Format and Settings */}
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

            {/* Problems */}
            <div className="bg-card rounded-2xl border p-6 space-y-4">
                <h3 className="text-lg font-bold">Problems</h3>
                <p className="text-sm text-muted-foreground">
                    Select problems to include in the contest. You can reorder them after creation.
                </p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-2 max-h-64 overflow-y-auto">
                    {problems?.data.map(p => (
                        <label key={p.id} className="flex items-center gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedProblems.includes(p.id)}
                                onChange={(e) => handleProblemSelect(p.id, e.target.checked)}
                                className="rounded w-5 h-5"
                            />
                            <div className="flex-1 min-w-0">
                                <div className="font-medium text-sm truncate">{p.name}</div>
                                <div className="text-xs text-muted-foreground">
                                    {p.code} • {p.points} pts
                                </div>
                            </div>
                        </label>
                    ))}
                </div>
                {selectedProblems.length > 0 && (
                    <div className="text-sm text-muted-foreground">
                        Selected: {selectedProblems.length} problem(s)
                    </div>
                )}
            </div>

            {/* Tags */}
            <div className="bg-card rounded-2xl border p-6 space-y-4">
                <h3 className="text-lg font-bold flex items-center gap-2">
                    <Tags size={20} className="text-primary" />
                    Contest Tags
                </h3>
                <p className="text-sm text-muted-foreground">
                    Select tags to categorize this contest.
                </p>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-48 overflow-y-auto">
                    {tags?.data.map(t => (
                        <label key={t.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedTags.includes(t.id)}
                                onChange={(e) => handleTagSelect(t.id, e.target.checked)}
                                className="rounded"
                            />
                            <div className="flex-1 min-w-0">
                                <div className="font-medium text-sm truncate" style={{ color: t.color }}>{t.name}</div>
                            </div>
                        </label>
                    ))}
                </div>
                {selectedTags.length > 0 && (
                    <div className="text-sm text-muted-foreground">
                        Selected: {selectedTags.length} tag(s)
                    </div>
                )}
            </div>

            {/* Staff */}
            <div className="bg-card rounded-2xl border p-6 space-y-4">
                <h3 className="text-lg font-bold">Contest Staff</h3>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Authors
                    </label>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-32 overflow-y-auto">
                        {users?.data.map(u => (
                            <label key={u.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                                <input
                                    type="checkbox"
                                    checked={selectedAuthors.includes(u.id)}
                                    onChange={(e) => handleMultiSelect('author_ids', u.id, e.target.checked)}
                                    className="rounded"
                                />
                                <span className="text-sm truncate">{u.username}</span>
                            </label>
                        ))}
                    </div>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Curators
                    </label>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-32 overflow-y-auto">
                        {users?.data.map(u => (
                            <label key={u.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                                <input
                                    type="checkbox"
                                    checked={selectedCurators.includes(u.id)}
                                    onChange={(e) => handleMultiSelect('curator_ids', u.id, e.target.checked)}
                                    className="rounded"
                                />
                                <span className="text-sm truncate">{u.username}</span>
                            </label>
                        ))}
                    </div>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Testers
                    </label>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-32 overflow-y-auto">
                        {users?.data.map(u => (
                            <label key={u.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                                <input
                                    type="checkbox"
                                    checked={selectedTesters.includes(u.id)}
                                    onChange={(e) => handleMultiSelect('tester_ids', u.id, e.target.checked)}
                                    className="rounded"
                                />
                                <span className="text-sm truncate">{u.username}</span>
                            </label>
                        ))}
                    </div>
                </div>
            </div>

            {/* Submit */}
            <div className="flex justify-end gap-3">
                <button
                    type="submit"
                    disabled={isLoading}
                    className="px-6 py-2.5 rounded-xl bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                    {isLoading ? 'Saving...' : (initialData?.key ? 'Update Contest' : 'Create Contest')}
                </button>
            </div>
        </form>
    );
}
