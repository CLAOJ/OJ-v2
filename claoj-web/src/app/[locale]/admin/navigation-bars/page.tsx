'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { adminNavigationBarApi } from '@/lib/adminApi';
import { navigationBarUpdatePayload } from '@/lib/adminForms';
import { AdminNavigationBar, AdminNavigationBarCreateRequest, AdminNavigationBarDetail } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/Dialog';
import { Label } from '@/components/ui/Label';
import { Select } from '@/components/ui/Select';
import { Textarea } from '@/components/ui/Textarea';
import { useAuth } from '@/components/providers/AuthProvider';
import { hasPerm } from '@/lib/permissions';
import { toast } from 'sonner';
import { Plus, Edit, Trash2, Search, Link as LinkIcon, Layers } from 'lucide-react';
import type { ApiError } from '@/lib/api';
import { apiErrorMessage } from '@/lib/api';

export default function NavigationBarsAdminPage() {
    const t = useTranslations('Admin');
    const { user } = useAuth();
    // Mirrors the API's judge.change_navigationbar gate on writes.
    const canWrite = hasPerm(user, 'judge.change_navigationbar');
    const [navBars, setNavBars] = useState<AdminNavigationBar[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(50);
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');

    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [editingDetail, setEditingDetail] = useState<AdminNavigationBarDetail | null>(null);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);

    const [formData, setFormData] = useState<AdminNavigationBarCreateRequest>({
        key: '',
        label: '',
        path: '',
        regex: '',
        parent_id: undefined,
        order: 0,
    });
    const [samplePath, setSamplePath] = useState('/problems');

    const loadNavBars = async () => {
        setLoading(true);
        try {
            const response = await adminNavigationBarApi.list(page, pageSize);
            setNavBars(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error(apiErrorMessage(error, t('navigationBars.loadFailed')));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadNavBars();
    }, [page]);

    const openCreateDialog = () => {
        setEditingDetail(null);
        setFormData({
            key: '',
            label: '',
            path: '',
            regex: '',
            parent_id: undefined,
            order: 0,
        });
        setIsCreateDialogOpen(true);
    };

    const openEditDialog = async (navBar: AdminNavigationBar) => {
        try {
            const response = await adminNavigationBarApi.detail(navBar.id);
            setFormData({
                key: response.data.key,
                label: response.data.label,
                path: response.data.path,
                regex: response.data.regex ?? '',
                parent_id: response.data.parent_id ?? undefined,
                order: response.data.order,
            });
            setEditingDetail(response.data);
            setIsCreateDialogOpen(true);
        } catch (error) {
            toast.error(apiErrorMessage(error, t('navigationBars.loadDetailFailed')));
        }
    };

    // A nav bar cannot be moved under itself or one of its descendants.
    const excludedParentIds = (() => {
        const excluded = new Set<number>();
        if (!editingDetail) return excluded;
        excluded.add(editingDetail.id);
        let changed = true;
        while (changed) {
            changed = false;
            for (const nb of navBars) {
                if (nb.parent_id != null && excluded.has(nb.parent_id) && !excluded.has(nb.id)) {
                    excluded.add(nb.id);
                    changed = true;
                }
            }
        }
        return excluded;
    })();

    // Preview the highlight regex against a sample path, same syntax as Go's
    // regexp package on the server.
    const regexPreview: 'empty' | 'invalid' | 'match' | 'no-match' = (() => {
        if (!formData.regex) return 'empty';
        try {
            return new RegExp(formData.regex).test(samplePath) ? 'match' : 'no-match';
        } catch {
            return 'invalid';
        }
    })();
    // A stored v1 pattern that does not compile in JS must stay saveable; only
    // a pattern the admin just typed is blocked.
    const regexBlocksSave = regexPreview === 'invalid'
        && (formData.regex ?? '') !== (editingDetail?.regex ?? '');

    const handleCreate = async () => {
        try {
            await adminNavigationBarApi.create(formData);
            toast.success(t('navigationBars.createSuccess'));
            setIsCreateDialogOpen(false);
            loadNavBars();
        } catch (caught) {
            const error = caught as ApiError;
            toast.error(error.response?.data?.error || t('navigationBars.createError'));
        }
    };

    const handleUpdate = async () => {
        if (!editingDetail) return;

        try {
            await adminNavigationBarApi.update(editingDetail.id, navigationBarUpdatePayload(formData, editingDetail));
            toast.success(t('navigationBars.updateSuccess'));
            setIsCreateDialogOpen(false);
            loadNavBars();
        } catch (caught) {
            const error = caught as ApiError;
            toast.error(error.response?.data?.error || t('navigationBars.updateError'));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminNavigationBarApi.delete(id);
            toast.success(t('navigationBars.deleteSuccess'));
            setDeleteConfirmId(null);
            loadNavBars();
        } catch (caught) {
            const error = caught as ApiError;
            toast.error(error.response?.data?.error || t('navigationBars.deleteError'));
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Layers className="text-primary" size={32} />
                        {t('navigationBars.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {t('navigationBars.subtitle')}
                    </p>
                </div>
                <Button onClick={openCreateDialog} disabled={!canWrite}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('navigationBars.addButton')}
                </Button>
            </div>
            {!canWrite && (
                <p className="text-xs text-muted-foreground">{t('navigationBars.writeHint')}</p>
            )}

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder={t('navigationBars.searchPlaceholder')}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-10"
                />
            </div>

            {/* Navigation Bars Table */}
            <div className="border rounded-lg overflow-hidden">
                <table className="w-full text-left">
                    <thead className="bg-muted/50 border-b">
                        <tr>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('navigationBars.colKey')}</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('navigationBars.colLabel')}</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('navigationBars.colPath')}</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('navigationBars.colParent')}</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('navigationBars.colOrder')}</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('navigationBars.colLevel')}</th>
                            <th className="px-6 py-4 text-right text-xs font-bold uppercase text-muted-foreground">{t('common.actions')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {loading ? (
                            <tr>
                                <td colSpan={7} className="text-center py-8">
                                    {t('common.loading')}
                                </td>
                            </tr>
                        ) : navBars.length === 0 ? (
                            <tr>
                                <td colSpan={7} className="text-center py-8 text-muted-foreground">
                                    {t('navigationBars.noNavBarsFound')}
                                </td>
                            </tr>
                        ) : (
                            navBars.map((navBar) => (
                                <tr key={navBar.id} className="border-b hover:bg-muted/30">
                                    <td className="px-6 py-4">
                                        <code className="text-xs bg-muted px-2 py-1 rounded">{navBar.key}</code>
                                    </td>
                                    <td className="px-6 py-4 font-medium">{navBar.label}</td>
                                    <td className="px-6 py-4">
                                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                            <LinkIcon size={14} />
                                            {navBar.path}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4">
                                        {navBar.parent ? (
                                            <Badge variant="secondary">{navBar.parent.label}</Badge>
                                        ) : (
                                            <span className="text-muted-foreground">-</span>
                                        )}
                                    </td>
                                    <td className="px-6 py-4 text-sm">{navBar.order}</td>
                                    <td className="px-6 py-4">
                                        <Badge variant="outline">{t('navigationBars.levelBadge', { level: navBar.level })}</Badge>
                                    </td>
                                    <td className="px-6 py-4 text-right">
                                        <div className="flex justify-end gap-2">
                                            {canWrite && (
                                                <>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => openEditDialog(navBar)}
                                                        title={t('common.edit')}
                                                        aria-label={t('common.edit')}
                                                    >
                                                        <Edit className="h-4 w-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => setDeleteConfirmId(navBar.id)}
                                                        className="text-destructive hover:text-destructive"
                                                        title={t('common.delete')}
                                                        aria-label={t('common.delete')}
                                                    >
                                                        <Trash2 className="h-4 w-4" />
                                                    </Button>
                                                </>
                                            )}
                                        </div>
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between">
                <div className="text-sm text-muted-foreground">
                    {t('common.showingRange', { from: (page - 1) * pageSize + 1, to: Math.min(page * pageSize, total), total })}
                </div>
                <div className="flex gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        disabled={page === 1}
                        onClick={() => setPage(page - 1)}
                    >
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

            {/* Create/Edit Dialog */}
            <Dialog open={isCreateDialogOpen} onOpenChange={(open) => {
                setIsCreateDialogOpen(open);
                if (!open) setEditingDetail(null);
            }}>
                <DialogContent className="max-w-lg">
                    <DialogHeader>
                        <DialogTitle>
                            {editingDetail ? t('navigationBars.editDialogTitle') : t('navigationBars.createDialogTitle')}
                        </DialogTitle>
                        <DialogDescription>
                            {editingDetail ? t('navigationBars.editDialogDesc') : t('navigationBars.createDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="key">{t('navigationBars.keyLabel')}</Label>
                                <Input
                                    id="key"
                                    value={formData.key}
                                    onChange={(e) => setFormData({
                ...formData, key: e.target.value })}
                                    placeholder={t('navigationBars.keyPlaceholder')}
                                    disabled={!!editingDetail}
                                />
                                <p className="text-xs text-muted-foreground">{t('navigationBars.keyHint')}</p>
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="label">{t('navigationBars.labelLabel')}</Label>
                                <Input
                                    id="label"
                                    value={formData.label}
                                    onChange={(e) => setFormData({
                ...formData, label: e.target.value })}
                                    placeholder={t('navigationBars.labelPlaceholder')}
                                />
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="path">{t('navigationBars.pathLabel')}</Label>
                            <Input
                                id="path"
                                value={formData.path}
                                onChange={(e) => setFormData({
                ...formData, path: e.target.value })}
                                placeholder={t('navigationBars.pathPlaceholder')}
                            />
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="regex">{t('navigationBars.regexLabel')}</Label>
                            <Textarea
                                id="regex"
                                rows={2}
                                className="font-mono"
                                value={formData.regex ?? ''}
                                onChange={(e) => setFormData({
                ...formData, regex: e.target.value })}
                                placeholder={t('navigationBars.regexPlaceholder')}
                            />
                            <p className="text-xs text-muted-foreground">{t('navigationBars.regexHint')}</p>
                            <div className="flex items-center gap-2">
                                <Input
                                    aria-label={t('navigationBars.regexPreviewLabel')}
                                    value={samplePath}
                                    onChange={(e) => setSamplePath(e.target.value)}
                                    className="h-9 font-mono"
                                />
                                <Badge
                                    variant={
                                        regexPreview === 'match' ? 'success'
                                            : regexPreview === 'invalid' ? 'destructive'
                                                : regexPreview === 'no-match' ? 'warning' : 'outline'
                                    }
                                    className="shrink-0"
                                >
                                    {regexPreview === 'empty' && t('navigationBars.regexPreviewEmpty')}
                                    {regexPreview === 'invalid' && t('navigationBars.regexPreviewInvalid')}
                                    {regexPreview === 'match' && t('navigationBars.regexPreviewMatch')}
                                    {regexPreview === 'no-match' && t('navigationBars.regexPreviewNoMatch')}
                                </Badge>
                            </div>
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="parent">{t('navigationBars.parentLabel')}</Label>
                                <Select
                                    id="parent"
                                    value={formData.parent_id ?? ''}
                                    onChange={(e) => setFormData({
                ...formData, parent_id: e.target.value ? Number(e.target.value) : undefined })}
                                >
                                    <option value="">{t('navigationBars.parentNone')}</option>
                                    {/* The current parent can sit on another page; inject it from the detail so the select is not blank. */}
                                    {editingDetail?.parent_id != null && !navBars.some((nb) => nb.id === editingDetail.parent_id) && (
                                        <option value={editingDetail.parent_id}>
                                            {'\u00A0'.repeat(Math.max(0, editingDetail.level - 1) * 2)}{editingDetail.parent_label}
                                        </option>
                                    )}
                                    {/* shortcut: options come from the loaded page (50), upgrade to a full fetch if nav items ever exceed one page */}
                                    {navBars
                                        .filter((nb) => !excludedParentIds.has(nb.id))
                                        .map((nb) => (
                                            <option key={nb.id} value={nb.id}>
                                                {'\u00A0'.repeat(nb.level * 2)}{nb.label}
                                            </option>
                                        ))}
                                </Select>
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="order">{t('navigationBars.orderLabel')}</Label>
                                <Input
                                    id="order"
                                    type="number"
                                    value={formData.order}
                                    onChange={(e) => setFormData({
                ...formData, order: parseInt(e.target.value) || 0 })}
                                />
                            </div>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateDialogOpen(false);
                            setEditingDetail(null);
                        }}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={editingDetail ? handleUpdate : handleCreate} disabled={regexBlocksSave}>
                            {editingDetail ? t('common.saveChanges') : t('common.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('navigationBars.deleteDialogTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('navigationBars.deleteDialogDesc')}
                        </DialogDescription>
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
