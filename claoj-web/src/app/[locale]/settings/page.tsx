'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { useAuth } from '@/components/providers/AuthProvider';
import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Loader2, CheckCircle, User, Info, Shield, Bell, Lock, Key, Globe, Smartphone, Copy, Download } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { UserDetail } from '@/types';
import { Badge } from '@/components/ui/Badge';
import WebAuthnSettings from '@/components/settings/WebAuthnSettings';

const settingsSchema = z.object({
    display_name: z.string().max(100).optional(),
    about: z.string().max(500).optional(),
    avatar_url: z.string().url().optional().or(z.literal('')),
});

const passwordSchema = z.object({
    current_password: z.string().min(1, 'Current password is required'),
    new_password: z.string().min(6, 'New password must be at least 6 characters'),
    confirm_password: z.string().min(6, 'Please confirm your new password'),
}).refine((data) => data.new_password === data.confirm_password, {
    message: "Passwords don't match",
    path: ["confirm_password"],
});

type SettingsFormValues = z.infer<typeof settingsSchema>;
type PasswordFormValues = z.infer<typeof passwordSchema>;

type ActiveTab = 'profile' | 'account' | 'oauth' | 'notifications' | 'api-token' | 'data-export' | 'webauthn';

type TwoFactorStatus = {
    enabled: boolean;
    backup_codes_remaining: number;
};

type TwoFactorSetup = {
    secret: string;
    url: string;
    qr_code: string;
    backup_codes?: string[];
};

