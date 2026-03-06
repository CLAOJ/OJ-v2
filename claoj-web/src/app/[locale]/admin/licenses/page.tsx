'use client';

import { useState, useEffect } from 'react';
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
            toast.error('Failed to load licenses');
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
            toast.error('Failed to load license details');
        }
    };

    const handleCreate = async () => {
        try {
            await adminLicenseApi.create(formData);
            toast.success('License created');
            setIsCreateDialogOpen(false);
            loadLicenses();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to create license');
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
            toast.success('License updated');
            setIsEditDialogOpen(false);
            loadLicenses();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to update license');
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminLicenseApi.delete(id);
            toast.success('License deleted');
            setDeleteConfirmId(null);
            loadLicenses();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to delete license');
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">Licenses</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage software licenses for problems
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add License
                </Button>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder="Search licenses..."
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
                            <TableHead>Key</TableHead>
                            <TableHead>Name</TableHead>
                            <TableHead>Display</TableHead>
                            <TableHead>Link</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8">
                                    Loading...
                                </TableCell>
                            </TableRow>
                        ) : licenses.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                                    No licenses found
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
                                            Link
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
            <Dialog open={isCreateDialogOpen || isEditDialogOpen} onOpenChange={(open) => {
                setIsCreateDialogOpen(open);
                setIsEditDialogOpen(open);
            }}>
                <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>
                            {editingLicense ? 'Edit License' : 'Create License'}
                        </DialogTitle>
                        <DialogDescription>
                            {editingLicense ? 'Update license details.' : 'Create a new software license.'}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="key">License Key</Label>
                                <Input
                                    id="key"
                                    value={formData.key}
                                    onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                                    placeholder="e.g., MIT"
                                    disabled={editingLicense !== null}
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="name">Name</Label>
                                <Input
                                    id="name"
                                    value={formData.name}
                                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                    placeholder="e.g., MIT License"
                                />
                            </div>
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="display">Display Text</Label>
                                <Input
                                    id="display"
                                    value={formData.display}
                                    onChange={(e) => setFormData({ ...formData, display: e.target.value })}
                                    placeholder="Short display text"
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="icon">Icon URL</Label>
                                <Input
                                    id="icon"
                                    value={formData.icon}
                                    onChange={(e) => setFormData({ ...formData, icon: e.target.value })}
                                    placeholder="https://..."
                                />
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="link">License Link</Label>
                            <Input
                                id="link"
                                value={formData.link}
                                onChange={(e) => setFormData({ ...formData, link: e.target.value })}
                                placeholder="https://opensource.org/licenses/MIT"
                            />
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="text">License Text</Label>
                            <Textarea
                                id="text"
                                value={formData.text}
                                onChange={(e) => setFormData({ ...formData, text: e.target.value })}
                                rows={10}
                                className="font-mono text-sm"
                                placeholder="Full license text"
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateDialogOpen(false);
                            setIsEditDialogOpen(false);
                        }}>
                            Cancel
                        </Button>
                        <Button onClick={editingLicense ? handleUpdate : handleCreate}>
                            {editingLicense ? 'Save Changes' : 'Create'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete License</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this license? This cannot be undone if the license is used by problems.
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
