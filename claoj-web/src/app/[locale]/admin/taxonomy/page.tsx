'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
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
    const t = useTranslations('Admin');
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
            toast.error(t('taxonomy.loadGroupsFailed'));
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
            toast.error(t('taxonomy.loadTypesFailed'));
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
            toast.success(t('taxonomy.groupCreated'));
            setIsCreateGroupDialogOpen(false);
            loadGroups();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('taxonomy.groupCreateFailed'));
        }
    };

    const handleUpdateGroup = async () => {
        if (!editingGroup) return;
        try {
            await adminProblemGroupApi.update(editingGroup.id, { full_name: groupFormData.full_name });
            toast.success(t('taxonomy.groupUpdated'));
            setIsEditGroupDialogOpen(false);
            loadGroups();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('taxonomy.groupUpdateFailed'));
        }
    };

    const handleDeleteGroup = async (id: number) => {
        try {
            await adminProblemGroupApi.delete(id);
            toast.success(t('taxonomy.groupDeleted'));
            setDeleteGroupConfirmId(null);
            loadGroups();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('taxonomy.groupDeleteFailed'));
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
            toast.success(t('taxonomy.typeCreated'));
            setIsCreateTypeDialogOpen(false);
            loadTypes();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('taxonomy.typeCreateFailed'));
        }
    };

    const handleUpdateType = async () => {
        if (!editingType) return;
        try {
            await adminProblemTypeApi.update(editingType.id, { full_name: typeFormData.full_name });
            toast.success(t('taxonomy.typeUpdated'));
            setIsEditTypeDialogOpen(false);
            loadTypes();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('taxonomy.typeUpdateFailed'));
        }
    };

    const handleDeleteType = async (id: number) => {
        try {
            await adminProblemTypeApi.delete(id);
            toast.success(t('taxonomy.typeDeleted'));
            setDeleteTypeConfirmId(null);
            loadTypes();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('taxonomy.typeDeleteFailed'));
        }
    };

    const groupTotalPages = Math.ceil(groupTotal / 20);
    const typeTotalPages = Math.ceil(typeTotal / 20);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">{t('taxonomy.title')}</h1>
                    <p className="text-muted-foreground mt-1">
                        {t('taxonomy.subtitle')}
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
                    {t('taxonomy.groupsTab', { count: groupTotal })}
                </Button>
                <Button
                    variant={activeTab === 'types' ? 'default' : 'ghost'}
                    onClick={() => setActiveTab('types')}
                    className="gap-2 rounded-t-lg"
                >
                    <Tag className="h-4 w-4" />
                    {t('taxonomy.typesTab', { count: typeTotal })}
                </Button>
            </div>

            {/* Problem Groups Tab */}
            {activeTab === 'groups' && (
                <div className="space-y-4">
                    <div className="flex justify-end">
                        <Button onClick={openCreateGroupDialog}>
                            <Plus className="h-4 w-4 mr-2" />
                            {t('taxonomy.addGroupButton')}
                        </Button>
                    </div>

                    <div className="border rounded-lg overflow-hidden">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>{t('taxonomy.colName')}</TableHead>
                                    <TableHead>{t('taxonomy.colFullName')}</TableHead>
                                    <TableHead className="text-right">{t('common.actions')}</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {groupLoading ? (
                                    <TableRow>
                                        <TableCell colSpan={3} className="text-center py-8">
                                            {t('common.loading')}
                                        </TableCell>
                                    </TableRow>
                                ) : groups.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={3} className="text-center py-8 text-muted-foreground">
                                            {t('taxonomy.noGroupsFound')}
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
                            {t('common.showingRange', { from: (groupPage - 1) * 20 + 1, to: Math.min(groupPage * 20, groupTotal), total: groupTotal })}
                        </div>
                        <div className="flex gap-2">
                            <Button
                                variant="outline"
                                size="sm"
                                disabled={groupPage === 1}
                                onClick={() => setGroupPage(groupPage - 1)}
                            >
                                {t('common.previous')}
                            </Button>
                            <Button
                                variant="outline"
                                size="sm"
                                disabled={groupPage >= groupTotalPages}
                                onClick={() => setGroupPage(groupPage + 1)}
                            >
                                {t('common.next')}
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
                            {t('taxonomy.addTypeButton')}
                        </Button>
                    </div>

                    <div className="border rounded-lg overflow-hidden">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>{t('taxonomy.colName')}</TableHead>
                                    <TableHead>{t('taxonomy.colFullName')}</TableHead>
                                    <TableHead className="text-right">{t('common.actions')}</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {typeLoading ? (
                                    <TableRow>
                                        <TableCell colSpan={3} className="text-center py-8">
                                            {t('common.loading')}
                                        </TableCell>
                                    </TableRow>
                                ) : types.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={3} className="text-center py-8 text-muted-foreground">
                                            {t('taxonomy.noTypesFound')}
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
                            {t('common.showingRange', { from: (typePage - 1) * 20 + 1, to: Math.min(typePage * 20, typeTotal), total: typeTotal })}
                        </div>
                        <div className="flex gap-2">
                            <Button
                                variant="outline"
                                size="sm"
                                disabled={typePage === 1}
                                onClick={() => setTypePage(typePage - 1)}
                            >
                                {t('common.previous')}
                            </Button>
                            <Button
                                variant="outline"
                                size="sm"
                                disabled={typePage >= typeTotalPages}
                                onClick={() => setTypePage(typePage + 1)}
                            >
                                {t('common.next')}
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
                            {editingGroup ? t('taxonomy.editGroupTitle') : t('taxonomy.createGroupTitle')}
                        </DialogTitle>
                        <DialogDescription>
                            {editingGroup ? t('taxonomy.editGroupDesc') : t('taxonomy.createGroupDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="group_name">{t('taxonomy.groupNameLabel')}</Label>
                            <Input
                                id="group_name"
                                value={groupFormData.name}
                                onChange={(e) => setGroupFormData({ ...groupFormData, name: e.target.value })}
                                placeholder={t('taxonomy.groupNamePlaceholder')}
                                disabled={editingGroup !== null}
                            />
                        </div>
                        <div className="grid gap-2">
                            <Label htmlFor="group_full_name">{t('taxonomy.groupFullNameLabel')}</Label>
                            <Input
                                id="group_full_name"
                                value={groupFormData.full_name}
                                onChange={(e) => setGroupFormData({ ...groupFormData, full_name: e.target.value })}
                                placeholder={t('taxonomy.groupFullNamePlaceholder')}
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateGroupDialogOpen(false);
                            setIsEditGroupDialogOpen(false);
                        }}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={editingGroup ? handleUpdateGroup : handleCreateGroup}>
                            {editingGroup ? t('common.saveChanges') : t('common.create')}
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
                            {editingType ? t('taxonomy.editTypeTitle') : t('taxonomy.createTypeTitle')}
                        </DialogTitle>
                        <DialogDescription>
                            {editingType ? t('taxonomy.editTypeDesc') : t('taxonomy.createTypeDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="type_name">{t('taxonomy.typeNameLabel')}</Label>
                            <Input
                                id="type_name"
                                value={typeFormData.name}
                                onChange={(e) => setTypeFormData({ ...typeFormData, name: e.target.value })}
                                placeholder={t('taxonomy.typeNamePlaceholder')}
                                disabled={editingType !== null}
                            />
                        </div>
                        <div className="grid gap-2">
                            <Label htmlFor="type_full_name">{t('taxonomy.typeFullNameLabel')}</Label>
                            <Input
                                id="type_full_name"
                                value={typeFormData.full_name}
                                onChange={(e) => setTypeFormData({ ...typeFormData, full_name: e.target.value })}
                                placeholder={t('taxonomy.typeFullNamePlaceholder')}
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateTypeDialogOpen(false);
                            setIsEditTypeDialogOpen(false);
                        }}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={editingType ? handleUpdateType : handleCreateType}>
                            {editingType ? t('common.saveChanges') : t('common.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmations */}
            <Dialog open={deleteGroupConfirmId !== null} onOpenChange={() => setDeleteGroupConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('taxonomy.deleteGroupTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('taxonomy.deleteGroupDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteGroupConfirmId(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteGroupConfirmId && handleDeleteGroup(deleteGroupConfirmId)}
                        >
                            {t('common.delete')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={deleteTypeConfirmId !== null} onOpenChange={() => setDeleteTypeConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('taxonomy.deleteTypeTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('taxonomy.deleteTypeDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteTypeConfirmId(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteTypeConfirmId && handleDeleteType(deleteTypeConfirmId)}
                        >
                            {t('common.delete')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
