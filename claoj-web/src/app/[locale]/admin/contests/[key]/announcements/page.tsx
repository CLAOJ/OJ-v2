'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { announcementApi } from '@/lib/adminApi';
import { ContestAnnouncement } from '@/types';
import { formatDateTime } from '@/lib/date';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Textarea } from '@/components/ui/Textarea';
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
import { Megaphone, Plus, Pencil, Trash2, AlertCircle } from 'lucide-react';

const emptyForm = { title: '', description: '' };

export default function ContestAnnouncementsPage() {
    const params = useParams();
    const t = useTranslations('admin');
    const tCommon = useTranslations('Common');
    const key = params.key as string;
    const queryClient = useQueryClient();

    const [isFormOpen, setIsFormOpen] = useState(false);
    const [editing, setEditing] = useState<ContestAnnouncement | null>(null);
    const [form, setForm] = useState(emptyForm);
    const [deleteTarget, setDeleteTarget] = useState<ContestAnnouncement | null>(null);

    const { data: announcements, isLoading } = useQuery({
        queryKey: ['contest-announcements', key],
        queryFn: async () => (await announcementApi.list(key)).data.data
    });

    const invalidate = () => queryClient.invalidateQueries({ queryKey: ['contest-announcements', key] });
    const toastError = (err: unknown) =>
        toast.error((err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? String(err));

    const { mutate: save, isPending } = useMutation({
        mutationFn: () =>
            editing
                ? announcementApi.update(key, editing.id, form)
                : announcementApi.create(key, form),
        onSuccess: () => {
            invalidate();
            setIsFormOpen(false);
            setEditing(null);
            setForm(emptyForm);
            toast.success(editing ? t('announcement_updated') : t('announcement_created'));
        },
        onError: toastError
    });

    const { mutate: remove } = useMutation({
        mutationFn: (id: number) => announcementApi.delete(key, id),
        onSuccess: () => {
            invalidate();
            setDeleteTarget(null);
            toast.success(t('announcement_deleted'));
        },
        onError: toastError
    });

    const openCreate = () => {
        setEditing(null);
        setForm(emptyForm);
        setIsFormOpen(true);
    };

    const openEdit = (announcement: ContestAnnouncement) => {
        setEditing(announcement);
        setForm({ title: announcement.title, description: announcement.description });
        setIsFormOpen(true);
    };

    const handleSubmit = () => {
        if (!form.title.trim()) {
            toast.error(t('announcement_title_required'));
            return;
        }
        if (!form.description.trim()) {
            toast.error(t('announcement_description_required'));
            return;
        }
        save();
    };

    return (
        <div className="container mx-auto py-8">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-3xl font-bold mb-2">{t('contest_announcements')}</h1>
                    <p className="text-muted-foreground">{t('manage_contest_announcements', { key })}</p>
                </div>
                <Dialog open={isFormOpen} onOpenChange={setIsFormOpen}>
                    <DialogTrigger asChild>
                        <Button onClick={openCreate}>
                            <Plus className="w-4 h-4 mr-2" />
                            {t('new_announcement')}
                        </Button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>{editing ? t('edit_announcement') : t('new_announcement')}</DialogTitle>
                            <DialogDescription>{t('manage_contest_announcements', { key })}</DialogDescription>
                        </DialogHeader>
                        <div className="py-4 space-y-4">
                            <div className="space-y-2">
                                <Label>{t('announcement_title')}</Label>
                                <Input
                                    value={form.title}
                                    maxLength={100}
                                    onChange={(e) => setForm({ ...form, title: e.target.value })}
                                />
                            </div>
                            <div className="space-y-2">
                                <Label>{t('announcement_description')}</Label>
                                <Textarea
                                    value={form.description}
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
            ) : !announcements || announcements.length === 0 ? (
                <Card>
                    <CardContent className="flex items-center gap-4 py-8">
                        <AlertCircle className="w-12 h-12 text-muted-foreground" />
                        <div>
                            <p className="font-medium">{t('no_announcements')}</p>
                            <p className="text-sm text-muted-foreground">{t('be_first_announcement')}</p>
                        </div>
                    </CardContent>
                </Card>
            ) : (
                <div className="space-y-4">
                    {announcements.map((announcement) => (
                        <Card key={announcement.id}>
                            <CardHeader className="pb-3">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-2">
                                        <Megaphone className="w-5 h-5 text-primary" />
                                        <CardTitle className="text-lg">{announcement.title}</CardTitle>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-muted-foreground">
                                            {formatDateTime(announcement.date)}
                                        </span>
                                        <Button variant="ghost" size="sm" onClick={() => openEdit(announcement)}>
                                            <Pencil className="w-4 h-4" />
                                        </Button>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => setDeleteTarget(announcement)}
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </Button>
                                    </div>
                                </div>
                            </CardHeader>
                            <CardContent>
                                <CardDescription className="whitespace-pre-wrap">
                                    {announcement.description}
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
                title={t('delete_announcement')}
                description={t('delete_announcement_desc', { title: deleteTarget?.title ?? '' })}
                confirmText={tCommon('delete')}
                cancelText={tCommon('cancel')}
                variant="danger"
            />
        </div>
    );
}