'use client';

import { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { adminGroupsApi, GroupCreateRequest, GroupUpdateRequest } from '@/lib/adminApi';
import { searchUsers } from '@/lib/adminSearch';
import { Group, GroupDetail, PermissionInfo } from '@/types';
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
import { useAuth } from '@/components/providers/AuthProvider';
import { cn } from '@/lib/utils';
import { Shield, Plus, Edit, Trash2, Users, Key, X, UserPlus, AlertCircle } from 'lucide-react';
import { apiErrorMessage } from '@/lib/api';

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
        onError: (err) => toast.error(apiErrorMessage(err, t('saveFailed'))),
    });

    const updateMutation = useMutation({
        mutationFn: ({ id, data }: { id: number; data: GroupUpdateRequest }) => adminGroupsApi.update(id, data),
        onSuccess: (_res, variables) => {
            toast.success(t('saved'));
            invalidateGroups();
            invalidateDetail(variables.id);
            setModalMode(null);
        },
        onError: (err) => toast.error(apiErrorMessage(err, t('saveFailed'))),
    });

    const deleteMutation = useMutation({
        mutationFn: (id: number) => adminGroupsApi.delete(id),
        onSuccess: (_res, id) => {
            toast.success(t('deleted'));
            invalidateGroups();
            if (selectedGroupId === id) setSelectedGroupId(null);
            setDeleteTarget(null);
        },
        onError: (err) => toast.error(apiErrorMessage(err, t('deleteFailed'))),
    });

    const togglePermissionMutation = useMutation({
        mutationFn: ({ id, permission_ids }: { id: number; permission_ids: number[] }) =>
            adminGroupsApi.update(id, { permission_ids }),
        onSuccess: (_res, variables) => {
            invalidateGroups();
            invalidateDetail(variables.id);
        },
        onError: (err) => toast.error(apiErrorMessage(err, t('saveFailed'))),
    });

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
                    <h1 className="text-3xl font-semibold flex items-center gap-3">
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
                        <Skeleton key={i} className="h-32 rounded-md" />
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
                                'bg-card rounded-md border p-6 cursor-pointer transition-all hover:border-primary/30',
                                selectedGroupId === group.id && 'ring-2 ring-primary border-primary'
                            )}
                            onClick={() => setSelectedGroupId(group.id)}
                        >
                            <div className="flex items-start justify-between mb-4">
                                <div className="flex items-center gap-3">
                                    <div className="w-12 h-12 rounded-xl flex items-center justify-center font-semibold text-lg bg-primary/10 text-primary">
                                        {group.name[0]?.toUpperCase()}
                                    </div>
                                    <h3 className="font-semibold text-lg">{group.name}</h3>
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
                <div className="bg-card rounded-md border p-6">
                    {detailLoading || !selectedGroup ? (
                        <Skeleton className="h-64 rounded-xl" />
                    ) : (
                        <>
                            <div className="flex items-center justify-between mb-6">
                                <h2 className="text-xl font-semibold">{selectedGroup.name}</h2>
                                <button
                                    onClick={() => setSelectedGroupId(null)}
                                    className="p-2 hover:bg-muted rounded-lg transition-colors"
                                    aria-label={t('close')}
                                >
                                    <X size={20} />
                                </button>
                            </div>

                            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                                {/* Members (managed in the edit modal) */}
                                <div>
                                    <h3 className="font-semibold mb-4 flex items-center gap-2">
                                        <Users size={18} />
                                        {t('members')} ({selectedGroup.users.length})
                                    </h3>

                                    <div className="space-y-2 max-h-72 overflow-y-auto">
                                        {selectedGroup.users.length === 0 ? (
                                            <p className="text-sm text-muted-foreground">{t('noMembers')}</p>
                                        ) : (
                                            selectedGroup.users.map((member) => (
                                                <div key={member.id} className="p-3 rounded-xl border text-sm font-medium">
                                                    {member.username}
                                                </div>
                                            ))
                                        )}
                                    </div>
                                </div>

                                {/* Permissions */}
                                <div>
                                    <h3 className="font-semibold mb-4 flex items-center gap-2">
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
                        <h4 className="text-xs font-medium text-muted-foreground mb-2">{appLabel}</h4>
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

                    {mode === 'edit' && group && (
                        <div className="space-y-2">
                            <p className="text-sm font-medium">{t('members')}</p>
                            <GroupMembersEditor groupId={group.id} t={t} />
                        </div>
                    )}

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

interface GroupMembersEditorProps {
    groupId: number;
    t: Translate;
}

/**
 * Immediate add/remove member editor for an existing group. Membership
 * changes hit the group-scoped endpoints and invalidate the group queries so
 * the card counts and the detail panel stay accurate.
 *
 * The search box only drives the add-candidate picker; the member list and
 * its count always reflect the group's real membership, so typing a name
 * never makes a populated group look empty.
 *
 * shortcut: the member list is capped at one page (100); larger groups need
 * paging or infinite scroll, which the endpoint already supports.
 */
function GroupMembersEditor({ groupId, t }: GroupMembersEditorProps) {
    const queryClient = useQueryClient();
    const { user: viewer } = useAuth();
    // The member add/remove endpoints are superuser-only; everyone else sees
    // the list read-only.
    const canManage = !!viewer?.is_admin;
    const [term, setTerm] = useState('');
    const debouncedTerm = useDebounce(term.trim(), 300);

    const { data, isLoading } = useQuery({
        queryKey: ['admin-group-members', groupId],
        queryFn: async () => (await adminGroupsApi.members(groupId, 1, 100)).data,
    });
    const members = data?.data || [];
    const total = data?.total ?? members.length;

    const invalidate = () => {
        queryClient.invalidateQueries({ queryKey: ['admin-group-members', groupId] });
        queryClient.invalidateQueries({ queryKey: ['admin-groups'] });
        queryClient.invalidateQueries({ queryKey: ['admin-group', groupId] });
    };

    const addMutation = useMutation({
        mutationFn: (profileId: number) => adminGroupsApi.addMember(groupId, profileId),
        onSuccess: () => {
            setTerm('');
            invalidate();
        },
        onError: (err) => toast.error(apiErrorMessage(err, t('addMemberFailed'))),
    });

    const removeMutation = useMutation({
        mutationFn: (profileId: number) => adminGroupsApi.removeMember(groupId, profileId),
        onSuccess: invalidate,
        onError: (err) => toast.error(apiErrorMessage(err, t('removeMemberFailed'))),
    });

    const { data: candidates = [], isFetching: searching } = useQuery({
        queryKey: ['admin-user-search-for-group', groupId, debouncedTerm],
        queryFn: () => searchUsers(debouncedTerm),
        enabled: debouncedTerm.length >= 2,
    });
    const memberIds = new Set(members.map((m) => m.id));
    const addable = candidates.filter((c) => !memberIds.has(c.id));

    return (
        <div className="space-y-2">
            {canManage && (
                <>
                    <Label htmlFor="group-member-search">{t('addMember')}</Label>
                    <Input
                        id="group-member-search"
                        value={term}
                        onChange={(e) => setTerm(e.target.value)}
                        placeholder={t('searchUsersPlaceholder')}
                    />
                </>
            )}

            {canManage && debouncedTerm.length >= 2 && (
                <div className="border rounded-xl divide-y max-h-40 overflow-y-auto">
                    {searching ? (
                        <div className="p-3 text-sm text-muted-foreground">{t('loading')}</div>
                    ) : addable.length === 0 ? (
                        <div className="p-3 text-sm text-muted-foreground">{t('noUsersFound')}</div>
                    ) : (
                        addable.map((u) => (
                            <button
                                key={u.id}
                                type="button"
                                onClick={() => addMutation.mutate(u.id)}
                                disabled={addMutation.isPending}
                                className="w-full flex items-center justify-between p-3 text-sm hover:bg-muted/50 transition-colors text-left"
                            >
                                <span>{u.label}</span>
                                <UserPlus size={14} className="text-primary" />
                            </button>
                        ))
                    )}
                </div>
            )}

            <div className="border rounded-xl divide-y max-h-56 overflow-y-auto">
                {isLoading ? (
                    <div className="p-3 text-sm text-muted-foreground">{t('loading')}</div>
                ) : total === 0 ? (
                    <div className="p-3 text-sm text-muted-foreground">{t('noMembers')}</div>
                ) : (
                    members.map((m) => (
                        <div key={m.id} className="flex items-center justify-between gap-3 p-3">
                            <div className="min-w-0">
                                <p className="text-sm font-medium truncate">{m.username}</p>
                                <p className="text-xs text-muted-foreground truncate">{m.email}</p>
                            </div>
                            {canManage && (
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    className="text-destructive hover:text-destructive"
                                    aria-label={t('removeMemberNamed', { name: m.username })}
                                    disabled={removeMutation.isPending}
                                    onClick={() => removeMutation.mutate(m.id)}
                                >
                                    {t('removeMember')}
                                </Button>
                            )}
                        </div>
                    ))
                )}
            </div>

            <p className="text-xs text-muted-foreground">
                {total > members.length
                    ? t('membersShowing', { shown: members.length, total })
                    : t('memberCount', { count: total })}
            </p>
            <p className="text-xs text-muted-foreground">{t('membersApplyImmediately')}</p>
        </div>
    );
}
