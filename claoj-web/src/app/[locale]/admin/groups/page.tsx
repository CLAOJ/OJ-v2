'use client';

import { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { adminGroupsApi, GroupCreateRequest, GroupUpdateRequest } from '@/lib/adminApi';
import api from '@/lib/api';
import { Group, GroupDetail, PermissionInfo, AdminUser } from '@/types';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Skeleton } from '@/components/ui/Skeleton';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from '@/components/ui/Dialog';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { useDebounce } from '@/hooks/useDebounce';
import { cn } from '@/lib/utils';
import { Shield, Plus, Edit, Trash2, Users, Key, X, UserPlus, AlertCircle } from 'lucide-react';

type Translate = (key: string, values?: Record<string, string | number>) => string;
type PermissionsByAppModel = Record<string, Record<string, PermissionInfo[]>>;

function groupPermissions(permissions: PermissionInfo[]): PermissionsByAppModel {
    const result: PermissionsByAppModel = {};
    for (const perm of permissions) {
        if (!result[perm.app_label]) result[perm.app_label] = {};
        if (!result[perm.app_label][perm.model]) result[perm.app_label][perm.model] = [];
        result[perm.app_label][perm.model].push(perm);
    }
    return result;
}

export default function AdminGroupsPage() {
    const t = useTranslations('AdminGroups') as unknown as Translate;
    const queryClient = useQueryClient();

    const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null);
    const [modalMode, setModalMode] = useState<'create' | 'edit' | null>(null);
    const [deleteTarget, setDeleteTarget] = useState<Group | null>(null);
    const [memberSearch, setMemberSearch] = useState('');
    const debouncedMemberSearch = useDebounce(memberSearch, 300);

    const { data: groupsData, isLoading: groupsLoading, isError: groupsError } = useQuery({
        queryKey: ['admin-groups'],
        queryFn: async () => (await adminGroupsApi.list()).data,
    });

    const { data: permissionsData, isError: permissionsError } = useQuery({
        queryKey: ['admin-permissions'],
        queryFn: async () => (await adminGroupsApi.permissions()).data,
    });

    const { data: selectedGroup, isLoading: detailLoading } = useQuery({
        queryKey: ['admin-group', selectedGroupId],
        queryFn: async () => (await adminGroupsApi.detail(selectedGroupId as number)).data,
        enabled: selectedGroupId !== null,
    });

    const groups = groupsData?.data || [];
    const permissions = permissionsData?.data || [];
    const permissionsByAppModel = useMemo(() => groupPermissions(permissions), [permissions]);

    const invalidateGroups = () => queryClient.invalidateQueries({ queryKey: ['admin-groups'] });
    const invalidateDetail = (id: number) => queryClient.invalidateQueries({ queryKey: ['admin-group', id] });

    const createMutation = useMutation({
        mutationFn: (data: GroupCreateRequest) => adminGroupsApi.create(data),
        onSuccess: () => {
            toast.success(t('saved'));
            invalidateGroups();
            setModalMode(null);
        },
        onError: () => toast.error(t('saveFailed')),
    });

    const updateMutation = useMutation({
        mutationFn: ({ id, data }: { id: number; data: GroupUpdateRequest }) => adminGroupsApi.update(id, data),
        onSuccess: (_res, variables) => {
            toast.success(t('saved'));
            invalidateGroups();
            invalidateDetail(variables.id);
            setModalMode(null);
        },
        onError: () => toast.error(t('saveFailed')),
    });

    const deleteMutation = useMutation({
        mutationFn: (id: number) => adminGroupsApi.delete(id),
        onSuccess: (_res, id) => {
            toast.success(t('deleted'));
            invalidateGroups();
            if (selectedGroupId === id) setSelectedGroupId(null);
            setDeleteTarget(null);
        },
        onError: () => toast.error(t('deleteFailed')),
    });

    const togglePermissionMutation = useMutation({
        mutationFn: ({ id, permission_ids }: { id: number; permission_ids: number[] }) =>
            adminGroupsApi.update(id, { permission_ids }),
        onSuccess: (_res, variables) => {
            invalidateGroups();
            invalidateDetail(variables.id);
        },
        onError: () => toast.error(t('saveFailed')),
    });

    const addMemberMutation = useMutation({
        mutationFn: ({ profileId, groupId }: { profileId: number; groupId: number }) =>
            adminGroupsApi.addUser(profileId, groupId),
        onSuccess: (_res, variables) => {
            invalidateGroups();
            invalidateDetail(variables.groupId);
            setMemberSearch('');
        },
        onError: () => toast.error(t('addMemberFailed')),
    });

    const removeMemberMutation = useMutation({
        mutationFn: ({ profileId, groupId }: { profileId: number; groupId: number }) =>
            adminGroupsApi.removeUser(profileId, groupId),
        onSuccess: (_res, variables) => {
            invalidateGroups();
            invalidateDetail(variables.groupId);
        },
        onError: () => toast.error(t('removeMemberFailed')),
    });

    const { data: userSearchResults, isFetching: userSearchLoading } = useQuery({
        queryKey: ['admin-user-search-for-group', debouncedMemberSearch],
        queryFn: async () => {
            const res = await api.get<{ data: AdminUser[] }>(
                `/admin/users?page=1&page_size=10&search=${encodeURIComponent(debouncedMemberSearch)}`
            );
            return res.data.data;
        },
        enabled: debouncedMemberSearch.trim().length >= 2 && selectedGroupId !== null,
    });

    const existingMemberIds = useMemo(
        () => new Set((selectedGroup?.users || []).map((u) => u.id)),
        [selectedGroup]
    );

    const availableSearchResults = (userSearchResults || []).filter((u) => !existingMemberIds.has(u.id));

    const handleTogglePermission = (permissionId: number) => {
        if (!selectedGroup) return;
        const current = selectedGroup.permission_ids;
        const next = current.includes(permissionId)
            ? current.filter((id) => id !== permissionId)
            : [...current, permissionId];
        togglePermissionMutation.mutate({ id: selectedGroup.id, permission_ids: next });
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Shield className="text-primary" size={32} />
                        {t('title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('subtitle')}</p>
                </div>

                <Button onClick={() => setModalMode('create')}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('createGroup')}
                </Button>
            </div>

            {/* Error State */}
            {(groupsError || permissionsError) && (
                <div className="flex items-center gap-2 rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                    <AlertCircle size={18} />
                    {t('loadFailed')}
                </div>
            )}

            {/* Groups Grid */}
            {groupsLoading ? (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {[1, 2, 3, 4, 5, 6].map((i) => (
                        <Skeleton key={i} className="h-32 rounded-2xl" />
                    ))}
                </div>
            ) : groupsError ? null : groups.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground">{t('noGroups')}</div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {groups.map((group) => (
                        <div
                            key={group.id}
                            className={cn(
                                'bg-card rounded-2xl border p-6 cursor-pointer transition-all hover:shadow-lg hover:border-primary/30',
                                selectedGroupId === group.id && 'ring-2 ring-primary border-primary'
                            )}
                            onClick={() => setSelectedGroupId(group.id)}
                        >
                            <div className="flex items-start justify-between mb-4">
                                <div className="flex items-center gap-3">
                                    <div className="w-12 h-12 rounded-xl flex items-center justify-center font-bold text-lg bg-primary/10 text-primary">
                                        {group.name[0]?.toUpperCase()}
                                    </div>
                                    <h3 className="font-bold text-lg">{group.name}</h3>
                                </div>
                                <div className="flex gap-1">
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            setSelectedGroupId(group.id);
                                            setModalMode('edit');
                                        }}
                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                        aria-label={t('editGroup')}
                                    >
                                        <Edit size={16} />
                                    </button>
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            setDeleteTarget(group);
                                        }}
                                        className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                        aria-label={t('deleteGroup')}
                                    >
                                        <Trash2 size={16} />
                                    </button>
                                </div>
                            </div>

                            <div className="flex items-center gap-4 text-sm text-muted-foreground">
                                <div className="flex items-center gap-1">
                                    <Users size={14} />
                                    {t('memberCount', { count: group.user_count })}
                                </div>
                                <div className="flex items-center gap-1">
                                    <Key size={14} />
                                    {t('permissionCount', { count: group.permission_count })}
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* Group Detail Panel */}
            {selectedGroupId !== null && (
                <div className="bg-card rounded-2xl border p-6">
                    {detailLoading || !selectedGroup ? (
                        <Skeleton className="h-64 rounded-xl" />
                    ) : (
                        <>
                            <div className="flex items-center justify-between mb-6">
                                <h2 className="text-xl font-bold">{selectedGroup.name}</h2>
                                <button
                                    onClick={() => setSelectedGroupId(null)}
                                    className="p-2 hover:bg-muted rounded-lg transition-colors"
                                    aria-label={t('close')}
                                >
                                    <X size={20} />
                                </button>
                            </div>

                            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                                {/* Members */}
                                <div>
                                    <h3 className="font-bold mb-4 flex items-center gap-2">
                                        <Users size={18} />
                                        {t('members')} ({selectedGroup.users.length})
                                    </h3>

                                    <div className="space-y-2 mb-4 max-h-64 overflow-y-auto">
                                        {selectedGroup.users.length === 0 ? (
                                            <p className="text-sm text-muted-foreground">{t('noMembers')}</p>
                                        ) : (
                                            selectedGroup.users.map((member) => (
                                                <div
                                                    key={member.id}
                                                    className="flex items-center justify-between p-3 rounded-xl border"
                                                >
                                                    <span className="text-sm font-medium">{member.username}</span>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        className="text-destructive hover:text-destructive"
                                                        disabled={removeMemberMutation.isPending}
                                                        onClick={() =>
                                                            removeMemberMutation.mutate({
                                                                profileId: member.id,
                                                                groupId: selectedGroup.id,
                                                            })
                                                        }
                                                    >
                                                        {t('removeMember')}
                                                    </Button>
                                                </div>
                                            ))
                                        )}
                                    </div>

                                    <div className="space-y-2">
                                        <Label htmlFor="add-member-search">{t('addMember')}</Label>
                                        <Input
                                            id="add-member-search"
                                            value={memberSearch}
                                            onChange={(e) => setMemberSearch(e.target.value)}
                                            placeholder={t('searchUsersPlaceholder')}
                                        />
                                        {debouncedMemberSearch.trim().length >= 2 && (
                                            <div className="border rounded-xl divide-y max-h-48 overflow-y-auto">
                                                {userSearchLoading ? (
                                                    <div className="p-3 text-sm text-muted-foreground">{t('loading')}</div>
                                                ) : availableSearchResults.length === 0 ? (
                                                    <div className="p-3 text-sm text-muted-foreground">{t('noUsersFound')}</div>
                                                ) : (
                                                    availableSearchResults.map((u) => (
                                                        <button
                                                            key={u.id}
                                                            onClick={() =>
                                                                addMemberMutation.mutate({
                                                                    profileId: u.id,
                                                                    groupId: selectedGroup.id,
                                                                })
                                                            }
                                                            disabled={addMemberMutation.isPending}
                                                            className="w-full flex items-center justify-between p-3 text-sm hover:bg-muted/50 transition-colors text-left"
                                                        >
                                                            <span>{u.username}</span>
                                                            <UserPlus size={14} className="text-primary" />
                                                        </button>
                                                    ))
                                                )}
                                            </div>
                                        )}
                                    </div>
                                </div>

                                {/* Permissions */}
                                <div>
                                    <h3 className="font-bold mb-4 flex items-center gap-2">
                                        <Key size={18} />
                                        {t('permissions')} ({selectedGroup.permission_ids.length})
                                    </h3>
                                    <PermissionPicker
                                        permissionsByAppModel={permissionsByAppModel}
                                        selectedIds={selectedGroup.permission_ids}
                                        onToggle={handleTogglePermission}
                                        emptyLabel={t('noPermissions')}
                                        disabled={togglePermissionMutation.isPending}
                                    />
                                </div>
                            </div>
                        </>
                    )}
                </div>
            )}

            {/* Create Modal */}
            {modalMode === 'create' && (
                <GroupFormModal
                    mode="create"
                    permissionsByAppModel={permissionsByAppModel}
                    isSaving={createMutation.isPending}
                    onClose={() => setModalMode(null)}
                    onSubmit={(data) => createMutation.mutate(data as GroupCreateRequest)}
                    t={t}
                />
            )}

            {/* Edit Modal */}
            {modalMode === 'edit' && selectedGroup && (
                <GroupFormModal
                    mode="edit"
                    group={selectedGroup}
                    permissionsByAppModel={permissionsByAppModel}
                    isSaving={updateMutation.isPending}
                    onClose={() => setModalMode(null)}
                    onSubmit={(data) => updateMutation.mutate({ id: selectedGroup.id, data })}
                    t={t}
                />
            )}

            {/* Delete Confirmation */}
            <ConfirmDialog
                isOpen={deleteTarget !== null}
                onClose={() => setDeleteTarget(null)}
                onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
                title={t('deleteGroup')}
                description={deleteTarget ? t('confirmDelete', { name: deleteTarget.name }) : ''}
                confirmText={t('deleteGroup')}
                cancelText={t('cancel')}
                variant="danger"
                isLoading={deleteMutation.isPending}
            />
        </div>
    );
}

