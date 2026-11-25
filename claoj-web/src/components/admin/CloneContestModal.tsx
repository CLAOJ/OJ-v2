'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { Copy } from 'lucide-react';
import { adminContestApi } from '@/lib/adminApi';
import { Dialog } from '@/components/ui/Dialog';
import { Button } from '@/components/ui/Button';
import { useRouter } from '@/navigation';

interface CloneContestModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    contestKey: string;
    contestName: string;
    onSuccess?: (contest: { key: string; name: string }) => void;
}

/** v1 CloneContest: a copy of the contest under a new key. */
export function CloneContestModal({ open, onOpenChange, contestKey, contestName, onSuccess }: CloneContestModalProps) {
    const t = useTranslations('Admin');
    const [newKey, setNewKey] = useState('');
    const [newName, setNewName] = useState('');
    const [copyProblems, setCopyProblems] = useState(true);
    const [copySettings, setCopySettings] = useState(true);

    const cloneMutation = useMutation({
        mutationFn: () => adminContestApi.clone(contestKey, {
            new_key: newKey,
            new_name: newName,
            copy_problems: copyProblems,
            copy_settings: copySettings,
        }),
        onSuccess: (data) => {
            onSuccess?.(data.data.new_contest);
            onOpenChange(false);
            setNewKey('');
            setNewName('');
            alert(t('contests.cloneSuccess', { name: data.data.new_contest.name, key: data.data.new_contest.key }));
        },
        onError: (err: unknown) => {
            const message = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
            alert(message || t('contests.cloneError'));
        },
    });

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        cloneMutation.mutate();
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4" />
            <div className="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none">
                <div className="bg-card rounded-md border shadow-md w-full max-w-md pointer-events-auto">
                    <form onSubmit={handleSubmit} className="space-y-6 p-6">
                        <div>
                            <h2 className="text-xl font-bold">{t('contests.cloneModalTitle')}</h2>
                            <p className="text-sm text-muted-foreground mt-1">
                                {t('contests.cloneModalSubtitle', { name: contestName })}
                            </p>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium mb-2">{t('contests.newKeyLabel')}</label>
                                <input
                                    type="text"
                                    value={newKey}
                                    onChange={(e) => setNewKey(e.target.value)}
                                    placeholder={t('contests.newKeyPlaceholder')}
                                    className="w-full px-4 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    required
                                />
                            </div>

                            <div>
                                <label className="block text-sm font-medium mb-2">{t('contests.newNameLabel')}</label>
                                <input
                                    type="text"
                                    value={newName}
                                    onChange={(e) => setNewName(e.target.value)}
                                    placeholder={t('contests.newNamePlaceholder')}
                                    className="w-full px-4 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    required
                                />
                            </div>

                            <div className="space-y-2">
                                <label className="flex items-center gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30">
                                    <input
                                        type="checkbox"
                                        checked={copyProblems}
                                        onChange={(e) => setCopyProblems(e.target.checked)}
                                        className="w-4 h-4"
                                    />
                                    <div>
                                        <div className="text-sm font-medium">{t('contests.copyProblemsLabel')}</div>
                                        <div className="text-xs text-muted-foreground">{t('contests.copyProblemsDesc')}</div>
                                    </div>
                                </label>

                                <label className="flex items-center gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30">
                                    <input
                                        type="checkbox"
                                        checked={copySettings}
                                        onChange={(e) => setCopySettings(e.target.checked)}
                                        className="w-4 h-4"
                                    />
                                    <div>
                                        <div className="text-sm font-medium">{t('contests.copySettingsLabel')}</div>
                                        <div className="text-xs text-muted-foreground">{t('contests.copySettingsDesc')}</div>
                                    </div>
                                </label>
                            </div>
                        </div>

                        <div className="flex items-center gap-3 pt-4 border-t">
                            <Button
                                type="button"
                                variant="secondary"
                                onClick={() => onOpenChange(false)}
                                className="flex-1"
                            >
                                {t('common.cancel')}
                            </Button>
                            <Button
                                type="submit"
                                variant="default"
                                disabled={cloneMutation.isPending}
                                className="flex-1"
                            >
                                {cloneMutation.isPending ? t('common.cloning') : t('contests.cloneModalTitle')}
                            </Button>
                        </div>
                    </form>
                </div>
            </div>
        </Dialog>
    );
}

/** A toolbar button that opens the clone dialog and lands on the new copy. */
export function CloneContestButton({ contestKey, contestName }: { contestKey: string; contestName: string }) {
    const t = useTranslations('Admin');
    const router = useRouter();
    const [open, setOpen] = useState(false);

    return (
        <>
            <Button variant="outline" onClick={() => setOpen(true)}>
                <Copy size={16} />
                {t('contests.cloneModalTitle')}
            </Button>
            <CloneContestModal
                open={open}
                onOpenChange={setOpen}
                contestKey={contestKey}
                contestName={contestName}
                onSuccess={(contest) => router.push(`/admin/contests/${contest.key}/edit`)}
            />
        </>
    );
}