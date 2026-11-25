'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { Copy } from 'lucide-react';
import { adminProblemApi } from '@/lib/adminApi';
import { Dialog } from '@/components/ui/Dialog';
import { Button } from '@/components/ui/Button';
import { useRouter } from '@/navigation';

interface CloneProblemModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    problemCode: string;
    problemName: string;
    onSuccess?: (problem: { id: number; code: string; name: string }) => void;
}

/** v1 CloneProblem: a copy of the problem under a new code. */
export function CloneProblemModal({ open, onOpenChange, problemCode, problemName, onSuccess }: CloneProblemModalProps) {
    const t = useTranslations('Admin');
    const [newCode, setNewCode] = useState('');
    const [newName, setNewName] = useState('');
    const [copyData, setCopyData] = useState(false);
    const [copyAuthors, setCopyAuthors] = useState(true);
    const [copySettings, setCopySettings] = useState(true);

    const cloneMutation = useMutation({
        mutationFn: () => adminProblemApi.clone(problemCode, {
            new_code: newCode,
            new_name: newName,
            copy_data: copyData,
            copy_authors: copyAuthors,
            copy_settings: copySettings,
        }),
        onSuccess: (data) => {
            onSuccess?.(data.data.new_problem);
            onOpenChange(false);
            setNewCode('');
            setNewName('');
            alert(t('problems.cloneSuccess', { name: data.data.new_problem.name, code: data.data.new_problem.code }));
        },
        onError: (err: unknown) => {
            const message = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
            alert(message || t('problems.cloneError'));
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
                            <h2 className="text-xl font-bold">{t('problems.cloneModalTitle')}</h2>
                            <p className="text-sm text-muted-foreground mt-1">
                                {t('problems.cloneModalSubtitle', { name: problemName })}
                            </p>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium mb-2">{t('problems.newCodeLabel')}</label>
                                <input
                                    type="text"
                                    value={newCode}
                                    onChange={(e) => setNewCode(e.target.value)}
                                    placeholder={t('problems.newCodePlaceholder')}
                                    className="w-full px-4 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    required
                                />
                            </div>

                            <div>
                                <label className="block text-sm font-medium mb-2">{t('problems.newNameLabel')}</label>
                                <input
                                    type="text"
                                    value={newName}
                                    onChange={(e) => setNewName(e.target.value)}
                                    placeholder={t('problems.newNamePlaceholder')}
                                    className="w-full px-4 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    required
                                />
                            </div>

                            <div className="space-y-2">
                                <label className="flex items-center gap-2 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={copyAuthors}
                                        onChange={(e) => setCopyAuthors(e.target.checked)}
                                        className="rounded"
                                    />
                                    <span className="text-sm">{t('problems.copyAuthorsLabel')}</span>
                                </label>

                                <label className="flex items-center gap-2 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={copySettings}
                                        onChange={(e) => setCopySettings(e.target.checked)}
                                        className="rounded"
                                    />
                                    <span className="text-sm">{t('problems.copySettingsLabel')}</span>
                                </label>

                                <label className="flex items-center gap-2 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={copyData}
                                        onChange={(e) => setCopyData(e.target.checked)}
                                        className="rounded"
                                    />
                                    <span className="text-sm">{t('problems.copyDataLabel')}</span>
                                </label>
                            </div>
                        </div>

                        <div className="flex gap-3 pt-4">
                            <button
                                type="button"
                                onClick={() => onOpenChange(false)}
                                className="flex-1 px-4 py-2 rounded-lg bg-muted hover:bg-muted/80 transition-colors font-medium"
                            >
                                {t('common.cancel')}
                            </button>
                            <button
                                type="submit"
                                disabled={cloneMutation.isPending}
                                className="flex-1 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors font-medium disabled:opacity-50"
                            >
                                {cloneMutation.isPending ? t('common.cloning') : t('problems.cloneModalTitle')}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </Dialog>
    );
}

/** A toolbar button that opens the clone dialog and lands on the new copy. */
export function CloneProblemButton({ code, name }: { code: string; name: string }) {
    const t = useTranslations('Admin');
    const router = useRouter();
    const [open, setOpen] = useState(false);

    return (
        <>
            <Button variant="outline" onClick={() => setOpen(true)}>
                <Copy size={16} />
                {t('problems.cloneModalTitle')}
            </Button>
            <CloneProblemModal
                open={open}
                onOpenChange={setOpen}
                problemCode={code}
                problemName={name}
                onSuccess={(problem) => router.push(`/admin/problems/${problem.code}/edit`)}
            />
        </>
    );
}