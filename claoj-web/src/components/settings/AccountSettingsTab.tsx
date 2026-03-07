'use client';

import React from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Loader2, CheckCircle, Lock, Key, Smartphone, Copy, Download, Info, Mail, AlertTriangle } from 'lucide-react';
import api from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Badge } from '@/components/ui/Badge';
import { useAuth } from '@/components/providers/AuthProvider';

const passwordSchema = z.object({
    current_password: z.string().min(1, 'Current password is required'),
    new_password: z.string().min(6, 'New password must be at least 6 characters'),
    confirm_password: z.string().min(6, 'Please confirm your new password'),
}).refine((data) => data.new_password === data.confirm_password, {
    message: "Passwords don't match",
    path: ["confirm_password"],
});

type PasswordFormValues = z.infer<typeof passwordSchema>;

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

export default function AccountSettingsTab() {
    const { user } = useAuth();
    const [passwordSuccess, setPasswordSuccess] = useState(false);

    // 2FA states
    const [twoFactorStep, setTwoFactorStep] = useState<'disabled' | 'setup' | 'confirm' | 'enabled'>('disabled');
    const [twoFactorSecret, setTwoFactorSecret] = useState<TwoFactorSetup | null>(null);
    const [twoFactorCode, setTwoFactorCode] = useState('');
    const [twoFactorPassword, setTwoFactorPassword] = useState('');
    const [backupCodes, setBackupCodes] = useState<string[] | null>(null);
    const [disablePassword, setDisablePassword] = useState('');
    const [disableCode, setDisableCode] = useState('');

    const { data: twoFactorStatus, refetch: refetchTwoFactor } = useQuery<{
        enabled: boolean;
        backup_codes_remaining: number;
    }>({
        queryKey: ['totp', 'status'],
        queryFn: async () => {
            const res = await api.get<TwoFactorStatus>('/auth/totp/status');
            return res.data;
        },
    });

    // Email verification query
    const { data: userProfile, refetch: refetchUserProfile } = useQuery({
        queryKey: ['user', 'me'],
        queryFn: async () => {
            const res = await api.get('/user/me');
            return res.data;
        },
    });

    const isEmailVerified = userProfile?.is_active ?? true;

    // Resend verification mutation
    const { mutate: resendVerification, isPending: isResendingVerification } = useMutation({
        mutationFn: async () => {
            const res = await api.post('/auth/resend-verification', {});
            return res.data;
        },
        onSuccess: (data) => {
            alert(data.message || 'Verification email sent successfully');
        },
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to send verification email');
        }
    });

    useEffect(() => {
        if (twoFactorStatus?.enabled) {
            setTwoFactorStep('enabled');
        } else {
            setTwoFactorStep('disabled');
        }
    }, [twoFactorStatus]);

    const {
        register: registerPassword,
        handleSubmit: handlePasswordSubmit,
        reset: resetPassword,
        formState: { errors: passwordErrors },
    } = useForm<PasswordFormValues>({
        resolver: zodResolver(passwordSchema),
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
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to change password');
        }
    });

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
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to setup 2FA');
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
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Invalid code');
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
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to disable 2FA');
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
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to regenerate backup codes');
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

    return (
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

            {/* Email Verification Section */}
            <section className="space-y-6">
                <div className="flex items-center gap-2 text-primary font-bold">
                    <Mail size={18} />
                    Email Verification
                </div>

                {isEmailVerified ? (
                    <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-between">
                        <p className="text-sm font-medium text-emerald-700 flex items-center gap-2">
                            <CheckCircle size={16} />
                            Your email is verified
                        </p>
                        <Badge variant="default" className="bg-emerald-500 text-white">
                            Verified
                        </Badge>
                    </div>
                ) : (
                    <div className="space-y-4">
                        <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 flex items-start gap-3">
                            <AlertTriangle size={18} className="text-amber-600 mt-0.5 flex-shrink-0" />
                            <div className="space-y-1">
                                <p className="text-sm font-medium text-amber-700">
                                    Email not verified
                                </p>
                                <p className="text-sm text-amber-600">
                                    Please verify your email address to access all features. Check your inbox for the verification link, or resend the email below.
                                </p>
                            </div>
                        </div>
                        <button
                            onClick={() => resendVerification()}
                            disabled={isResendingVerification}
                            className="px-6 h-12 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-all disabled:opacity-50 flex items-center gap-2"
                        >
                            {isResendingVerification && <Loader2 size={18} className="animate-spin" />}
                            Resend Verification Email
                        </button>
                    </div>
                )}
            </section>

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
    );
}
