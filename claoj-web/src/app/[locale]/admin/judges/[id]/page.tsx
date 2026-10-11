'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminJudgeApi } from '@/lib/adminApi';
import { Link } from '@/navigation';
import { toast } from 'sonner';
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
    ArrowLeft,
    Eye,
    EyeOff
} from 'lucide-react';

export default function AdminJudgeDetailPage() {
    const t = useTranslations('Admin');
    const params = useParams();
    const router = useRouter();
    const id = params.id as string;
    const queryClient = useQueryClient();

    const { data: judgeResponse, isLoading } = useQuery({
        queryKey: ['admin-judge', id],
        queryFn: () => adminJudgeApi.detail(parseInt(id)),
    });

    const judge = judgeResponse?.data;
    // null = untouched, so the field tracks the server value until edited.
    const [draft, setDraft] = useState<string | null>(null);
    const description = draft ?? judge?.description ?? '';
    const [showKey, setShowKey] = useState(false);

    const updateMutation = useMutation({
        mutationFn: (judgeId: number) => adminJudgeApi.update(judgeId, { description }),
        onSuccess: () => {
            toast.success(t('judgeDetail.descriptionSaved'));
            setDraft(null);
            queryClient.invalidateQueries({ queryKey: ['admin-judge', id] });
        },
        onError: () => toast.error(t('judgeDetail.descriptionSaveFailed')),
    });

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
                <Skeleton className="h-64 rounded-md" />
                <Skeleton className="h-64 rounded-md" />
            </div>
        );
    }

    if (!judge) {
        return (
            <div className="text-center py-12">
                <Server size={48} className="mx-auto text-muted-foreground opacity-20" />
                <h2 className="text-xl font-semibold mt-4">{t('judgeDetail.notFoundTitle')}</h2>
                <Link href="/admin/judges" className="text-accent hover:underline mt-2 inline-block">
                    {t('judgeDetail.backToJudges')}
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
                    <p className="text-muted-foreground mt-1">{t('judgeDetail.subtitle')}</p>
                </div>
                <div className="flex gap-2">
                    {judge.online ? (
                        <Badge variant="success" className="flex items-center gap-1">
                            <CheckCircle size={14} /> {t('judges.online')}
                        </Badge>
                    ) : (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <WifiOff size={14} /> {t('judges.offline')}
                        </Badge>
                    )}
                    {judge.is_blocked && (
                        <Badge variant="destructive" className="flex items-center gap-1">
                            <ShieldOff size={14} /> {t('judges.blocked')}
                        </Badge>
                    )}
                    {judge.is_disabled && (
                        <Badge variant="warning" className="flex items-center gap-1">
                            <Power size={14} /> {t('judges.disabled')}
                        </Badge>
                    )}
                </div>
            </div>

            {/* Status Card */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <div className="bg-card rounded-md p-4 border">
                    <div className="flex items-center gap-3">
                        <div className={`p-2 rounded-lg ${judge.online ? 'bg-success/10 text-success' : 'bg-muted/50 text-muted-foreground'}`}>
                            {judge.online ? <Wifi size={20} /> : <WifiOff size={20} />}
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">{t('judgeDetail.statusLabel')}</div>
                            <div className="font-semibold">{judge.online ? t('judges.online') : t('judges.offline')}</div>
                        </div>
                    </div>
                </div>
                <div className="bg-card rounded-md p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-accent/10 text-accent">
                            <Activity size={20} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">{t('judgeDetail.loadLabel')}</div>
                            <div className="font-semibold">{judge.load?.toFixed(2) ?? t('common.notAvailable')}</div>
                        </div>
                    </div>
                </div>
                <div className="bg-card rounded-md p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-accent/10 text-accent">
                            <Wifi size={20} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">{t('judgeDetail.pingLabel')}</div>
                            <div className="font-semibold">{judge.ping?.toFixed(0) ?? t('common.notAvailable')}ms</div>
                        </div>
                    </div>
                </div>
                <div className="bg-card rounded-md p-4 border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-warning/10 text-warning">
                            <Clock size={20} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">{t('judgeDetail.startTimeLabel')}</div>
                            <div className="font-semibold text-sm">
                                {judge.start_time ? new Date(judge.start_time).toLocaleString() : t('common.notAvailable')}
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Info Cards */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Problems */}
                <div className="bg-card rounded-md p-6 border">
                    <h3 className="font-bold text-lg flex items-center gap-2 mb-4">
                        <FileText size={20} className="text-primary" />
                        {t('judgeDetail.assignedProblems', { count: judge.problems.length })}
                    </h3>
                    {judge.problems.length === 0 ? (
                        <p className="text-muted-foreground text-sm">{t('judgeDetail.noProblemsAssigned')}</p>
                    ) : (
                        <div className="space-y-2 max-h-64 overflow-y-auto">
                            {judge.problems.map((problem) => (
                                <div key={problem.code} className="flex items-center justify-between p-2 rounded-lg hover:bg-muted/50">
                                    <Link
                                        href={`/problem/${problem.code}`}
                                        className="font-medium text-accent hover:underline"
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
                <div className="bg-card rounded-md p-6 border">
                    <h3 className="font-bold text-lg flex items-center gap-2 mb-4">
                        <Code size={20} className="text-primary" />
                        {t('judgeDetail.supportedLanguages', { count: judge.runtimes.length })}
                    </h3>
                    {judge.runtimes.length === 0 ? (
                        <p className="text-muted-foreground text-sm">{t('judgeDetail.noLanguagesReported')}</p>
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
            <div className="bg-card rounded-md p-6 border">
                <label htmlFor="judge-description" className="block font-bold text-lg mb-3">{t('judgeDetail.descriptionTitle')}</label>
                <textarea
                    id="judge-description"
                    rows={4}
                    value={description}
                    onChange={e => setDraft(e.target.value)}
                    placeholder={t('judgeDetail.descriptionPlaceholder')}
                    className="w-full rounded-md border bg-background p-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring/40"
                />
                <div className="mt-3 flex justify-end">
                    <button
                        onClick={() => updateMutation.mutate(judge.id)}
                        disabled={updateMutation.isPending || description === (judge.description ?? '')}
                        className="h-9 px-4 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50"
                    >
                        {updateMutation.isPending ? t('common.saving') : t('common.saveChanges')}
                    </button>
                </div>
            </div>

            {/* Additional Info */}
            <div className="bg-card rounded-md p-6 border">
                <h3 className="font-bold text-lg mb-4">{t('judgeDetail.additionalInfoTitle')}</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-muted/50">
                            <MapPin size={18} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">{t('judgeDetail.lastIpLabel')}</div>
                            <div className="font-mono text-sm">{judge.last_ip || t('common.notAvailable')}</div>
                        </div>
                    </div>
                    <div className="flex items-center gap-3">
                        <div className="p-2 rounded-lg bg-muted/50">
                            <Shield size={18} />
                        </div>
                        <div>
                            <div className="text-sm text-muted-foreground">{t('judgeDetail.authKeyLabel')}</div>
                            <div className="flex items-center gap-2">
                                <span className="font-mono text-sm truncate max-w-[200px]">
                                    {showKey ? judge.auth_key : '•'.repeat(16)}
                                </span>
                                <button
                                    onClick={() => setShowKey(v => !v)}
                                    aria-label={showKey ? t('judgeDetail.hideKey') : t('judgeDetail.showKey')}
                                    className="p-1 rounded hover:bg-muted text-muted-foreground"
                                >
                                    {showKey ? <EyeOff size={14} /> : <Eye size={14} />}
                                </button>
                            </div>
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
                        <Shield size={18} /> {t('judgeDetail.unblockButton')}
                    </button>
                ) : (
                    <button
                        onClick={() => blockMutation.mutate(judge.id)}
                        disabled={blockMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-destructive/10 text-destructive hover:bg-destructive/20 flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <ShieldOff size={18} /> {t('judgeDetail.blockButton')}
                    </button>
                )}
                {judge.is_disabled ? (
                    <button
                        onClick={() => enableMutation.mutate(judge.id)}
                        disabled={enableMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-success/10 text-success hover:bg-success/20 flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <Power size={18} /> {t('judgeDetail.enableButton')}
                    </button>
                ) : (
                    <button
                        onClick={() => disableMutation.mutate(judge.id)}
                        disabled={disableMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-warning/10 text-warning hover:bg-warning/20 flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <Power size={18} className="rotate-180" /> {t('judgeDetail.disableButton')}
                    </button>
                )}
            </div>
        </div>
    );
}
