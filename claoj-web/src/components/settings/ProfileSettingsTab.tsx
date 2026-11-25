'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useTranslations } from 'next-intl';
import api, { type ApiError } from '@/lib/api';
import { uploadDirect } from '@/lib/directUpload';
import { useState, useEffect, useRef } from 'react';
import { toast } from 'sonner';
import { motion } from 'framer-motion';
import { CheckCircle, Info, Loader2, Trash2, Upload } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { UserDetail } from '@/types';
import { useAuth } from '@/components/providers/AuthProvider';

const settingsSchema = z.object({
    display_name: z.string().max(100).optional(),
    about: z.string().max(500).optional(),
});

type SettingsFormValues = z.infer<typeof settingsSchema>;

interface ProfileSettingsTabProps {
    onSuccess?: () => void;
}

export default function ProfileSettingsTab({ onSuccess }: ProfileSettingsTabProps) {
    const t = useTranslations('Settings');
    const tc = useTranslations('Common');
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
            alert(error.response?.data?.error || t('updateProfileError'));
        }
    });

    // Avatar: the photo goes from the browser straight to R2, then PUT /user/avatar
    // verifies and keeps it. No photo = the Gravatar identicon, as in v1.
    const fileInput = useRef<HTMLInputElement>(null);
    const hasUploadedAvatar = !!profile?.avatar_url && !profile.avatar_url.includes('gravatar.com');
    const refreshProfile = () => queryClient.invalidateQueries({ queryKey: ['user', user?.username] });
    const avatarError = (err: unknown) =>
        toast.error((err as ApiError).response?.data?.error || t('avatarUploadError'));

    const uploadAvatar = useMutation({
        mutationFn: (file: File) => uploadDirect('/user/avatar/sign', file, key => api.put('/user/avatar', { key })),
        onSuccess: (res) => {
            if (res === null) {
                toast.error(t('avatarUploadUnavailable'));
                return;
            }
            toast.success(t('avatarUpdated'));
            refreshProfile();
        },
        onError: avatarError,
    });
    const removeAvatar = useMutation({
        mutationFn: () => api.delete('/user/avatar'),
        onSuccess: () => {
            toast.success(t('avatarRemoved'));
            refreshProfile();
        },
        onError: avatarError,
    });
    const avatarBusy = uploadAvatar.isPending || removeAvatar.isPending;

    const onAvatarFile = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        e.target.value = ''; // the same file can be picked again after an error
        if (!file) return;
        if (file.size > 1 << 20) {
            toast.error(t('avatarTooLarge'));
            return;
        }
        uploadAvatar.mutate(file);
    };

    if (isLoading) {
        return <div className="p-8 text-center text-muted-foreground">{tc('loading')}</div>;
    }

    return (
        <form onSubmit={handleSubmit((data) => updateProfile(data))} className="space-y-8">
            <section className="space-y-6">
                <div className="flex items-center gap-2 text-primary font-medium">
                    <Info size={18} />
                    {t('basicInfo')}
                </div>

                <div className="grid grid-cols-1 gap-6">
                    <div className="flex items-center gap-4 p-4 rounded-xl border bg-muted/30">
                        <img
                            src={profile?.avatar_url || `https://www.gravatar.com/avatar/${user?.username}?s=80&d=mp`}
                            width={64}
                            height={64}
                            alt={t('avatar')}
                            className="w-16 h-16 rounded-full object-cover border-2 border-primary/20"
                        />
                        <div className="min-w-0 space-y-2">
                            <p className="text-sm font-medium">{t('avatar')}</p>
                            <div className="flex flex-wrap gap-2">
                                <input
                                    ref={fileInput}
                                    type="file"
                                    accept="image/png,image/jpeg,image/gif,image/webp"
                                    className="sr-only"
                                    tabIndex={-1}
                                    aria-hidden="true"
                                    onChange={onAvatarFile}
                                />
                                <button
                                    type="button"
                                    onClick={() => fileInput.current?.click()}
                                    disabled={avatarBusy}
                                    className="inline-flex h-9 items-center gap-2 rounded-md bg-primary px-3 text-sm font-medium text-primary-foreground hover:opacity-90 disabled:opacity-50"
                                >
                                    {uploadAvatar.isPending ? <Loader2 size={16} className="animate-spin" /> : <Upload size={16} />}
                                    {t('uploadAvatar')}
                                </button>
                                {hasUploadedAvatar && (
                                    <button
                                        type="button"
                                        onClick={() => removeAvatar.mutate()}
                                        disabled={avatarBusy}
                                        className="inline-flex h-9 items-center gap-2 rounded-md border px-3 text-sm font-medium hover:bg-muted disabled:opacity-50"
                                    >
                                        {removeAvatar.isPending ? <Loader2 size={16} className="animate-spin" /> : <Trash2 size={16} />}
                                        {t('removeAvatar')}
                                    </button>
                                )}
                            </div>
                            <p className="text-xs text-muted-foreground">{t('avatarHint')}</p>
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
                        <label className="text-sm font-medium">{t('about')}</label>
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
                className="px-8 h-12 rounded-xl bg-primary text-primary-foreground font-medium hover:opacity-90 disabled:opacity-50 transition-all flex items-center justify-center gap-2"
            >
                {t('save')}
            </button>
        </form>
    );
}
