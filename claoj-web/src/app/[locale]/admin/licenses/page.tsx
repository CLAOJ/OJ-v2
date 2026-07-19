'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { adminLicenseApi } from '@/lib/adminApi';
import { AdminLicense, AdminLicenseCreateRequest, AdminLicenseUpdateRequest } from '@/types';
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
import { Textarea } from '@/components/ui/Textarea';
import { toast } from 'sonner';
import { Plus, Edit, Trash2, Search, Scale } from 'lucide-react';

export default function LicenseAdminPage() {
    const t = useTranslations('Admin');
    const [licenses, setLicenses] = useState<AdminLicense[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(20);
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');

    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [editingLicense, setEditingLicense] = useState<AdminLicense | null>(null);
    const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);

    const [formData, setFormData] = useState<AdminLicenseCreateRequest>({
        key: '',
        link: '',
        name: '',
        display: '',
        icon: '',
        text: '',
    });

    const loadLicenses = async () => {
        setLoading(true);
        try {
            const response = await adminLicenseApi.list(page, pageSize);
            setLicenses(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error(t('licenses.loadFailed'));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadLicenses();
    }, [page]);

    const openCreateDialog = () => {
        setFormData({
            key: '',
            link: '',
            name: '',
            display: '',
            icon: '',
            text: '',
        });
        setIsCreateDialogOpen(true);
    };

    const openEditDialog = async (license: AdminLicense) => {
        try {
            const response = await adminLicenseApi.detail(license.id);
            setFormData({
                key: response.data.key,
                link: response.data.link,
                name: response.data.name,
                display: response.data.display,
                icon: response.data.icon,
                text: response.data.text,
            });
            setEditingLicense(license);
            setIsEditDialogOpen(true);
        } catch (error) {
            toast.error(t('licenses.loadDetailFailed'));
        }
    };

    const handleCreate = async () => {
        try {
            await adminLicenseApi.create(formData);
            toast.success(t('licenses.createSuccess'));
            setIsCreateDialogOpen(false);
            loadLicenses();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('licenses.createError'));
        }
    };

    const handleUpdate = async () => {
        if (!editingLicense) return;

        try {
            const updateData: AdminLicenseUpdateRequest = {
                link: formData.link,
                name: formData.name,
                display: formData.display,
                icon: formData.icon,
                text: formData.text,
            };
            await adminLicenseApi.update(editingLicense.id, updateData);
            toast.success(t('licenses.updateSuccess'));
            setIsEditDialogOpen(false);
            loadLicenses();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('licenses.updateError'));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminLicenseApi.delete(id);
            toast.success(t('licenses.deleteSuccess'));
            setDeleteConfirmId(null);
            loadLicenses();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('licenses.deleteError'));
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">{t('licenses.title')}</h1>
                    <p className="text-muted-foreground mt-1">
                        {t('licenses.subtitle')}
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('licenses.addButton')}
                </Button>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder={t('licenses.searchPlaceholder')}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-10"
                />
            </div>

            {/* Licenses Table */}
            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t('licenses.colKey')}</TableHead>
                            <TableHead>{t('licenses.colName')}</TableHead>
                            <TableHead>{t('licenses.colDisplay')}</TableHead>
                            <TableHead>{t('licenses.colLink')}</TableHead>
                            <TableHead className="text-right">{t('common.actions')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8">
                                    {t('common.loading')}
                                </TableCell>
                            </TableRow>
                        ) : licenses.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                                    {t('licenses.noLicensesFound')}
                                </TableCell>
                            </TableRow>
                        ) : (
                            licenses.map((license) => (
                                <TableRow key={license.id}>
                                    <TableCell className="font-mono font-bold">{license.key}</TableCell>
                                    <TableCell>
                                        <div className="flex items-center gap-2">
                                            {license.icon && (
                                                <img src={license.icon} alt="" className="w-5 h-5" />
                                            )}
                                            <span className="font-medium">{license.name}</span>
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        {license.display ? (
                                            <Badge variant="default">{license.display}</Badge>
                                        ) : (
                                            <span className="text-muted-foreground">-</span>
                                        )}
                                    </TableCell>
                                    <TableCell>
                                        <a
                                            href={license.link}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-primary hover:underline text-sm"
                                        >
                                            <Scale className="inline h-3 w-3 mr-1" />
                                            {t('licenses.linkText')}
                                        </a>
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => openEditDialog(license)}
                                            >
                                                <Edit className="h-4 w-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => setDeleteConfirmId(license.id)}
                                                className="text-destructive hover:text-destructive"
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
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
            <Dialog open={isCreateDialogOpen || isEditDialogOpen} onOpenChange={(open) => {
                setIsCreateDialogOpen(open);
                setIsEditDialogOpen(open);
            }}>
                <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>
                            {editingLicense ? t('licenses.editDialogTitle') : t('licenses.createDialogTitle')}
                        </DialogTitle>
                        <DialogDescription>
                            {editingLicense ? t('licenses.editDialogDesc') : t('licenses.createDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="key">{t('licenses.keyLabel')}</Label>
                                <Input
                                    id="key"
                                    value={formData.key}
                                    onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                                    placeholder={t('licenses.keyPlaceholder')}
                                    disabled={editingLicense !== null}
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="name">{t('licenses.nameLabel')}</Label>
                                <Input
                                    id="name"
                                    value={formData.name}
                                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                    placeholder={t('licenses.namePlaceholder')}
                                />
                            </div>
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="display">{t('licenses.displayLabel')}</Label>
                                <Input
                                    id="display"
                                    value={formData.display}
                                    onChange={(e) => setFormData({ ...formData, display: e.target.value })}
                                    placeholder={t('licenses.displayPlaceholder')}
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="icon">{t('licenses.iconLabel')}</Label>
                                <Input
                                    id="icon"
                                    value={formData.icon}
                                    onChange={(e) => setFormData({ ...formData, icon: e.target.value })}
                                    placeholder={t('licenses.iconPlaceholder')}
                                />
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="link">{t('licenses.linkLabel')}</Label>
                            <Input
                                id="link"
                                value={formData.link}
                                onChange={(e) => setFormData({ ...formData, link: e.target.value })}
                                placeholder={t('licenses.linkPlaceholder')}
                            />
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="text">{t('licenses.textLabel')}</Label>
                            <Textarea
                                id="text"
                                value={formData.text}
                                onChange={(e) => setFormData({ ...formData, text: e.target.value })}
                                rows={10}
                                className="font-mono text-sm"
                                placeholder={t('licenses.textPlaceholder')}
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateDialogOpen(false);
                            setIsEditDialogOpen(false);
                        }}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={editingLicense ? handleUpdate : handleCreate}>
                            {editingLicense ? t('common.saveChanges') : t('common.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('licenses.deleteDialogTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('licenses.deleteDialogDesc')}
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
