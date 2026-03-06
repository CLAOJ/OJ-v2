'use client';

import { useState, useEffect } from 'react';
import { adminProblemGroupApi, adminProblemTypeApi } from '@/lib/adminApi';
import { AdminProblemGroup, AdminProblemType, AdminProblemGroupCreateRequest, AdminProblemTypeCreateRequest } from '@/types';
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
import { toast } from 'sonner';
import { Plus, Edit, Trash2, Search, Folder, Tag } from 'lucide-react';

type TabType = 'groups' | 'types';

export default function ProblemTaxonomyAdminPage() {
    const [activeTab, setActiveTab] = useState<TabType>('groups');

    // Problem Groups state
    const [groups, setGroups] = useState<AdminProblemGroup[]>([]);
    const [groupTotal, setGroupTotal] = useState(0);
    const [groupPage, setGroupPage] = useState(1);
    const [groupLoading, setGroupLoading] = useState(false);
    const [isCreateGroupDialogOpen, setIsCreateGroupDialogOpen] = useState(false);
    const [editingGroup, setEditingGroup] = useState<AdminProblemGroup | null>(null);
    const [isEditGroupDialogOpen, setIsEditGroupDialogOpen] = useState(false);
    const [deleteGroupConfirmId, setDeleteGroupConfirmId] = useState<number | null>(null);
    const [groupFormData, setGroupFormData] = useState<AdminProblemGroupCreateRequest>({
        name: '',
        full_name: '',
    });

    // Problem Types state
    const [types, setTypes] = useState<AdminProblemType[]>([]);
    const [typeTotal, setTypeTotal] = useState(0);
    const [typePage, setTypePage] = useState(1);
    const [typeLoading, setTypeLoading] = useState(false);
    const [isCreateTypeDialogOpen, setIsCreateTypeDialogOpen] = useState(false);
    const [editingType, setEditingType] = useState<AdminProblemType | null>(null);
    const [isEditTypeDialogOpen, setIsEditTypeDialogOpen] = useState(false);
    const [deleteTypeConfirmId, setDeleteTypeConfirmId] = useState<number | null>(null);
    const [typeFormData, setTypeFormData] = useState<AdminProblemTypeCreateRequest>({
        name: '',
        full_name: '',
    });

    const loadGroups = async () => {
        setGroupLoading(true);
        try {
            const response = await adminProblemGroupApi.list(groupPage, 20);
            setGroups(response.data.data);
            setGroupTotal(response.data.total);
        } catch (error) {
            toast.error('Failed to load problem groups');
        } finally {
            setGroupLoading(false);
        }
    };

    const loadTypes = async () => {
        setTypeLoading(true);
        try {
            const response = await adminProblemTypeApi.list(typePage, 20);
            setTypes(response.data.data);
            setTypeTotal(response.data.total);
        } catch (error) {
            toast.error('Failed to load problem types');
        } finally {
            setTypeLoading(false);
        }
    };

    useEffect(() => {
        if (activeTab === 'groups') {
            loadGroups();
        } else {
            loadTypes();
        }
    }, [activeTab, groupPage, typePage]);

    // Group handlers
    const openCreateGroupDialog = () => {
        setGroupFormData({ name: '', full_name: '' });
        setIsCreateGroupDialogOpen(true);
    };

    const openEditGroupDialog = async (group: AdminProblemGroup) => {
        const response = await adminProblemGroupApi.detail(group.id);
        setGroupFormData({
            name: response.data.name,
            full_name: response.data.full_name,
        });
        setEditingGroup(group);
        setIsEditGroupDialogOpen(true);
    };

    const handleCreateGroup = async () => {
        try {
            await adminProblemGroupApi.create(groupFormData);
            toast.success('Problem group created');
            setIsCreateGroupDialogOpen(false);
            loadGroups();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to create problem group');
        }
    };

    const handleUpdateGroup = async () => {
        if (!editingGroup) return;
        try {
            await adminProblemGroupApi.update(editingGroup.id, { full_name: groupFormData.full_name });
            toast.success('Problem group updated');
            setIsEditGroupDialogOpen(false);
            loadGroups();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to update problem group');
        }
    };

    const handleDeleteGroup = async (id: number) => {
        try {
            await adminProblemGroupApi.delete(id);
            toast.success('Problem group deleted');
            setDeleteGroupConfirmId(null);
            loadGroups();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to delete problem group');
        }
    };

    // Type handlers
    const openCreateTypeDialog = () => {
        setTypeFormData({ name: '', full_name: '' });
        setIsCreateTypeDialogOpen(true);
    };

    const openEditTypeDialog = async (ptype: AdminProblemType) => {
        const response = await adminProblemTypeApi.detail(ptype.id);
        setTypeFormData({
            name: response.data.name,
            full_name: response.data.full_name,
        });
        setEditingType(ptype);
        setIsEditTypeDialogOpen(true);
    };

    const handleCreateType = async () => {
        try {
            await adminProblemTypeApi.create(typeFormData);
            toast.success('Problem type created');
            setIsCreateTypeDialogOpen(false);
            loadTypes();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to create problem type');
        }
    };

    const handleUpdateType = async () => {
        if (!editingType) return;
        try {
            await adminProblemTypeApi.update(editingType.id, { full_name: typeFormData.full_name });
            toast.success('Problem type updated');
            setIsEditTypeDialogOpen(false);
            loadTypes();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to update problem type');
        }
    };

    const handleDeleteType = async (id: number) => {
        try {
            await adminProblemTypeApi.delete(id);
            toast.success('Problem type deleted');
            setDeleteTypeConfirmId(null);
            loadTypes();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to delete problem type');
        }
    };

    const groupTotalPages = Math.ceil(groupTotal / 20);
    const typeTotalPages = Math.ceil(typeTotal / 20);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">Problem Taxonomy</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage problem groups and types
                    </p>
                </div>
            </div>

            {/* Tabs */}
            <div className="flex gap-2 border-b">
                <Button
                    variant={activeTab === 'groups' ? 'default' : 'ghost'}
                    onClick={() => setActiveTab('groups')}
                    className="gap-2 rounded-t-lg"
                >
                    <Folder className="h-4 w-4" />
                    Problem Groups ({groupTotal})
                </Button>
                <Button
                    variant={activeTab === 'types' ? 'default' : 'ghost'}
                    onClick={() => setActiveTab('types')}
                    className="gap-2 rounded-t-lg"
                >
                    <Tag className="h-4 w-4" />
                    Problem Types ({typeTotal})
                </Button>
            </div>

            {/* Problem Groups Tab */}
            {activeTab === 'groups' && (
                <div className="space-y-4">
                    <div className="flex justify-end">
                        <Button onClick={openCreateGroupDialog}>
                            <Plus className="h-4 w-4 mr-2" />
                            Add Group
                        </Button>
                    </div>

                    <div className="border rounded-lg overflow-hidden">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>Name</TableHead>
                                    <TableHead>Full Name</TableHead>
                                    <TableHead className="text-right">Actions</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {groupLoading ? (
                                    <TableRow>
                                        <TableCell colSpan={3} className="text-center py-8">
                                            Loading...
                                        </TableCell>
                                    </TableRow>
                                ) : groups.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={3} className="text-center py-8 text-muted-foreground">
                                            No problem groups found
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    groups.map((group) => (
                                        <TableRow key={group.id}>
                                            <TableCell className="font-mono font-bold">{group.name}</TableCell>
                                            <TableCell>{group.full_name}</TableCell>
                                            <TableCell className="text-right">
                                                <div className="flex justify-end gap-2">
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => openEditGroupDialog(group)}
                                                    >
                                                        <Edit className="h-4 w-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => setDeleteGroupConfirmId(group.id)}
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
                            Showing {(groupPage - 1) * 20 + 1} to {Math.min(groupPage * 20, groupTotal)} of {groupTotal}
                        </div>
                        <div className="flex gap-2">
                            <Button
                                variant="outline"
                                size="sm"
                                disabled={groupPage === 1}
                                onClick={() => setGroupPage(groupPage - 1)}
                            >
                                Previous
                            </Button>
                            <Button
                                variant="outline"
                                size="sm"
                                disabled={groupPage >= groupTotalPages}
                                onClick={() => setGroupPage(groupPage + 1)}
                            >
                                Next
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            {/* Problem Types Tab */}
            {activeTab === 'types' && (
                <div className="space-y-4">
                    <div className="flex justify-end">
                        <Button onClick={openCreateTypeDialog}>
                            <Plus className="h-4 w-4 mr-2" />
                            Add Type
                        </Button>
                    </div>

                    <div className="border rounded-lg overflow-hidden">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>Name</TableHead>
                                    <TableHead>Full Name</TableHead>
                                    <TableHead className="text-right">Actions</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {typeLoading ? (
                                    <TableRow>
                                        <TableCell colSpan={3} className="text-center py-8">
                                            Loading...
                                        </TableCell>
                                    </TableRow>
                                ) : types.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={3} className="text-center py-8 text-muted-foreground">
                                            No problem types found
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    types.map((ptype) => (
                                        <TableRow key={ptype.id}>
                                            <TableCell className="font-mono font-bold">{ptype.name}</TableCell>
                                            <TableCell>{ptype.full_name}</TableCell>
                                            <TableCell className="text-right">
                                                <div className="flex justify-end gap-2">
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => openEditTypeDialog(ptype)}
                                                    >
                                                        <Edit className="h-4 w-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => setDeleteTypeConfirmId(ptype.id)}
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
                            Showing {(typePage - 1) * 20 + 1} to {Math.min(typePage * 20, typeTotal)} of {typeTotal}
                        </div>
                        <div className="flex gap-2">
                            <Button
                                variant="outline"
                                size="sm"
                                disabled={typePage === 1}
                                onClick={() => setTypePage(typePage - 1)}
                            >
                                Previous
                            </Button>
                            <Button
                                variant="outline"
                                size="sm"
                                disabled={typePage >= typeTotalPages}
                                onClick={() => setTypePage(typePage + 1)}
                            >
                                Next
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            {/* Create/Edit Group Dialog */}
            <Dialog open={isCreateGroupDialogOpen || isEditGroupDialogOpen} onOpenChange={(open) => {
                setIsCreateGroupDialogOpen(open);
                setIsEditGroupDialogOpen(open);
            }}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>
                            {editingGroup ? 'Edit Problem Group' : 'Create Problem Group'}
                        </DialogTitle>
                        <DialogDescription>
                            {editingGroup ? 'Update problem group details.' : 'Create a new problem group.'}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="group_name">Name (Key)</Label>
                            <Input
                                id="group_name"
                                value={groupFormData.name}
                                onChange={(e) => setGroupFormData({ ...groupFormData, name: e.target.value })}
                                placeholder="e.g., IOI"
                                disabled={editingGroup !== null}
                            />
                        </div>
                        <div className="grid gap-2">
                            <Label htmlFor="group_full_name">Full Name</Label>
                            <Input
                                id="group_full_name"
                                value={groupFormData.full_name}
                                onChange={(e) => setGroupFormData({ ...groupFormData, full_name: e.target.value })}
                                placeholder="e.g., International Olympiad in Informatics"
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateGroupDialogOpen(false);
                            setIsEditGroupDialogOpen(false);
                        }}>
                            Cancel
                        </Button>
                        <Button onClick={editingGroup ? handleUpdateGroup : handleCreateGroup}>
                            {editingGroup ? 'Save Changes' : 'Create'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Create/Edit Type Dialog */}
            <Dialog open={isCreateTypeDialogOpen || isEditTypeDialogOpen} onOpenChange={(open) => {
                setIsCreateTypeDialogOpen(open);
                setIsEditTypeDialogOpen(open);
            }}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>
                            {editingType ? 'Edit Problem Type' : 'Create Problem Type'}
                        </DialogTitle>
                        <DialogDescription>
                            {editingType ? 'Update problem type details.' : 'Create a new problem type.'}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="type_name">Name (Key)</Label>
                            <Input
                                id="type_name"
                                value={typeFormData.name}
                                onChange={(e) => setTypeFormData({ ...typeFormData, name: e.target.value })}
                                placeholder="e.g., DP"
                                disabled={editingType !== null}
                            />
                        </div>
                        <div className="grid gap-2">
                            <Label htmlFor="type_full_name">Full Name</Label>
                            <Input
                                id="type_full_name"
                                value={typeFormData.full_name}
                                onChange={(e) => setTypeFormData({ ...typeFormData, full_name: e.target.value })}
                                placeholder="e.g., Dynamic Programming"
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateTypeDialogOpen(false);
                            setIsEditTypeDialogOpen(false);
                        }}>
                            Cancel
                        </Button>
                        <Button onClick={editingType ? handleUpdateType : handleCreateType}>
                            {editingType ? 'Save Changes' : 'Create'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmations */}
            <Dialog open={deleteGroupConfirmId !== null} onOpenChange={() => setDeleteGroupConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Problem Group</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this problem group? This cannot be undone if the group is used by problems.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteGroupConfirmId(null)}>
                            Cancel
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteGroupConfirmId && handleDeleteGroup(deleteGroupConfirmId)}
                        >
                            Delete
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={deleteTypeConfirmId !== null} onOpenChange={() => setDeleteTypeConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Problem Type</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this problem type? This cannot be undone.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteTypeConfirmId(null)}>
                            Cancel
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteTypeConfirmId && handleDeleteType(deleteTypeConfirmId)}
                        >
                            Delete
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
