'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { AdminJudge } from '@/types';
import { adminJudgeApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import Link from 'next/link';
import {
    Search,
    Server,
    XCircle,
    CheckCircle,
    RefreshCw,
    Power,
    Activity,
    Wifi,
    WifiOff,
    Shield,
    ShieldOff
} from 'lucide-react';

export default function AdminJudgesPage() {
    const [search, setSearch] = useState('');

    const queryClient = useQueryClient();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-judges'],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminJudge[];
                total: number;
                page: number;
                page_size: number;
            }>('/admin/judges');
            return res.data;
        }
    });

    const blockMutation = useMutation({
        mutationFn: (id: number) => adminJudgeApi.block(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-judges'] });
        }
    });

    const unblockMutation = useMutation({
        mutationFn: (id: number) => adminJudgeApi.unblock(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-judges'] });
        }
    });

    const enableMutation = useMutation({
        mutationFn: (id: number) => adminJudgeApi.enable(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-judges'] });
        }
    });

    const disableMutation = useMutation({
        mutationFn: (id: number) => adminJudgeApi.disable(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-judges'] });
        }
    });

    const judges = data?.data || [];

    const filteredJudges = judges.filter(j =>
        j.name.toLowerCase().includes(search.toLowerCase())
    );

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Server className="text-primary" size={32} />
                        Judges
                    </h1>
                    <p className="text-muted-foreground mt-1">Manage judging nodes and monitor status</p>
                </div>

                <div className="flex gap-2">
                    <button
                        onClick={() => refetch()}
                        className="px-4 py-2 rounded-xl bg-card border hover:bg-primary/5 flex items-center gap-2 font-medium"
                    >
                        <RefreshCw size={18} className={isLoading ? 'animate-spin' : ''} /> Refresh
                    </button>
                    <div className="relative">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder="Search judges..."
                            className="w-full md:w-64 h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                </div>
            </div>

            {/* Stats */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="bg-card rounded-2xl p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-500">
                            <Wifi size={20} />
                        </div>
                        <div>
                            <div className="text-2xl font-bold">{judges.filter(j => j.online).length}</div>
                            <div className="text-sm text-muted-foreground">Online</div>
                        </div>
                    </div>
                </div>
                <div className="bg-card rounded-2xl p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-red-500/10 text-red-500">
                            <WifiOff size={20} />
                        </div>
                        <div>
                            <div className="text-2xl font-bold">{judges.filter(j => !j.online).length}</div>
                            <div className="text-sm text-muted-foreground">Offline</div>
                        </div>
                    </div>
                </div>
                <div className="bg-card rounded-2xl p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-amber-500/10 text-amber-500">
                            <Shield size={20} />
                        </div>
                        <div>
                            <div className="text-2xl font-bold">{judges.filter(j => j.is_blocked).length}</div>
                            <div className="text-sm text-muted-foreground">Blocked</div>
                        </div>
                    </div>
                </div>
                <div className="bg-card rounded-2xl p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-blue-500/10 text-blue-500">
                            <Power size={20} />
                        </div>
                        <div>
                            <div className="text-2xl font-bold">{judges.filter(j => j.is_disabled).length}</div>
                            <div className="text-sm text-muted-foreground">Disabled</div>
                        </div>
                    </div>
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-32 rounded-2xl" />)}
                </div>
            ) : (
                <div className="grid gap-4">
                    {filteredJudges.length === 0 ? (
                        <div className="text-center py-12 rounded-2xl border border-dashed bg-muted/30">
                            <Server size={48} className="mx-auto text-muted-foreground opacity-20" />
                            <p className="text-muted-foreground mt-4">No judges found</p>
                        </div>
                    ) : (
                        filteredJudges.map((judge) => (
                            <div key={judge.id} className="bg-card rounded-2xl p-6 border hover:shadow-lg transition-shadow">
                                <div className="flex items-start justify-between gap-4">
                                    <div className="flex items-start gap-4 flex-1">
                                        <div className={`w-14 h-14 rounded-xl flex items-center justify-center shrink-0 ${
                                            judge.online ? 'bg-success/10 text-success' : 'bg-muted/50 text-muted-foreground'
                                        }`}>
                                            {judge.online ? <Wifi size={28} /> : <WifiOff size={28} />}
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <div className="flex flex-wrap items-center gap-2 mb-2">
                                                <h3 className="font-bold text-lg">{judge.name}</h3>
                                                <div className="flex gap-1">
                                                    {judge.online ? (
                                                        <Badge variant="success" className="flex items-center gap-1 text-xs">
                                                            <CheckCircle size={12} /> Online
                                                        </Badge>
                                                    ) : (
                                                        <Badge variant="secondary" className="text-xs">Offline</Badge>
                                                    )}
                                                    {judge.is_blocked && (
                                                        <Badge variant="destructive" className="text-xs flex items-center gap-1">
                                                            <ShieldOff size={12} /> Blocked
                                                        </Badge>
                                                    )}
                                                    {judge.is_disabled && (
                                                        <Badge variant="warning" className="text-xs flex items-center gap-1">
                                                            <Power size={12} /> Disabled
                                                        </Badge>
                                                    )}
                                                </div>
                                            </div>
                                            <div className="flex flex-wrap gap-4 text-sm">
                                                <div className="flex items-center gap-1.5 text-muted-foreground">
                                                    <Activity size={14} />
                                                    <span>Load: <span className="font-medium text-foreground">{judge.load?.toFixed(2) ?? 'N/A'}</span></span>
                                                </div>
                                                <div className="flex items-center gap-1.5 text-muted-foreground">
                                                    <Wifi size={14} />
                                                    <span>Ping: <span className="font-medium text-foreground">{judge.ping?.toFixed(0) ?? 'N/A'}ms</span></span>
                                                </div>
                                                {judge.last_ip && (
                                                    <div className="flex items-center gap-1.5 text-muted-foreground">
                                                        <Server size={14} />
                                                        <span className="font-mono">{judge.last_ip}</span>
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                    <div className="flex flex-col gap-2">
                                        <Link
                                            href={`/admin/judges/${judge.id}`}
                                            className="px-4 py-2 rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 flex items-center gap-2 font-medium text-sm whitespace-nowrap"
                                        >
                                            <Server size={16} /> Details
                                        </Link>
                                        <div className="flex gap-2">
                                            {judge.is_blocked ? (
                                                <button
                                                    onClick={() => unblockMutation.mutate(judge.id)}
                                                    disabled={unblockMutation.isPending}
                                                    className="px-3 py-2 rounded-lg bg-success/10 text-success hover:bg-success/20 flex items-center gap-1.5 font-medium text-sm"
                                                    title="Unblock judge"
                                                >
                                                    <Shield size={16} />
                                                </button>
                                            ) : (
                                                <button
                                                    onClick={() => blockMutation.mutate(judge.id)}
                                                    disabled={blockMutation.isPending}
                                                    className="px-3 py-2 rounded-lg bg-destructive/10 text-destructive hover:bg-destructive/20 flex items-center gap-1.5 font-medium text-sm"
                                                    title="Block judge"
                                                >
                                                    <ShieldOff size={16} />
                                                </button>
                                            )}
                                            {judge.is_disabled ? (
                                                <button
                                                    onClick={() => enableMutation.mutate(judge.id)}
                                                    disabled={enableMutation.isPending}
                                                    className="px-3 py-2 rounded-lg bg-success/10 text-success hover:bg-success/20 flex items-center gap-1.5 font-medium text-sm"
                                                    title="Enable judge"
                                                >
                                                    <Power size={16} />
                                                </button>
                                            ) : (
                                                <button
                                                    onClick={() => disableMutation.mutate(judge.id)}
                                                    disabled={disableMutation.isPending}
                                                    className="px-3 py-2 rounded-lg bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 flex items-center gap-1.5 font-medium text-sm"
                                                    title="Disable judge"
                                                >
                                                    <Power size={16} className="rotate-180" />
                                                </button>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        ))
                    )}
                </div>
            )}
        </div>
    );
}
