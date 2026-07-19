'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { adminMiscConfigApi } from '@/lib/adminApi';
import { AdminMiscConfig, AdminMiscConfigUpdateRequest } from '@/types';
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
import { toast } from 'sonner';
import { Plus, Edit, Trash2, Search, Settings } from 'lucide-react';

interface FormData {
    key: string;
    value: string;
}

export default function MiscConfigsAdminPage() {
    const t = useTranslations('Admin');
    const [miscConfigs, setMiscConfigs] = useState<AdminMiscConfig[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(50);
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');

    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [editingConfig, setEditingConfig] = useState<AdminMiscConfig | null>(null);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);

    const [formData, setFormData] = useState<FormData>({
        key: '',
        value: '',
    });

    const loadMiscConfigs = async () => {
        setLoading(true);
        try {
            const response = await adminMiscConfigApi.list(page, pageSize, search || undefined);
            setMiscConfigs(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error(t('miscConfigs.loadFailed'));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadMiscConfigs();
    }, [page]);

    const openCreateDialog = () => {
        setFormData({
            key: '',
            value: '',
        });
        setIsCreateDialogOpen(true);
    };

    const openEditDialog = async (config: AdminMiscConfig) => {
        try {
            const response = await adminMiscConfigApi.detail(config.id);
            setFormData({
                key: response.data.key,
                value: response.data.value,
            });
            setEditingConfig(config);
            setIsCreateDialogOpen(true);
        } catch (error) {
            toast.error(t('miscConfigs.loadDetailFailed'));
        }
    };

    const handleCreate = async () => {
        try {
            await adminMiscConfigApi.create(formData);
            toast.success(t('miscConfigs.createSuccess'));
            setIsCreateDialogOpen(false);
            loadMiscConfigs();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('miscConfigs.createError'));
        }
    };

    const handleUpdate = async () => {
        if (!editingConfig) return;

        try {
            const updateData: AdminMiscConfigUpdateRequest = {
                value: formData.value || '',
            };
            await adminMiscConfigApi.update(editingConfig.id, updateData);
            toast.success(t('miscConfigs.updateSuccess'));
            setIsCreateDialogOpen(false);
            loadMiscConfigs();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('miscConfigs.updateError'));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminMiscConfigApi.delete(id);
            toast.success(t('miscConfigs.deleteSuccess'));
            setDeleteConfirmId(null);
            loadMiscConfigs();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('miscConfigs.deleteError'));
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Settings className="text-primary" size={32} />
                        {t('miscConfigs.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {t('miscConfigs.subtitle')}
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('miscConfigs.addButton')}
                </Button>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder={t('miscConfigs.searchPlaceholder')}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-10"
                    onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                            setPage(1);
                            loadMiscConfigs();
                        }
                    }}
                />
            </div>

            {/* Misc Configs Table */}
            <div className="border rounded-lg overflow-hidden">
                <table className="w-full text-left">
                    <thead className="bg-muted/50 border-b">
                        <tr>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('miscConfigs.colKey')}</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('miscConfigs.colValue')}</th>
                            <th className="px-6 py-4 text-right text-xs font-bold uppercase text-muted-foreground">{t('common.actions')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {loading ? (
                            <tr>
                                <td colSpan={3} className="text-center py-8">
                                    {t('common.loading')}
                                </td>
                            </tr>
                        ) : miscConfigs.length === 0 ? (
                            <tr>
                                <td colSpan={3} className="text-center py-8 text-muted-foreground">
                                    {t('miscConfigs.noConfigsFound')}
                                </td>
                            </tr>
                        ) : (
                            miscConfigs.map((config) => (
                                <tr key={config.id} className="border-b hover:bg-muted/30">
                                    <td className="px-6 py-4">
                                        <code className="text-xs bg-muted px-2 py-1 rounded font-mono">{config.key}</code>
                                    </td>
                                    <td className="px-6 py-4">
                                        <span className="text-sm font-mono break-all">{config.value || <span className="text-muted-foreground italic">{t('miscConfigs.emptyValue')}</span>}</span>
                                    </td>
                                    <td className="px-6 py-4 text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => openEditDialog(config)}
                                            >
                                                <Edit className="h-4 w-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => setDeleteConfirmId(config.id)}
                                                className="text-destructive hover:text-destructive"
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </Button>
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
                if (!open) setEditingConfig(null);
            }}>
                <DialogContent className="max-w-lg">
                    <DialogHeader>
                        <DialogTitle>
                            {editingConfig ? t('miscConfigs.editDialogTitle') : t('miscConfigs.createDialogTitle')}
                        </DialogTitle>
                        <DialogDescription>
                            {editingConfig ? t('miscConfigs.editDialogDesc') : t('miscConfigs.createDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="key">{t('miscConfigs.keyLabel')}</Label>
                            <Input
                                id="key"
                                value={formData.key}
                                onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                                placeholder={t('miscConfigs.keyPlaceholder')}
                                disabled={!!editingConfig}
                            />
                            <p className="text-xs text-muted-foreground">{t('miscConfigs.keyHint')}</p>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="value">{t('miscConfigs.valueLabel')}</Label>
                            <Input
                                id="value"
                                value={formData.value}
                                onChange={(e) => setFormData({ ...formData, value: e.target.value })}
                                placeholder={t('miscConfigs.valuePlaceholder')}
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateDialogOpen(false);
                            setEditingConfig(null);
                        }}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={editingConfig ? handleUpdate : handleCreate}>
                            {editingConfig ? t('common.saveChanges') : t('common.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('miscConfigs.deleteDialogTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('miscConfigs.deleteDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteConfirmId(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteConfirmId !== null && handleDelete(deleteConfirmId)}
                        >
                            {t('common.delete')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
