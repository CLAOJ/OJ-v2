'use client';

import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useConfirm } from '@/components/providers/ConfirmProvider';
import api from '@/lib/api';
import { OrganizationDetail, OrganizationMember, OrganizationRequest } from '@/types';
import { Link } from '@/navigation';
import { useEffect, useState } from 'react';
import { parseLeadingId } from '@/utils/route';
import { parseOrgRequestTab, type OrgRequestTab } from '@/lib/organizationView';
import {
    Building2,
    Users,
    Shield,
    Crown,
    ArrowLeft,
    Settings,
    Loader2,
    Check,
    X,
    UserX,
    Bell
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { Badge } from '@/components/ui/Badge';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/Dialog';
import dayjs from 'dayjs';

export default function OrganizationManagePage() {
    const t = useTranslations('Organization');
    const confirm = useConfirm();
    const params = useParams();
    const router = useRouter();
    const searchParams = useSearchParams();
    const idParam = params.id as string;
    const id = parseLeadingId(idParam);
    const queryClient = useQueryClient();
    const [activeTab, setActiveTab] = useState<'requests' | 'members' | 'edit'>('requests');
    // v1 request tabs: pending / approved / rejected / log (approved+rejected).
    const [requestTab, setRequestTab] = useState<OrgRequestTab>(() => parseOrgRequestTab(searchParams.get('state')));
    const [selectedRequest, setSelectedRequest] = useState<OrganizationRequest | null>(null);

    // Back/forward changes the query string without remounting; follow it.
    useEffect(() => {
        setRequestTab(parseOrgRequestTab(searchParams.get('state')));
    }, [searchParams]);

    const selectRequestTab = (tab: OrgRequestTab) => {
        setRequestTab(tab);
        const next = new URLSearchParams(searchParams.toString());
        next.set('state', tab);
        router.replace(`?${next.toString()}`, { scroll: false });
    };

    const { data: org, isLoading: orgLoading } = useQuery({
        queryKey: ['organization', id],
        queryFn: async () => {
            const res = await api.get<OrganizationDetail>(`/organization/${id}`);
            return res.data;
        }
    });

    const { data: requests, isLoading: requestsLoading } = useQuery({
        queryKey: ['organization-requests', id, requestTab],
        queryFn: async () => {
            const res = await api.get<{ data: OrganizationRequest[] }>(`/organization/${id}/requests`, {
                params: { state: requestTab },
            });
            return res.data;
        },
        enabled: !!org?.viewer_is_admin
    });

    const kickMutation = useMutation({
        mutationFn: async (username: string) => {
            const res = await api.post(`/organization/${id}/kick`, { username });
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
            queryClient.invalidateQueries({ queryKey: ['organization', id] });
            setSelectedRequest(null);
        }
    });

    const handleKick = async (member: OrganizationMember) => {
        if (await confirm(t('kickConfirm', { username: member.username }), { variant: 'danger' })) {
            kickMutation.mutate(member.username);
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
                <h2 className="text-2xl font-semibold mb-2">{t('notFound')}</h2>
                <Link href="/organizations" className="text-primary hover:underline">
                    {t('backToOrganizations')}
                </Link>
            </div>
        );
    }

    const isAdmin = org.viewer_is_admin ?? false;

    if (!isAdmin) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <Shield size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-semibold mb-2">{t('accessDenied')}</h2>
                <p className="text-muted-foreground">{t('noManagePermission')}</p>
                <Link href={`/organization/${org.id}-${org.slug}`} className="text-primary hover:underline mt-4 block">
                    {t('backToOrganization')}
                </Link>
            </div>
        );
    }

    return (
        <div className="max-w-5xl mx-auto space-y-6 py-8">
            {/* Header */}
            <div className="flex items-center gap-4 mb-6">
                <Link
                    href={`/organization/${org.id}-${org.slug}`}
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                >
                    <ArrowLeft size={20} />
                </Link>
                <div>
                    <h1 className="text-3xl font-semibold flex items-center gap-3">
                        <Settings className="text-primary" size={32} />
                        {t('manage')}
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
                    {t('joinRequests')}
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
                    {t('members')}
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
                    {t('edit')}
                </button>
            </div>

            {/* Requests Tab */}
            {activeTab === 'requests' && (
                <div className="space-y-4">
                    <div className="flex gap-2 overflow-x-auto">
                        {([
                            ['P', t('requestPending')],
                            ['A', t('requestApproved')],
                            ['R', t('requestRejected')],
                            ['log', t('requestLog')],
                        ] as [OrgRequestTab, string][]).map(([tab, label]) => (
                            <button
                                key={tab}
                                onClick={() => selectRequestTab(tab)}
                                className={cn(
                                    "px-3 py-1.5 rounded-xl text-sm font-medium border transition-colors",
                                    requestTab === tab
                                        ? "bg-primary text-primary-foreground border-primary"
                                        : "border-transparent text-muted-foreground hover:bg-muted"
                                )}
                            >
                                {label}
                            </button>
                        ))}
                    </div>
                    {requestsLoading ? (
                        <div className="flex items-center justify-center py-12">
                            <Loader2 size={32} className="animate-spin text-muted-foreground" />
                        </div>
                    ) : requests?.data.length ? (
                        <div className="border rounded-xl overflow-hidden">
                            <div className="bg-muted/30 px-4 py-3 border-b font-medium">
                                {t('requestCount', { count: requests.data.length })}
                            </div>
                            <div className="divide-y">
                                {requests.data.map((request) => (
                                    <div
                                        key={request.id}
                                        className="flex items-center gap-4 p-4 hover:bg-muted/20 transition-colors cursor-pointer"
                                        onClick={() => setSelectedRequest(request)}
                                    >
                                        <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary font-semibold">
                                            {request.username[0]?.toUpperCase()}
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <div className="font-medium flex items-center gap-2">
                                                {request.username}
                                                <RequestStateBadge state={request.state} />
                                            </div>
                                            <div className="text-sm text-muted-foreground">
                                                {t('requestedAt', { time: dayjs(request.time).fromNow() })}
                                            </div>
                                        </div>
                                        {request.state === 'P' && (
                                            <div className="flex items-center gap-2" onClick={e => e.stopPropagation()}>
                                                <button
                                                    onClick={() => handleRequest(request.id, 'approve')}
                                                    disabled={handleRequestMutation.isPending}
                                                    className="p-2 bg-green-500/10 text-green-500 rounded-lg hover:bg-green-500/20 transition-colors disabled:opacity-50"
                                                    title={t('approve')}
                                                >
                                                    <Check size={18} />
                                                </button>
                                                <button
                                                    onClick={() => handleRequest(request.id, 'reject')}
                                                    disabled={handleRequestMutation.isPending}
                                                    className="p-2 bg-red-500/10 text-red-500 rounded-lg hover:bg-red-500/20 transition-colors disabled:opacity-50"
                                                    title={t('reject')}
                                                >
                                                    <X size={18} />
                                                </button>
                                            </div>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </div>
                    ) : (
                        <div className="p-8 text-center text-muted-foreground border rounded-xl">
                            <Bell size={48} className="mx-auto mb-4 opacity-20" />
                            <p className="font-medium">{t('noRequests')}</p>
                        </div>
                    )}
                </div>
            )}

            {/* Request detail (v1 OrganizationRequestDetail) */}
            <Dialog open={!!selectedRequest} onOpenChange={open => !open && setSelectedRequest(null)}>
                <DialogContent className="max-w-lg">
                    <DialogHeader>
                        <DialogTitle>{t('requestDetail')}</DialogTitle>
                        <DialogDescription>{t('requestDetailDesc')}</DialogDescription>
                    </DialogHeader>
                    {selectedRequest && (
                        <div className="space-y-4">
                            <div className="flex items-center gap-3">
                                <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary font-semibold">
                                    {selectedRequest.username[0]?.toUpperCase()}
                                </div>
                                <div>
                                    <Link
                                        href={`/user/${selectedRequest.username}`}
                                        className="font-medium hover:text-primary transition-colors"
                                    >
                                        {selectedRequest.username}
                                    </Link>
                                    <div className="text-sm text-muted-foreground">
                                        {t('requestedAt', { time: dayjs(selectedRequest.time).fromNow() })}
                                    </div>
                                </div>
                                <div className="ml-auto">
                                    <RequestStateBadge state={selectedRequest.state} />
                                </div>
                            </div>
                            <div>
                                <div className="text-sm font-medium mb-1">{t('requestReason')}</div>
                                <p className="whitespace-pre-wrap rounded-md border bg-muted/30 p-3 text-sm text-muted-foreground">
                                    {selectedRequest.reason || t('noReason')}
                                </p>
                            </div>
                            {selectedRequest.state === 'P' && (
                                <div className="flex justify-end gap-3">
                                    <button
                                        onClick={() => handleRequest(selectedRequest.id, 'reject')}
                                        disabled={handleRequestMutation.isPending}
                                        className="px-4 py-2 rounded-xl bg-red-500/10 text-red-500 font-medium hover:bg-red-500/20 transition-colors disabled:opacity-50"
                                    >
                                        {t('reject')}
                                    </button>
                                    <button
                                        onClick={() => handleRequest(selectedRequest.id, 'approve')}
                                        disabled={handleRequestMutation.isPending}
                                        className="px-4 py-2 rounded-xl bg-green-500/10 text-green-500 font-medium hover:bg-green-500/20 transition-colors disabled:opacity-50"
                                    >
                                        {t('approve')}
                                    </button>
                                </div>
                            )}
                        </div>
                    )}
                </DialogContent>
            </Dialog>

            {/* Members Tab */}
            {activeTab === 'members' && (
                <div className="border rounded-xl overflow-hidden">
                    <div className="bg-muted/30 px-4 py-3 border-b font-medium">
                        {t('membersCount', { count: org.members?.length || 0 })}
                    </div>
                    <div className="divide-y">
                        {org.members?.map((member) => (
                            <div key={member.id} className="flex items-center gap-4 p-4 hover:bg-muted/30 transition-colors">
                                <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary font-semibold">
                                    {member.username[0]?.toUpperCase()}
                                </div>
                                <div className="flex-1">
                                    <div className="flex items-center gap-2">
                                        <span className="font-medium">{member.username}</span>
                                        {member.role === 'admin' && (
                                            <Badge className="text-xs bg-amber-500/10 text-amber-500 border-amber-500/20">
                                                <Crown size={12} className="inline mr-1" /> {t('admin')}
                                            </Badge>
                                        )}
                                    </div>
                                    <div className="text-sm text-muted-foreground">
                                        {t('joinedAt', { time: dayjs(member.joined_at).fromNow() })}
                                    </div>
                                </div>
                                {member.role !== 'admin' && (
                                    <button
                                        onClick={() => handleKick(member)}
                                        disabled={kickMutation.isPending}
                                        className="p-2 hover:bg-destructive/10 rounded-lg transition-colors text-destructive"
                                        title={t('kickFromOrganization')}
                                    >
                                        <UserX size={18} />
                                    </button>
                                )}
                            </div>
                        ))}
                        {!org.members?.length && (
                            <div className="p-8 text-center text-muted-foreground">
                                <Users size={48} className="mx-auto mb-4 opacity-20" />
                                <p className="font-medium">{t('noMembersYet')}</p>
                            </div>
                        )}
                    </div>
                </div>
            )}

            {/* Edit Tab */}
            {activeTab === 'edit' && (
                <EditOrganizationForm organization={org} onSuccess={() => router.push(`/organization/${org.id}-${org.slug}`)} />
            )}
        </div>
    );
}

/** State badge shared by the request list and the detail dialog. */
function RequestStateBadge({ state }: { state: OrganizationRequest['state'] }) {
    const t = useTranslations('Organization');
    const label = state === 'A' ? t('requestApproved') : state === 'R' ? t('requestRejected') : t('requestPending');
    const styles = state === 'A'
        ? 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20'
        : state === 'R'
            ? 'bg-red-500/10 text-red-500 border-red-500/20'
            : 'bg-amber-500/10 text-amber-500 border-amber-500/20';
    return <Badge className={cn('text-xs font-medium', styles)}>{label}</Badge>;
}

function EditOrganizationForm({ organization, onSuccess }: { organization: OrganizationDetail; onSuccess: () => void }) {
    const t = useTranslations('Organization');
    const tCommon = useTranslations('Common');
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
            <div className="bg-card rounded-md border p-6 space-y-4">
                <h3 className="text-lg font-semibold">{t('basicInformation')}</h3>

                <div>
                    <label className="block text-sm font-medium mb-2">{t('name')}</label>
                    <input
                        type="text"
                        value={formData.name}
                        onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        required
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium mb-2">{t('shortName')}</label>
                    <input
                        type="text"
                        value={formData.short_name}
                        onChange={(e) => setFormData(prev => ({ ...prev, short_name: e.target.value }))}
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={t('shortNamePlaceholder')}
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium mb-2">{t('about')}</label>
                    <textarea
                        value={formData.about}
                        onChange={(e) => setFormData(prev => ({ ...prev, about: e.target.value }))}
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none min-h-[150px]"
                        placeholder={t('aboutPlaceholder')}
                    />
                </div>
            </div>

            <div className="bg-card rounded-md border p-6 space-y-4">
                <h3 className="text-lg font-semibold">{t('settings')}</h3>

                <div className="flex flex-wrap gap-4">
                    <label className="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            checked={formData.is_open}
                            onChange={(e) => setFormData(prev => ({ ...prev, is_open: e.target.checked }))}
                            className="rounded w-5 h-5"
                        />
                        <span className="text-sm font-medium">{t('openForJoining')}</span>
                    </label>

                    <label className="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            checked={formData.is_unlisted}
                            onChange={(e) => setFormData(prev => ({ ...prev, is_unlisted: e.target.checked }))}
                            className="rounded w-5 h-5"
                        />
                        <span className="text-sm font-medium">{t('unlistedHide')}</span>
                    </label>
                </div>

                <div>
                    <label className="block text-sm font-medium mb-2">{t('memberSlots')}</label>
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
                    href={`/organization/${organization.id}-${organization.slug}`}
                    className="px-6 py-2.5 rounded-xl border hover:bg-muted transition-colors font-medium"
                >
                    {tCommon('cancel')}
                </Link>
                <button
                    type="submit"
                    disabled={isLoading}
                    className="px-6 py-2.5 rounded-xl bg-primary text-white font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
                >
                    {isLoading && <Loader2 size={16} className="animate-spin" />}
                    {tCommon('saveChanges')}
                </button>
            </div>
        </form>
    );
}
