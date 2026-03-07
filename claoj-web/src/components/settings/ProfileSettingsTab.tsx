'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { CheckCircle, Info } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { UserDetail } from '@/types';
import { useAuth } from '@/components/providers/AuthProvider';

const settingsSchema = z.object({
    display_name: z.string().max(100).optional(),
    about: z.string().max(500).optional(),
    avatar_url: z.string().url().optional().or(z.literal('')),
});

type SettingsFormValues = z.infer<typeof settingsSchema>;

interface ProfileSettingsTabProps {
    onSuccess?: () => void;
}

export default function ProfileSettingsTab({ onSuccess }: ProfileSettingsTabProps) {
    const t = useTranslations('Settings');
    const { user } = useAuth();
    const queryClient = useQueryClient();
    const [success, setSuccess] = useState(false);

    const { data: profile, isLoading } = useQuery({
        queryKey: ['user', user?.username],
        queryFn: async () => {
            const res = await api.get<UserDetail>(`/user/${user?.username}`);
            return res.data;
        },
        enabled: !!user?.username,
    });

    const {
        register,
        handleSubmit,
        reset,
        formState: { errors, isDirty },
    } = useForm<SettingsFormValues>({
        resolver: zodResolver(settingsSchema),
    });

    useEffect(() => {
        if (profile) {
            reset({
                display_name: profile.display_name,
                about: profile.about || '',
                avatar_url: profile.avatar_url || '',
            });
        }
    }, [profile, reset]);

    const { mutate: updateProfile, isPending } = useMutation({
        mutationFn: async (data: SettingsFormValues) => {
            await api.patch('/user/me', data);
        },
        onSuccess: () => {
            setSuccess(true);
            queryClient.invalidateQueries({ queryKey: ['user', user?.username] });
            onSuccess?.();
            setTimeout(() => setSuccess(false), 3000);
        },
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to update profile');
        }
    });

    if (isLoading) {
        return <div className="p-8 text-center text-muted-foreground">Loading...</div>;
    }

    return (
        <form onSubmit={handleSubmit((data) => updateProfile(data))} className="space-y-8">
            <section className="space-y-6">
                <div className="flex items-center gap-2 text-primary font-bold">
                    <Info size={18} />
                    Basic Info
                </div>

                <div className="grid grid-cols-1 gap-6">
                    <div className="flex items-center gap-4 p-4 rounded-xl border bg-muted/30">
                        <img
                            src={profile?.avatar_url || `https://www.gravatar.com/avatar/${user?.username}?s=80&d=mp`}
                            alt="Avatar"
                            className="w-16 h-16 rounded-full object-cover border-2 border-primary/20"
                        />
                        <div>
                            <p className="text-sm font-bold">Avatar</p>
                            <p className="text-xs text-muted-foreground">Avatar is managed via Gravatar</p>
                            <a
                                href="https://gravatar.com"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-xs text-primary hover:underline"
                            >
                                Manage at Gravatar
                            </a>
                        </div>
                    </div>

                    <div className="space-y-2">
                        <label className="text-sm font-medium">{t('displayName')}</label>
                        <input
                            {...register('display_name')}
                            className="flex h-12 w-full rounded-xl border border-input bg-background px-4 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                        />
                        {errors.display_name && <p className="text-xs text-destructive">{errors.display_name.message}</p>}
                    </div>

                    <div className="space-y-2">
                        <label className="text-sm font-medium">Avatar URL (optional)</label>
                        <input
                            {...register('avatar_url')}
                            placeholder="https://example.com/avatar.jpg"
                            className="flex h-12 w-full rounded-xl border border-input bg-background px-4 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                        />
                        {errors.avatar_url && <p className="text-xs text-destructive">{errors.avatar_url.message}</p>}
                    </div>

                    <div className="space-y-2">
                        <label className="text-sm font-medium">About</label>
                        <textarea
                            {...register('about')}
                            placeholder={t('aboutPlaceholder')}
                            className="flex min-h-[120px] w-full rounded-xl border border-input bg-background px-4 py-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all resize-none leading-relaxed"
                        />
                        {errors.about && <p className="text-xs text-destructive">{errors.about.message}</p>}
                    </div>
                </div>
            </section>

            {success && (
                <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: 'auto' }}
                    className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 text-sm flex items-center gap-3 font-medium"
                >
                    <CheckCircle size={18} />
                    {t('updateSuccess')}
                </motion.div>
            )}

            <button
                type="submit"
                disabled={isPending || !isDirty}
                className="px-8 h-12 rounded-xl bg-primary text-primary-foreground font-bold hover:opacity-90 active:scale-95 disabled:opacity-50 transition-all flex items-center justify-center gap-2"
            >
                {t('save')}
            </button>
        </form>
    );
}
