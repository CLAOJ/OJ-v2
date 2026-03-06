'use client';

import { useParams } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { adminContestApi } from '@/lib/adminApi';
import { RankingResponse } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import {
    Trophy,
    Users,
    ShieldAlert,
    CheckCircle,
    XCircle,
    ArrowLeft
} from 'lucide-react';
import { cn } from '@/lib/utils';

export default function ContestParticipationsPage() {
    const params = useParams();
    const key = params.key as string;
    const queryClient = useQueryClient();

    const { data: ranking, isLoading } = useQuery({
        queryKey: ['contest-ranking', key],
        queryFn: async () => {
            const res = await api.get<RankingResponse>(`/contest/${key}/ranking`);
            return res.data;
        }
    });

    const disqualifyMutation = useMutation({
        mutationFn: (participationId: number) =>
            adminContestApi.disqualifyParticipation(key, participationId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['contest-ranking', key] });
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to disqualify participation');
        }
    });

    const undisqualifyMutation = useMutation({
        mutationFn: (participationId: number) =>
            adminContestApi.undisqualifyParticipation(key, participationId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['contest-ranking', key] });
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to undisqualify participation');
        }
    });

    return (
        <div className="container mx-auto py-8">
            <div className="flex items-center justify-between mb-8">
                <div className="flex items-center gap-4">
                    <Link
                        href={`/admin/contests/${key}/edit`}
                        className="p-2 hover:bg-muted rounded-xl transition-colors"
                    >
                        <ArrowLeft size={20} />
                    </Link>
                    <div>
                        <h1 className="text-3xl font-bold flex items-center gap-3">
                            <Users className="text-primary" size={32} />
                            Contest Participations
                        </h1>
                        <p className="text-muted-foreground mt-1">
                            Manage participations for contest {key}
                        </p>
                    </div>
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3, 4, 5].map((i) => (
                        <Skeleton key={i} className="h-16 rounded-xl" />
                    ))}
                </div>
            ) : !ranking?.rankings || ranking.rankings.length === 0 ? (
                <div className="bg-card border rounded-2xl p-12 text-center">
                    <Users className="w-16 h-16 text-muted-foreground mx-auto mb-4" />
                    <p className="text-lg font-medium">No participations yet</p>
                    <p className="text-sm text-muted-foreground mt-1">
                        No users have joined this contest yet
                    </p>
                </div>
            ) : (
                <div className="bg-card border rounded-2xl overflow-hidden">
                    <table className="w-full text-left">
                        <thead className="bg-muted/50 border-b">
                            <tr>
                                <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Rank</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">User</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Score</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Time</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">Status</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {ranking.rankings.map((row) => (
                                <tr
                                    key={row.username}
                                    className={cn(
                                        "hover:bg-muted/5 transition-colors",
                                        row.is_disqualified && "bg-red-50/50 dark:bg-red-950/20"
                                    )}
                                >
                                    <td className="px-6 py-4 text-center">
                                        <span className={cn(
                                            "inline-flex items-center justify-center w-8 h-8 rounded-lg font-black text-xs",
                                            row.rank === 1 ? "bg-amber-500 text-white" :
                                            row.rank === 2 ? "bg-zinc-400 text-white" :
                                            row.rank === 3 ? "bg-orange-600 text-white" :
                                            "bg-muted text-muted-foreground"
                                        )}>
                                            {row.rank}
                                        </span>
                                    </td>
                                    <td className="px-6 py-4">
                                        <Link
                                            href={`/user/${row.username}`}
                                            className={cn(
                                                "font-bold text-sm hover:text-primary transition-colors",
                                                row.is_disqualified && "line-through text-muted-foreground"
                                            )}
                                        >
                                            {row.username}
                                        </Link>
                                    </td>
                                    <td className="px-6 py-4">
                                        <span className={cn(
                                            "font-black text-primary",
                                            row.is_disqualified && "text-muted-foreground line-through"
                                        )}>
                                            {Math.round(row.score)}
                                        </span>
                                    </td>
                                    <td className="px-6 py-4 text-sm text-muted-foreground">
                                        {Math.floor(row.cumtime / 60)}m {row.cumtime % 60}s
                                    </td>
                                    <td className="px-6 py-4">
                                        {row.is_disqualified ? (
                                            <Badge variant="destructive" className="gap-1">
                                                <XCircle size={12} />
                                                Disqualified
                                            </Badge>
                                        ) : (
                                            <Badge variant="success" className="gap-1">
                                                <CheckCircle size={12} />
                                                Active
                                            </Badge>
                                        )}
                                    </td>
                                    <td className="px-6 py-4 text-right">
                                        {row.is_disqualified ? (
                                            <Button
                                                size="sm"
                                                variant="outline"
                                                onClick={() => undisqualifyMutation.mutate(row.participation_id!)}
                                                disabled={undisqualifyMutation.isPending}
                                            >
                                                <CheckCircle size={16} className="mr-1" />
                                                Undisqualify
                                            </Button>
                                        ) : (
                                            <Button
                                                size="sm"
                                                variant="destructive"
                                                onClick={() => disqualifyMutation.mutate(row.participation_id!)}
                                                disabled={disqualifyMutation.isPending}
                                            >
                                                <ShieldAlert size={16} className="mr-1" />
                                                Disqualify
                                            </Button>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}
