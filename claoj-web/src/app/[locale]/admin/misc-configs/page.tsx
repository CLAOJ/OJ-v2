'use client';

import { useState, useEffect } from 'react';
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
            toast.error('Failed to load misc configs');
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
            toast.error('Failed to load config details');
        }
    };

    const handleCreate = async () => {
        try {
            await adminMiscConfigApi.create(formData);
            toast.success('Config created');
            setIsCreateDialogOpen(false);
            loadMiscConfigs();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to create config');
        }
    };

    const handleUpdate = async () => {
        if (!editingConfig) return;

        try {
            const updateData: AdminMiscConfigUpdateRequest = {
                value: formData.value || '',
            };
            await adminMiscConfigApi.update(editingConfig.id, updateData);
            toast.success('Config updated');
            setIsCreateDialogOpen(false);
            loadMiscConfigs();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to update config');
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminMiscConfigApi.delete(id);
            toast.success('Config deleted');
            setDeleteConfirmId(null);
            loadMiscConfigs();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to delete config');
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Settings className="text-primary" size={32} />
                        Misc Configs
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        Manage site-wide configuration key-value pairs
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Config
                </Button>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder="Search configs by key..."
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
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Key</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Value</th>
                            <th className="px-6 py-4 text-right text-xs font-bold uppercase text-muted-foreground">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {loading ? (
                            <tr>
                                <td colSpan={3} className="text-center py-8">
                                    Loading...
                                </td>
                            </tr>
                        ) : miscConfigs.length === 0 ? (
                            <tr>
                                <td colSpan={3} className="text-center py-8 text-muted-foreground">
                                    No configs found
                                </td>
                            </tr>
                        ) : (
                            miscConfigs.map((config) => (
                                <tr key={config.id} className="border-b hover:bg-muted/30">
                                    <td className="px-6 py-4">
                                        <code className="text-xs bg-muted px-2 py-1 rounded font-mono">{config.key}</code>
                                    </td>
                                    <td className="px-6 py-4">
                                        <span className="text-sm font-mono break-all">{config.value || <span className="text-muted-foreground italic">empty</span>}</span>
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
                    Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, total)} of {total}
                </div>
                <div className="flex gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        disabled={page === 1}
                        onClick={() => setPage(page - 1)}
                    >
                        Previous
                    </Button>
                    <Button
                        variant="outline"
                        size="sm"
                        disabled={page >= totalPages}
                        onClick={() => setPage(page + 1)}
                    >
                        Next
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
                            {editingConfig ? 'Edit Config' : 'Create Config'}
                        </DialogTitle>
                        <DialogDescription>
                            {editingConfig ? 'Update configuration value.' : 'Add a new configuration key-value pair.'}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="key">Key</Label>
                            <Input
                                id="key"
                                value={formData.key}
                                onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                                placeholder="e.g., site_name"
                                disabled={!!editingConfig}
                            />
                            <p className="text-xs text-muted-foreground">Unique identifier (cannot be changed after creation)</p>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="value">Value</Label>
                            <Input
                                id="value"
                                value={formData.value}
                                onChange={(e) => setFormData({ ...formData, value: e.target.value })}
                                placeholder="e.g., My Online Judge"
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateDialogOpen(false);
                            setEditingConfig(null);
                        }}>
                            Cancel
                        </Button>
                        <Button onClick={editingConfig ? handleUpdate : handleCreate}>
                            {editingConfig ? 'Save Changes' : 'Create'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Config</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this config? This action cannot be undone.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteConfirmId(null)}>
                            Cancel
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteConfirmId !== null && handleDelete(deleteConfirmId)}
                        >
                            Delete
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
