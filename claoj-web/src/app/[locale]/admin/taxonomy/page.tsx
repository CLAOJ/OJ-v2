'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { adminProblemGroupApi, adminProblemTypeApi } from '@/lib/adminApi';
import { searchProblems } from '@/lib/adminSearch';
import { EntityPicker, EntityOption } from '@/components/admin/EntityPicker';
import { AdminProblemGroup, AdminProblemType, AdminProblemGroupCreateRequest, AdminProblemTypeCreateRequest } from '@/types';
import { Link } from '@/navigation';
import { useDebounce } from '@/hooks/useDebounce';
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
import type { ApiError } from '@/lib/api';

type TabType = 'groups' | 'types';

export default function ProblemTaxonomyAdminPage() {
    const t = useTranslations('Admin');
    const [activeTab, setActiveTab] = useState<TabType>('groups');

    // Problem Groups state
    const [groups, setGroups] = useState<AdminProblemGroup[]>([]);
    const [groupTotal, setGroupTotal] = useState(0);
    const [groupPage, setGroupPage] = useState(1);
    const [groupSearch, setGroupSearch] = useState('');
    const [groupLoading, setGroupLoading] = useState(false);
    const [isCreateGroupDialogOpen, setIsCreateGroupDialogOpen] = useState(false);
    const [editingGroup, setEditingGroup] = useState<AdminProblemGroup | null>(null);
    const [editingGroupProblemCount, setEditingGroupProblemCount] = useState<number | null>(null);
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
    const [typeSearch, setTypeSearch] = useState('');
    const [typeLoading, setTypeLoading] = useState(false);
    const [isCreateTypeDialogOpen, setIsCreateTypeDialogOpen] = useState(false);
    const [editingType, setEditingType] = useState<AdminProblemType | null>(null);
    const [typeProblemOptions, setTypeProblemOptions] = useState<EntityOption[]>([]);
    const [isEditTypeDialogOpen, setIsEditTypeDialogOpen] = useState(false);
    const [deleteTypeConfirmId, setDeleteTypeConfirmId] = useState<number | null>(null);
    const [typeFormData, setTypeFormData] = useState<AdminProblemTypeCreateRequest>({
        name: '',
        full_name: '',
        problem_ids: [],
    });

    const groupSearchDebounced = useDebounce(groupSearch.trim(), 300);
    const typeSearchDebounced = useDebounce(typeSearch.trim(), 300);

    const loadGroups = async () => {
        setGroupLoading(true);
        try {
            const response = await adminProblemGroupApi.list(groupPage, 20, groupSearchDebounced);
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
            const response = await adminProblemTypeApi.list(typePage, 20, typeSearchDebounced);
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
    }, [activeTab, groupPage, typePage, groupSearchDebounced, typeSearchDebounced]);

    // Group handlers
    const openCreateGroupDialog = () => {
        setGroupFormData({ name: '', full_name: '' });
        setEditingGroup(null);
        setEditingGroupProblemCount(null);
        setIsCreateGroupDialogOpen(true);
    };

    const openEditGroupDialog = async (group: AdminProblemGroup) => {
        try {
            const response = await adminProblemGroupApi.detail(group.id);
            setGroupFormData({
                name: response.data.name,
                full_name: response.data.full_name,
            });
            setEditingGroupProblemCount(response.data.problem_count ?? 0);
            setEditingGroup(group);
            setIsEditGroupDialogOpen(true);
        } catch {
            toast.error(t('taxonomy.loadDetailFailed'));
        }
    };

    const handleCreateGroup = async () => {
        try {
            await adminProblemGroupApi.create(groupFormData);
            toast.success(t('taxonomy.groupCreated'));
            setIsCreateGroupDialogOpen(false);
            loadGroups();
        } catch (caught) {
            const error = caught as ApiError;
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
        } catch (caught) {
            const error = caught as ApiError;
            toast.error(error.response?.data?.error || t('taxonomy.groupUpdateFailed'));
        }
    };

    const handleDeleteGroup = async (id: number) => {
        try {
            await adminProblemGroupApi.delete(id);
            toast.success(t('taxonomy.groupDeleted'));
            setDeleteGroupConfirmId(null);
            loadGroups();
        } catch (caught) {
            const error = caught as ApiError;
            toast.error(error.response?.data?.error || t('taxonomy.groupDeleteFailed'));
        }
    };

    // Type handlers
    const openCreateTypeDialog = () => {
        setTypeFormData({ name: '', full_name: '', problem_ids: [] });
        setTypeProblemOptions([]);
        setEditingType(null);
        setIsCreateTypeDialogOpen(true);
    };

    const openEditTypeDialog = async (ptype: AdminProblemType) => {
        try {
            const response = await adminProblemTypeApi.detail(ptype.id);
            setTypeFormData({
                name: response.data.name,
                full_name: response.data.full_name,
                problem_ids: response.data.problem_ids ?? [],
            });
            setTypeProblemOptions((response.data.problems ?? []).map((p) => ({ id: p.id, label: p.code, detail: p.name })));
            setEditingType(ptype);
            setIsEditTypeDialogOpen(true);
        } catch {
            // Never open with an unknown selection: saving would clear the type's problems.
            toast.error(t('taxonomy.loadDetailFailed'));
        }
    };

    const handleCreateType = async () => {
        try {
            await adminProblemTypeApi.create(typeFormData);
            toast.success(t('taxonomy.typeCreated'));
            setIsCreateTypeDialogOpen(false);
            loadTypes();
        } catch (caught) {
            const error = caught as ApiError;
            toast.error(error.response?.data?.error || t('taxonomy.typeCreateFailed'));
        }
    };

    const handleUpdateType = async () => {
        if (!editingType) return;
        try {
            await adminProblemTypeApi.update(editingType.id, {
                full_name: typeFormData.full_name,
                problem_ids: typeFormData.problem_ids,
            });
            toast.success(t('taxonomy.typeUpdated'));
            setIsEditTypeDialogOpen(false);
            loadTypes();
        } catch (caught) {
            const error = caught as ApiError;
            toast.error(error.response?.data?.error || t('taxonomy.typeUpdateFailed'));
        }
    };

    const handleDeleteType = async (id: number) => {
        try {
            await adminProblemTypeApi.delete(id);
            toast.success(t('taxonomy.typeDeleted'));
            setDeleteTypeConfirmId(null);
            loadTypes();
        } catch (caught) {
            const error = caught as ApiError;
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
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <div className="relative w-full sm:w-72">
                            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                            <Input
                                value={groupSearch}
                                onChange={(e) => {
                                    setGroupSearch(e.target.value);
                                    setGroupPage(1);
                                }}
                                placeholder={t('taxonomy.searchGroupsPlaceholder')}
                                className="pl-9"
                                aria-label={t('taxonomy.searchGroupsPlaceholder')}
                            />
                        </div>
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
                                                        title={t('common.edit')}
                                                        aria-label={t('common.edit')}
                                                    >
                                                        <Edit className="h-4 w-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => setDeleteGroupConfirmId(group.id)}
                                                        className="text-destructive hover:text-destructive"
                                                        title={t('common.delete')}
                                                        aria-label={t('common.delete')}
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
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <div className="relative w-full sm:w-72">
                            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                            <Input
                                value={typeSearch}
                                onChange={(e) => {
                                    setTypeSearch(e.target.value);
                                    setTypePage(1);
                                }}
                                placeholder={t('taxonomy.searchTypesPlaceholder')}
                                className="pl-9"
                                aria-label={t('taxonomy.searchTypesPlaceholder')}
                            />
                        </div>
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
                                    <TableHead className="text-right">{t('taxonomy.colProblems')}</TableHead>
                                    <TableHead className="text-right">{t('common.actions')}</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {typeLoading ? (
                                    <TableRow>
                                        <TableCell colSpan={4} className="text-center py-8">
                                            {t('common.loading')}
                                        </TableCell>
                                    </TableRow>
                                ) : types.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={4} className="text-center py-8 text-muted-foreground">
                                            {t('taxonomy.noTypesFound')}
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    types.map((ptype) => (
                                        <TableRow key={ptype.id}>
                                            <TableCell className="font-mono font-bold">{ptype.name}</TableCell>
                                            <TableCell>{ptype.full_name}</TableCell>
                                            <TableCell className="text-right">
                                                <Badge variant="secondary">{ptype.problem_count ?? 0}</Badge>
                                            </TableCell>
                                            <TableCell className="text-right">
                                                <div className="flex justify-end gap-2">
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => openEditTypeDialog(ptype)}
                                                        title={t('common.edit')}
                                                        aria-label={t('common.edit')}
                                                    >
                                                        <Edit className="h-4 w-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => setDeleteTypeConfirmId(ptype.id)}
                                                        className="text-destructive hover:text-destructive"
                                                        title={t('common.delete')}
                                                        aria-label={t('common.delete')}
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
                        {editingGroup && editingGroupProblemCount !== null && (
                            <div className="text-sm text-muted-foreground">
                                {t('taxonomy.includedProblemsCount', { count: editingGroupProblemCount })}{' '}
                                <Link
                                    href={`/admin/problems?group_id=${editingGroup.id}`}
                                    className="text-primary hover:underline"
                                >
                                    {t('taxonomy.viewProblems')}
                                </Link>
                            </div>
                        )}
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
                        <EntityPicker
                            id="type_problems"
                            label={t('taxonomy.includedProblems')}
                            value={typeFormData.problem_ids ?? []}
                            onChange={(ids) => setTypeFormData({ ...typeFormData, problem_ids: ids })}
                            search={searchProblems}
                            queryKey="problems"
                            initialOptions={typeProblemOptions}
                        />
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
