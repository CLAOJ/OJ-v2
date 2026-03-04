'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { AdminOrganization } from '@/types';
import { adminOrganizationApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import OrganizationEditModal from '@/components/admin/OrganizationEditModal';
import {
    Search,
    Database,
    Plus,
    Users,
    Edit
} from 'lucide-react';

export default function AdminOrganizationsPage() {
    const [search, setSearch] = useState('');
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [editingOrg, setEditingOrg] = useState<any>(null);

    const queryClient = useQueryClient();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-organizations', search],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminOrganization[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/organizations?search=${search}`);
            return res.data;
        }
    });

    const createMutation = useMutation({
        mutationFn: (data: any) => adminOrganizationApi.create(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-organizations'] });
        }
    });

    const [newOrg, setNewOrg] = useState({ name: '', slug: '', short_name: '' });

    const organizations = data?.data || [];

    const filteredOrgs = organizations.filter(o =>
        o.name.toLowerCase().includes(search.toLowerCase()) ||
        o.short_name.toLowerCase().includes(search.toLowerCase())
    );

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Database className="text-primary" size={32} />
                        Organizations
                    </h1>
                    <p className="text-muted-foreground mt-1">Manage contest organizations</p>
                </div>

                <div className="flex gap-3">
                    <div className="relative w-full md:w-80">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder="Search organizations..."
                            className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                    <button
                        onClick={() => setShowCreateModal(true)}
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                    >
                        <Plus size={18} /> Create
                    </button>
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-20 rounded-2xl" />)}
                </div>
            ) : (
                <div className="grid gap-4">
                    {filteredOrgs.length === 0 ? (
                        <div className="text-center py-12 rounded-2xl border border-dashed bg-muted/30">
                            <Database size={48} className="mx-auto text-muted-foreground opacity-20" />
                            <p className="text-muted-foreground mt-4">No organizations found</p>
                        </div>
                    ) : (
                        filteredOrgs.map((org) => (
                            <div key={org.id} className="bg-card rounded-2xl p-6 border hover:border-primary/30 hover:shadow-lg transition-all">
                                <div className="flex items-center justify-between">
                                    <Link href={`/organization/${org.id}`} className="flex items-center gap-4 flex-1">
                                        <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center text-primary font-bold text-lg">
                                            {org.name[0].toUpperCase()}
                                        </div>
                                        <div>
                                            <h3 className="font-bold text-lg hover:text-primary transition-colors">{org.name}</h3>
                                            <div className="text-sm text-muted-foreground mt-1 flex items-center gap-3">
                                                <span className="font-mono text-xs">{org.slug}</span>
                                                <span className="text-muted-foreground">|</span>
                                                <div className="flex items-center gap-1">
                                                    <Users size={14} />
                                                    {org.member_count} members
                                                </div>
                                            </div>
                                        </div>
                                    </Link>
                                    <div className="flex items-center gap-2">
                                        {org.is_open && (
                                            <Badge variant="success" className="text-xs">Open</Badge>
                                        )}
                                        {org.is_unlisted && (
                                            <Badge variant="secondary" className="text-xs">Unlisted</Badge>
                                        )}
                                        <button
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                setEditingOrg(org);
                                            }}
                                            className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                            title="Edit organization"
                                        >
                                            <Edit size={18} />
                                        </button>
                                    </div>
                                </div>
                            </div>
                        ))
                    )}
                </div>
            )}

            {/* Create Modal */}
            {showCreateModal && (
                <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
                    <div className="bg-card rounded-2xl w-full max-w-md p-6">
                        <h2 className="text-xl font-bold mb-4">Create Organization</h2>

                        <div className="space-y-4">
                            <div>
                                <label className="text-sm font-medium mb-2 block">Name</label>
                                <input
                                    type="text"
                                    className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    value={newOrg.name}
                                    onChange={(e) => setNewOrg({ ...newOrg, name: e.target.value })}
                                />
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="text-sm font-medium mb-2 block">Slug</label>
                                    <input
                                        type="text"
                                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                        value={newOrg.slug}
                                        onChange={(e) => setNewOrg({ ...newOrg, slug: e.target.value })}
                                    />
                                </div>
                                <div>
                                    <label className="text-sm font-medium mb-2 block">Short Name</label>
                                    <input
                                        type="text"
                                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                        value={newOrg.short_name}
                                        onChange={(e) => setNewOrg({ ...newOrg, short_name: e.target.value })}
                                    />
                                </div>
                            </div>
                        </div>

                        <div className="flex justify-end gap-3 mt-6">
                            <button
                                onClick={() => setShowCreateModal(false)}
                                className="px-4 py-2 rounded-xl hover:bg-muted transition-colors"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={() => {
                                    createMutation.mutate(newOrg);
                                    setShowCreateModal(false);
                                    setNewOrg({ name: '', slug: '', short_name: '' });
                                }}
                                disabled={createMutation.isPending}
                                className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors"
                            >
                                Create
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Edit Modal */}
            {editingOrg && (
                <OrganizationEditModal
                    organization={editingOrg}
                    onClose={() => setEditingOrg(null)}
                />
            )}
        </div>
    );
}
