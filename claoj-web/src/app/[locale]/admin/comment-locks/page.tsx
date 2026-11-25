'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminCommentLockApi } from '@/lib/adminApi';
import { apiErrorMessage } from '@/lib/apiError';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import { PaginationBar } from '@/components/ui/PaginationBar';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/Table';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/Dialog';
import { Edit, Lock, Plus, Search, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

const FILTER_KEYS = ['search'] as const;
const PAGE_SIZE = 20;

export default function AdminCommentLocksPage() {
    const t = useTranslations('Admin');
    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);

    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const [editing, setEditing] = useState<{ id: number; page: string } | null>(null);
    const [deletingId, setDeletingId] = useState<number | null>(null);
    const [formPage, setFormPage] = useState('');

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-comment-locks', page, search],
        queryFn: async () => (await adminCommentLockApi.list(page, PAGE_SIZE, search)).data,
    });

    const locks = data?.data ?? [];

    const openCreate = () => {
        setFormPage('');
        setIsCreateOpen(true);
    };

    const handleCreate = async () => {
        try {
            await adminCommentLockApi.create(formPage.trim());
            toast.success(t('commentLocks.createSuccess'));
            setIsCreateOpen(false);
            refetch();
        } catch (error) {
            toast.error(apiErrorMessage(error, t('commentLocks.createError')));
        }
    };

    const handleUpdate = async () => {
        if (!editing) return;
        try {
            await adminCommentLockApi.update(editing.id, formPage.trim());
            toast.success(t('commentLocks.updateSuccess'));
            setEditing(null);
            refetch();
        } catch (error) {
            toast.error(apiErrorMessage(error, t('commentLocks.updateError')));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminCommentLockApi.delete(id);
            toast.success(t('commentLocks.deleteSuccess'));
            setDeletingId(null);
            refetch();
        } catch (error) {
            toast.error(apiErrorMessage(error, t('commentLocks.deleteError')));
        }
    };

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Lock className="text-primary" size={30} />
                        {t('commentLocks.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('commentLocks.subtitle')}</p>
                </div>
                <Button onClick={openCreate}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('commentLocks.addButton')}
                </Button>
            </div>

            <div className="relative w-full sm:w-80">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    value={filters.search ?? ''}
                    onChange={(e) => setFilters({ search: e.target.value })}
                    placeholder={t('commentLocks.searchPlaceholder')}
                    aria-label={t('commentLocks.searchPlaceholder')}
                    className="pl-9"
                />
            </div>

            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t('commentLocks.colPage')}</TableHead>
                            <TableHead className="text-right">{t('common.actions')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={2} className="text-center py-8">{t('common.loading')}</TableCell>
                            </TableRow>
                        ) : locks.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={2} className="text-center py-8 text-muted-foreground">
                                    {t('commentLocks.empty')}
                                </TableCell>
                            </TableRow>
                        ) : locks.map((lock) => (
                            <TableRow key={lock.id}>
                                <TableCell className="font-mono text-sm">{lock.page}</TableCell>
                                <TableCell className="text-right">
                                    <div className="flex justify-end gap-2">
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => {
                                                setFormPage(lock.page);
                                                setEditing(lock);
                                            }}
                                            title={t('common.edit')}
                                            aria-label={t('common.edit')}
                                        >
                                            <Edit className="h-4 w-4" />
                                        </Button>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => setDeletingId(lock.id)}
                                            className="text-destructive hover:text-destructive"
                                            title={t('common.delete')}
                                            aria-label={t('common.delete')}
                                        >
                                            <Trash2 className="h-4 w-4" />
                                        </Button>
                                    </div>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>

            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />

            <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('commentLocks.createTitle')}</DialogTitle>
                        <DialogDescription>{t('commentLocks.createDesc')}</DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-2 py-4">
                        <Label htmlFor="lock-page">{t('commentLocks.pageLabel')}</Label>
                        <Input
                            id="lock-page"
                            value={formPage}
                            onChange={(e) => setFormPage(e.target.value)}
                            placeholder={t('commentLocks.pagePlaceholder')}
                            className="font-mono"
                        />
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={handleCreate}>{t('common.create')}</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('commentLocks.editTitle')}</DialogTitle>
                        <DialogDescription>{t('commentLocks.editDesc')}</DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-2 py-4">
                        <Label htmlFor="lock-page-edit">{t('commentLocks.pageLabel')}</Label>
                        <Input
                            id="lock-page-edit"
                            value={formPage}
                            onChange={(e) => setFormPage(e.target.value)}
                            placeholder={t('commentLocks.pagePlaceholder')}
                            className="font-mono"
                        />
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setEditing(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={handleUpdate}>{t('common.saveChanges')}</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={deletingId !== null} onOpenChange={() => setDeletingId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('commentLocks.deleteTitle')}</DialogTitle>
                        <DialogDescription>{t('commentLocks.deleteDesc')}</DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeletingId(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button variant="destructive" onClick={() => deletingId && handleDelete(deletingId)}>
                            {t('common.delete')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}