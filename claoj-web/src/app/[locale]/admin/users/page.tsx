'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { AdminUser, AdminUserUpdateRequest } from '@/types';
import { adminUserApi, adminErrorMessage } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Switch } from '@/components/ui/Switch';
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from '@/components/ui/Dialog';
import { Link } from '@/navigation';
import {
    Search,
    User,
    Ban,
    UserCheck,
    Trash2,
    Edit,
    RefreshCw,
    Settings,
    Shield,
    Clock,
    Users,
    Plus,
    KeyRound
} from 'lucide-react';

export default function AdminUserPage() {
    const t = useTranslations('Admin');
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');
    const [selectedUser, setSelectedUser] = useState<AdminUser | null>(null);
    const [showEditModal, setShowEditModal] = useState(false);
    const [editForm, setEditForm] = useState<AdminUserUpdateRequest>({});
    const emptyCreate = { username: '', email: '', password: '', full_name: '', is_active: true };
    const [showCreate, setShowCreate] = useState(false);
    const [createForm, setCreateForm] = useState(emptyCreate);
    const [passwordTarget, setPasswordTarget] = useState<AdminUser | null>(null);
    const [newPassword, setNewPassword] = useState('');

    const queryClient = useQueryClient();
    const router = useRouter();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-users', page, search],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminUser[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/users?page=${page}&page_size=50&search=${search}`);
            return res.data;
        }
    });

    const banMutation = useMutation({
        mutationFn: ({ id, reason }: { id: number; reason: string }) =>
            adminUserApi.ban(id, reason),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-users'] });
            setShowEditModal(false);
            setSelectedUser(null);
        }
    });

    const createMutation = useMutation({
        mutationFn: () => adminUserApi.create({
            username: createForm.username.trim(),
            email: createForm.email.trim(),
            password: createForm.password,
            full_name: createForm.full_name || undefined,
            is_active: createForm.is_active,
        }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-users'] });
            setShowCreate(false);
            setCreateForm(emptyCreate);
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

    const unbanMutation = useMutation({
        mutationFn: (id: number) => adminUserApi.unban(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-users'] });
            setShowEditModal(false);
            setSelectedUser(null);
        }
    });

    const deleteMutation = useMutation({
        mutationFn: (id: number) => adminUserApi.delete(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-users'] });
            setShowEditModal(false);
            setSelectedUser(null);
        }
    });

    const users = data?.data || [];

    const filteredUsers = users.filter(u =>
        u.username.toLowerCase().includes(search.toLowerCase()) ||
        u.email.toLowerCase().includes(search.toLowerCase())
    );

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

                <div className="flex gap-2 w-full md:w-auto">
                <Button onClick={() => { createMutation.reset(); setShowCreate(true); }}>
                    <Plus size={18} /> {t('users.createButton')}
                </Button>
                <div className="relative w-full md:w-80">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                    <input
                        type="text"
                        placeholder={t('users.searchPlaceholder')}
                        className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                    />
                </div>
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-24 rounded-md" />)}
                </div>
            ) : (
                <div className="bg-card rounded-md border overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left">
                            <thead className="bg-muted/50 border-b">
                                <tr>
                                    <th className="px-6 py-4 text-xs font-medium uppercase text-muted-foreground">{t('users.colUser')}</th>
                                    <th className="px-6 py-4 text-xs font-medium uppercase text-muted-foreground">{t('users.colStats')}</th>
                                    <th className="px-6 py-4 text-xs font-medium uppercase text-muted-foreground">{t('users.colStatus')}</th>
                                    <th className="px-6 py-4 text-xs font-medium uppercase text-muted-foreground">{t('users.colRole')}</th>
                                    <th className="px-6 py-4 text-xs font-medium uppercase text-muted-foreground text-right">{t('common.actions')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {filteredUsers.length === 0 ? (
                                    <tr>
                                        <td colSpan={5} className="px-6 py-12 text-center text-muted-foreground">
                                            {t('users.noUsersFound')}
                                        </td>
                                    </tr>
                                ) : (
                                    filteredUsers.map((user) => (
                                        <tr key={user.id} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-3">
                                                    <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary font-medium">
                                                        {user.username[0].toUpperCase()}
                                                    </div>
                                                    <div>
                                                        <div className="font-medium text-sm">{user.username}</div>
                                                        <div className="text-xs text-muted-foreground">{user.email}</div>
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col gap-1 text-sm">
                                                    <span className="text-muted-foreground">
                                                        {t('users.pointsLabel', { count: Math.round(user.points) })}
                                                    </span>
                                                    <span className="text-xs text-muted-foreground">
                                                        {t('users.problemsLabel', { count: user.problem_count })}
                                                    </span>
                                                    <span className="text-xs text-muted-foreground">
                                                        {user.date_joined && new Date(user.date_joined).toLocaleDateString()}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-2">
                                                    {user.is_active ? (
                                                        <Badge variant="success" className="flex items-center gap-1 text-xs">
                                                            <UserCheck size={12} /> {t('users.active')}
                                                        </Badge>
                                                    ) : (
                                                        <Badge variant="destructive" className="flex items-center gap-1 text-xs">
                                                            <Ban size={12} /> {t('users.deactivated')}
                                                        </Badge>
                                                    )}
                                                    {user.is_unlisted && (
                                                        <Badge variant="warning" className="text-xs">{t('users.hidden')}</Badge>
                                                    )}
                                                    {user.is_muted && (
                                                        <Badge variant="destructive" className="text-xs">{t('users.muted')}</Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-2">
                                                    {user.is_staff && (
                                                        <Badge variant="secondary" className="flex items-center gap-1 text-xs">
                                                            <Shield size={12} /> {t('users.staff')}
                                                        </Badge>
                                                    )}
                                                    {user.is_super_user && (
                                                        <Badge variant="warning" className="flex items-center gap-1 text-xs">
                                                            <Shield size={12} /> {t('users.superuser')}
                                                        </Badge>
                                                    )}
                                                    {user.display_rank && (
                                                        <span className="text-xs text-muted-foreground">
                                                            {user.display_rank}
                                                        </span>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    <button
                                                        onClick={() => {
                                                            setSelectedUser(user);
                                                            setEditForm({});
                                                            setShowEditModal(true);
                                                        }}
                                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                    >
                                                        <Edit size={18} />
                                                    </button>
                                                    <button
                                                        onClick={() => { passwordMutation.reset(); setNewPassword(''); setPasswordTarget(user); }}
                                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('users.setPasswordButton')}
                                                        aria-label={t('users.setPasswordButton')}
                                                    >
                                                        <KeyRound size={18} />
                                                    </button>
                                                    {!user.is_active ? (
                                                        <button
                                                            onClick={() => unbanMutation.mutate(user.id)}
                                                            disabled={unbanMutation.isPending}
                                                            className="p-2 hover:bg-success/10 text-success rounded-lg transition-colors"
                                                        >
                                                            <UserCheck size={18} />
                                                        </button>
                                                    ) : (
                                                        <button
                                                            onClick={() => {
                                                                setSelectedUser(user);
                                                                setEditForm({ ban_reason: '' });
                                                                setShowEditModal(true);
                                                            }}
                                                            className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                        >
                                                            <Ban size={18} />
                                                        </button>
                                                    )}
                                                    <button
                                                        onClick={() => deleteMutation.mutate(user.id)}
                                                        disabled={deleteMutation.isPending}
                                                        className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                    >
                                                        <Trash2 size={18} />
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>

                    {/* Pagination */}
                    {filteredUsers.length > 0 && (
                        <div className="flex items-center justify-between px-6 py-4 border-t bg-muted/30">
                            <div className="text-sm text-muted-foreground">
                                {t('users.showingCount', { shown: filteredUsers.length, total: data?.total || 0 })}
                            </div>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => setPage(p => Math.max(1, p - 1))}
                                    disabled={page === 1}
                                    className="px-3 py-1.5 rounded-lg bg-card border disabled:opacity-50 hover:bg-muted transition-colors"
                                >
                                    {t('common.previous')}
                                </button>
                                <div className="px-3 py-1.5 rounded-lg bg-primary text-primary-foreground font-medium">
                                    {page}
                                </div>
                                <button
                                    onClick={() => setPage(p => p + 1)}
                                    disabled={filteredUsers.length < 50}
                                    className="px-3 py-1.5 rounded-lg bg-card border disabled:opacity-50 hover:bg-muted transition-colors"
                                >
                                    {t('common.next')}
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            )}

            {/* User Actions Modal */}
            {selectedUser && showEditModal && (
                <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
                    <div className="bg-card rounded-md w-full max-w-md p-6">
                        <h2 className="text-xl font-semibold mb-4">{t('users.manageUserTitle', { username: selectedUser.username })}</h2>

                        <div className="space-y-4">
                            <div>
                                <label className="text-sm font-medium text-muted-foreground block mb-2">
                                    {t('users.banReasonLabel')}
                                </label>
                                <input
                                    type="text"
                                    className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    placeholder={t('users.banReasonPlaceholder')}
                                    value={editForm.ban_reason || ''}
                                    onChange={(e) => setEditForm({ ...editForm, ban_reason: e.target.value })}
                                />
                            </div>

                            <div>
                                <div>
                                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                                        {t('users.newStatusLabel')}
                                    </label>
                                    <select
                                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                        onChange={(e) => setEditForm({ ...editForm, is_active: e.target.value === 'active' })}
                                    >
                                        <option value="active">{t('users.optionActivate')}</option>
                                        <option value="inactive">{t('users.optionDeactivate')}</option>
                                        <option value="hidden">{t('users.optionHide')}</option>
                                        <option value="unhide">{t('users.optionUnhide')}</option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        <div className="flex justify-end gap-3 mt-6">
                            <button
                                onClick={() => setShowEditModal(false)}
                                className="px-4 py-2 rounded-xl hover:bg-muted transition-colors"
                            >
                                {t('common.cancel')}
                            </button>
                            {editForm.ban_reason ? (
                                <button
                                    onClick={() => {
                                        banMutation.mutate({ id: selectedUser.id, reason: editForm.ban_reason || '' });
                                    }}
                                    disabled={banMutation.isPending}
                                    className="px-4 py-2 rounded-xl bg-destructive text-white hover:bg-destructive/90 transition-colors"
                                >
                                    {t('users.banUserButton')}
                                </button>
                            ) : (
                                <button
                                    onClick={() => {
                                        if (editForm.is_active !== undefined) {
                                            // Handle activate/deactivate/hide logic
                                            adminUserApi.update(selectedUser.id, editForm).then(() => {
                                                setShowEditModal(false);
                                                refetch();
                                            });
                                        }
                                    }}
                                    disabled={banMutation.isPending || deleteMutation.isPending}
                                    className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors"
                                >
                                    {t('common.saveChanges')}
                                </button>
                            )}
                        </div>
                    </div>
                </div>
            )}

            <Dialog open={showCreate} onOpenChange={setShowCreate} className="max-w-md">
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
                        <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>{t('common.cancel')}</Button>
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
