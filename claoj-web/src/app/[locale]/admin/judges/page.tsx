'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { AdminJudge } from '@/types';
import { adminJudgeApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import {
    Search,
    Server,
    XCircle,
    CheckCircle,
    RefreshCw
} from 'lucide-react';

export default function AdminJudgesPage() {
    const [search, setSearch] = useState('');

    const queryClient = useQueryClient();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-judges', search],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminJudge[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/judges?search=${search}`);
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
                    <p className="text-muted-foreground mt-1">Manage judging nodes</p>
                </div>

                <div className="relative w-full md:w-80">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                    <input
                        type="text"
                        placeholder="Search judges..."
                        className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                    />
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-20 rounded-2xl" />)}
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
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-4">
                                        <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${
                                            judge.online ? 'bg-success/10 text-success' : 'bg-muted/50 text-muted-foreground'
                                        }`}>
                                            <Server size={24} />
                                        </div>
                                        <div>
                                            <div className="flex items-center gap-2">
                                                <h3 className="font-bold text-lg">{judge.name}</h3>
                                                {judge.is_blocked && (
                                                    <Badge variant="destructive" className="text-xs">Blocked</Badge>
                                                )}
                                                {judge.online ? (
                                                    <Badge variant="success" className="flex items-center gap-1 text-xs">
                                                        <CheckCircle size={12} /> Online
                                                    </Badge>
                                                ) : (
                                                    <Badge variant="secondary" className="text-xs">Offline</Badge>
                                                )}
                                            </div>
                                            <div className="text-sm text-muted-foreground mt-1">
                                                Auth Key: <span className="font-mono">{judge.auth_key}</span>
                                                {judge.last_ip && ` | Last IP: ${judge.last_ip}`}
                                            </div>
                                        </div>
                                    </div>
                                    <div className="flex gap-2">
                                        {judge.is_blocked ? (
                                            <button
                                                onClick={() => unblockMutation.mutate(judge.id)}
                                                disabled={unblockMutation.isPending}
                                                className="px-4 py-2 rounded-xl bg-success/10 text-success hover:bg-success/20 flex items-center gap-2 font-medium"
                                            >
                                                <CheckCircle size={18} /> Unblock
                                            </button>
                                        ) : (
                                            <button
                                                onClick={() => blockMutation.mutate(judge.id)}
                                                disabled={blockMutation.isPending}
                                                className="px-4 py-2 rounded-xl bg-destructive/10 text-destructive hover:bg-destructive/20 flex items-center gap-2 font-medium"
                                            >
                                                <XCircle size={18} /> Block
                                            </button>
                                        )}
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
