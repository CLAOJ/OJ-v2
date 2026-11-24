'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useAuth } from '@/components/providers/AuthProvider';
import { adminUserApi, adminGroupsApi, adminErrorMessage } from '@/lib/adminApi';
import { toggleSort, type SortOrder, type UserSort } from '@/lib/listSort';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import type { AdminUser } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Switch } from '@/components/ui/Switch';
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from '@/components/ui/Dialog';
import { PaginationBar } from '@/components/ui/PaginationBar';
import { SortableHeader } from '@/components/ui/SortableHeader';
import { UserEditorModal } from '@/components/admin/UserEditorModal';
import {
    Search,
    Ban,
    UserCheck,
    Trash2,
    Edit,
    Shield,
    Users,
    Plus,
    KeyRound
} from 'lucide-react';

const FILTER_KEYS = ['search', 'role', 'status', 'group_id', 'sort', 'order'] as const;
const PAGE_SIZE = 50;
const DEFAULT_SORT: UserSort = 'date_joined';

export default function AdminUserPage() {
    const t = useTranslations('Admin');
    const { user: viewer } = useAuth();
    const [editorUserId, setEditorUserId] = useState<number | null>(null);
    const emptyCreate = { username: '', email: '', password: '', full_name: '', is_active: true };
    const [showCreate, setShowCreate] = useState(false);
    const [createForm, setCreateForm] = useState(emptyCreate);
    const [passwordTarget, setPasswordTarget] = useState<AdminUser | null>(null);
    const [newPassword, setNewPassword] = useState('');
    const [banTarget, setBanTarget] = useState<AdminUser | null>(null);
    const [banReason, setBanReason] = useState('');

    const queryClient = useQueryClient();

    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);
    const sort = (filters.sort ?? DEFAULT_SORT) as UserSort;
    const order: SortOrder = filters.order === 'asc' ? 'asc' : 'desc';
    const role = filters.role ?? '';
    const status = filters.status ?? '';

    const handleSort = (column: UserSort, firstOrder: SortOrder) => {
        const next = toggleSort({ sort, order }, column, firstOrder);
        setFilters({ sort: next.sort, order: next.order });
    };

    const { data: groupsData } = useQuery({
        queryKey: ['admin-groups'],
        queryFn: async () => (await adminGroupsApi.list()).data.data,
    });

    const { data, isLoading } = useQuery({
        queryKey: ['admin-users', page, search, role, status, filters.group_id, sort, order],
        queryFn: async () => (await adminUserApi.list(page, PAGE_SIZE, {
            search,
            staff: role === 'staff' ? 'true' : undefined,
            superuser: role === 'superuser' ? 'true' : undefined,
            active: status === 'active' ? 'true' : status === 'inactive' ? 'false' : undefined,
            group_id: filters.group_id ? Number(filters.group_id) : undefined,
            sort,
            order,
        })).data,
    });

    const invalidateUsers = () => queryClient.invalidateQueries({ queryKey: ['admin-users'] });

    const createMutation = useMutation({
        mutationFn: () => adminUserApi.create({
            username: createForm.username.trim(),
            email: createForm.email.trim(),
            password: createForm.password,
            full_name: createForm.full_name || undefined,
            is_active: createForm.is_active,
        }),
        onSuccess: () => {
            invalidateUsers();
            closeCreate();
        }
    });

    const passwordMutation = useMutation({
        mutationFn: ({ id, password }: { id: number; password: string }) =>
            adminUserApi.setPassword(id, password),
        onSuccess: () => {
            setPasswordTarget(null);
            setNewPassword('');
        }
    });

    const banMutation = useMutation({
        mutationFn: ({ id, reason }: { id: number; reason: string }) =>
            adminUserApi.ban(id, reason),
        onSuccess: () => {
            invalidateUsers();
            setBanTarget(null);
            setBanReason('');
        }
    });

    const unbanMutation = useMutation({
        mutationFn: (id: number) => adminUserApi.unban(id),
        onSuccess: invalidateUsers,
    });

    const deleteMutation = useMutation({
        mutationFn: (id: number) => adminUserApi.delete(id),
        onSuccess: invalidateUsers,
    });

    const closeCreate = () => {
        setShowCreate(false);
        setCreateForm(emptyCreate);
    };

    const users = data?.data || [];

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-semibold flex items-center gap-3">
                        <Users className="text-primary" size={32} />
                        {t('users.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('users.subtitle')}</p>
                </div>

                <div className="flex flex-wrap gap-2 w-full md:w-auto">
                    <Button onClick={() => { createMutation.reset(); setShowCreate(true); }}>
                        <Plus size={18} /> {t('users.createButton')}
                    </Button>
                    <div className="relative w-full md:w-72">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder={t('users.searchPlaceholder')}
                            className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={filters.search ?? ''}
                            onChange={(e) => setFilters({ search: e.target.value })}
                        />
                    </div>
                </div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
                <select
                    aria-label={t('users.roleFilter')}
                    className="h-10 px-3 rounded-xl bg-card border"
                    value={role}
                    onChange={(e) => setFilters({ role: e.target.value })}
                >
                    <option value="">{t('users.roleAll')}</option>
                    <option value="staff">{t('users.staff')}</option>
                    <option value="superuser">{t('users.superuser')}</option>
                </select>
                <select
                    aria-label={t('users.statusFilter')}
                    className="h-10 px-3 rounded-xl bg-card border"
                    value={status}
                    onChange={(e) => setFilters({ status: e.target.value })}
                >
                    <option value="">{t('users.statusAll')}</option>
                    <option value="active">{t('users.active')}</option>
                    <option value="inactive">{t('users.deactivated')}</option>
                </select>
                <select
                    aria-label={t('users.groupFilter')}
                    className="h-10 px-3 rounded-xl bg-card border"
                    value={filters.group_id ?? ''}
                    onChange={(e) => setFilters({ group_id: e.target.value })}
                >
                    <option value="">{t('users.groupAll')}</option>
                    {(groupsData ?? []).map(group => (
                        <option key={group.id} value={group.id}>{group.name}</option>
                    ))}
                </select>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-20 rounded-md" />)}
                </div>
            ) : (
                <div className="bg-card rounded-md border overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left text-sm">
                            <thead className="bg-muted/50 border-b">
                                <tr>
                                    <SortableHeader
                                        label={t('users.colUser')}
                                        active={sort === 'username'}
                                        order={order}
                                        onClick={() => handleSort('username', 'asc')}
                                    />
                                    <SortableHeader
                                        label={t('users.colPoints')}
                                        active={sort === 'points'}
                                        order={order}
                                        onClick={() => handleSort('points', 'desc')}
                                    />
                                    <SortableHeader
                                        label={t('users.colJoined')}
                                        active={sort === 'date_joined'}
                                        order={order}
                                        onClick={() => handleSort('date_joined', 'desc')}
                                    />
                                    <SortableHeader
                                        label={t('users.colLastLogin')}
                                        active={sort === 'last_login'}
                                        order={order}
                                        onClick={() => handleSort('last_login', 'desc')}
                                    />
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground">{t('users.colStatus')}</th>
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground">{t('users.colRole')}</th>
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground text-right">{t('common.actions')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {users.length === 0 ? (
                                    <tr>
                                        <td colSpan={7} className="px-4 py-12 text-center text-muted-foreground">
                                            {t('users.noUsersFound')}
                                        </td>
                                    </tr>
                                ) : (
                                    users.map((user) => (
                                        <tr key={user.id} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-4 py-2">
                                                <div className="flex items-center gap-2">
                                                    <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary text-sm font-medium shrink-0">
                                                        {user.username[0]?.toUpperCase()}
                                                    </div>
                                                    <div className="min-w-0">
                                                        <div className="font-medium truncate">{user.username}</div>
                                                        <div className="text-xs text-muted-foreground truncate">{user.email}</div>
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-4 py-2 whitespace-nowrap">
                                                <div className="text-sm">{Math.round(user.points)}</div>
                                                <div className="text-xs text-muted-foreground">
                                                    {t('users.problemsLabel', { count: user.problem_count })}
                                                </div>
                                            </td>
                                            <td className="px-4 py-2 text-xs text-muted-foreground whitespace-nowrap">
                                                {user.date_joined ? new Date(user.date_joined).toLocaleDateString() : t('common.notAvailable')}
                                            </td>
                                            <td className="px-4 py-2 text-xs text-muted-foreground whitespace-nowrap">
                                                {user.last_login ? new Date(user.last_login).toLocaleDateString() : t('users.never')}
                                            </td>
                                            <td className="px-4 py-2">
                                                <div className="flex flex-wrap items-center gap-1">
                                                    {user.is_active ? (
                                                        <Badge variant="success" className="flex items-center gap-1 text-[10px] py-0 px-1.5">
                                                            <UserCheck size={10} /> {t('users.active')}
                                                        </Badge>
                                                    ) : (
                                                        <Badge variant="destructive" className="flex items-center gap-1 text-[10px] py-0 px-1.5">
                                                            <Ban size={10} /> {t('users.deactivated')}
                                                        </Badge>
                                                    )}
                                                    {user.is_unlisted && (
                                                        <Badge variant="warning" className="text-[10px] py-0 px-1.5">{t('users.hidden')}</Badge>
                                                    )}
                                                    {user.is_muted && (
                                                        <Badge variant="destructive" className="text-[10px] py-0 px-1.5">{t('users.muted')}</Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-4 py-2">
                                                <div className="flex flex-wrap items-center gap-1">
                                                    {user.is_staff && (
                                                        <Badge variant="secondary" className="flex items-center gap-1 text-[10px] py-0 px-1.5">
                                                            <Shield size={10} /> {t('users.staff')}
                                                        </Badge>
                                                    )}
                                                    {user.is_super_user && (
                                                        <Badge variant="warning" className="flex items-center gap-1 text-[10px] py-0 px-1.5">
                                                            <Shield size={10} /> {t('users.superuser')}
                                                        </Badge>
                                                    )}
                                                    {(user.groups ?? []).map(group => (
                                                        <Badge key={group.id} variant="outline" className="text-[10px] py-0 px-1.5">
                                                            {group.name}
                                                        </Badge>
                                                    ))}
                                                </div>
                                            </td>
                                            <td className="px-4 py-2 text-right">
                                                <div className="flex justify-end gap-1">
                                                    <button
                                                        onClick={() => setEditorUserId(user.id)}
                                                        className="p-1.5 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('common.edit')}
                                                        aria-label={t('common.edit')}
                                                    >
                                                        <Edit size={16} />
                                                    </button>
                                                    <button
                                                        onClick={() => { passwordMutation.reset(); setNewPassword(''); setPasswordTarget(user); }}
                                                        className="p-1.5 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('users.setPasswordButton')}
                                                        aria-label={t('users.setPasswordButton')}
                                                    >
                                                        <KeyRound size={16} />
                                                    </button>
                                                    {!user.is_active ? (
                                                        <button
                                                            onClick={() => unbanMutation.mutate(user.id)}
                                                            disabled={unbanMutation.isPending}
                                                            className="p-1.5 hover:bg-success/10 text-success rounded-lg transition-colors"
                                                            title={t('users.unbanButton')}
                                                            aria-label={t('users.unbanButton')}
                                                        >
                                                            <UserCheck size={16} />
                                                        </button>
                                                    ) : (
                                                        <button
                                                            onClick={() => { banMutation.reset(); setBanReason(''); setBanTarget(user); }}
                                                            className="p-1.5 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                            title={t('users.banUserButton')}
                                                            aria-label={t('users.banUserButton')}
                                                        >
                                                            <Ban size={16} />
                                                        </button>
                                                    )}
                                                    <button
                                                        onClick={() => deleteMutation.mutate(user.id)}
                                                        disabled={deleteMutation.isPending}
                                                        className="p-1.5 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                        title={t('common.delete')}
                                                        aria-label={t('common.delete')}
                                                    >
                                                        <Trash2 size={16} />
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}

            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />

            {editorUserId !== null && (
                <UserEditorModal
                    userId={editorUserId}
                    viewer={{ username: viewer?.username, isSuperuser: !!viewer?.is_admin }}
                    onClose={() => setEditorUserId(null)}
                />
            )}

            <Dialog open={banTarget !== null} onOpenChange={(open) => { if (!open) setBanTarget(null); }} className="max-w-md">
                <DialogHeader>
                    <DialogTitle id="ban-user-title">{t('users.banUserTitle', { username: banTarget?.username ?? '' })}</DialogTitle>
                </DialogHeader>
                <form onSubmit={(e) => { e.preventDefault(); if (banTarget) banMutation.mutate({ id: banTarget.id, reason: banReason }); }}>
                    <DialogContent className="space-y-4">
                        <div>
                            <label htmlFor="ban-reason" className="text-sm font-medium text-muted-foreground block mb-2">{t('users.banReasonLabel')}</label>
                            <Input id="ban-reason" required value={banReason}
                                placeholder={t('users.banReasonPlaceholder')}
                                onChange={(e) => setBanReason(e.target.value)} />
                        </div>
                        {banMutation.isError && (
                            <p role="alert" className="text-sm text-destructive">
                                {adminErrorMessage(banMutation.error, t('users.saveError'))}
                            </p>
                        )}
                    </DialogContent>
                    <DialogFooter>
                        <Button type="button" variant="outline" onClick={() => setBanTarget(null)}>{t('common.cancel')}</Button>
                        <Button type="submit" variant="destructive" disabled={banMutation.isPending}>{t('users.banUserButton')}</Button>
                    </DialogFooter>
                </form>
            </Dialog>

            <Dialog open={showCreate} onOpenChange={(open) => { if (!open) closeCreate(); }} className="max-w-md">
                <DialogHeader>
                    <DialogTitle id="create-user-title">{t('users.createTitle')}</DialogTitle>
                </DialogHeader>
                <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(); }}>
                    <DialogContent className="space-y-4">
                        <div>
                            <label htmlFor="new-username" className="text-sm font-medium text-muted-foreground block mb-2">{t('users.usernameLabel')}</label>
                            <Input id="new-username" required autoComplete="off" value={createForm.username}
                                onChange={(e) => setCreateForm({ ...createForm, username: e.target.value })} />
                        </div>
                        <div>
                            <label htmlFor="new-email" className="text-sm font-medium text-muted-foreground block mb-2">{t('users.emailLabel')}</label>
                            <Input id="new-email" type="email" required autoComplete="off" value={createForm.email}
                                onChange={(e) => setCreateForm({ ...createForm, email: e.target.value })} />
                        </div>
                        <div>
                            <label htmlFor="new-password" className="text-sm font-medium text-muted-foreground block mb-2">{t('users.passwordLabel')}</label>
                            <Input id="new-password" type="password" required autoComplete="new-password" value={createForm.password}
                                onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })} />
                            <p className="text-xs text-muted-foreground mt-1">{t('users.passwordHint')}</p>
                        </div>
                        <div>
                            <label htmlFor="new-fullname" className="text-sm font-medium text-muted-foreground block mb-2">{t('users.fullNameLabel')}</label>
                            <Input id="new-fullname" value={createForm.full_name}
                                onChange={(e) => setCreateForm({ ...createForm, full_name: e.target.value })} />
                        </div>
                        <div className="flex items-center gap-3">
                            <Switch id="new-active" checked={createForm.is_active}
                                onCheckedChange={(v) => setCreateForm({ ...createForm, is_active: v })} />
                            <label htmlFor="new-active" className="text-sm font-medium">{t('users.activeLabel')}</label>
                        </div>
                        {createMutation.isError && (
                            <p role="alert" className="text-sm text-destructive">
                                {adminErrorMessage(createMutation.error, t('users.saveError'))}
                            </p>
                        )}
                    </DialogContent>
                    <DialogFooter>
                        <Button type="button" variant="outline" onClick={closeCreate}>{t('common.cancel')}</Button>
                        <Button type="submit" disabled={createMutation.isPending}>{t('users.createSubmit')}</Button>
                    </DialogFooter>
                </form>
            </Dialog>

            <Dialog open={passwordTarget !== null} onOpenChange={(open) => { if (!open) setPasswordTarget(null); }} className="max-w-md">
                <DialogHeader>
                    <DialogTitle id="set-password-title">{t('users.setPasswordTitle', { username: passwordTarget?.username ?? '' })}</DialogTitle>
                </DialogHeader>
                <form onSubmit={(e) => { e.preventDefault(); if (passwordTarget) passwordMutation.mutate({ id: passwordTarget.id, password: newPassword }); }}>
                    <DialogContent className="space-y-4">
                        <div>
                            <label htmlFor="set-password" className="text-sm font-medium text-muted-foreground block mb-2">{t('users.newPasswordLabel')}</label>
                            <Input id="set-password" type="password" required autoComplete="new-password" value={newPassword}
                                onChange={(e) => setNewPassword(e.target.value)} />
                            <p className="text-xs text-muted-foreground mt-1">{t('users.passwordHint')}</p>
                        </div>
                        {passwordMutation.isError && (
                            <p role="alert" className="text-sm text-destructive">
                                {adminErrorMessage(passwordMutation.error, t('users.saveError'))}
                            </p>
                        )}
                    </DialogContent>
                    <DialogFooter>
                        <Button type="button" variant="outline" onClick={() => setPasswordTarget(null)}>{t('common.cancel')}</Button>
                        <Button type="submit" disabled={passwordMutation.isPending}>{t('users.setPasswordSubmit')}</Button>
                    </DialogFooter>
                </form>
            </Dialog>
        </div>
    );
}
