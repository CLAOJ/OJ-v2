'use client';

import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { cn } from '@/lib/utils';

export interface ProblemFormData {
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

export interface ProblemFormProps {
    initialData?: ProblemFormData;
    onSubmit: (data: ProblemFormData) => Promise<void>;
    isLoading?: boolean;
}

interface ProblemGroup {
    id: number;
    name: string;
}

interface ProblemType {
    id: number;
    full_name: string;
}

interface Language {
    id: number;
    name: string;
    key: string;
}

interface UserProfile {
    id: number;
    username: string;
}

export default function ProblemForm({ initialData, onSubmit, isLoading }: ProblemFormProps) {
    const { register, handleSubmit, formState: { errors }, setValue, watch } = useForm<ProblemFormData>({
        defaultValues: {
            code: initialData?.code || '',
            name: initialData?.name || '',
            description: initialData?.description || '',
            points: initialData?.points || 100,
            partial: initialData?.partial ?? true,
            is_public: initialData?.is_public ?? false,
            time_limit: initialData?.time_limit || 1,
            memory_limit: initialData?.memory_limit || 256,
            is_manually_managed: initialData?.is_manually_managed ?? false,
            pdf_url: initialData?.pdf_url || '',
        }
    });

    const [description, setDescription] = useState(initialData?.description || '');

    // Fetch groups, types, languages, and users for selects
    const { data: groups } = useQuery<{ data: ProblemGroup[] }>({
        queryKey: ['problem-groups'],
        queryFn: async () => {
            const res = await api.get('/problem/groups');
            return res.data;
        }
    });

    const { data: types } = useQuery<{ data: ProblemType[] }>({
        queryKey: ['problem-types'],
        queryFn: async () => {
            const res = await api.get('/problem/types');
            return res.data;
        }
    });

    const { data: languages } = useQuery<{ data: Language[] }>({
        queryKey: ['languages'],
        queryFn: async () => {
            const res = await api.get('/languages');
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

    const selectedGroup = watch('group_id');
    const selectedTypes = watch('type_ids') || [];
    const selectedAuthors = watch('author_ids') || [];
    const selectedLangs = watch('allowed_lang_ids') || [];

    const handleMultiSelect = (
        field: 'type_ids' | 'author_ids' | 'allowed_lang_ids',
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

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            {/* Basic Info */}
            <div className="bg-card rounded-2xl border p-6 space-y-4">
                <h3 className="text-lg font-bold">Basic Information</h3>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            Problem Code *
                        </label>
                        <input
                            type="text"
                            className={cn(
                                "w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none",
                                errors.code && "border-destructive"
                            )}
                            placeholder="e.g., SAMPLE"
                            {...register('code', { required: 'Problem code is required' })}
                            disabled={!!initialData?.code}
                        />
                        {errors.code && (
                            <p className="text-destructive text-xs mt-1">{errors.code.message}</p>
                        )}
                    </div>

                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            Problem Name *
                        </label>
                        <input
                            type="text"
                            className={cn(
                                "w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none",
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
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Description *
                    </label>
                    <textarea
                        className={cn(
                            "w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none min-h-[300px] font-mono text-sm",
                            errors.description && "border-destructive"
                        )}
                        placeholder="Problem description in Markdown..."
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

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            Points *
                        </label>
                        <input
                            type="number"
                            step="0.01"
                            className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            {...register('points', { required: true, min: 0 })}
                        />
                    </div>

                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            Time Limit (seconds) *
                        </label>
                        <input
                            type="number"
                            step="0.1"
                            min="0.1"
                            className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            {...register('time_limit', { required: true, min: 0.1 })}
                        />
                    </div>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Memory Limit (MB) *
                    </label>
                    <input
                        type="number"
                        min="1"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        {...register('memory_limit', { required: true, min: 1 })}
                    />
                </div>
            </div>

            {/* Group and Types */}
            <div className="bg-card rounded-2xl border p-6 space-y-4">
                <h3 className="text-lg font-bold">Classification</h3>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Problem Group
                    </label>
                    <select
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={selectedGroup || ''}
                        onChange={(e) => setValue('group_id', e.target.value ? Number(e.target.value) : undefined)}
                    >
                        <option value="">Select a group...</option>
                        {groups?.data.map(g => (
                            <option key={g.id} value={g.id}>{g.name}</option>
                        ))}
                    </select>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Problem Types
                    </label>
                    <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                        {types?.data.map(t => (
                            <label key={t.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                                <input
                                    type="checkbox"
                                    checked={selectedTypes.includes(t.id)}
                                    onChange={(e) => handleMultiSelect('type_ids', t.id, e.target.checked)}
                                    className="rounded"
                                />
                                <span className="text-sm">{t.full_name}</span>
                            </label>
                        ))}
                    </div>
                </div>
            </div>

            {/* Authors and Languages */}
            <div className="bg-card rounded-2xl border p-6 space-y-4">
                <h3 className="text-lg font-bold">Authors & Languages</h3>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        Authors
                    </label>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-48 overflow-y-auto">
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
                        Allowed Languages
                    </label>
                    <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                        {languages?.data.map(lang => (
                            <label key={lang.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                                <input
                                    type="checkbox"
                                    checked={selectedLangs.includes(lang.id)}
                                    onChange={(e) => handleMultiSelect('allowed_lang_ids', lang.id, e.target.checked)}
                                    className="rounded"
                                />
                                <span className="text-sm">{lang.name}</span>
                            </label>
                        ))}
                    </div>
                </div>
            </div>

            {/* Settings */}
            <div className="bg-card rounded-2xl border p-6 space-y-4">
                <h3 className="text-lg font-bold">Settings</h3>

                <div className="flex items-center gap-4">
                    <label className="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            className="rounded w-5 h-5"
                            {...register('is_public')}
                        />
                        <span className="text-sm font-medium">Public (visible to users)</span>
                    </label>

                    <label className="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            className="rounded w-5 h-5"
                            {...register('partial')}
                        />
                        <span className="text-sm font-medium">Partial scoring</span>
                    </label>

                    <label className="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            className="rounded w-5 h-5"
                            {...register('is_manually_managed')}
                        />
                        <span className="text-sm font-medium">Manually managed</span>
                    </label>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        PDF URL (optional)
                    </label>
                    <input
                        type="url"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder="https://example.com/problem.pdf"
                        {...register('pdf_url')}
                    />
                </div>
            </div>

            {/* Submit */}
            <div className="flex justify-end gap-3">
                <button
                    type="submit"
                    disabled={isLoading}
                    className="px-6 py-2.5 rounded-xl bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                    {isLoading ? 'Saving...' : (initialData?.code ? 'Update Problem' : 'Create Problem')}
                </button>
            </div>
        </form>
    );
}
