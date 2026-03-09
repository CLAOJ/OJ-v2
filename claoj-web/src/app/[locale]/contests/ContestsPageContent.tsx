'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Contest, APIResponse } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link, useRouter } from '@/navigation';
import { useState, useEffect } from 'react';
import {
    Trophy,
    Clock,
    Calendar,
    Users,
    Zap,
    ChevronRight,
    Search,
    Play,
    Timer,
    History,
    Eye,
    Star
} from 'lucide-react';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import duration from 'dayjs/plugin/duration';

dayjs.extend(relativeTime);
dayjs.extend(duration);

export default function ContestsPageContent() {
    const t = useTranslations('Contest');
    const queryClient = useQueryClient();
    const [search, setSearch] = useState('');
    const [now, setNow] = useState(dayjs());

    useEffect(() => {
        const timer = setInterval(() => setNow(dayjs()), 1000);
        return () => clearInterval(timer);
    }, []);

    const { data: contestsData, isLoading } = useQuery({
        queryKey: ['contests'],
        queryFn: async () => {
            const res = await api.get<APIResponse<Contest[]>>('/contests', {
                params: { page_size: 1000 }
            });
            return res.data;
        }
    });

    const joinMutation = useMutation({
        mutationFn: (key: string) => api.post(`/contest/${key}/join`),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['contests'] });
        }
    });

    const allContests = contestsData?.data || [];

    const filtered = allContests.filter(c =>
        c.name.toLowerCase().includes(search.toLowerCase()) ||
        c.key.toLowerCase().includes(search.toLowerCase())
    );

    const active = filtered.filter(c => c.is_joined && now.isBefore(dayjs(c.end_time)));
    const ongoing = filtered.filter(c => !c.is_joined && now.isAfter(dayjs(c.start_time)) && now.isBefore(dayjs(c.end_time)));
    const upcoming = filtered.filter(c => now.isBefore(dayjs(c.start_time)));
    const past = filtered.filter(c => now.isAfter(dayjs(c.end_time)));

    const formatDuration = (start: string, end: string) => {
        const diff = dayjs(end).diff(dayjs(start));
        const d = dayjs.duration(diff);
        if (d.asDays() >= 1) return `${Math.floor(d.asDays())}d ${d.hours()}h`;
        return `${d.hours()}h ${d.minutes()}m`;
    };

    const ContestTable = ({ contests, title, icon: Icon, emptyMsg }: { contests: Contest[], title: string, icon: any, emptyMsg: string }) => (
        <div className="space-y-4">
            <div className="flex items-center gap-2 border-b pb-2">
                <Icon size={20} className="text-primary" />
                <h2 className="text-xl font-bold tracking-tight">{title}</h2>
                <span className="ml-auto text-xs font-bold px-2 py-0.5 bg-muted rounded-full">
                    {contests.length}
                </span>
            </div>

            {contests.length === 0 ? (
                <p className="text-sm text-muted-foreground italic py-4 px-6 bg-muted/20 rounded-xl border border-dashed text-center">
                    {emptyMsg}
                </p>
            ) : (
                <div className="border rounded-xl bg-card overflow-hidden shadow-sm">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left border-collapse text-sm">
                            <thead className="bg-muted/50 border-b text-xs font-bold uppercase tracking-wider text-muted-foreground">
                                <tr>
                                    <th className="px-6 py-3">{t('title')}</th>
                                    <th className="px-6 py-3">{t('duration')}</th>
                                    <th className="px-6 py-3 text-center">{t('score')}</th>
                                    <th className="px-6 py-3 text-center">{t('users')}</th>
                                    <th className="px-6 py-3 text-right"></th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {contests.map((c) => {
                                    const isRunning = now.isAfter(dayjs(c.start_time)) && now.isBefore(dayjs(c.end_time));

                                    return (
                                        <tr key={c.key} className="hover:bg-muted/30 transition-colors group">
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col gap-1">
                                                    <Link href={`/contests/${c.key}`} className="font-bold text-base hover:text-primary transition-colors">
                                                        {c.name}
                                                    </Link>
                                                    <div className="flex items-center gap-2 flex-wrap">
                                                        {c.is_rated && <Badge variant="warning" className="text-[10px] h-4 uppercase">{t('rated')}</Badge>}
                                                        {c.format && <Badge variant="outline" className="text-[10px] h-4 uppercase">{c.format}</Badge>}
                                                        {c.tags && c.tags.map(tag => (
                                                            <Badge
                                                                key={tag.id}
                                                                variant="secondary"
                                                                className="text-[10px] h-4 uppercase"
                                                                style={{ backgroundColor: tag.color + '20', color: tag.color, border: `1px solid ${tag.color}40` }}
                                                            >
                                                                #{tag.name}
                                                            </Badge>
                                                        ))}
                                                        <span className="text-[10px] text-muted-foreground font-mono">{c.key}</span>
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col">
                                                    <span className="font-medium whitespace-nowrap">
                                                        {formatDuration(c.start_time, c.end_time)}
                                                    </span>
                                                    <span className="text-[10px] text-muted-foreground">
                                                        {dayjs(c.start_time).format('MMM D, HH:mm')}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-center">
                                                {c.is_rated ? <Star size={14} className="mx-auto text-amber-500" fill="currentColor" /> : <span className="text-muted-foreground">-</span>}
                                            </td>
                                            <td className="px-6 py-4 text-center text-muted-foreground font-medium">
                                                <div className="flex items-center justify-center gap-1">
                                                    <Users size={14} /> {c.user_count}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    {!c.is_joined && isRunning && (
                                                        <button
                                                            onClick={() => joinMutation.mutate(c.key)}
                                                            className="bg-emerald-500 hover:bg-emerald-600 text-white px-4 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center gap-1 shadow-sm shadow-emerald-200"
                                                        >
                                                            <Play size={14} fill="currentColor" /> {t('join')}
                                                        </button>
                                                    )}
                                                    {c.is_joined && isRunning && (
                                                        <Link
                                                            href={`/contests/${c.key}`}
                                                            className="bg-primary hover:bg-primary/90 text-primary-foreground px-4 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center gap-1 shadow-sm shadow-primary/20"
                                                        >
                                                            <Trophy size={14} /> {t('enter')}
                                                        </Link>
                                                    )}
                                                    {!isRunning && !now.isAfter(dayjs(c.end_time)) && (
                                                        <Link
                                                            href={`/contests/${c.key}`}
                                                            className="border border-muted-foreground/20 hover:bg-muted px-4 py-1.5 rounded-lg text-xs font-bold transition-all"
                                                        >
                                                            {t('details')}
                                                        </Link>
                                                    )}
                                                    {now.isAfter(dayjs(c.end_time)) && (
                                                        <>
                                                            <button className="text-primary hover:text-primary/80 px-2 py-1 text-xs font-bold transition-colors border rounded-md">
                                                                {t('virtualJoin')}
                                                            </button>
                                                            <Link href={`/contests/${c.key}`} className="p-1.5 hover:bg-muted rounded-md transition-colors">
                                                                <ChevronRight size={18} />
                                                            </Link>
                                                        </>
                                                    )}
                                                </div>
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}
        </div>
    );

    return (
        <div className="max-w-6xl mx-auto space-y-12 pb-20 animate-in fade-in duration-700">
            <header className="space-y-4">
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                    <div>
                        <h1 className="text-4xl font-black tracking-tight flex items-center gap-3">
                            <Trophy className="text-primary" size={36} />
                            {t('title')}
                        </h1>
                        <p className="text-muted-foreground mt-1">Participate in challenges, win recognition, and grow your rating.</p>
                    </div>

                    <div className="flex items-center gap-3">
                        <Link
                            href="/contests/calendar"
                            className="flex items-center gap-2 px-4 py-2.5 bg-primary/10 text-primary hover:bg-primary/20 rounded-xl text-sm font-bold transition-all border border-primary/20"
                        >
                            <Calendar size={18} />
                            <span>View Calendar</span>
                        </Link>
                        <div className="relative w-full md:w-64">
                            <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/50" size={18} />
                            <input
                                type="text"
                                placeholder="Search contests..."
                                className="w-full h-12 bg-card border rounded-[1.5rem] pl-12 pr-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 transition-all outline-none"
                                value={search}
                                onChange={(e) => setSearch(e.target.value)}
                            />
                        </div>
                    </div>
                </div>
            </header>

            {isLoading ? (
                <div className="space-y-12">
                    {[1, 2, 3].map(i => (
                        <div key={i} className="space-y-4">
                            <Skeleton className="h-8 w-48" />
                            <Skeleton className="h-48 w-full rounded-2xl" />
                        </div>
                    ))}
                </div>
            ) : (
                <div className="space-y-12">
                    {active.length > 0 && (
                        <ContestTable
                            contests={active}
                            title={t('active')}
                            icon={Timer}
                            emptyMsg=""
                        />
                    )}

                    <ContestTable
                        contests={ongoing}
                        title={t('ongoing')}
                        icon={Play}
                        emptyMsg={t('noScheduled')}
                    />

                    <ContestTable
                        contests={upcoming}
                        title={t('upcoming')}
                        icon={Calendar}
                        emptyMsg={t('noScheduled')}
                    />

                    <ContestTable
                        contests={past.slice(0, 10)}
                        title={t('pastContests')}
                        icon={History}
                        emptyMsg={t('noPast')}
                    />

                    {past.length > 10 && (
                        <div className="flex justify-center pt-4">
                            <button className="text-sm font-bold text-primary flex items-center gap-1 hover:underline">
                                View all past contests <ChevronRight size={16} />
                            </button>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
