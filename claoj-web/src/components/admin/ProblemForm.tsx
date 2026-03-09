'use client';

import { useState } from 'react';
import { useForm, UseFormRegister } from 'react-hook-form';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { BasicInfoSection } from './problem-form/BasicInfoSection';
import { ClassificationSection } from './problem-form/ClassificationSection';
import { AuthorsSection } from './problem-form/AuthorsSection';
import { SettingsSection } from './problem-form/SettingsSection';

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
        const current = (watch(field) as number[] | undefined) || [];
        if (checked) {
            setValue(field, [...current, id]);
        } else {
            setValue(field, current.filter(i => i !== id));
        }
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            <BasicInfoSection
                formData={{
                    code: watch('code'),
                    name: watch('name'),
                    description: watch('description'),
                    points: watch('points'),
                    time_limit: watch('time_limit'),
                    memory_limit: watch('memory_limit')
                }}
                errors={errors as any}
                register={register}
                onDescriptionChange={(value) => {
                    setDescription(value);
                    setValue('description', value);
                }}
                isEditMode={!!initialData?.code}
            />

            <ClassificationSection
                groups={groups}
                types={types}
                selectedGroup={selectedGroup}
                selectedTypes={selectedTypes}
                onGroupChange={(groupId) => setValue('group_id', groupId)}
                onTypeToggle={(typeId, checked) => handleMultiSelect('type_ids', typeId, checked)}
            />

            <AuthorsSection
                users={users}
                languages={languages}
                selectedAuthors={selectedAuthors}
                selectedLangs={selectedLangs}
                onAuthorToggle={(userId, checked) => handleMultiSelect('author_ids', userId, checked)}
                onLangToggle={(langId, checked) => handleMultiSelect('allowed_lang_ids', langId, checked)}
            />

            <SettingsSection
                settings={{
                    is_public: watch('is_public') || false,
                    partial: watch('partial') || false,
                    is_manually_managed: watch('is_manually_managed') || false,
                    pdf_url: watch('pdf_url')
                }}
                register={register}
            />

            <div className="flex justify-end gap-3">
                <button
                    type="submit"
                    disabled={isLoading}
                    className="px-6 py-2.5 rounded-lg bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                    {isLoading ? 'Saving...' : (initialData?.code ? 'Update Problem' : 'Create Problem')}
                </button>
            </div>
        </form>
    );
}
