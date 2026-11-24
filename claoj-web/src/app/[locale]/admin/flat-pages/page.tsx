'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { adminFlatPageApi, adminErrorMessage } from '@/lib/adminApi';
import { useDebounce } from '@/hooks/useDebounce';
import {
    AdminFlatPage,
    AdminFlatPageCreateRequest,
    AdminFlatPageDetail,
} from '@/types';
import { Badge } from '@/components/ui/Badge';
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
import { Switch } from '@/components/ui/Switch';
import { Textarea } from '@/components/ui/Textarea';
import { MarkdownImageButton } from '@/components/admin/MarkdownImageButton';
import { useAuth } from '@/components/providers/AuthProvider';
import { hasPerm } from '@/lib/permissions';
import { toast } from 'sonner';
import { Plus, Edit, Trash2, Search } from 'lucide-react';

const emptyForm: AdminFlatPageCreateRequest = {
    url: '',
    title: '',
    content: '',
    registration_required: false,
};

export default function FlatPagesAdminPage() {
    const t = useTranslations('Admin');
    const { user } = useAuth();
    // Mirrors the API's flatpages.change_flatpage gate on create/update/delete.
    const canWrite = hasPerm(user, 'flatpages.change_flatpage');
    const [pages, setPages] = useState<AdminFlatPage[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(20);
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');
    const debouncedSearch = useDebounce(search.trim(), 300);

    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [editingPage, setEditingPage] = useState<AdminFlatPageDetail | null>(null);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);
    const [formData, setFormData] = useState<AdminFlatPageCreateRequest>(emptyForm);

    const loadPages = async () => {
        setLoading(true);
        try {
            const response = await adminFlatPageApi.list(page, pageSize, debouncedSearch || undefined);
            setPages(response.data.data);
            setTotal(response.data.total);
        } catch {
            toast.error(t('flatPages.loadFailed'));
        } finally {
            setLoading(false);
        }
    };

    // Reset to page 1 in the same render a filter changes so the fetch below
    // does not race a stale-page request (same pattern as blog posts).
    const [appliedSearch, setAppliedSearch] = useState(debouncedSearch);
    if (appliedSearch !== debouncedSearch) {
        setAppliedSearch(debouncedSearch);
        setPage(1);
    }

    useEffect(() => {
        loadPages();
    }, [page, debouncedSearch]);

    const openCreateDialog = () => {
        setEditingPage(null);
        setFormData(emptyForm);
        setIsCreateDialogOpen(true);
    };

    const openEditDialog = async (flatPage: AdminFlatPage) => {
        try {
            const response = await adminFlatPageApi.detail(flatPage.id);
            setFormData({
                url: response.data.url,
                title: response.data.title,
                content: response.data.content,
                registration_required: response.data.registration_required,
            });
            setEditingPage(response.data);
            setIsCreateDialogOpen(true);
        } catch {
            toast.error(t('flatPages.loadDetailFailed'));
        }
    };

    const handleCreate = async () => {
        try {
            await adminFlatPageApi.create(formData);
            toast.success(t('flatPages.createSuccess'));
            setIsCreateDialogOpen(false);
            loadPages();
        } catch (error) {
            toast.error(adminErrorMessage(error, t('flatPages.createError')));
        }
    };

    const handleUpdate = async () => {
        if (!editingPage) return;
        try {
            await adminFlatPageApi.update(editingPage.id, formData);
            toast.success(t('flatPages.updateSuccess'));
            setIsCreateDialogOpen(false);
            loadPages();
        } catch (error) {
            toast.error(adminErrorMessage(error, t('flatPages.updateError')));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminFlatPageApi.delete(id);
            toast.success(t('flatPages.deleteSuccess'));
            setDeleteConfirmId(null);
            loadPages();
        } catch (error) {
            toast.error(adminErrorMessage(error, t('flatPages.deleteError')));
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">{t('flatPages.title')}</h1>
                    <p className="text-muted-foreground mt-1">{t('flatPages.subtitle')}</p>
                </div>
                <Button onClick={openCreateDialog} disabled={!canWrite}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('flatPages.addButton')}
                </Button>
            </div>
            {!canWrite && (
                <p className="text-xs text-muted-foreground">{t('flatPages.writeHint')}</p>
            )}

            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder={t('flatPages.searchPlaceholder')}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-10"
                />
            </div>

            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t('flatPages.colUrl')}</TableHead>
                            <TableHead>{t('flatPages.colTitle')}</TableHead>
                            <TableHead>{t('flatPages.colRegistration')}</TableHead>
                            <TableHead className="text-right">{t('common.actions')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={4} className="text-center py-8">
                                    {t('common.loading')}
                                </TableCell>
                            </TableRow>
                        ) : pages.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={4} className="text-center py-8 text-muted-foreground">
                                    {t('flatPages.noPagesFound')}
                                </TableCell>
                            </TableRow>
                        ) : (
                            pages.map((flatPage) => (
                                <TableRow key={flatPage.id}>
                                    <TableCell className="font-mono">{flatPage.url}</TableCell>
                                    <TableCell className="font-medium">{flatPage.title}</TableCell>
                                    <TableCell>
                                        {flatPage.registration_required ? (
                                            <Badge variant="default">{t('flatPages.registrationRequired')}</Badge>
                                        ) : (
                                            <span className="text-muted-foreground">-</span>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            {canWrite && (
                                                <>
                                                    <Button variant="ghost" size="sm" onClick={() => openEditDialog(flatPage)}>
                                                        <Edit className="h-4 w-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => setDeleteConfirmId(flatPage.id)}
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
                <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>
                            {editingPage ? t('flatPages.editDialogTitle') : t('flatPages.createDialogTitle')}
                        </DialogTitle>
                        <DialogDescription>
                            {editingPage ? t('flatPages.editDialogDesc') : t('flatPages.createDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="url">{t('flatPages.urlLabel')}</Label>
                                <Input
                                    id="url"
                                    value={formData.url}
                                    onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                                    placeholder={t('flatPages.urlPlaceholder')}
                                    className="font-mono"
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="title">{t('flatPages.titleLabel')}</Label>
                                <Input
                                    id="title"
                                    value={formData.title}
                                    onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                                    placeholder={t('flatPages.titlePlaceholder')}
                                />
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <div className="flex items-center justify-between">
                                <Label htmlFor="content">{t('flatPages.contentLabel')}</Label>
                                <MarkdownImageButton
                                    id="content"
                                    value={formData.content ?? ''}
                                    onChange={(v) => setFormData((prev) => ({ ...prev, content: v }))}
                                />
                            </div>
                            <Textarea
                                id="content"
                                value={formData.content}
                                onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                                rows={14}
                                className="font-mono text-sm"
                                placeholder={t('flatPages.contentPlaceholder')}
                            />
                        </div>

                        <div className="flex items-center gap-2">
                            <Switch
                                id="registration_required"
                                checked={formData.registration_required ?? false}
                                onCheckedChange={(checked) =>
                                    setFormData({ ...formData, registration_required: checked })
                                }
                            />
                            <Label htmlFor="registration_required">{t('flatPages.registrationLabel')}</Label>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsCreateDialogOpen(false)}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={editingPage ? handleUpdate : handleCreate}>
                            {editingPage ? t('common.saveChanges') : t('common.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('flatPages.deleteDialogTitle')}</DialogTitle>
                        <DialogDescription>{t('flatPages.deleteDialogDesc')}</DialogDescription>
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
