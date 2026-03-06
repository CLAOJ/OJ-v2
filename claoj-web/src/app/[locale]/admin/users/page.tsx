'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import api from '@/lib/api';
import { AdminUser, AdminUserUpdateRequest } from '@/types';
import { adminUserApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
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
    Users
} from 'lucide-react';

export default function AdminUserPage() {
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');
    const [selectedUser, setSelectedUser] = useState<AdminUser | null>(null);
    const [showEditModal, setShowEditModal] = useState(false);
    const [editForm, setEditForm] = useState<AdminUserUpdateRequest>({});
    const [banDays, setBanDays] = useState(7);

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
        mutationFn: ({ id, reason, days }: { id: number; reason: string; days: number }) =>
            adminUserApi.ban(id, reason, days),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-users'] });
            setShowEditModal(false);
            setSelectedUser(null);
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
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Users className="text-primary" size={32} />
                        Users
                    </h1>
                    <p className="text-muted-foreground mt-1">Manage all users and their accounts</p>
                </div>

                <div className="relative w-full md:w-80">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                    <input
                        type="text"
                        placeholder="Search users..."
                        className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                    />
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-24 rounded-2xl" />)}
                </div>
            ) : (
                <div className="bg-card rounded-2xl border overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left">
                            <thead className="bg-muted/50 border-b">
                                <tr>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">User</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Stats</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Status</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Role</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground text-right">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {filteredUsers.length === 0 ? (
                                    <tr>
                                        <td colSpan={5} className="px-6 py-12 text-center text-muted-foreground">
                                            No users found
                                        </td>
                                    </tr>
                                ) : (
                                    filteredUsers.map((user) => (
                                        <tr key={user.id} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-3">
                                                    <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary font-bold">
                                                        {user.username[0].toUpperCase()}
                                                    </div>
                                                    <div>
                                                        <div className="font-bold text-sm">{user.username}</div>
                                                        <div className="text-xs text-muted-foreground">{user.email}</div>
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col gap-1 text-sm">
                                                    <span className="text-muted-foreground">
                                                        {Math.round(user.points)} Points
                                                    </span>
                                                    <span className="text-xs text-muted-foreground">
                                                        {user.problem_count} Problems
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
                                                            <UserCheck size={12} /> Active
                                                        </Badge>
                                                    ) : (
                                                        <Badge variant="destructive" className="flex items-center gap-1 text-xs">
                                                            <Ban size={12} /> Deactivated
                                                        </Badge>
                                                    )}
                                                    {user.is_unlisted && (
                                                        <Badge variant="warning" className="text-xs">Hidden</Badge>
                                                    )}
                                                    {user.is_muted && (
                                                        <Badge variant="destructive" className="text-xs">Muted</Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-2">
                                                    {user.is_staff && (
                                                        <Badge variant="secondary" className="flex items-center gap-1 text-xs">
                                                            <Shield size={12} /> Staff
                                                        </Badge>
                                                    )}
                                                    {user.is_super_user && (
                                                        <Badge variant="warning" className="flex items-center gap-1 text-xs">
                                                            <Shield size={12} /> Superuser
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
                                                                setBanDays(7);
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
                                Showing {filteredUsers.length} of {data?.total || 0} users
                            </div>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => setPage(p => Math.max(1, p - 1))}
                                    disabled={page === 1}
                                    className="px-3 py-1.5 rounded-lg bg-card border disabled:opacity-50 hover:bg-muted transition-colors"
                                >
                                    Previous
                                </button>
                                <div className="px-3 py-1.5 rounded-lg bg-primary text-primary-foreground font-bold">
                                    {page}
                                </div>
                                <button
                                    onClick={() => setPage(p => p + 1)}
                                    disabled={filteredUsers.length < 50}
                                    className="px-3 py-1.5 rounded-lg bg-card border disabled:opacity-50 hover:bg-muted transition-colors"
                                >
                                    Next
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            )}

            {/* User Actions Modal */}
            {selectedUser && showEditModal && (
                <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
                    <div className="bg-card rounded-2xl w-full max-w-md p-6">
                        <h2 className="text-xl font-bold mb-4">Manage User: {selectedUser.username}</h2>

                        <div className="space-y-4">
                            <div>
                                <label className="text-sm font-medium text-muted-foreground block mb-2">
                                    Ban Reason
                                </label>
                                <input
                                    type="text"
                                    className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    placeholder="Enter ban reason..."
                                    value={editForm.ban_reason || ''}
                                    onChange={(e) => setEditForm({ ...editForm, ban_reason: e.target.value })}
                                />
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                                        Ban Duration (days)
                                    </label>
                                    <input
                                        type="number"
                                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                        value={banDays}
                                        onChange={(e) => setBanDays(Number(e.target.value))}
                                        min={1}
                                        max={365}
                                    />
                                </div>
                                <div>
                                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                                        New Status
                                    </label>
                                    <select
                                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                        onChange={(e) => setEditForm({ ...editForm, is_active: e.target.value === 'active' })}
                                    >
                                        <option value="active">Activate</option>
                                        <option value="inactive">Deactivate</option>
                                        <option value="hidden">Hide</option>
                                        <option value="unhide">Unhide</option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        <div className="flex justify-end gap-3 mt-6">
                            <button
                                onClick={() => setShowEditModal(false)}
                                className="px-4 py-2 rounded-xl hover:bg-muted transition-colors"
                            >
                                Cancel
                            </button>
                            {editForm.ban_reason ? (
                                <button
                                    onClick={() => {
                                        banMutation.mutate({ id: selectedUser.id, reason: editForm.ban_reason || '', days: banDays });
                                    }}
                                    disabled={banMutation.isPending}
                                    className="px-4 py-2 rounded-xl bg-destructive text-white hover:bg-destructive/90 transition-colors"
                                >
                                    Ban User
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
                                    Save Changes
                                </button>
                            )}
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
