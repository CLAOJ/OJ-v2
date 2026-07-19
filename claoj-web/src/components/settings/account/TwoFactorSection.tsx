'use client';

import React, { useState, useEffect } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { motion } from 'framer-motion';
import { Loader2, CheckCircle, Smartphone, Lock, Key, Copy, Download } from 'lucide-react';
import api from '@/lib/api';
import { Badge } from '@/components/ui/Badge';

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

interface TwoFactorSectionProps {
    onStepChange?: (step: string) => void;
}

export function TwoFactorSection({ onStepChange }: TwoFactorSectionProps) {
    const t = useTranslations('Settings');
    const tc = useTranslations('Common');
    const [twoFactorStep, setTwoFactorStep] = useState<'disabled' | 'setup' | 'confirm' | 'enabled'>('disabled');
    const [twoFactorSecret, setTwoFactorSecret] = useState<TwoFactorSetup | null>(null);
    const [twoFactorCode, setTwoFactorCode] = useState('');
    const [twoFactorPassword, setTwoFactorPassword] = useState('');
    const [backupCodes, setBackupCodes] = useState<string[] | null>(null);
    const [disablePassword, setDisablePassword] = useState('');
    const [disableCode, setDisableCode] = useState('');

    const { data: twoFactorStatus, refetch: refetchTwoFactor } = useQuery<TwoFactorStatus>({
        queryKey: ['totp', 'status'],
        queryFn: async () => {
            const res = await api.get<TwoFactorStatus>('/auth/totp/status');
            return res.data;
        },
    });

    // Sync twoFactorStep with twoFactorStatus.enabled
    useEffect(() => {
        if (twoFactorStatus?.enabled) {
            setTwoFactorStep('enabled');
        } else if (twoFactorStatus !== undefined) {
            setTwoFactorStep('disabled');
        }
    }, [twoFactorStatus]);

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
            alert(error.response?.data?.error || t('setupFailedError'));
        }
    });

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
            alert(error.response?.data?.error || t('invalidCodeError'));
        }
    });

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
            alert(error.response?.data?.error || t('disableFailedError'));
        }
    });

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
            alert(error.response?.data?.error || t('regenerateBackupCodesError'));
        }
    });

    const copyToClipboard = async (text: string) => {
        try {
            await navigator.clipboard.writeText(text);
            alert(t('copiedToClipboard'));
        } catch {
            alert(t('copyFailed'));
        }
    };

    const downloadBackupCodes = () => {
        if (!backupCodes) return;
        const content = `${t('backupCodesFileHeader')}\n==================\n\n${t('saveBackupCodesWarning')}\n\n${backupCodes.join('\n')}\n\n${t('backupCodesFileGeneratedOn', { date: new Date().toLocaleString() })}`;
        const blob = new Blob([content], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'claoj-backup-codes.txt';
        a.click();
        URL.revokeObjectURL(url);
    };

    return (
        <section className="space-y-6">
            <div className="flex items-center gap-2 text-primary font-bold">
                <Smartphone size={18} />
                {t('twoFactor')}
            </div>

            {twoFactorStep === 'disabled' && (
                <div className="space-y-4">
                    <p className="text-sm text-muted-foreground">
                        {t('twoFactorDesc')}
                    </p>
                    <div className="space-y-2">
                        <label className="text-sm font-medium">{t('passwordLabel')}</label>
                        <div className="relative">
                            <Lock className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                            <input
                                type="password"
                                value={twoFactorPassword}
                                onChange={(e) => setTwoFactorPassword(e.target.value)}
                                placeholder={t('enterPasswordContinue')}
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
                        {t('setup2FA')}
                    </button>
                </div>
            )}

            {twoFactorStep === 'setup' && twoFactorSecret && (
                <div className="space-y-4">
                    <div className="p-4 rounded-xl bg-muted/50 border text-center">
                        <p className="text-sm font-medium mb-4">{t('scanQrCode')}</p>
                        <div className="w-48 h-48 mx-auto bg-white rounded-xl overflow-hidden flex items-center justify-center">
                            <img src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(twoFactorSecret.url)}`} alt={t('qrCodeAlt')} className="w-full h-full object-contain" />
                        </div>
                        <p className="text-xs text-muted-foreground mt-4 font-mono">
                            {t('secretLabel')} <span className="select-all">{twoFactorSecret.secret}</span>
                        </p>
                        <button
                            onClick={() => copyToClipboard(twoFactorSecret.secret)}
                            className="mt-2 text-xs text-primary hover:underline flex items-center gap-1"
                        >
                            <Copy size={12} /> {t('copySecret')}
                        </button>
                    </div>
                    <div className="space-y-2">
                        <label className="text-sm font-medium">{t('verificationCodeLabel')}</label>
                        <input
                            type="text"
                            value={twoFactorCode}
                            onChange={(e) => setTwoFactorCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                            placeholder={t('enterCodeFromApp')}
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
                            {tc('confirm')}
                        </button>
                        <button
                            onClick={() => { setTwoFactorStep('disabled'); setTwoFactorPassword(''); }}
                            className="px-6 h-12 rounded-xl border font-bold hover:bg-muted transition-all"
                        >
                            {tc('cancel')}
                        </button>
                    </div>
                </div>
            )}

            {twoFactorStep === 'confirm' && backupCodes && (
                <div className="space-y-4">
                    <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20">
                        <p className="text-sm font-bold text-emerald-600 flex items-center gap-2">
                            <CheckCircle size={16} />
                            {t('twoFactorEnabledSuccess')}
                        </p>
                    </div>
                    <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20">
                        <p className="text-sm font-medium text-amber-700 mb-2">
                            <strong>{t('importantLabel')}</strong> {t('saveBackupCodesWarning')}
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
                            <Download size={14} /> {t('downloadBackupCodesButton')}
                        </button>
                    </div>
                </div>
            )}

            {twoFactorStep === 'enabled' && (
                <div className="space-y-4">
                    <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-between">
                        <p className="text-sm font-medium text-emerald-700 flex items-center gap-2">
                            <CheckCircle size={16} />
                            {t('twoFactorEnabledMsg')}
                        </p>
                        <Badge variant="default" className="bg-emerald-500 text-white">
                            {tc('active')}
                        </Badge>
                    </div>
                    <div className="space-y-2">
                        <label className="text-sm font-medium">{t('disable2FA')}</label>
                        <div className="space-y-2">
                            <div className="relative">
                                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                                <input
                                    type="password"
                                    value={disableCode}
                                    onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                                    placeholder={t('enter6DigitCode')}
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
                                    placeholder={t('enterYourPasswordPlaceholder')}
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
                            {t('disable2FA')}
                        </button>
                    </div>
                    {twoFactorStatus?.backup_codes_remaining !== undefined && (
                        <p className="text-xs text-muted-foreground">
                            {t('backupCodesRemaining', { count: twoFactorStatus.backup_codes_remaining })}
                        </p>
                    )}
                </div>
            )}
        </section>
    );
}
