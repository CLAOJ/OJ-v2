'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { APIResponse, Submission, PaginatedList } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import { useState } from 'react';
import {
    ChevronLeft,
    ChevronRight,
    User,
    Hash,
    Code2,
    Clock,
    HardDrive,
    Calendar,
    Filter,
    RefreshCw,
    Search,
    BrainCircuit
} from 'lucide-react';
import { cn, getStatusColor, getStatusVariant } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

export default function SubmissionListPage() {
    const t = useTranslations('Submissions');
    const [page, setPage] = useState(1);
    const [userFilter, setUserFilter] = useState('');
    const [problemFilter, setProblemFilter] = useState('');
    const [resultFilter, setResultFilter] = useState('');
    const [langFilter, setLangFilter] = useState('');

    const { data: languages } = useQuery({
        queryKey: ['languages'],
        queryFn: async () => {
            const res = await api.get<APIResponse<{ key: string, name: string }[]>>('/languages');
            return res.data.data;
        }
    });

    const { data, isLoading } = useQuery({
        queryKey: ['submissions', page, userFilter, problemFilter, resultFilter, langFilter],
        queryFn: async () => {
            const res = await api.get<APIResponse<Submission[]>>('/submissions', {
                params: {
                    page,
                    page_size: 50,
                    user: userFilter,
                    problem: problemFilter,
                    result: resultFilter,
                    language: langFilter
                }
            });
            return res.data;
        }
    });

    const submissions = data?.data || [];

    const results = ['AC', 'WA', 'TLE', 'MLE', 'OLE', 'RE', 'IE', 'CE', 'AB'];

    return (
        <div className="max-w-7xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            <div className="flex flex-col md:flex-row justify-between items-end gap-6">
                <header className="space-y-2">
                    <h1 className="text-4xl md:text-5xl font-black tracking-tighter flex items-center gap-4">
                        <BrainCircuit className="text-primary" size={48} />
                        {t('title')}
                    </h1>
                    <p className="text-muted-foreground font-black opacity-80">Real-time judging results from the globally distributed nodes.</p>
                </header>

                <div className="flex flex-wrap items-center gap-3 bg-muted/30 p-4 rounded-[2.5rem] border border-dashed">
                    <div className="flex flex-col gap-1">
                        <span className="text-[10px] font-black uppercase text-muted-foreground ml-1">Page</span>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setPage(p => Math.max(1, p - 1))}
                                disabled={page === 1}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-20 transition-all"
                            >
                                <ChevronLeft size={18} />
                            </button>
                            <div className="h-10 px-4 rounded-xl bg-primary text-primary-foreground font-black text-xs flex items-center shadow-lg shadow-primary/20">
                                {page}
                            </div>
                            <button
                                onClick={() => setPage(p => p + 1)}
                                disabled={submissions.length < 50}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-20 transition-all"
                            >
                                <ChevronRight size={18} />
                            </button>
                        </div>
                    </div>

                    <button
                        onClick={() => {
                            setUserFilter('');
                            setProblemFilter('');
                            setResultFilter('');
                            setLangFilter('');
                            setPage(1);
                        }}
                        className="h-10 px-6 rounded-xl bg-muted/50 hover:bg-muted font-black text-[10px] uppercase tracking-widest flex items-center gap-2 items-end mt-auto"
                    >
                        <RefreshCw size={14} /> Reset
                    </button>
                </div>
            </div>

            {/* Advanced Filters Bar */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 p-6 rounded-[2.5rem] bg-card border shadow-sm">
                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Username</label>
                    <div className="relative">
                        <User className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <input
                            type="text"
                            placeholder="Search user..."
                            className="w-full h-12 bg-muted/30 border border-transparent rounded-2xl pl-12 pr-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none"
                            value={userFilter}
                            onChange={(e) => { setUserFilter(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Problem Code</label>
                    <div className="relative">
                        <Hash className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <input
                            type="text"
                            placeholder="P01, APB..."
                            className="w-full h-12 bg-muted/30 border border-transparent rounded-2xl pl-12 pr-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none"
                            value={problemFilter}
                            onChange={(e) => { setProblemFilter(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Verdict</label>
                    <select
                        className="w-full h-12 bg-muted/30 border border-transparent rounded-2xl px-4 text-sm font-black focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none cursor-pointer"
                        value={resultFilter}
                        onChange={(e) => { setResultFilter(e.target.value); setPage(1); }}
                    >
                        <option value="">All Results</option>
                        {results.map(r => <option key={r} value={r}>{r}</option>)}
                    </select>
                </div>

                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Language</label>
                    <select
                        className="w-full h-12 bg-muted/30 border border-transparent rounded-2xl px-4 text-sm font-black focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none cursor-pointer"
                        value={langFilter}
                        onChange={(e) => { setLangFilter(e.target.value); setPage(1); }}
                    >
                        <option value="">All Languages</option>
                        {languages?.map(l => <option key={l.key} value={l.key}>{l.name}</option>)}
                    </select>
                </div>
            </div>

            <div className="bg-card border rounded-[3rem] overflow-hidden shadow-2xl shadow-primary/5">
                <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                        <thead>
                            <tr className="bg-muted/30 border-b">
                                <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground w-40">{t('user')}</th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">{t('problem')}</th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground w-40 text-center">{t('status')}</th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center w-24">{t('score')}</th>
                                <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center w-32">{t('language')}</th>
                                <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-right w-40">{t('date')}</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {isLoading ? (
                                Array.from({ length: 15 }).map((_, i) => (
                                    <tr key={i}>
                                        <td colSpan={6} className="px-10 py-6"><Skeleton className="h-10 w-full rounded-2xl" /></td>
                                    </tr>
                                ))
                            ) : (
                                submissions.map((s) => (
                                    <tr key={s.id} className="hover:bg-muted/10 transition-colors group">
                                        <td className="px-10 py-8">
                                            <Link href={`/user/${s.user}`} className="flex items-center gap-3 group/user outline-none">
                                                <div className="w-9 h-9 rounded-xl bg-muted flex items-center justify-center text-muted-foreground font-black text-xs group-hover/user:bg-primary/10 group-hover/user:text-primary transition-all">
                                                    {s.user[0]?.toUpperCase()}
                                                </div>
                                                <span className="font-black text-sm group-hover/user:text-primary transition-colors">{s.user}</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-8">
                                            <Link href={`/problems/${s.problem}`} className="inline-flex flex-col gap-0.5 outline-none hover:text-primary transition-colors">
                                                <span className="font-black text-sm tracking-tight">{s.problem}</span>
                                                <span className="text-[10px] font-black uppercase text-muted-foreground opacity-40">Problem Code</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            <Link href={`/submissions/${s.id}`} className="outline-none">
                                                <Badge
                                                    variant={getStatusVariant(s.status)}
                                                    className="px-4 py-1.5 rounded-full text-[10px] font-black uppercase tracking-widest shadow-sm hover:scale-105 transition-transform"
                                                >
                                                    {s.status}
                                                </Badge>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            <div className={cn(
                                                "inline-flex items-center justify-center w-12 h-8 rounded-xl font-black text-xs border tracking-tighter",
                                                s.score === 100 ? "bg-emerald-500/10 text-emerald-500 border-emerald-500/20 shadow-sm" :
                                                    s.score > 0 ? "bg-amber-500/10 text-amber-500 border-amber-500/20" :
                                                        "bg-muted/50 text-muted-foreground border-transparent"
                                            )}>
                                                {s.score !== null ? Math.round(s.score) : '-'}
                                            </div>
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground opacity-60">
                                                {s.language}
                                            </span>
                                        </td>
                                        <td className="px-10 py-8 text-right">
                                            <div className="flex flex-col items-end gap-0.5">
                                                <span className="text-[10px] font-black text-muted-foreground uppercase opacity-80">{dayjs(s.date).fromNow()}</span>
                                                <span className="text-[9px] font-mono text-muted-foreground opacity-30">{dayjs(s.date).format('HH:mm:ss')}</span>
                                            </div>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    );
}
