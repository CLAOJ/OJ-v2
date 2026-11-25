'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import api from '@/lib/api';
import { APIResponse, Submission, PaginatedList } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import { useState } from 'react';
import {
    User,
    Hash,
    Building2,
    BrainCircuit,
    RefreshCw
} from 'lucide-react';
import { cn, getStatusVariant } from '@/lib/utils';
import { PaginationBar, PAGE_SIZE_OPTIONS } from '@/components/ui/PaginationBar';
import { buildSubmissionListParams } from '@/lib/submissionFilters';
import { VERDICT_CODES } from '@/lib/submissionStatus';
import { formatMemoryMb } from '@/lib/submissionDetail';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const RESULTS = VERDICT_CODES;

type LanguageOption = { key: string; name: string; short_name?: string };

/** Toggle one value in a chip-list filter. */
function toggleValue(values: string[], value: string): string[] {
    return values.includes(value) ? values.filter(v => v !== value) : [...values, value];
}

// v1 row.html:86-97: no usage figures while a submission is queued/grading or
// errored, and TLE's time is meaningless.
const NO_USAGE_STATUSES = ['QU', 'P', 'G', 'CE', 'IE', 'AB'];

function submissionTime(s: Submission): string {
    if (NO_USAGE_STATUSES.includes(s.status) || s.result === 'TLE' || s.time == null) return '-';
    return `${(s.time * 1000).toFixed(0)}ms`;
}

function submissionMemory(s: Submission): string {
    return NO_USAGE_STATUSES.includes(s.status) ? '-' : formatMemoryMb(s.memory);
}

