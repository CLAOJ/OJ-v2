'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminRolesApi } from '@/lib/adminApi';
import { Role, Permission } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import {
    Shield,
    Plus,
    Edit,
    Trash2,
    Users,
    Key,
    X
} from 'lucide-react';
import { cn } from '@/lib/utils';

export default function AdminRolesPage() {
    const [selectedRole, setSelectedRole] = useState<Role | null>(null);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [showEditModal, setShowEditModal] = useState(false);

    const queryClient = useQueryClient();

    const { data, isLoading } = useQuery<{ data: Role[] }>({
        queryKey: ['admin-roles'],
        queryFn: async () => {
            const res = await adminRolesApi.list();
            return res.data;
        }
    });

    const { data: permissions } = useQuery({
        queryKey: ['admin-permissions'],
        queryFn: async () => {
            const res = await adminRolesApi.permissions();
            return res.data;
        }
    });

    const deleteMutation = useMutation({
        mutationFn: (id: number) => adminRolesApi.delete(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-roles'] });
            setSelectedRole(null);
        }
    });

    const roles = data?.data || [];

    const getCategoryColor = (category: string) => {
        const colors: Record<string, string> = {
            problems: 'bg-blue-500/10 text-blue-500 border-blue-500/20',
            contests: 'bg-green-500/10 text-green-500 border-green-500/20',
            submissions: 'bg-purple-500/10 text-purple-500 border-purple-500/20',
            users: 'bg-orange-500/10 text-orange-500 border-orange-500/20',
            organizations: 'bg-pink-500/10 text-pink-500 border-pink-500/20',
            comments: 'bg-cyan-500/10 text-cyan-500 border-cyan-500/20',
            tickets: 'bg-indigo-500/10 text-indigo-500 border-indigo-500/20',
            blogs: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20',
            system: 'bg-red-500/10 text-red-500 border-red-500/20',
        };
        return colors[category] || 'bg-gray-500/10 text-gray-500 border-gray-500/20';
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Shield className="text-primary" size={32} />
                        Roles & Permissions
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        Manage user roles and permissions
                    </p>
                </div>

                <button
                    onClick={() => setShowCreateModal(true)}
                    className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                >
                    <Plus size={18} />
                    Create Role
                </button>
            </div>

            {/* Roles Grid */}
            {isLoading ? (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {[1, 2, 3, 4, 5, 6].map(i => (
                        <Skeleton key={i} className="h-40 rounded-2xl" />
                    ))}
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {roles.map((role) => (
                        <div
                            key={role.id}
                            className={cn(
                                "bg-card rounded-2xl border p-6 cursor-pointer transition-all hover:shadow-lg hover:border-primary/30",
                                selectedRole?.id === role.id && "ring-2 ring-primary border-primary"
                            )}
                            onClick={() => setSelectedRole(role)}
                        >
                            <div className="flex items-start justify-between mb-4">
                                <div className="flex items-center gap-3">
                                    <div
                                        className="w-12 h-12 rounded-xl flex items-center justify-center font-bold text-lg"
                                        style={{ backgroundColor: `${role.color}20`, color: role.color }}
                                    >
                                        {role.display_name[0]}
                                    </div>
                                    <div>
                                        <h3 className="font-bold text-lg">{role.display_name}</h3>
                                        <p className="text-sm text-muted-foreground">{role.name}</p>
                                    </div>
                                </div>
                                {role.is_default && (
                                    <Badge variant="secondary" className="text-xs">Default</Badge>
                                )}
                            </div>

                            <p className="text-sm text-muted-foreground mb-4 line-clamp-2">
                                {role.description || 'No description'}
                            </p>

                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-1 text-sm text-muted-foreground">
                                    <Key size={14} />
                                    {role.permissions?.length || 0} permissions
                                </div>
                                <div className="flex gap-2">
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            setSelectedRole(role);
                                            setShowEditModal(true);
                                        }}
                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                    >
                                        <Edit size={16} />
                                    </button>
                                    {!role.is_default && (
                                        <button
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                if (confirm(`Delete role "${role.display_name}"?`)) {
                                                    deleteMutation.mutate(role.id);
                                                }
                                            }}
                                            className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                        >
                                            <Trash2 size={16} />
                                        </button>
                                    )}
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* Role Detail Panel */}
            {selectedRole && (
                <div className="bg-card rounded-2xl border p-6">
                    <div className="flex items-center justify-between mb-6">
                        <div className="flex items-center gap-3">
                            <div
                                className="w-12 h-12 rounded-xl flex items-center justify-center font-bold text-lg"
                                style={{ backgroundColor: `${selectedRole.color}20`, color: selectedRole.color }}
                            >
                                {selectedRole.display_name[0]}
                            </div>
                            <div>
                                <h2 className="text-xl font-bold">{selectedRole.display_name}</h2>
                                <p className="text-sm text-muted-foreground">{selectedRole.name}</p>
                            </div>
                        </div>
                        <button
                            onClick={() => setSelectedRole(null)}
                            className="p-2 hover:bg-muted rounded-lg transition-colors"
                        >
                            <X size={20} />
                        </button>
                    </div>

                    <p className="text-muted-foreground mb-6">{selectedRole.description}</p>

                    <h3 className="font-bold mb-4 flex items-center gap-2">
                        <Key size={18} />
                        Permissions ({selectedRole.permissions?.length || 0})
                    </h3>

                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                        {selectedRole.permissions?.map((perm) => (
                            <div
                                key={perm.id}
                                className={cn(
                                    "p-3 rounded-xl border",
                                    getCategoryColor(perm.category)
                                )}
                            >
                                <div className="font-medium text-sm">{perm.name}</div>
                                <div className="text-xs opacity-70 mt-1">{perm.code}</div>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* Create Modal */}
            {showCreateModal && (
                <RoleFormModal
                    permissions={permissions?.data || []}
                    onClose={() => setShowCreateModal(false)}
                    mode="create"
                />
            )}

            {/* Edit Modal */}
            {showEditModal && selectedRole && (
                <RoleFormModal
                    role={selectedRole}
                    permissions={permissions?.data || []}
                    onClose={() => {
                        setShowEditModal(false);
                        setSelectedRole(null);
                    }}
                    mode="edit"
                />
            )}
        </div>
    );
}

interface RoleFormModalProps {
    role?: Role;
    permissions: Permission[];
    onClose: () => void;
    mode: 'create' | 'edit';
}

function RoleFormModal({ role, permissions, onClose, mode }: RoleFormModalProps) {
    const queryClient = useQueryClient();
    const [formData, setFormData] = useState({
        name: role?.name || '',
        display_name: role?.display_name || '',
        description: role?.description || '',
        color: role?.color || '#6b7280',
        is_default: role?.is_default || false,
        permission_ids: role?.permissions?.map(p => p.id) || [],
    });

    const createMutation = useMutation({
        mutationFn: (data: any) => adminRolesApi.create(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-roles'] });
            onClose();
        }
    });

    const updateMutation = useMutation({
        mutationFn: (data: any) => adminRolesApi.update(role!.id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-roles'] });
            onClose();
        }
    });

    const handleSubmit = () => {
        if (mode === 'create') {
            createMutation.mutate(formData);
        } else {
            updateMutation.mutate(formData);
        }
    };

    const togglePermission = (permissionId: number) => {
        const current = formData.permission_ids;
        if (current.includes(permissionId)) {
            setFormData({ ...formData, permission_ids: current.filter(id => id !== permissionId) });
        } else {
            setFormData({ ...formData, permission_ids: [...current, permissionId] });
        }
    };

    const getCategoryColor = (category: string) => {
        const colors: Record<string, string> = {
            problems: 'bg-blue-500/10 text-blue-500 border-blue-500/20',
            contests: 'bg-green-500/10 text-green-500 border-green-500/20',
            submissions: 'bg-purple-500/10 text-purple-500 border-purple-500/20',
            users: 'bg-orange-500/10 text-orange-500 border-orange-500/20',
            organizations: 'bg-pink-500/10 text-pink-500 border-pink-500/20',
            comments: 'bg-cyan-500/10 text-cyan-500 border-cyan-500/20',
            tickets: 'bg-indigo-500/10 text-indigo-500 border-indigo-500/20',
            blogs: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20',
            system: 'bg-red-500/10 text-red-500 border-red-500/20',
        };
        return colors[category] || 'bg-gray-500/10 text-gray-500 border-gray-500/20';
    };

    // Group permissions by category
    const permissionsByCategory = permissions.reduce((acc, perm) => {
        if (!acc[perm.category]) acc[perm.category] = [];
        acc[perm.category].push(perm);
        return acc;
    }, {} as Record<string, Permission[]>);

    return (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
            <div className="bg-card rounded-2xl w-full max-w-2xl p-6 max-h-[90vh] overflow-y-auto">
                <div className="flex items-center justify-between mb-6">
                    <h2 className="text-xl font-bold">
                        {mode === 'create' ? 'Create Role' : 'Edit Role'}
                    </h2>
                    <button onClick={onClose} className="p-2 hover:bg-muted rounded-lg transition-colors">
                        <X size={20} />
                    </button>
                </div>

                <div className="space-y-4 mb-6">
                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            Role Name (slug)
                        </label>
                        <input
                            type="text"
                            className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                            placeholder="e.g., moderator"
                            disabled={mode === 'edit'}
                        />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="text-sm font-medium text-muted-foreground block mb-2">
                                Display Name
                            </label>
                            <input
                                type="text"
                                className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                value={formData.display_name}
                                onChange={(e) => setFormData({ ...formData, display_name: e.target.value })}
                                placeholder="e.g., Moderator"
                            />
                        </div>

                        <div>
                            <label className="text-sm font-medium text-muted-foreground block mb-2">
                                Color
                            </label>
                            <input
                                type="color"
                                className="w-full h-10 px-1 py-1 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                value={formData.color}
                                onChange={(e) => setFormData({ ...formData, color: e.target.value })}
                            />
                        </div>
                    </div>

                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            Description
                        </label>
                        <textarea
                            className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none min-h-[80px]"
                            value={formData.description}
                            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                            placeholder="Describe what this role does..."
                        />
                    </div>

                    <label className="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            className="rounded w-5 h-5"
                            checked={formData.is_default}
                            onChange={(e) => setFormData({ ...formData, is_default: e.target.checked })}
                        />
                        <span className="text-sm font-medium">Default role (assigned to new users)</span>
                    </label>
                </div>

                <h3 className="font-bold mb-4">Permissions</h3>
                <div className="space-y-4 mb-6 max-h-64 overflow-y-auto">
                    {Object.entries(permissionsByCategory).map(([category, perms]) => (
                        <div key={category}>
                            <h4 className="text-sm font-medium text-muted-foreground uppercase mb-2">
                                {category}
                            </h4>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                                {perms.map((perm) => (
                                    <label
                                        key={perm.id}
                                        className={cn(
                                            "flex items-center gap-3 p-3 rounded-xl border cursor-pointer transition-colors",
                                            formData.permission_ids.includes(perm.id)
                                                ? getCategoryColor(perm.category)
                                                : "hover:bg-muted/30"
                                        )}
                                    >
                                        <input
                                            type="checkbox"
                                            className="rounded w-5 h-5"
                                            checked={formData.permission_ids.includes(perm.id)}
                                            onChange={() => togglePermission(perm.id)}
                                        />
                                        <div>
                                            <div className="font-medium text-sm">{perm.name}</div>
                                            <div className="text-xs opacity-70">{perm.code}</div>
                                        </div>
                                    </label>
                                ))}
                            </div>
                        </div>
                    ))}
                </div>

                <div className="flex justify-end gap-3">
                    <button
                        type="button"
                        onClick={onClose}
                        className="px-4 py-2 rounded-xl hover:bg-muted transition-colors"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={handleSubmit}
                        disabled={createMutation.isPending || updateMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors disabled:opacity-50"
                    >
                        {mode === 'create' ? 'Create Role' : 'Save Changes'}
                    </button>
                </div>
            </div>
        </div>
    );
}
