'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
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
    const t = useTranslations('Admin');
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
                        {t('judges.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('judges.subtitle')}</p>
                </div>

                <div className="flex gap-2">
                    <button
                        onClick={() => refetch()}
                        className="px-4 py-2 rounded-xl bg-card border hover:bg-primary/5 flex items-center gap-2 font-medium"
                    >
                        <RefreshCw size={18} className={isLoading ? 'animate-spin' : ''} /> {t('judges.refreshButton')}
                    </button>
                    <div className="relative">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder={t('judges.searchPlaceholder')}
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
                            <div className="text-sm text-muted-foreground">{t('judges.online')}</div>
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
                            <div className="text-sm text-muted-foreground">{t('judges.offline')}</div>
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
                            <div className="text-sm text-muted-foreground">{t('judges.blocked')}</div>
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
                            <div className="text-sm text-muted-foreground">{t('judges.disabled')}</div>
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
                            <p className="text-muted-foreground mt-4">{t('judges.noJudgesFound')}</p>
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
                                                            <CheckCircle size={12} /> {t('judges.online')}
                                                        </Badge>
                                                    ) : (
                                                        <Badge variant="secondary" className="text-xs">{t('judges.offline')}</Badge>
                                                    )}
                                                    {judge.is_blocked && (
                                                        <Badge variant="destructive" className="text-xs flex items-center gap-1">
                                                            <ShieldOff size={12} /> {t('judges.blocked')}
                                                        </Badge>
                                                    )}
                                                    {judge.is_disabled && (
                                                        <Badge variant="warning" className="text-xs flex items-center gap-1">
                                                            <Power size={12} /> {t('judges.disabled')}
                                                        </Badge>
                                                    )}
                                                </div>
                                            </div>
                                            <div className="flex flex-wrap gap-4 text-sm">
                                                <div className="flex items-center gap-1.5 text-muted-foreground">
                                                    <Activity size={14} />
                                                    <span>{t('judges.loadPrefix')} <span className="font-medium text-foreground">{judge.load?.toFixed(2) ?? t('common.notAvailable')}</span></span>
                                                </div>
                                                <div className="flex items-center gap-1.5 text-muted-foreground">
                                                    <Wifi size={14} />
                                                    <span>{t('judges.pingPrefix')} <span className="font-medium text-foreground">{judge.ping?.toFixed(0) ?? t('common.notAvailable')}ms</span></span>
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
                                            <Server size={16} /> {t('judges.detailsButton')}
                                        </Link>
                                        <div className="flex gap-2">
                                            {judge.is_blocked ? (
                                                <button
                                                    onClick={() => unblockMutation.mutate(judge.id)}
                                                    disabled={unblockMutation.isPending}
                                                    className="px-3 py-2 rounded-lg bg-success/10 text-success hover:bg-success/20 flex items-center gap-1.5 font-medium text-sm"
                                                    title={t('judges.unblockTitle')}
                                                >
                                                    <Shield size={16} />
                                                </button>
                                            ) : (
                                                <button
                                                    onClick={() => blockMutation.mutate(judge.id)}
                                                    disabled={blockMutation.isPending}
                                                    className="px-3 py-2 rounded-lg bg-destructive/10 text-destructive hover:bg-destructive/20 flex items-center gap-1.5 font-medium text-sm"
                                                    title={t('judges.blockTitle')}
                                                >
                                                    <ShieldOff size={16} />
                                                </button>
                                            )}
                                            {judge.is_disabled ? (
                                                <button
                                                    onClick={() => enableMutation.mutate(judge.id)}
                                                    disabled={enableMutation.isPending}
                                                    className="px-3 py-2 rounded-lg bg-success/10 text-success hover:bg-success/20 flex items-center gap-1.5 font-medium text-sm"
                                                    title={t('judges.enableTitle')}
                                                >
                                                    <Power size={16} />
                                                </button>
                                            ) : (
                                                <button
                                                    onClick={() => disableMutation.mutate(judge.id)}
                                                    disabled={disableMutation.isPending}
                                                    className="px-3 py-2 rounded-lg bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 flex items-center gap-1.5 font-medium text-sm"
                                                    title={t('judges.disableTitle')}
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
