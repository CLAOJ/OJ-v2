'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { OrganizationDetail, OrganizationMember } from '@/types';
import { Link } from '@/navigation';
import { useState } from 'react';
import {
    Building2,
    Users,
    UserPlus,
    UserCheck,
    Shield,
    Crown,
    ArrowLeft,
    Settings,
    Trash2,
    Loader2,
    Check,
    X,
    UserX,
    Bell
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { Badge } from '@/components/ui/Badge';
import { motion, AnimatePresence } from 'framer-motion';
import dayjs from 'dayjs';

export default function OrganizationManagePage() {
    const params = useParams();
    const router = useRouter();
    const id = params.id as string;
    const queryClient = useQueryClient();
    const [activeTab, setActiveTab] = useState<'requests' | 'members' | 'edit'>('requests');

    const { data: org, isLoading: orgLoading } = useQuery({
        queryKey: ['organization', id],
        queryFn: async () => {
            const res = await api.get<OrganizationDetail>(`/organization/${id}`);
            return res.data;
        }
    });

    const { data: requests, isLoading: requestsLoading } = useQuery({
        queryKey: ['organization-requests', id],
        queryFn: async () => {
            const res = await api.get<{ data: OrganizationRequest[] }>(`/organization/${id}/requests`);
            return res.data;
        },
        enabled: !!org?.admins?.some(admin => admin.id === org.user_id)
    });

    const kickMutation = useMutation({
        mutationFn: async (userId: number) => {
            const res = await api.post(`/organization/${id}/kick`, { user_id: userId });
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['organization', id] });
        }
    });

    const handleRequestMutation = useMutation({
        mutationFn: async ({ requestId, action }: { requestId: number; action: 'approve' | 'reject' }) => {
            const res = await api.post(`/organization/request/${requestId}/handle`, { action });
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['organization-requests', id] });
        }
    });

    const handleKick = (member: OrganizationMember) => {
        if (confirm(`Are you sure you want to kick ${member.username} from the organization?`)) {
            kickMutation.mutate(member.id);
        }
    };

    const handleRequest = (requestId: number, action: 'approve' | 'reject') => {
        handleRequestMutation.mutate({ requestId, action });
    };

    if (orgLoading) {
        return (
            <div className="flex items-center justify-center py-12">
                <Loader2 size={32} className="animate-spin text-muted-foreground" />
            </div>
        );
    }

    if (!org) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <Building2 size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-black mb-2">Organization Not Found</h2>
                <Link href="/organizations" className="text-primary hover:underline">
                    Back to Organizations
                </Link>
            </div>
        );
    }

    const isAdmin = org.admins?.some(admin => admin.id === org.user_id);

    if (!isAdmin) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <Shield size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-black mb-2">Access Denied</h2>
                <p className="text-muted-foreground">You don&apos;t have permission to manage this organization.</p>
                <Link href={`/organization/${id}`} className="text-primary hover:underline mt-4 block">
                    Back to Organization
                </Link>
            </div>
        );
    }

    return (
        <div className="max-w-5xl mx-auto space-y-6 py-8">
            {/* Header */}
            <div className="flex items-center gap-4 mb-6">
                <Link
                    href={`/organization/${id}`}
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                >
                    <ArrowLeft size={20} />
                </Link>
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Settings className="text-primary" size={32} />
                        Manage Organization
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {org.name}
                    </p>
                </div>
            </div>

            {/* Tabs */}
            <div className="flex gap-2 border-b">
                <button
                    onClick={() => setActiveTab('requests')}
                    className={cn(
                        "px-4 py-2 font-medium transition-colors border-b-2 flex items-center gap-2",
                        activeTab === 'requests'
                            ? "border-primary text-primary"
                            : "border-transparent text-muted-foreground hover:text-foreground"
                    )}
                >
                    <Bell size={18} />
                    Join Requests
                    {requests?.data.length ? (
                        <Badge className="ml-1 bg-primary text-primary-foreground">{requests.data.length}</Badge>
                    ) : null}
                </button>
                <button
                    onClick={() => setActiveTab('members')}
                    className={cn(
                        "px-4 py-2 font-medium transition-colors border-b-2 flex items-center gap-2",
                        activeTab === 'members'
                            ? "border-primary text-primary"
                            : "border-transparent text-muted-foreground hover:text-foreground"
                    )}
                >
                    <Users size={18} />
                    Members
                </button>
                <button
                    onClick={() => setActiveTab('edit')}
                    className={cn(
                        "px-4 py-2 font-medium transition-colors border-b-2 flex items-center gap-2",
                        activeTab === 'edit'
                            ? "border-primary text-primary"
                            : "border-transparent text-muted-foreground hover:text-foreground"
                    )}
                >
                    <Settings size={18} />
                    Edit Organization
                </button>
            </div>

            {/* Requests Tab */}
            {activeTab === 'requests' && (
                <div className="space-y-4">
                    {requestsLoading ? (
                        <div className="flex items-center justify-center py-12">
                            <Loader2 size={32} className="animate-spin text-muted-foreground" />
                        </div>
                    ) : requests?.data.length ? (
                        <div className="border rounded-xl overflow-hidden">
                            <div className="bg-muted/30 px-4 py-3 border-b font-medium">
                                Pending Join Requests ({requests.data.length})
                            </div>
                            <div className="divide-y">
                                {requests.data.map((request) => (
                                    <div key={request.id} className="flex items-center gap-4 p-4">
                                        <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary font-bold">
                                            {request.user.username[0]?.toUpperCase()}
                                        </div>
                                        <div className="flex-1">
                                            <div className="font-medium">{request.user.username}</div>
                                            <div className="text-sm text-muted-foreground">
                                                Requested {dayjs(request.created_at).fromNow()}
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <button
                                                onClick={() => handleRequest(request.id, 'approve')}
                                                disabled={handleRequestMutation.isPending}
                                                className="p-2 bg-green-500/10 text-green-500 rounded-lg hover:bg-green-500/20 transition-colors disabled:opacity-50"
                                                title="Approve"
                                            >
                                                <Check size={18} />
                                            </button>
                                            <button
                                                onClick={() => handleRequest(request.id, 'reject')}
                                                disabled={handleRequestMutation.isPending}
                                                className="p-2 bg-red-500/10 text-red-500 rounded-lg hover:bg-red-500/20 transition-colors disabled:opacity-50"
                                                title="Reject"
                                            >
                                                <X size={18} />
                                            </button>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    ) : (
                        <div className="p-8 text-center text-muted-foreground border rounded-xl">
                            <Bell size={48} className="mx-auto mb-4 opacity-20" />
                            <p className="font-medium">No pending requests</p>
                        </div>
                    )}
                </div>
            )}

            {/* Members Tab */}
            {activeTab === 'members' && (
                <div className="border rounded-xl overflow-hidden">
                    <div className="bg-muted/30 px-4 py-3 border-b font-medium">
                        Organization Members ({org.members?.length || 0})
                    </div>
                    <div className="divide-y">
                        {org.members?.map((member) => (
                            <div key={member.id} className="flex items-center gap-4 p-4 hover:bg-muted/30 transition-colors">
                                <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary font-bold">
                                    {member.username[0]?.toUpperCase()}
                                </div>
                                <div className="flex-1">
                                    <div className="flex items-center gap-2">
                                        <span className="font-medium">{member.username}</span>
                                        {member.role === 'admin' && (
                                            <Badge className="text-[10px] bg-amber-500/10 text-amber-500 border-amber-500/20">
                                                <Crown size={12} className="inline mr-1" /> Admin
                                            </Badge>
                                        )}
                                    </div>
                                    <div className="text-sm text-muted-foreground">
                                        Joined {dayjs(member.joined_at).fromNow()}
                                    </div>
                                </div>
                                {member.role !== 'admin' && (
                                    <button
                                        onClick={() => handleKick(member)}
                                        disabled={kickMutation.isPending}
                                        className="p-2 hover:bg-destructive/10 rounded-lg transition-colors text-destructive"
                                        title="Kick from organization"
                                    >
                                        <UserX size={18} />
                                    </button>
                                )}
                            </div>
                        ))}
                        {!org.members?.length && (
                            <div className="p-8 text-center text-muted-foreground">
                                <Users size={48} className="mx-auto mb-4 opacity-20" />
                                <p className="font-medium">No members yet</p>
                            </div>
                        )}
                    </div>
                </div>
            )}

            {/* Edit Tab */}
            {activeTab === 'edit' && (
                <EditOrganizationForm organization={org} onSuccess={() => router.push(`/organization/${id}`)} />
            )}
        </div>
    );
}

interface OrganizationRequest {
    id: number;
    user: { id: number; username: string };
    created_at: string;
}

function EditOrganizationForm({ organization, onSuccess }: { organization: OrganizationDetail; onSuccess: () => void }) {
    const queryClient = useQueryClient();
    const [formData, setFormData] = useState({
        name: organization.name,
        short_name: organization.short_name || '',
        about: organization.about || '',
        is_open: organization.is_open,
        is_unlisted: organization.is_unlisted,
        slots: organization.slots || 0,
    });
    const [isLoading, setIsLoading] = useState(false);

    const updateMutation = useMutation({
        mutationFn: async (data: typeof formData) => {
            const res = await api.patch(`/organization/${organization.id}`, data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['organization', organization.id] });
            onSuccess();
        }
    });

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsLoading(true);
        await updateMutation.mutateAsync(formData);
        setIsLoading(false);
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-6">
            <div className="bg-card rounded-2xl border p-6 space-y-4">
                <h3 className="text-lg font-bold">Basic Information</h3>

                <div>
                    <label className="block text-sm font-medium mb-2">Organization Name</label>
                    <input
                        type="text"
                        value={formData.name}
                        onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        required
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium mb-2">Short Name</label>
                    <input
                        type="text"
                        value={formData.short_name}
                        onChange={(e) => setFormData(prev => ({ ...prev, short_name: e.target.value }))}
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder="e.g., CLA"
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium mb-2">About</label>
                    <textarea
                        value={formData.about}
                        onChange={(e) => setFormData(prev => ({ ...prev, about: e.target.value }))}
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none min-h-[150px]"
                        placeholder="Describe your organization..."
                    />
                </div>
            </div>

            <div className="bg-card rounded-2xl border p-6 space-y-4">
                <h3 className="text-lg font-bold">Settings</h3>

                <div className="flex flex-wrap gap-4">
                    <label className="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            checked={formData.is_open}
                            onChange={(e) => setFormData(prev => ({ ...prev, is_open: e.target.checked }))}
                            className="rounded w-5 h-5"
                        />
                        <span className="text-sm font-medium">Open for joining (anyone can join)</span>
                    </label>

                    <label className="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            checked={formData.is_unlisted}
                            onChange={(e) => setFormData(prev => ({ ...prev, is_unlisted: e.target.checked }))}
                            className="rounded w-5 h-5"
                        />
                        <span className="text-sm font-medium">Unlisted (hide from public list)</span>
                    </label>
                </div>

                <div>
                    <label className="block text-sm font-medium mb-2">Member Slots (0 for unlimited)</label>
                    <input
                        type="number"
                        min="0"
                        value={formData.slots}
                        onChange={(e) => setFormData(prev => ({ ...prev, slots: parseInt(e.target.value) || 0 }))}
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    />
                </div>
            </div>

            <div className="flex justify-end gap-3">
                <Link
                    href={`/organization/${organization.id}`}
                    className="px-6 py-2.5 rounded-xl border hover:bg-muted transition-colors font-medium"
                >
                    Cancel
                </Link>
                <button
                    type="submit"
                    disabled={isLoading}
                    className="px-6 py-2.5 rounded-xl bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
                >
                    {isLoading && <Loader2 size={16} className="animate-spin" />}
                    Save Changes
                </button>
            </div>
        </form>
    );
}