interface PermissionPickerProps {
    permissionsByAppModel: PermissionsByAppModel;
    selectedIds: number[];
    onToggle: (id: number) => void;
    emptyLabel: string;
    disabled?: boolean;
}

function PermissionPicker({ permissionsByAppModel, selectedIds, onToggle, emptyLabel, disabled }: PermissionPickerProps) {
    const apps = Object.keys(permissionsByAppModel).sort();
    const selected = new Set(selectedIds);

    if (apps.length === 0) {
        return <p className="text-sm text-muted-foreground">{emptyLabel}</p>;
    }

    return (
        <div className="space-y-4 max-h-80 overflow-y-auto pr-1">
            {apps.map((appLabel) => {
                const models = Object.keys(permissionsByAppModel[appLabel]).sort();
                return (
                    <div key={appLabel}>
                        <h4 className="text-xs font-bold uppercase text-muted-foreground mb-2">{appLabel}</h4>
                        {models.map((model) => (
                            <div key={model} className="mb-3">
                                <p className="text-xs text-muted-foreground mb-1">{model}</p>
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                                    {permissionsByAppModel[appLabel][model].map((perm) => (
                                        <label
                                            key={perm.id}
                                            className={cn(
                                                'flex items-center gap-2 p-2 rounded-lg border cursor-pointer transition-colors text-sm',
                                                selected.has(perm.id) ? 'bg-primary/10 border-primary/30' : 'hover:bg-muted/30'
                                            )}
                                        >
                                            <input
                                                type="checkbox"
                                                className="rounded w-4 h-4"
                                                checked={selected.has(perm.id)}
                                                disabled={disabled}
                                                onChange={() => onToggle(perm.id)}
                                            />
                                            <span>{perm.name}</span>
                                        </label>
                                    ))}
                                </div>
                            </div>
                        ))}
                    </div>
                );
            })}
        </div>
    );
}