export default function SubmissionListPage() {
    const t = useTranslations('Submissions');
    const tCommon = useTranslations('Common');
    const searchParams = useSearchParams();
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(PAGE_SIZE_OPTIONS[1]);
    // The profile's "View submissions" link deep-links into a prefiltered list.
    // Follow the URL when it changes, not only on mount: navigating from
    // another profile while this page is mounted must reseed the filter. The
    // adjustment runs during render (React's documented pattern) so the first
    // query already uses the new value.
    const urlUser = searchParams.get('user');
    const [userFilter, setUserFilter] = useState(urlUser ?? '');
    const [problemFilter, setProblemFilter] = useState('');
    const [resultFilters, setResultFilters] = useState<string[]>([]);
    const [langFilters, setLangFilters] = useState<string[]>([]);
    const [orgFilter, setOrgFilter] = useState('');
    const [seenUrlUser, setSeenUrlUser] = useState(urlUser);
    if (urlUser !== seenUrlUser) {
        setSeenUrlUser(urlUser);
        setUserFilter(urlUser ?? '');
        setPage(1);
    }

    const { data: languages } = useQuery({
        queryKey: ['languages'],
        queryFn: async () => {
            const res = await api.get<APIResponse<LanguageOption[]>>('/languages');
            return res.data.data;
        }
    });
    // Friendly names for the language column: v1 prints the short name.
    const languageNames = new Map((languages ?? []).map(l => [l.key, l]));

    const { data: organizations } = useQuery({
        queryKey: ['organizations', 'filter'],
        queryFn: async () => {
            const res = await api.get<PaginatedList<{ id: number; name: string }>>('/organizations', {
                params: { page_size: 100 }
            });
            return res.data.data;
        }
    });

    const { data, isLoading } = useQuery({
        queryKey: ['submissions', page, pageSize, userFilter, problemFilter, resultFilters, langFilters, orgFilter],
        queryFn: async () => {
            const res = await api.get<PaginatedList<Submission>>('/submissions', {
                params: buildSubmissionListParams({
                    user: userFilter,
                    problem: problemFilter,
                    results: resultFilters,
                    languages: langFilters,
                    organization: orgFilter,
                    page,
                    pageSize,
                })
            });
            return res.data;
        }
    });

    const submissions = data?.data || [];

    const resetFilters = () => {
        setUserFilter('');
        setProblemFilter('');
        setResultFilters([]);
        setLangFilters([]);
        setOrgFilter('');
        setPage(1);
    };

    const chipClass = (active: boolean) => cn(
        'px-2.5 py-1 rounded-md border text-xs font-medium cursor-pointer select-none transition-colors',
        active
            ? 'bg-primary/10 border-primary/40 text-primary'
            : 'bg-muted/30 border-transparent hover:bg-muted text-muted-foreground'
    );

    return (
        <div className="max-w-7xl mx-auto space-y-4 mt-4 pb-20">
            <div className="flex flex-col md:flex-row justify-between items-end gap-6">
                <header className="space-y-2">
                    <h1 className="text-4xl md:text-5xl font-semibold tracking-tighter flex items-center gap-4">
                        <BrainCircuit className="text-muted-foreground" size={48} />
                        {t('title')}
                    </h1>
                    <p className="text-muted-foreground font-medium opacity-80">{t('subtitle')}</p>
                </header>

                <div className="flex flex-wrap items-center gap-3">
                    <button
                        onClick={resetFilters}
                        className="h-9 px-6 rounded-md bg-muted/30 border hover:bg-muted font-medium text-xs flex items-center gap-2"
                    >
                        <RefreshCw size={14} /> {tCommon('reset')}
                    </button>
                </div>
            </div>

            {/* Advanced Filters Bar */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 p-6 rounded-md bg-card border shadow-sm">
                <div className="space-y-2">
                    <label className="text-xs font-medium text-muted-foreground ml-1">{t('usernameLabel')}</label>
                    <div className="relative">
                        <User className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <input
                            type="text"
                            placeholder={t('searchUserPlaceholder')}
                            className="w-full h-12 bg-muted/30 border border-transparent rounded-md pl-12 pr-4 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none"
                            value={userFilter}
                            onChange={(e) => { setUserFilter(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-xs font-medium text-muted-foreground ml-1">{t('problemCodeLabel')}</label>
                    <div className="relative">
                        <Hash className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <input
                            type="text"
                            placeholder={t('problemCodePlaceholder')}
                            className="w-full h-12 bg-muted/30 border border-transparent rounded-md pl-12 pr-4 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none"
                            value={problemFilter}
                            onChange={(e) => { setProblemFilter(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-xs font-medium text-muted-foreground ml-1">{t('organizationLabel')}</label>
                    <div className="relative">
                        <Building2 className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <select
                            className="w-full h-12 bg-muted/30 border border-transparent rounded-md pl-12 pr-4 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none cursor-pointer"
                            value={orgFilter}
                            onChange={(e) => { setOrgFilter(e.target.value); setPage(1); }}
                        >
                            <option value="">{t('allOrganizationsOption')}</option>
                            {organizations?.map(o => <option key={o.id} value={o.id}>{o.name}</option>)}
                        </select>
                    </div>
                </div>

                <div className="space-y-2 sm:col-span-2 lg:col-span-1">
                    <label className="text-xs font-medium text-muted-foreground ml-1">{t('verdictLabel')}</label>
                    <div className="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto py-1">
                        {RESULTS.map(r => (
                            <label key={r} className={chipClass(resultFilters.includes(r))}>
                                <input
                                    type="checkbox"
                                    className="sr-only"
                                    checked={resultFilters.includes(r)}
                                    onChange={() => { setResultFilters(prev => toggleValue(prev, r)); setPage(1); }}
                                />
                                {r}
                            </label>
                        ))}
                    </div>
                </div>

                <div className="space-y-2 sm:col-span-2">
                    <label className="text-xs font-medium text-muted-foreground ml-1">{t('language')}</label>
                    <div className="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto py-1">
                        {(languages ?? []).map(l => (
                            <label key={l.key} className={chipClass(langFilters.includes(l.key))} title={l.name}>
                                <input
                                    type="checkbox"
                                    className="sr-only"
                                    checked={langFilters.includes(l.key)}
                                    onChange={() => { setLangFilters(prev => toggleValue(prev, l.key)); setPage(1); }}
                                />
                                {l.short_name || l.key}
                            </label>
                        ))}
                    </div>
                </div>
            </div>

            <div className="bg-card border rounded-md overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                        <thead>
                            <tr className="bg-muted/30 border-b">
                                <th className="px-10 py-6 text-xs font-medium text-muted-foreground w-40">{t('user')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground">{t('problem')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground w-40 text-center">{tCommon('status')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-24">{t('score')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-28">{t('time')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-28">{t('memory')}</th>
                                <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-32">{t('language')}</th>
                                <th className="px-10 py-6 text-xs font-medium text-muted-foreground text-right w-40">{t('date')}</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {isLoading ? (
                                Array.from({ length: 15 }).map((_, i) => (
                                    <tr key={i}>
                                        <td colSpan={8} className="px-10 py-6"><Skeleton className="h-10 w-full rounded-md" /></td>
                                    </tr>
                                ))
                            ) : (
                                submissions.map((s) => (
                                    <tr key={s.id} className="hover:bg-muted/10 transition-colors group">
                                        <td className="px-10 py-8">
                                            <Link href={`/user/${s.user}`} className="flex items-center gap-3 group/user outline-none">
                                                <div className="w-9 h-9 rounded-xl bg-muted flex items-center justify-center text-muted-foreground font-medium text-xs group-hover/user:bg-primary/10 group-hover/user:text-primary transition-all">
                                                    {s.user[0]?.toUpperCase()}
                                                </div>
                                                <span className="font-medium text-sm group-hover/user:text-primary transition-colors">{s.user}</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-8">
                                            <Link href={`/problem/${s.problem}`} className="inline-flex flex-col gap-0.5 outline-none hover:text-primary transition-colors">
                                                <span className="font-medium text-sm tracking-tight">{s.problem_name || s.problem}</span>
                                                <span className="text-xs uppercase font-mono font-medium text-muted-foreground opacity-40">{s.problem}</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            <Link href={`/submission/${s.id}`} className="outline-none">
                                                <Badge
                                                    variant={getStatusVariant(s.result || s.status)}
                                                    className="px-4 py-1.5 rounded text-xs font-medium uppercase shadow-sm"
                                                >
                                                    {s.result || s.status}
                                                </Badge>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            <div className={cn(
                                                "inline-flex items-center justify-center min-w-12 h-8 px-2 rounded-xl font-medium text-xs border tracking-tighter",
                                                s.result === 'AC' ? "bg-emerald-500/10 text-emerald-500 border-emerald-500/20 shadow-sm" :
                                                    (s.points ?? 0) > 0 ? "bg-amber-500/10 text-amber-500 border-amber-500/20" :
                                                        "bg-muted/50 text-muted-foreground border-transparent"
                                            )}>
                                                {s.points != null ? (Number.isInteger(s.points) ? s.points : s.points.toFixed(2)) : '-'}
                                            </div>
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            <span className="text-xs font-mono text-muted-foreground">
                                                {submissionTime(s)}
                                            </span>
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            <span className="text-xs font-mono text-muted-foreground">
                                                {submissionMemory(s)}
                                            </span>
                                        </td>
                                        <td className="px-6 py-8 text-center">
                                            <span
                                                className="text-xs font-medium text-muted-foreground opacity-60"
                                                title={languageNames.get(s.language)?.name}
                                            >
                                                {languageNames.get(s.language)?.short_name || s.language}
                                            </span>
                                        </td>
                                        <td className="px-10 py-8 text-right">
                                            <div className="flex flex-col items-end gap-0.5">
                                                <span className="text-xs font-medium text-muted-foreground opacity-80">{dayjs(s.date).fromNow()}</span>
                                                <span className="text-xs font-mono text-muted-foreground opacity-30">{dayjs(s.date).format('HH:mm:ss')}</span>
                                            </div>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            <PaginationBar
                page={page}
                onPageChange={setPage}
                total={data?.total}
                pageSize={pageSize}
                onPageSizeChange={size => { setPageSize(size); setPage(1); }}
            />
        </div>
    );
}