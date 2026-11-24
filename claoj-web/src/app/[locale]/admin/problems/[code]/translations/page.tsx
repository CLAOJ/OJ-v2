'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { problemTranslationApi } from '@/lib/api';
import { ProblemTranslation } from '@/types';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Textarea } from '@/components/ui/Textarea';
import { Select, SelectOption } from '@/components/ui/Select';
import { Badge } from '@/components/ui/Badge';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger
} from '@/components/ui/Dialog';
import { Languages, Plus, Pencil, Trash2, AlertCircle } from 'lucide-react';

const LANGUAGE_OPTIONS = [
    { value: 'en', labelKey: 'translation_language_en' },
    { value: 'vi', labelKey: 'translation_language_vi' }
] as const;

const emptyForm = { language: 'vi', name: '', description: '' };

export default function ProblemTranslationsPage() {
    const params = useParams();
    const t = useTranslations('admin');
    const tCommon = useTranslations('Common');
    const code = params.code as string;
    const queryClient = useQueryClient();

    const [isFormOpen, setIsFormOpen] = useState(false);
    const [editing, setEditing] = useState<ProblemTranslation | null>(null);
    const [form, setForm] = useState(emptyForm);
    const [deleteTarget, setDeleteTarget] = useState<ProblemTranslation | null>(null);

    const { data: translations, isLoading } = useQuery({
        queryKey: ['problem-translations', code],
        queryFn: async () => (await problemTranslationApi.list(code)).data.data
    });

    const invalidate = () => queryClient.invalidateQueries({ queryKey: ['problem-translations', code] });
    const toastError = (err: unknown) =>
        toast.error((err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? String(err));

    const { mutate: save, isPending } = useMutation({
        mutationFn: () =>
            editing
                ? problemTranslationApi.update(code, editing.id, form)
                : problemTranslationApi.create(code, form),
        onSuccess: () => {
            invalidate();
            setIsFormOpen(false);
            setEditing(null);
            setForm(emptyForm);
            toast.success(t('translation_saved'));
        },
        onError: toastError
    });

    const { mutate: remove } = useMutation({
        mutationFn: (id: number) => problemTranslationApi.delete(code, id),
        onSuccess: () => {
            invalidate();
            setDeleteTarget(null);
            toast.success(t('translation_deleted'));
        },
        onError: toastError
    });

    const openCreate = () => {
        setEditing(null);
        setForm(emptyForm);
        setIsFormOpen(true);
    };

    const openEdit = (translation: ProblemTranslation) => {
        setEditing(translation);
        setForm({
            language: translation.language,
            name: translation.name,
            description: translation.description
        });
        setIsFormOpen(true);
    };

    const handleSubmit = () => {
        if (!form.name.trim() || !form.description.trim()) {
            toast.error(t('translation_fields_required'));
            return;
        }
        save();
    };

    return (
        <div className="container mx-auto py-8">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-3xl font-bold mb-2">{t('problem_translations')}</h1>
                    <p className="text-muted-foreground">{t('manage_problem_translations', { code })}</p>
                </div>
                <Dialog open={isFormOpen} onOpenChange={setIsFormOpen}>
                    <DialogTrigger asChild>
                        <Button onClick={openCreate}>
                            <Plus className="w-4 h-4 mr-2" />
                            {t('add_translation')}
                        </Button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>{editing ? t('edit_translation') : t('add_translation')}</DialogTitle>
                            <DialogDescription>{t('add_problem_translation_desc', { code })}</DialogDescription>
                        </DialogHeader>
                        <div className="py-4 space-y-4">
                            <div className="space-y-2">
                                <Label>{t('translation_language')}</Label>
                                <Select
                                    value={form.language}
                                    onChange={(e) => setForm({ ...form, language: e.target.value })}
                                >
                                    {LANGUAGE_OPTIONS.map((opt) => (
                                        <SelectOption key={opt.value} value={opt.value}>
                                            {t(opt.labelKey)}
                                        </SelectOption>
                                    ))}
                                </Select>
                            </div>
                            <div className="space-y-2">
                                <Label>{t('translation_name')}</Label>
                                <Input
                                    value={form.name}
                                    maxLength={100}
                                    placeholder={t('translation_name_placeholder')}
                                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                                />
                            </div>
                            <div className="space-y-2">
                                <Label>{t('translation_description')}</Label>
                                <Textarea
                                    value={form.description}
                                    placeholder={t('translation_description_placeholder')}
                                    onChange={(e) => setForm({ ...form, description: e.target.value })}
                                    className="min-h-[150px]"
                                />
                            </div>
                        </div>
                        <DialogFooter>
                            <Button variant="outline" onClick={() => setIsFormOpen(false)}>
                                {tCommon('cancel')}
                            </Button>
                            <Button onClick={handleSubmit} disabled={isPending}>
                                {tCommon('save')}
                            </Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map((i) => (
                        <Card key={i} className="animate-pulse">
                            <CardHeader className="pb-3">
                                <div className="h-4 bg-muted rounded w-3/4" />
                            </CardHeader>
                            <CardContent>
                                <div className="h-3 bg-muted rounded w-1/4" />
                            </CardContent>
                        </Card>
                    ))}
                </div>
            ) : !translations || translations.length === 0 ? (
                <Card>
                    <CardContent className="flex items-center gap-4 py-8">
                        <AlertCircle className="w-12 h-12 text-muted-foreground" />
                        <div>
                            <p className="font-medium">{t('no_translations')}</p>
                            <p className="text-sm text-muted-foreground">{t('be_first_translation')}</p>
                        </div>
                    </CardContent>
                </Card>
            ) : (
                <div className="space-y-4">
                    {translations.map((translation) => (
                        <Card key={translation.id}>
                            <CardHeader className="pb-3">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-2">
                                        <Languages className="w-5 h-5 text-primary" />
                                        <CardTitle className="text-lg">{translation.name}</CardTitle>
                                        <Badge variant="secondary">{translation.language}</Badge>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <Button variant="ghost" size="sm" onClick={() => openEdit(translation)}>
                                            <Pencil className="w-4 h-4" />
                                        </Button>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => setDeleteTarget(translation)}
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </Button>
                                    </div>
                                </div>
                            </CardHeader>
                            <CardContent>
                                <CardDescription className="whitespace-pre-wrap line-clamp-4">
                                    {translation.description}
                                </CardDescription>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}

            <ConfirmDialog
                isOpen={deleteTarget !== null}
                onClose={() => setDeleteTarget(null)}
                onConfirm={() => deleteTarget && remove(deleteTarget.id)}
                title={t('delete_translation')}
                description={t('delete_translation_desc', { name: deleteTarget?.name ?? '' })}
                confirmText={tCommon('delete')}
                cancelText={tCommon('cancel')}
                variant="danger"
            />
        </div>
    );
}