export default function SettingsPage() {
    const t = useTranslations('Settings');
    const { user } = useAuth();
    const queryClient = useQueryClient();
    const [success, setSuccess] = useState(false);
    const [activeTab, setActiveTab] = useState<ActiveTab>('profile');
    const [passwordSuccess, setPasswordSuccess] = useState(false);

    // 2FA states
    const [twoFactorStep, setTwoFactorStep] = useState<'disabled' | 'setup' | 'confirm' | 'enabled'>('disabled');
    const [twoFactorSecret, setTwoFactorSecret] = useState<TwoFactorSetup | null>(null);
    const [twoFactorCode, setTwoFactorCode] = useState('');
    const [twoFactorPassword, setTwoFactorPassword] = useState('');
    const [backupCodes, setBackupCodes] = useState<string[] | null>(null);
    const [disablePassword, setDisablePassword] = useState('');
    const [disableCode, setDisableCode] = useState('');

    const { data: twoFactorStatus, refetch: refetchTwoFactor } = useQuery({
        queryKey: ['totp', 'status'],
        queryFn: async () => {
            const res = await api.get<TwoFactorStatus>('/auth/totp/status');
            return res.data;
        },
    });

    useEffect(() => {
        if (twoFactorStatus?.enabled) {
            setTwoFactorStep('enabled');
        } else {
            setTwoFactorStep('disabled');
        }
    }, [twoFactorStatus]);

    const { data: profile, isLoading: isFetching } = useQuery({
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

    const {
        register: registerPassword,
        handleSubmit: handlePasswordSubmit,
        reset: resetPassword,
        formState: { errors: passwordErrors },
    } = useForm<PasswordFormValues>({
        resolver: zodResolver(passwordSchema),
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

    const { mutate: updateProfile, isPending: isUpdating } = useMutation({
        mutationFn: async (data: SettingsFormValues) => {
            await api.patch('/user/me', data);
        },
        onSuccess: () => {
            setSuccess(true);
            queryClient.invalidateQueries({ queryKey: ['user', user?.username] });
            setTimeout(() => setSuccess(false), 3000);
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to update profile');
        }
    });

    const { mutate: changePassword, isPending: isChangingPassword } = useMutation({
        mutationFn: async (data: PasswordFormValues) => {
            await api.post('/auth/password/change', {
                current_password: data.current_password,
                new_password: data.new_password,
            });
        },
        onSuccess: () => {
            setPasswordSuccess(true);
            resetPassword();
            setTimeout(() => setPasswordSuccess(false), 3000);
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to change password');
        }
    });

    const handleOAuthConnect = (provider: string) => {
        window.location.href = `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v2'}/auth/oauth/${provider}`;
    };

    // 2FA Setup mutation
    const { mutate: setupTwoFactor, isPending: isSettingUpTwoFactor } = useMutation({
        mutationFn: async (password: string) => {
            const res = await api.post<TwoFactorSetup>('/auth/totp/setup', { password });
            return res.data;
        },
        onSuccess: (data) => {
            setTwoFactorSecret(data);
            setTwoFactorStep('confirm');
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to setup 2FA');
        }
    });

    // 2FA Confirm mutation
    const { mutate: confirmTwoFactor, isPending: isConfirmingTwoFactor } = useMutation({
        mutationFn: async (code: string) => {
            const res = await api.post<{ backup_codes: string[] }>('/auth/totp/confirm', { code });
            return res.data;
        },
        onSuccess: (data) => {
            setBackupCodes(data.backup_codes);
            setTwoFactorStep('enabled');
            refetchTwoFactor();
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Invalid code');
        }
    });

    // 2FA Disable mutation
    const { mutate: disableTwoFactor, isPending: isDisablingTwoFactor } = useMutation({
        mutationFn: async ({ code, password }: { code: string; password: string }) => {
            await api.post('/auth/totp/disable', { code, password });
        },
        onSuccess: () => {
            setTwoFactorStep('disabled');
            setTwoFactorSecret(null);
            setBackupCodes(null);
            refetchTwoFactor();
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to disable 2FA');
        }
    });

    // 2FA Backup codes regeneration
    const { mutate: regenerateBackupCodes, isPending: isRegeneratingBackupCodes } = useMutation({
        mutationFn: async (password: string) => {
            const res = await api.post<{ backup_codes: string[] }>('/auth/totp/backup-codes', { password });
            return res.data;
        },
        onSuccess: (data) => {
            setBackupCodes(data.backup_codes);
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to regenerate backup codes');
        }
    });

    const copyToClipboard = async (text: string) => {
        try {
            await navigator.clipboard.writeText(text);
            alert('Copied to clipboard!');
        } catch {
            alert('Failed to copy');
        }
    };

    const downloadBackupCodes = () => {
        if (!backupCodes) return;
        const content = `CLAOJ Backup Codes\n==================\n\nSave these codes in a safe place. Each code can only be used once.\n\n${backupCodes.join('\n')}\n\nGenerated on: ${new Date().toLocaleString()}`;
        const blob = new Blob([content], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'claoj-backup-codes.txt';
        a.click();
        URL.revokeObjectURL(url);
    };

    if (isFetching) return <div className="p-8"><Skeleton className="h-[40vh] w-full" /></div>;

    return (
        <div className="max-w-4xl mx-auto space-y-8 animate-in fade-in duration-500">
            <header>
                <h1 className="text-3xl font-bold tracking-tight">{t('title')}</h1>
                <p className="text-muted-foreground mt-1">Manage your profile and account settings.</p>
            </header>

            <div className="flex flex-col md:flex-row gap-8">
                {/* Sidebar Tabs */}
                <aside className="w-full md:w-64 space-y-1">
                    <button
                        onClick={() => setActiveTab('profile')}
                        className={cn(
                            "w-full flex items-center gap-3 px-4 py-2.5 rounded-xl font-bold text-sm text-left transition-all",
                            activeTab === 'profile'
                                ? "bg-primary/10 text-primary"
                                : "hover:bg-muted text-muted-foreground"
                        )}
                    >
                        <User size={18} />
                        {t('profile')}
                    </button>
                    <button
                        onClick={() => setActiveTab('account')}
                        className={cn(
                            "w-full flex items-center gap-3 px-4 py-2.5 rounded-xl font-bold text-sm text-left transition-all",
                            activeTab === 'account'
                                ? "bg-primary/10 text-primary"
                                : "hover:bg-muted text-muted-foreground"
                        )}
                    >
                        <Shield size={18} />
                        {t('account')}
                    </button>
                    <button
                        onClick={() => setActiveTab('oauth')}
                        className={cn(
                            "w-full flex items-center gap-3 px-4 py-2.5 rounded-xl font-bold text-sm text-left transition-all",
                            activeTab === 'oauth'
                                ? "bg-primary/10 text-primary"
                                : "hover:bg-muted text-muted-foreground"
                        )}
                    >
                        <Globe size={18} />
                        OAuth
                    </button>
                    <button
                        onClick={() => setActiveTab('notifications')}
                        className={cn(
                            "w-full flex items-center gap-3 px-4 py-2.5 rounded-xl font-bold text-sm text-left transition-all",
                            activeTab === 'notifications'
                                ? "bg-primary/10 text-primary"
                                : "hover:bg-muted text-muted-foreground"
                        )}
                    >
                        <Bell size={18} />
                        Notifications
                    </button>
                    <button
                        onClick={() => setActiveTab('api-token')}
                        className={cn(
                            "w-full flex items-center gap-3 px-4 py-2.5 rounded-xl font-bold text-sm text-left transition-all",
                            activeTab === 'api-token'
                                ? "bg-primary/10 text-primary"
                                : "hover:bg-muted text-muted-foreground"
                        )}
                    >
                        <Key size={18} />
                        API Token
                    </button>
                    <button
                        onClick={() => setActiveTab('data-export')}
                        className={cn(
                            "w-full flex items-center gap-3 px-4 py-2.5 rounded-xl font-bold text-sm text-left transition-all",
                            activeTab === 'data-export'
                                ? "bg-primary/10 text-primary"
                                : "hover:bg-muted text-muted-foreground"
                        )}
                    >
                        <Download size={18} />
                        Data Export
                    </button>
                    <button
                        onClick={() => setActiveTab('webauthn')}
                        className={cn(
                            "w-full flex items-center gap-3 px-4 py-2.5 rounded-xl font-bold text-sm text-left transition-all",
                            activeTab === 'webauthn'
                                ? "bg-primary/10 text-primary"
                                : "hover:bg-muted text-muted-foreground"
                        )}
                    >
                        <Shield size={18} />
                        Passkey (WebAuthn)
                    </button>
                </aside>

                {/* Content Area */}
                <div className="flex-grow p-8 rounded-3xl border bg-card shadow-sm">
                    {/* Profile Tab */}
                    {activeTab === 'profile' && (
                        <form onSubmit={handleSubmit((data) => updateProfile(data))} className="space-y-8">
                            <section className="space-y-6">
                                <div className="flex items-center gap-2 text-primary font-bold">
                                    <Info size={18} />
                                    Basic Info
                                </div>

                                <div className="grid grid-cols-1 gap-6">
                                    {/* Avatar Preview */}
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
                                disabled={isUpdating || !isDirty}
                                className="px-8 h-12 rounded-xl bg-primary text-primary-foreground font-bold hover:opacity-90 active:scale-95 disabled:opacity-50 transition-all flex items-center justify-center gap-2"
                            >
                                {isUpdating && <Loader2 size={18} className="animate-spin" />}
                                {t('save')}
                            </button>
                        </form>
                    )}

                    {/* Account Tab */}
                    {activeTab === 'account' && (
                        <div className="space-y-8">
                            {/* Change Password Section */}
                            <form onSubmit={handlePasswordSubmit((data) => changePassword(data))} className="space-y-8">
                                <section className="space-y-6">
                                    <div className="flex items-center gap-2 text-primary font-bold">
                                        <Lock size={18} />
                                        Change Password
                                    </div>

                                    <div className="grid grid-cols-1 gap-6">
                                        <div className="space-y-2">
                                            <label className="text-sm font-medium">Current Password</label>
                                            <div className="relative">
                                                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                                                <input
                                                    {...registerPassword('current_password')}
                                                    type="password"
                                                    className="flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                                                />
                                            </div>
                                            {passwordErrors.current_password && <p className="text-xs text-destructive">{passwordErrors.current_password.message}</p>}
                                        </div>

                                        <div className="space-y-2">
                                            <label className="text-sm font-medium">New Password</label>
                                            <div className="relative">
                                                <Key className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                                                <input
                                                    {...registerPassword('new_password')}
                                                    type="password"
                                                    className="flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                                                />
                                            </div>
                                            {passwordErrors.new_password && <p className="text-xs text-destructive">{passwordErrors.new_password.message}</p>}
                                        </div>

                                        <div className="space-y-2">
                                            <label className="text-sm font-medium">Confirm New Password</label>
                                            <div className="relative">
                                                <Key className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                                                <input
                                                    {...registerPassword('confirm_password')}
                                                    type="password"
                                                    className="flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                                                />
                                            </div>
                                            {passwordErrors.confirm_password && <p className="text-xs text-destructive">{passwordErrors.confirm_password.message}</p>}
                                        </div>
                                    </div>
                                </section>

                                {passwordSuccess && (
                                    <motion.div
                                        initial={{ opacity: 0, height: 0 }}
                                        animate={{ opacity: 1, height: 'auto' }}
                                        className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 text-sm flex items-center gap-3 font-medium"
                                    >
                                        <CheckCircle size={18} />
                                        Password changed successfully
                                    </motion.div>
                                )}

                                <button
                                    type="submit"
                                    disabled={isChangingPassword}
                                    className="px-8 h-12 rounded-xl bg-primary text-primary-foreground font-bold hover:opacity-90 active:scale-95 disabled:opacity-50 transition-all flex items-center justify-center gap-2"
                                >
                                    {isChangingPassword && <Loader2 size={18} className="animate-spin" />}
                                    Change Password
                                </button>
                            </form>

                            <hr className="border-border" />

                            {/* Two-Factor Authentication Section */}
                            <section className="space-y-6">
                                <div className="flex items-center gap-2 text-primary font-bold">
                                    <Smartphone size={18} />
                                    Two-Factor Authentication
                                </div>

                                {twoFactorStep === 'disabled' && (
                                    <div className="space-y-4">
                                        <p className="text-sm text-muted-foreground">
                                            Add an extra layer of security to your account by enabling two-factor authentication.
                                        </p>
                                        <div className="space-y-2">
                                            <label className="text-sm font-medium">Password</label>
                                            <div className="relative">
                                                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                                                <input
                                                    type="password"
                                                    value={twoFactorPassword}
                                                    onChange={(e) => setTwoFactorPassword(e.target.value)}
                                                    placeholder="Enter your password to continue"
                                                    className="flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                                                />
                                            </div>
                                        </div>
                                        <button
                                            onClick={() => setupTwoFactor(twoFactorPassword)}
                                            disabled={isSettingUpTwoFactor || !twoFactorPassword}
                                            className="px-6 h-12 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-all disabled:opacity-50 flex items-center gap-2"
                                        >
                                            {isSettingUpTwoFactor && <Loader2 size={18} className="animate-spin" />}
                                            Setup 2FA
                                        </button>
                                    </div>
                                )}

                                {twoFactorStep === 'setup' && twoFactorSecret && (
                                    <div className="space-y-4">
                                        <div className="p-4 rounded-xl bg-muted/50 border text-center">
                                            <p className="text-sm font-medium mb-4">Scan this QR code with your authenticator app</p>
                                            <div className="w-48 h-48 mx-auto bg-white rounded-xl overflow-hidden flex items-center justify-center">
                                                <img src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(twoFactorSecret.url)}`} alt="QR Code" className="w-full h-full object-contain" />
                                            </div>
                                            <p className="text-xs text-muted-foreground mt-4 font-mono">
                                                Secret: <span className="select-all">{twoFactorSecret.secret}</span>
                                            </p>
                                            <button
                                                onClick={() => copyToClipboard(twoFactorSecret.secret)}
                                                className="mt-2 text-xs text-primary hover:underline flex items-center gap-1"
                                            >
                                                <Copy size={12} /> Copy secret
                                            </button>
                                        </div>
                                        <div className="space-y-2">
                                            <label className="text-sm font-medium">Verification Code</label>
                                            <input
                                                type="text"
                                                value={twoFactorCode}
                                                onChange={(e) => setTwoFactorCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                                                placeholder="Enter 6-digit code from your app"
                                                className="flex h-12 w-full rounded-xl border border-input bg-background px-4 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium text-center tracking-widest"
                                                maxLength={6}
                                            />
                                        </div>
                                        <div className="flex gap-2">
                                            <button
                                                onClick={() => confirmTwoFactor(twoFactorCode)}
                                                disabled={isConfirmingTwoFactor || twoFactorCode.length !== 6}
                                                className="flex-1 px-6 h-12 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-all disabled:opacity-50 flex items-center justify-center gap-2"
                                            >
                                                {isConfirmingTwoFactor && <Loader2 size={18} className="animate-spin" />}
                                                Confirm
                                            </button>
                                            <button
                                                onClick={() => { setTwoFactorStep('disabled'); setTwoFactorPassword(''); }}
                                                className="px-6 h-12 rounded-xl border font-bold hover:bg-muted transition-all"
                                            >
                                                Cancel
                                            </button>
                                        </div>
                                    </div>
                                )}

                                {twoFactorStep === 'confirm' && backupCodes && (
                                    <div className="space-y-4">
                                        <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20">
                                            <p className="text-sm font-bold text-emerald-600 flex items-center gap-2">
                                                <CheckCircle size={16} />
                                                2FA Enabled Successfully!
                                            </p>
                                        </div>
                                        <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20">
                                            <p className="text-sm font-medium text-amber-700 mb-2">
                                                <strong>Important:</strong> Save these backup codes in a safe place. Each code can only be used once.
                                            </p>
                                            <div className="grid grid-cols-2 gap-2 font-mono text-xs bg-white p-3 rounded-lg border">
                                                {backupCodes.map((code, i) => (
                                                    <div key={i} className="flex items-center justify-between p-1.5 bg-muted rounded">
                                                        <span>{code}</span>
                                                        <button onClick={() => copyToClipboard(code)} className="text-muted-foreground hover:text-primary">
                                                            <Copy size={12} />
                                                        </button>
                                                    </div>
                                                ))}
                                            </div>
                                            <button
                                                onClick={downloadBackupCodes}
                                                className="mt-3 text-sm text-primary hover:underline flex items-center gap-2"
                                            >
                                                <Download size={14} /> Download backup codes
                                            </button>
                                        </div>
                                    </div>
                                )}

                                {twoFactorStep === 'enabled' && (
                                    <div className="space-y-4">
                                        <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-between">
                                            <p className="text-sm font-medium text-emerald-700 flex items-center gap-2">
                                                <CheckCircle size={16} />
                                                Two-factor authentication is enabled
                                            </p>
                                            <Badge variant="default" className="bg-emerald-500 text-white">
                                                Active
                                            </Badge>
                                        </div>
                                        <div className="space-y-2">
                                            <label className="text-sm font-medium">Disable 2FA</label>
                                            <div className="space-y-2">
                                                <div className="relative">
                                                    <Lock className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                                                    <input
                                                        type="password"
                                                        value={disableCode}
                                                        onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                                                        placeholder="Enter 6-digit code"
                                                        className="flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                                                        maxLength={6}
                                                    />
                                                </div>
                                                <div className="relative">
                                                    <Key className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                                                    <input
                                                        type="password"
                                                        value={disablePassword}
                                                        onChange={(e) => setDisablePassword(e.target.value)}
                                                        placeholder="Enter your password"
                                                        className="flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                                                    />
                                                </div>
                                            </div>
                                            <button
                                                onClick={() => disableTwoFactor({ code: disableCode, password: disablePassword })}
                                                disabled={isDisablingTwoFactor || !disableCode || !disablePassword}
                                                className="px-6 h-12 rounded-xl border font-bold hover:bg-destructive/10 hover:text-destructive transition-all disabled:opacity-50 flex items-center gap-2"
                                            >
                                                {isDisablingTwoFactor && <Loader2 size={18} className="animate-spin" />}
                                                Disable 2FA
                                            </button>
                                        </div>
                                        {twoFactorStatus?.backup_codes_remaining !== undefined && (
                                            <p className="text-xs text-muted-foreground">
                                                Backup codes remaining: {twoFactorStatus.backup_codes_remaining}
                                            </p>
                                        )}
                                    </div>
                                )}
                            </section>

                            {/* Danger Zone */}
                            <hr className="border-destructive/50" />
                            <section className="space-y-4">
                                <div className="flex items-center gap-2 text-destructive font-bold">
                                    <Info size={18} />
                                    Danger Zone
                                </div>
                                <p className="text-sm text-muted-foreground">
                                    Once you delete your account, there is no going back. This action is permanent and irreversible.
                                </p>
                                <button className="px-6 h-12 rounded-xl bg-destructive text-destructive-foreground font-bold hover:bg-destructive/90 transition-all flex items-center gap-2">
                                    Delete Account
                                </button>
                            </section>
                        </div>
                    )}

                    {/* OAuth Tab */}
                    {activeTab === 'oauth' && (
                        <div className="space-y-6">
                            <section className="space-y-6">
                                <div className="flex items-center gap-2 text-primary font-bold">
                                    <Globe size={18} />
                                    Connected Accounts
                                </div>

                                <p className="text-sm text-muted-foreground">
                                    Connect your Google or GitHub account for easier sign-in.
                                </p>

                                <div className="space-y-4">
                                    {/* Google */}
                                    <div className="flex items-center justify-between p-4 rounded-xl border bg-muted/30">
                                        <div className="flex items-center gap-4">
                                            <div className="w-12 h-12 rounded-xl bg-white flex items-center justify-center">
                                                <svg className="h-6 w-6" viewBox="0 0 24 24">
                                                    <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                                                    <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                                                    <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                                                    <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
                                                </svg>
                                            </div>
                                            <div>
                                                <h3 className="font-bold">Google</h3>
                                                <p className="text-xs text-muted-foreground">Connect your Google account</p>
                                            </div>
                                        </div>
                                        <button
                                            onClick={() => handleOAuthConnect('google')}
                                            className="px-4 py-2 rounded-xl bg-primary text-primary-foreground font-bold text-sm hover:bg-primary/90 transition-colors"
                                        >
                                            Connect
                                        </button>
                                    </div>

                                    {/* GitHub */}
                                    <div className="flex items-center justify-between p-4 rounded-xl border bg-muted/30">
                                        <div className="flex items-center gap-4">
                                            <div className="w-12 h-12 rounded-xl bg-zinc-900 flex items-center justify-center">
                                                <svg className="h-6 w-6 text-white" fill="currentColor" viewBox="0 0 24 24">
                                                    <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd"/>
                                                </svg>
                                            </div>
                                            <div>
                                                <h3 className="font-bold">GitHub</h3>
                                                <p className="text-xs text-muted-foreground">Connect your GitHub account</p>
                                            </div>
                                        </div>
                                        <button
                                            onClick={() => handleOAuthConnect('github')}
                                            className="px-4 py-2 rounded-xl bg-primary text-primary-foreground font-bold text-sm hover:bg-primary/90 transition-colors"
                                        >
                                            Connect
                                        </button>
                                    </div>
                                </div>
                            </section>
                        </div>
                    )}

                    {/* API Token Tab */}
                    {activeTab === 'api-token' && (
                        <APITokenTab />
                    )}

                    {/* Data Export Tab */}
                    {activeTab === 'webauthn' && (
                        <WebAuthnSettings />
                    )}

                    {activeTab === 'data-export' && (
                        <DataExportTab />
                    )}

                    {/* Notifications Tab */}
                    {activeTab === 'notifications' && (
                        <NotificationsTab />
                    )}
                </div>
            </div>
        </div>
    );
}

// Notifications Tab Component
function NotificationsTab() {
    const queryClient = useQueryClient();

    const { data: preferences, isLoading } = useQuery({
        queryKey: ['notifications', 'preferences'],
        queryFn: async () => {
            const res = await api.get('/notifications/preferences');
            return res.data;
        },
    });

    const { mutate: updatePreferences, isPending } = useMutation({
        mutationFn: async (data: Record<string, boolean>) => {
            await api.patch('/notifications/preferences', data);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['notifications', 'preferences'] });
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to update preferences');
        }
    });

    const ToggleSwitch = ({ label, description, checked, onChange }: {
        label: string;
        description: string;
        checked: boolean;
        onChange: (checked: boolean) => void;
    }) => (
        <div className="flex items-center justify-between p-4 rounded-xl border bg-muted/30">
            <div className="space-y-1">
                <p className="text-sm font-bold">{label}</p>
                <p className="text-xs text-muted-foreground">{description}</p>
            </div>
            <button
                onClick={() => onChange(!checked)}
                className={cn(
                    "relative w-12 h-6 rounded-full transition-colors",
                    checked ? "bg-primary" : "bg-muted"
                )}
            >
                <div
                    className={cn(
                        "absolute top-1 w-4 h-4 rounded-full bg-white transition-transform",
                        checked ? "left-7" : "left-1"
                    )}
                />
            </button>
        </div>
    );

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-12">
                <Loader2 size={24} className="animate-spin text-muted-foreground" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <section className="space-y-4">
                <div className="flex items-center gap-2 text-primary font-bold">
                    <Bell size={18} />
                    Notification Preferences
                </div>
                <p className="text-sm text-muted-foreground">
                    Choose how you want to receive notifications.
                </p>

                <div className="space-y-4">
                    <h3 className="text-sm font-bold text-muted-foreground uppercase tracking-wider">Email Notifications</h3>
                    <ToggleSwitch
                        label="Submission Results"
                        description="Receive email when your submission is graded"
                        checked={preferences?.email_on_submission_result ?? true}
                        onChange={(val) => updatePreferences({ email_on_submission_result: val })}
                    />
                    <ToggleSwitch
                        label="Contest Start"
                        description="Receive email when a contest you joined is about to start"
                        checked={preferences?.email_on_contest_start ?? true}
                        onChange={(val) => updatePreferences({ email_on_contest_start: val })}
                    />
                    <ToggleSwitch
                        label="Ticket Replies"
                        description="Receive email when a ticket receives a reply"
                        checked={preferences?.email_on_ticket_reply ?? true}
                        onChange={(val) => updatePreferences({ email_on_ticket_reply: val })}
                    />

                    <h3 className="text-sm font-bold text-muted-foreground uppercase tracking-wider mt-6">Web Notifications</h3>
                    <ToggleSwitch
                        label="Submission Results"
                        description="Show web notification when your submission is graded"
                        checked={preferences?.web_on_submission_result ?? true}
                        onChange={(val) => updatePreferences({ web_on_submission_result: val })}
                    />
                    <ToggleSwitch
                        label="Contest Start"
                        description="Show web notification when a contest you joined is about to start"
                        checked={preferences?.web_on_contest_start ?? true}
                        onChange={(val) => updatePreferences({ web_on_contest_start: val })}
                    />
                    <ToggleSwitch
                        label="Ticket Replies"
                        description="Show web notification when a ticket receives a reply"
                        checked={preferences?.web_on_ticket_reply ?? true}
                        onChange={(val) => updatePreferences({ web_on_ticket_reply: val })}
                    />
                </div>
            </section>
        </div>
    );
}

function Skeleton({ className }: { className?: string }) {
    return <div className={cn("animate-pulse bg-muted rounded-xl", className)} />;
}

// API Token Tab Component
function APITokenTab() {
    const queryClient = useQueryClient();

    const { data: tokenInfo, isLoading } = useQuery({
        queryKey: ['api-token'],
        queryFn: async () => {
            const res = await api.get('/user/api-token');
            return res.data;
        },
    });

    const { mutate: generateToken, isPending: isGenerating } = useMutation({
        mutationFn: async () => {
            const res = await api.post('/user/api-token');
            return res.data;
        },
        onSuccess: (data: any) => {
            queryClient.invalidateQueries({ queryKey: ['api-token'] });
            if (data.token) {
                navigator.clipboard.writeText(data.token);
            }
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to generate API token');
        },
    });

    const { mutate: revokeToken, isPending: isRevoking } = useMutation({
        mutationFn: async () => {
            const res = await api.delete('/user/api-token');
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['api-token'] });
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to revoke API token');
        },
    });

    const [showToken, setShowToken] = useState(false);
    const [generatedToken, setGeneratedToken] = useState<string | null>(null);

    const handleGenerate = () => {
        generateToken(undefined, {
            onSuccess: (data: any) => {
                if (data.token) {
                    setGeneratedToken(data.token);
                    setShowToken(true);
                }
            },
        });
    };

    return (
        <div className="space-y-6">
            <div className="space-y-2">
                <h2 className="text-2xl font-bold">API Token</h2>
                <p className="text-muted-foreground text-sm">
                    Generate a token to access your account programmatically.
                </p>
            </div>

            {tokenInfo?.has_token && !generatedToken ? (
                <div className="p-6 rounded-2xl border bg-card space-y-4">
                    <div className="flex items-center gap-2 text-amber-600">
                        <Info size={20} />
                        <span className="font-bold text-sm">Existing Token</span>
                    </div>
                    <p className="text-sm text-muted-foreground">
                        You already have an API token. For security reasons, you cannot view it again.
                        Generate a new token if you&apos;ve lost yours (this will invalidate the old token).
                    </p>
                    <button
                        onClick={() => revokeToken()}
                        disabled={isRevoking}
                        className="px-6 py-3 rounded-xl bg-destructive text-destructive-foreground font-bold text-sm hover:bg-destructive/90 transition-colors disabled:opacity-50"
                    >
                        {isRevoking ? 'Revoking...' : 'Revoke Token'}
                    </button>
                </div>
            ) : generatedToken ? (
                <div className="p-6 rounded-2xl border bg-card space-y-4">
                    <div className="flex items-center gap-2 text-green-600">
                        <CheckCircle size={20} />
                        <span className="font-bold text-sm">Token Generated</span>
                    </div>
                    <div className="p-4 bg-muted rounded-xl">
                        <code className="text-xs break-all">{generatedToken}</code>
                    </div>
                    <button
                        onClick={() => {
                            navigator.clipboard.writeText(generatedToken);
                        }}
                        className="px-4 py-2 rounded-lg bg-primary/10 text-primary font-medium text-sm hover:bg-primary/20 transition-colors flex items-center gap-2"
                    >
                        <Copy size={14} />
                        Copy to Clipboard
                    </button>
                    <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-700 text-sm">
                        <p className="font-medium flex items-center gap-2">
                            <Info size={16} />
                            Important: Save this token now
                        </p>
                        <p className="mt-1">
                            For security reasons, the token will not be shown again. Store it in a secure location.
                        </p>
                    </div>
                    <button
                        onClick={() => {
                            setShowToken(false);
                            setGeneratedToken(null);
                        }}
                        className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold text-sm hover:bg-primary/90 transition-colors"
                    >
                        Done
                    </button>
                </div>
            ) : (
                <div className="p-6 rounded-2xl border bg-card">
                    <button
                        onClick={handleGenerate}
                        disabled={isGenerating}
                        className="w-full px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold text-sm hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                    >
                        {isGenerating && <Loader2 size={16} className="animate-spin" />}
                        Generate API Token
                    </button>
                </div>
            )}

            <div className="p-4 rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-600 text-sm space-y-2">
                <p className="font-medium flex items-center gap-2">
                    <Shield size={16} />
                    How to use your API token
                </p>
                <p>
                    Include the token in the Authorization header of your API requests:
                </p>
                <pre className="bg-muted p-3 rounded-lg text-xs overflow-x-auto mt-2">
                    <code>Authorization: Bearer YOUR_API_TOKEN</code>
                </pre>
            </div>
        </div>
    );
}

// Data Export Tab Component
function DataExportTab() {
    const { data: exportStatus, refetch } = useQuery({
        queryKey: ['user-export', 'status'],
        queryFn: async () => {
            const res = await api.get('/user/export/status');
            return res.data;
        },
    });

    const { mutate: requestExport, isPending: isRequesting } = useMutation({
        mutationFn: async () => {
            const res = await api.post('/user/export/request');
            return res.data;
        },
        onSuccess: () => {
            refetch();
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to request data export');
        },
    });

    return (
        <div className="space-y-6">
            <div className="space-y-2">
                <h2 className="text-2xl font-bold">Download Your Data</h2>
                <p className="text-muted-foreground text-sm">
                    Export all your personal data including submissions, comments, blog posts, and contest participations.
                </p>
            </div>

            <div className="p-6 rounded-2xl border bg-muted/50 space-y-4">
                <div className="flex items-start gap-3">
                    <Info className="w-5 h-5 text-primary mt-0.5" />
                    <div className="space-y-2 text-sm">
                        <p className="font-medium">What&apos;s included in your export:</p>
                        <ul className="list-disc list-inside space-y-1 text-muted-foreground ml-2">
                            <li>Profile information and preferences</li>
                            <li>All submissions with source code</li>
                            <li>Comments and blog posts</li>
                            <li>Support tickets</li>
                            <li>Contest participations and ratings</li>
                            <li>Organization memberships</li>
                        </ul>
                    </div>
                </div>
            </div>

            <div className="p-6 rounded-2xl border bg-card">
                <h3 className="text-lg font-bold mb-4">Export Status</h3>
                
                {exportStatus ? (
                    <div className="space-y-4">
                        <div className="flex items-center justify-between">
                            <span className="text-sm text-muted-foreground">Last export:</span>
                            <span className="text-sm font-medium">
                                {exportStatus.last_export 
                                    ? new Date(exportStatus.last_export).toLocaleDateString()
                                    : 'Never'}
                            </span>
                        </div>
                        
                        <div className="flex items-center justify-between">
                            <span className="text-sm text-muted-foreground">Can request export:</span>
                            <Badge variant={exportStatus.can_request ? 'success' : 'secondary'}>
                                {exportStatus.can_request ? 'Yes' : 'No'}
                            </Badge>
                        </div>

                        {!exportStatus.can_request && (
                            <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-700 text-sm">
                                <p className="font-medium flex items-center gap-2">
                                    <Info size={16} />
                                    Rate limit active
                                </p>
                                <p className="mt-1">
                                    You can request a new data export in {exportStatus.days_until_request} days.
                                    This helps us manage server resources.
                                </p>
                            </div>
                        )}

                        {exportStatus.can_request && (
                            <button
                                onClick={() => requestExport()}
                                disabled={isRequesting}
                                className="w-full px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold text-sm hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                            >
                                {isRequesting && <Loader2 size={16} className="animate-spin" />}
                                Request Data Export
                            </button>
                        )}
                    </div>
                ) : (
                    <div className="flex items-center justify-center py-8">
                        <Loader2 className="animate-spin" size={24} />
                    </div>
                )}
            </div>

            <div className="p-4 rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-600 text-sm space-y-2">
                <p className="font-medium flex items-center gap-2">
                    <Shield size={16} />
                    Export details
                </p>
                <ul className="list-disc list-inside space-y-1 ml-2">
                    <li>Exports are prepared asynchronously (5-10 minutes)</li>
                    <li>Download links expire after 24 hours</li>
                    <li>You can request a new export once per week</li>
                    <li>Data is provided in JSON format within a ZIP archive</li>
                </ul>
            </div>
        </div>
    );
}
