'use client';

import { useState, useEffect } from 'react';
import { adminNavigationBarApi } from '@/lib/adminApi';
import { AdminNavigationBar, AdminNavigationBarCreateRequest, AdminNavigationBarUpdateRequest } from '@/types';
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
import { Plus, Edit, Trash2, Search, Link as LinkIcon, Layers } from 'lucide-react';

export default function NavigationBarsAdminPage() {
    const [navBars, setNavBars] = useState<AdminNavigationBar[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(50);
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');

    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [editingNavBar, setEditingNavBar] = useState<AdminNavigationBar | null>(null);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);

    const [formData, setFormData] = useState<AdminNavigationBarCreateRequest>({
        key: '',
        label: '',
        path: '',
        parent_id: undefined,
        order: 0,
    });

    const loadNavBars = async () => {
        setLoading(true);
        try {
            const response = await adminNavigationBarApi.list(page, pageSize);
            setNavBars(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error('Failed to load navigation bars');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadNavBars();
    }, [page]);

    const openCreateDialog = () => {
        setFormData({
            key: '',
            label: '',
            path: '',
            parent_id: undefined,
            order: 0,
        });
        setIsCreateDialogOpen(true);
    };

    const openEditDialog = async (navBar: AdminNavigationBar) => {
        try {
            const response = await adminNavigationBarApi.detail(navBar.id);
            setFormData({
                label: response.data.label,
                path: response.data.path,
                order: response.data.order,
            });
            setEditingNavBar(navBar);
            setIsCreateDialogOpen(true);
        } catch (error) {
            toast.error('Failed to load navigation bar details');
        }
    };

    const handleCreate = async () => {
        try {
            await adminNavigationBarApi.create(formData);
            toast.success('Navigation bar created');
            setIsCreateDialogOpen(false);
            loadNavBars();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to create navigation bar');
        }
    };

    const handleUpdate = async () => {
        if (!editingNavBar) return;

        try {
            const updateData: AdminNavigationBarUpdateRequest = {
                label: formData.label,
                path: formData.path,
                order: formData.order,
            };
            await adminNavigationBarApi.update(editingNavBar.id, updateData);
            toast.success('Navigation bar updated');
            setIsCreateDialogOpen(false);
            loadNavBars();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to update navigation bar');
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminNavigationBarApi.delete(id);
            toast.success('Navigation bar deleted');
            setDeleteConfirmId(null);
            loadNavBars();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to delete navigation bar');
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Layers className="text-primary" size={32} />
                        Navigation Bars
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        Manage site navigation menu items
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Navigation Item
                </Button>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder="Search navigation bars..."
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
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Key</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Label</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Path</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Parent</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Order</th>
                            <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Level</th>
                            <th className="px-6 py-4 text-right text-xs font-bold uppercase text-muted-foreground">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {loading ? (
                            <tr>
                                <td colSpan={7} className="text-center py-8">
                                    Loading...
                                </td>
                            </tr>
                        ) : navBars.length === 0 ? (
                            <tr>
                                <td colSpan={7} className="text-center py-8 text-muted-foreground">
                                    No navigation bars found
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
                                        <Badge variant="outline">Level {navBar.level}</Badge>
                                    </td>
                                    <td className="px-6 py-4 text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => openEditDialog(navBar)}
                                            >
                                                <Edit className="h-4 w-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => setDeleteConfirmId(navBar.id)}
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
                if (!open) setEditingNavBar(null);
            }}>
                <DialogContent className="max-w-lg">
                    <DialogHeader>
                        <DialogTitle>
                            {editingNavBar ? 'Edit Navigation Item' : 'Create Navigation Item'}
                        </DialogTitle>
                        <DialogDescription>
                            {editingNavBar ? 'Update navigation item details.' : 'Add a new navigation menu item.'}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="key">Key</Label>
                                <Input
                                    id="key"
                                    value={formData.key}
                                    onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                                    placeholder="e.g., home"
                                    disabled={!!editingNavBar}
                                />
                                <p className="text-xs text-muted-foreground">Unique identifier</p>
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="label">Label</Label>
                                <Input
                                    id="label"
                                    value={formData.label}
                                    onChange={(e) => setFormData({ ...formData, label: e.target.value })}
                                    placeholder="e.g., Home"
                                />
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="path">Path</Label>
                            <Input
                                id="path"
                                value={formData.path}
                                onChange={(e) => setFormData({ ...formData, path: e.target.value })}
                                placeholder="e.g., / or /problems"
                            />
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="order">Order</Label>
                                <Input
                                    id="order"
                                    type="number"
                                    value={formData.order}
                                    onChange={(e) => setFormData({ ...formData, order: parseInt(e.target.value) || 0 })}
                                />
                            </div>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateDialogOpen(false);
                            setEditingNavBar(null);
                        }}>
                            Cancel
                        </Button>
                        <Button onClick={editingNavBar ? handleUpdate : handleCreate}>
                            {editingNavBar ? 'Save Changes' : 'Create'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Navigation Item</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this navigation item? This action cannot be undone.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteConfirmId(null)}>
                            Cancel
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}
                        >
                            Delete
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
