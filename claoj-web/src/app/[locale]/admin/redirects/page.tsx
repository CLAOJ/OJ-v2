'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { adminRedirectApi, adminErrorMessage } from '@/lib/adminApi';
import { useDebounce } from '@/hooks/useDebounce';
import { AdminRedirect, AdminRedirectCreateRequest } from '@/types';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
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
import { Label } from '@/components/ui/Label';
import { useAuth } from '@/components/providers/AuthProvider';
import { hasPerm } from '@/lib/permissions';
import { toast } from 'sonner';
import { Plus, Edit, Trash2, Search, ArrowRight } from 'lucide-react';

const emptyForm: AdminRedirectCreateRequest = { old_path: '', new_path: '' };

export default function RedirectsAdminPage() {
    const t = useTranslations('Admin');
    const { user } = useAuth();
    // Mirrors the API's redirects.change_redirect gate on create/update/delete.
    const canWrite = hasPerm(user, 'redirects.change_redirect');
    const [redirects, setRedirects] = useState<AdminRedirect[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(20);
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');
    const debouncedSearch = useDebounce(search.trim(), 300);

    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [editing, setEditing] = useState<AdminRedirect | null>(null);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);
    const [formData, setFormData] = useState<AdminRedirectCreateRequest>(emptyForm);

    const loadRedirects = async () => {
        setLoading(true);
        try {
            const response = await adminRedirectApi.list(page, pageSize, debouncedSearch || undefined);
            setRedirects(response.data.data);
            setTotal(response.data.total);
        } catch {
            toast.error(t('redirects.loadFailed'));
        } finally {
            setLoading(false);
        }
    };

    const [appliedSearch, setAppliedSearch] = useState(debouncedSearch);
    if (appliedSearch !== debouncedSearch) {
        setAppliedSearch(debouncedSearch);
        setPage(1);
    }

    useEffect(() => {
        loadRedirects();
    }, [page, debouncedSearch]);

    const openCreateDialog = () => {
        setEditing(null);
        setFormData(emptyForm);
        setIsCreateDialogOpen(true);
    };

    const openEditDialog = async (redirect: AdminRedirect) => {
        try {
            const response = await adminRedirectApi.detail(redirect.id);
            setFormData({ old_path: response.data.old_path, new_path: response.data.new_path });
            setEditing(response.data);
            setIsCreateDialogOpen(true);
        } catch {
            toast.error(t('redirects.loadDetailFailed'));
        }
    };

    const handleCreate = async () => {
        try {
            await adminRedirectApi.create(formData);
            toast.success(t('redirects.createSuccess'));
            setIsCreateDialogOpen(false);
            loadRedirects();
        } catch (error) {
            toast.error(adminErrorMessage(error, t('redirects.createError')));
        }
    };

    const handleUpdate = async () => {
        if (!editing) return;
        try {
            await adminRedirectApi.update(editing.id, formData);
            toast.success(t('redirects.updateSuccess'));
            setIsCreateDialogOpen(false);
            loadRedirects();
        } catch (error) {
            toast.error(adminErrorMessage(error, t('redirects.updateError')));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminRedirectApi.delete(id);
            toast.success(t('redirects.deleteSuccess'));
            setDeleteConfirmId(null);
            loadRedirects();
        } catch (error) {
            toast.error(adminErrorMessage(error, t('redirects.deleteError')));
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">{t('redirects.title')}</h1>
                    <p className="text-muted-foreground mt-1">{t('redirects.subtitle')}</p>
                </div>
                <Button onClick={openCreateDialog} disabled={!canWrite}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('redirects.addButton')}
                </Button>
            </div>
            {!canWrite && (
                <p className="text-xs text-muted-foreground">{t('redirects.writeHint')}</p>
            )}

            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder={t('redirects.searchPlaceholder')}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-10"
                />
            </div>

            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t('redirects.colOldPath')}</TableHead>
                            <TableHead>{t('redirects.colNewPath')}</TableHead>
                            <TableHead className="text-right">{t('common.actions')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={3} className="text-center py-8">
                                    {t('common.loading')}
                                </TableCell>
                            </TableRow>
                        ) : redirects.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={3} className="text-center py-8 text-muted-foreground">
                                    {t('redirects.noRedirectsFound')}
                                </TableCell>
                            </TableRow>
                        ) : (
                            redirects.map((redirect) => (
                                <TableRow key={redirect.id}>
                                    <TableCell className="font-mono">{redirect.old_path}</TableCell>
                                    <TableCell className="font-mono">
                                        <ArrowRight className="inline h-3 w-3 mr-1 text-muted-foreground" />
                                        {redirect.new_path}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            {canWrite && (
                                                <>
                                                    <Button variant="ghost" size="sm" onClick={() => openEditDialog(redirect)}>
                                                        <Edit className="h-4 w-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => setDeleteConfirmId(redirect.id)}
                                                        className="text-destructive hover:text-destructive"
                                                    >
                                                        <Trash2 className="h-4 w-4" />
                                                    </Button>
                                                </>
                                            )}
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            <div className="flex items-center justify-between">
                <div className="text-sm text-muted-foreground">
                    {t('common.showingRange', {
                        from: (page - 1) * pageSize + 1,
                        to: Math.min(page * pageSize, total),
                        total,
                    })}
                </div>
                <div className="flex gap-2">
                    <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage(page - 1)}>
                        {t('common.previous')}
                    </Button>
                    <Button
                        variant="outline"
                        size="sm"
                        disabled={page >= totalPages}
                        onClick={() => setPage(page + 1)}
                    >
                        {t('common.next')}
                    </Button>
                </div>
            </div>

            <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
                <DialogContent className="max-w-xl">
                    <DialogHeader>
                        <DialogTitle>
                            {editing ? t('redirects.editDialogTitle') : t('redirects.createDialogTitle')}
                        </DialogTitle>
                        <DialogDescription>
                            {editing ? t('redirects.editDialogDesc') : t('redirects.createDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="old_path">{t('redirects.oldPathLabel')}</Label>
                            <Input
                                id="old_path"
                                value={formData.old_path}
                                onChange={(e) => setFormData({ ...formData, old_path: e.target.value })}
                                placeholder="/old-page/"
                                className="font-mono"
                            />
                        </div>
                        <div className="grid gap-2">
                            <Label htmlFor="new_path">{t('redirects.newPathLabel')}</Label>
                            <Input
                                id="new_path"
                                value={formData.new_path}
                                onChange={(e) => setFormData({ ...formData, new_path: e.target.value })}
                                placeholder="/new-page/"
                                className="font-mono"
                            />
                            <p className="text-xs text-muted-foreground">{t('redirects.newPathHint')}</p>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsCreateDialogOpen(false)}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={editing ? handleUpdate : handleCreate}>
                            {editing ? t('common.saveChanges') : t('common.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('redirects.deleteDialogTitle')}</DialogTitle>
                        <DialogDescription>{t('redirects.deleteDialogDesc')}</DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteConfirmId(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}
                        >
                            {t('common.delete')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
