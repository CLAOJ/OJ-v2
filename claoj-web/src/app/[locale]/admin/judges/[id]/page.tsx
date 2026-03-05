'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { adminJudgeApi } from '@/lib/adminApi';
import { AdminJudgeDetail } from '@/types';
import Link from 'next/link';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import {
    Server,
    Wifi,
    WifiOff,
    Activity,
    Shield,
    ShieldOff,
    Power,
    Clock,
    MapPin,
    FileText,
    Code,
    CheckCircle,
    XCircle,
    ArrowLeft
} from 'lucide-react';

export default function AdminJudgeDetailPage() {
    const params = useParams();
    const router = useRouter();
    const id = params.id as string;
    const queryClient = useQueryClient();

    const { data: judgeResponse, isLoading } = useQuery({
        queryKey: ['admin-judge', id],
        queryFn: () => adminJudgeApi.detail(parseInt(id)),
    });

    const judge = judgeResponse?.data;

    const blockMutation = useMutation({
        mutationFn: (judgeId: number) => adminJudgeApi.block(judgeId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-judge', id] });
        }
    });

    const unblockMutation = useMutation({
        mutationFn: (judgeId: number) => adminJudgeApi.unblock(judgeId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-judge', id] });
        }
    });

    const enableMutation = useMutation({
        mutationFn: (judgeId: number) => adminJudgeApi.enable(judgeId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-judge', id] });
        }
    });

    const disableMutation = useMutation({
        mutationFn: (judgeId: number) => adminJudgeApi.disable(judgeId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-judge', id] });
        }
    });

    if (isLoading) {
        return (
            <div className="space-y-6">
                <Skeleton className="h-10 w-48" />
                <Skeleton className="h-64 rounded-2xl" />
                <Skeleton className="h-64 rounded-2xl" />
            </div>
        );
    }

    if (!judge) {
        return (
            <div className="text-center py-12">
                <Server size={48} className="mx-auto text-muted-foreground opacity-20" />
                <h2 className="text-xl font-semibold mt-4">Judge not found</h2>
                <Link href="/admin/judges" className="text-blue-600 dark:text-blue-400 hover:underline mt-2 inline-block">
                    Back to judges
                </Link>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center gap-4">
                <button
                    onClick={() => router.back()}
                    className="p-2 rounded-lg hover:bg-card border"
                >
                    <ArrowLeft size={20} />
                </button>
                <div className="flex-1">
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Server className="text-primary" size={32} />
                        {judge.name}
                    </h1>
                    <p className="text-muted-foreground mt-1">Judge details and configuration</p>
                </div>
                <div className="flex gap-2">
                    {judge.online ? (
                        <Badge variant="success" className="flex items-center gap-1">
                            <CheckCircle size={14} /> Online
                        </Badge>
                    ) : (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <WifiOff size={14} /> Offline
                        </Badge>
                    )}
                    {judge.is_blocked && (
                        <Badge variant="destructive" className="flex items-center gap-1">
                            <ShieldOff size={14} /> Blocked
                        </Badge>
                    )}
                    {judge.is_disabled && (
                        <Badge variant="warning" className="flex items-center gap-1">
                            <Power size={14} /> Disabled
                        </Badge>
                    )}
                </div>
            </div>

            {/* Status Card */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <div className="bg-card rounded-2xl p-4 border">
                    <div className="flex items-center gap-3">
                        <div className={`p-2 rounded-lg ${judge.online ? 'bg-emerald-500/10 text-emerald-500' : 'bg-muted/50 text-muted-foreground'}`}>
                            {judge.online ? <Wifi size={20} /> : <WifiOff size={20} />}
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">Status</div>
                            <div className="font-semibold">{judge.online ? 'Online' : 'Offline'}</div>
                        </div>
                    </div>
                </div>
                <div className="bg-card rounded-2xl p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-blue-500/10 text-blue-500">
                            <Activity size={20} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">Load</div>
                            <div className="font-semibold">{judge.load?.toFixed(2) ?? 'N/A'}</div>
                        </div>
                    </div>
                </div>
                <div className="bg-card rounded-2xl p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-purple-500/10 text-purple-500">
                            <Wifi size={20} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">Ping</div>
                            <div className="font-semibold">{judge.ping?.toFixed(0) ?? 'N/A'}ms</div>
                        </div>
                    </div>
                </div>
                <div className="bg-card rounded-2xl p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-amber-500/10 text-amber-500">
                            <Clock size={20} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">Start Time</div>
                            <div className="font-semibold text-sm">
                                {judge.start_time ? new Date(judge.start_time).toLocaleString() : 'N/A'}
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Info Cards */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Problems */}
                <div className="bg-card rounded-2xl p-6 border">
                    <h3 className="font-bold text-lg flex items-center gap-2 mb-4">
                        <FileText size={20} className="text-primary" />
                        Assigned Problems ({judge.problems.length})
                    </h3>
                    {judge.problems.length === 0 ? (
                        <p className="text-muted-foreground text-sm">No problems assigned - this judge can handle all problems</p>
                    ) : (
                        <div className="space-y-2 max-h-64 overflow-y-auto">
                            {judge.problems.map((problem) => (
                                <div key={problem.code} className="flex items-center justify-between p-2 rounded-lg hover:bg-muted/50">
                                    <Link
                                        href={`/problems/${problem.code}`}
                                        className="font-medium text-blue-600 dark:text-blue-400 hover:underline"
                                    >
                                        {problem.code}
                                    </Link>
                                    <span className="text-sm text-muted-foreground truncate max-w-[200px]">{problem.name}</span>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Runtimes */}
                <div className="bg-card rounded-2xl p-6 border">
                    <h3 className="font-bold text-lg flex items-center gap-2 mb-4">
                        <Code size={20} className="text-primary" />
                        Supported Languages ({judge.runtimes.length})
                    </h3>
                    {judge.runtimes.length === 0 ? (
                        <p className="text-muted-foreground text-sm">No language versions reported</p>
                    ) : (
                        <div className="space-y-2 max-h-64 overflow-y-auto">
                            {judge.runtimes.map((runtime) => (
                                <div key={runtime.key} className="flex items-center justify-between p-2 rounded-lg hover:bg-muted/50">
                                    <div className="font-medium">{runtime.name}</div>
                                    <div className="text-sm text-muted-foreground font-mono">{runtime.version}</div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>

            {/* Description */}
            {judge.description && (
                <div className="bg-card rounded-2xl p-6 border">
                    <h3 className="font-bold text-lg mb-3">Description</h3>
                    <p className="text-muted-foreground whitespace-pre-wrap">{judge.description}</p>
                </div>
            )}

            {/* Additional Info */}
            <div className="bg-card rounded-2xl p-6 border">
                <h3 className="font-bold text-lg mb-4">Additional Information</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-muted/50">
                            <MapPin size={18} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">Last IP</div>
                            <div className="font-mono text-sm">{judge.last_ip || 'N/A'}</div>
                        </div>
                    </div>
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-muted/50">
                            <Shield size={18} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">Auth Key</div>
                            <div className="font-mono text-sm truncate max-w-[200px]">{judge.auth_key}</div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Actions */}
            <div className="flex flex-wrap gap-3">
                {judge.is_blocked ? (
                    <button
                        onClick={() => unblockMutation.mutate(judge.id)}
                        disabled={unblockMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-success/10 text-success hover:bg-success/20 flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <Shield size={18} /> Unblock Judge
                    </button>
                ) : (
                    <button
                        onClick={() => blockMutation.mutate(judge.id)}
                        disabled={blockMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-destructive/10 text-destructive hover:bg-destructive/20 flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <ShieldOff size={18} /> Block Judge
                    </button>
                )}
                {judge.is_disabled ? (
                    <button
                        onClick={() => enableMutation.mutate(judge.id)}
                        disabled={enableMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-success/10 text-success hover:bg-success/20 flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <Power size={18} /> Enable Judge
                    </button>
                ) : (
                    <button
                        onClick={() => disableMutation.mutate(judge.id)}
                        disabled={disableMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <Power size={18} className="rotate-180" /> Disable Judge
                    </button>
                )}
            </div>
        </div>
    );
}