interface GroupFormModalProps {
    mode: 'create' | 'edit';
    group?: GroupDetail;
    permissionsByAppModel: PermissionsByAppModel;
    isSaving: boolean;
    onClose: () => void;
    onSubmit: (data: GroupCreateRequest | GroupUpdateRequest) => void;
    t: Translate;
}

function GroupFormModal({ mode, group, permissionsByAppModel, isSaving, onClose, onSubmit, t }: GroupFormModalProps) {
    const [name, setName] = useState(group?.name || '');
    const [permissionIds, setPermissionIds] = useState<number[]>(group?.permission_ids || []);

    const togglePermission = (id: number) => {
        setPermissionIds((current) =>
            current.includes(id) ? current.filter((p) => p !== id) : [...current, id]
        );
    };

    const handleSubmit = () => {
        if (!name.trim()) {
            toast.error(t('nameRequired'));
            return;
        }
        onSubmit({ name: name.trim(), permission_ids: permissionIds });
    };

    return (
        <Dialog open onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="max-w-2xl">
                <DialogHeader>
                    <DialogTitle>{mode === 'create' ? t('createGroup') : t('editGroup')}</DialogTitle>
                </DialogHeader>

                <div className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="group-name">{t('groupName')}</Label>
                        <Input
                            id="group-name"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            placeholder={t('groupNamePlaceholder')}
                        />
                    </div>

                    <div>
                        <Label className="mb-2 block">{t('permissions')}</Label>
                        <PermissionPicker
                            permissionsByAppModel={permissionsByAppModel}
                            selectedIds={permissionIds}
                            onToggle={togglePermission}
                            emptyLabel={t('noPermissions')}
                        />
                    </div>
                </div>

                <DialogFooter>
                    <Button variant="outline" onClick={onClose} disabled={isSaving}>
                        {t('cancel')}
                    </Button>
                    <Button onClick={handleSubmit} disabled={isSaving}>
                        {mode === 'create' ? t('create') : t('save')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
