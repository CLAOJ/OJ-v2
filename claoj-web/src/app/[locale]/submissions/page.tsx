'use client';

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import api from '@/lib/api';
import { APIResponse, Submission, PaginatedList, ContestDetail } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import { useState } from 'react';
import {
    User,
    Hash,
    Building2,
    BrainCircuit,
    RefreshCw,
    Shield,
    X
} from 'lucide-react';
import { cn, getStatusVariant } from '@/lib/utils';
import { PaginationBar, PAGE_SIZE_OPTIONS } from '@/components/ui/PaginationBar';
import { buildSubmissionListParams, addFilterValue, removeFilterValue } from '@/lib/submissionFilters';
import { VERDICT_CODES } from '@/lib/submissionStatus';
import { formatMemoryMb } from '@/lib/submissionDetail';
import { useAuth } from '@/components/providers/AuthProvider';
import { useConfirm } from '@/components/providers/ConfirmProvider';
import { toast } from 'sonner';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const RESULTS = VERDICT_CODES;

type LanguageOption = { key: string; name: string; short_name?: string };

// v1 row.html:86-97: no usage figures while a submission is queued/grading or
// errored, and TLE's time is meaningless.
const NO_USAGE_STATUSES = ['QU', 'P', 'G', 'CE', 'IE', 'AB'];

function submissionTime(s: Submission): string {
    if (NO_USAGE_STATUSES.includes(s.status) || s.result === 'TLE' || s.time == null) return '-';
    return `${(s.time * 1000).toFixed(0)}ms`;
}

/** Case score as v1 prints it: integers bare, fractions to two places. */
function formatScore(n: number): string {
    return Number.isInteger(n) ? String(n) : n.toFixed(2);
}

function submissionMemory(s: Submission): string {
    return NO_USAGE_STATUSES.includes(s.status) ? '-' : formatMemoryMb(s.memory);
}

export default function SubmissionListPage() {
    const t = useTranslations('Submissions');
    const tCommon = useTranslations('Common');
    const searchParams = useSearchParams();
    const { user } = useAuth();
    const confirm = useConfirm();
    const queryClient = useQueryClient();
    const isAdmin = !!(user?.is_admin || user?.is_staff);
    const [rejudging, setRejudging] = useState<number | null>(null);
    const rejudge = async (id: number) => {
        if (!(await confirm(t('rejudgeConfirm')))) return;
        setRejudging(id);
        try {
            await api.post(`/admin/submission/${id}/rejudge`);
            queryClient.invalidateQueries({ queryKey: ['submissions'] });
        } catch {
            toast.error(t('rejudgeError'));
        } finally {
            setRejudging(null);
        }
    };
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(PAGE_SIZE_OPTIONS[1]);
    // The profile's "View submissions" link deep-links into a prefiltered list.
    // Follow the URL when it changes, not only on mount: navigating from
    // another profile while this page is mounted must reseed the filter. The
    // adjustment runs during render (React's documented pattern) so the first
    // query already uses the new value.
    const urlUser = searchParams.get('user');
    const urlBest = searchParams.get('best') === '1';
    const urlContest = searchParams.get('contest') ?? '';
    const urlProblem = searchParams.get('problem') ?? '';
    const [userFilter, setUserFilter] = useState(urlUser ?? '');
    const [problemFilter, setProblemFilter] = useState(urlProblem);
    const [resultFilters, setResultFilters] = useState<string[]>([]);
    const [langFilters, setLangFilters] = useState<string[]>([]);
    const [orgFilter, setOrgFilter] = useState('');
    // `best=1` comes from the problem page's "Best solutions" link; it is a
    // mode, not a filter the visitor edits here.
    const [bestOnly, setBestOnly] = useState(urlBest);
    const [seenUrlUser, setSeenUrlUser] = useState(urlUser);
    if (urlUser !== seenUrlUser) {
        setSeenUrlUser(urlUser);
        setUserFilter(urlUser ?? '');
        setPage(1);
    }
    const [seenUrlProblem, setSeenUrlProblem] = useState(urlProblem);
    if (urlProblem !== seenUrlProblem) {
        setSeenUrlProblem(urlProblem);
        setProblemFilter(urlProblem);
        setPage(1);
    }
    const [seenUrlBest, setSeenUrlBest] = useState(urlBest);
    if (urlBest !== seenUrlBest) {
        setSeenUrlBest(urlBest);
        setBestOnly(urlBest);
        setPage(1);
    }
    // `?contest=<key>` scopes the list to one contest (v1's contest submission
    // pages). It is a mode like best: the visitor cannot edit it here.
    const contestFilter = urlContest;

    const { data: contest } = useQuery({
        queryKey: ['contest', contestFilter],
        queryFn: async () => {
            const res = await api.get<ContestDetail>(`/contest/${contestFilter}`);
            return res.data;
        },
        enabled: !!contestFilter,
    });

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
        queryKey: ['submissions', page, pageSize, userFilter, problemFilter, resultFilters, langFilters, orgFilter, bestOnly, contestFilter],
        queryFn: async () => {
            const res = await api.get<PaginatedList<Submission>>('/submissions', {
                params: buildSubmissionListParams({
                    user: userFilter,
                    problem: problemFilter,
                    results: resultFilters,
                    languages: langFilters,
                    organization: orgFilter,
                    contest: contestFilter,
                    best: bestOnly,
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

        const filterSelectClass = 'w-full h-12 bg-muted/30 border border-transparent rounded-md px-4 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none cursor-pointer';

    return (
        <div className="max-w-7xl mx-auto space-y-4 mt-4 pb-20">
            <div className="flex flex-col md:flex-row justify-between items-end gap-6">
                <header className="space-y-2">
                    <h1 className="text-4xl md:text-5xl font-semibold tracking-tighter flex items-center gap-4">
                        <BrainCircuit className="text-muted-foreground" size={48} />
                        {t('title')}
                    </h1>
                    <p className="text-muted-foreground font-medium opacity-80">{t('subtitle')}</p>
                    {bestOnly && (
                        <Badge variant="warning" title={t('bestModeHint')}>
                            {t('bestModeBadge')}
                        </Badge>
                    )}
                    {contestFilter && (
                        <div className="flex flex-wrap items-center gap-2">
                            <Badge variant="outline" title={t('contestModeHint')}>
                                {t('contestModeBadge')}
                            </Badge>
                            <Link
                                href={`/contest/${contestFilter}`}
                                className="text-sm font-medium text-primary hover:underline"
                            >
                                {contest?.name ?? contestFilter}
                            </Link>
                        </div>
                    )}
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

                {/* v1's multi-select shape: chosen values as removable tags
                    above one closed "add" select, instead of a chip cloud. */}
                <div className="space-y-2 sm:col-span-2 lg:col-span-1">
                    <label htmlFor="verdict-filter" className="text-xs font-medium text-muted-foreground ml-1">{t('verdictLabel')}</label>
                    {resultFilters.length > 0 && (
                        <div className="flex flex-wrap gap-1.5">
                            {resultFilters.map(r => (
                                <span key={r} className="inline-flex items-center gap-1 pl-2.5 pr-1 h-7 rounded-md bg-primary/15 text-primary text-xs font-medium">
                                    {r}
                                    <button
                                        type="button"
                                        aria-label={t('removeFilter', { name: r })}
                                        onClick={() => { setResultFilters(removeFilterValue(resultFilters, r)); setPage(1); }}
                                        className="p-0.5 rounded hover:bg-primary/20 transition-colors"
                                    >
                                        <X size={12} />
                                    </button>
                                </span>
                            ))}
                        </div>
                    )}
                    <select
                        id="verdict-filter"
                        value=""
                        className={filterSelectClass}
                        onChange={(e) => {
                            if (!e.target.value) return;
                            setResultFilters(addFilterValue(resultFilters, e.target.value));
                            setPage(1);
                        }}
                    >
                        <option value="">{t('addVerdictPlaceholder')}</option>
                        {RESULTS.filter(r => !resultFilters.includes(r)).map(r => (
                            <option key={r} value={r}>{r}</option>
                        ))}
                    </select>
                </div>

                <div className="space-y-2 sm:col-span-2">
                    <label htmlFor="language-filter" className="text-xs font-medium text-muted-foreground ml-1">{t('language')}</label>
                    {langFilters.length > 0 && (
                        <div className="flex flex-wrap gap-1.5">
                            {langFilters.map(key => {
                                const lang = (languages ?? []).find(l => l.key === key);
                                const label = lang?.short_name || key;
                                return (
                                    <span key={key} title={lang?.name} className="inline-flex items-center gap-1 pl-2.5 pr-1 h-7 rounded-md bg-primary/15 text-primary text-xs font-medium">
                                        {label}
                                        <button
                                            type="button"
                                            aria-label={t('removeFilter', { name: label })}
                                            onClick={() => { setLangFilters(removeFilterValue(langFilters, key)); setPage(1); }}
                                            className="p-0.5 rounded hover:bg-primary/20 transition-colors"
                                        >
                                            <X size={12} />
                                        </button>
                                    </span>
                                );
                            })}
                        </div>
                    )}
                    <select
                        id="language-filter"
                        value=""
                        className={filterSelectClass}
                        onChange={(e) => {
                            if (!e.target.value) return;
                            setLangFilters(addFilterValue(langFilters, e.target.value));
                            setPage(1);
                        }}
                    >
                        <option value="">{t('addLanguagePlaceholder')}</option>
                        {(languages ?? []).filter(l => !langFilters.includes(l.key)).map(l => (
                            <option key={l.key} value={l.key}>{l.short_name || l.key}</option>
                        ))}
                    </select>
                </div>
            </div>

            <div className="bg-card border rounded-md overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse text-sm tabular-nums">
                        <thead>
                            <tr className="bg-muted/30 border-b text-xs font-medium text-muted-foreground">
                                <th className="px-3 py-2">{tCommon('status')}</th>
                                <th className="px-3 py-2">{t('problem')}</th>
                                <th className="px-3 py-2">{t('user')}</th>
                                <th className="hidden sm:table-cell px-3 py-2 text-right">{t('time')}</th>
                                <th className="hidden md:table-cell px-3 py-2 text-right">{t('memory')}</th>
                                <th className="px-3 py-2 text-right">{t('date')}</th>
                                {isAdmin && <th className="px-3 py-2 text-right"><span className="sr-only">{t('adminActions')}</span></th>}
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                            {isLoading ? (
                                Array.from({ length: 15 }).map((_, i) => (
                                    <tr key={i}>
                                        <td colSpan={isAdmin ? 7 : 6} className="px-3 py-2"><Skeleton className="h-5 w-full rounded" /></td>
                                    </tr>
                                ))
                            ) : (
                                submissions.map((s) => {
                                    const verdict = s.result || s.status;
                                    const lang = languageNames.get(s.language);
                                    return (
                                        <tr key={s.id} className="h-10 hover:bg-muted/40 transition-colors">
                                            {/* v1 row: verdict, case score and language read as one unit. */}
                                            <td className="px-3 whitespace-nowrap">
                                                <Link href={`/submission/${s.id}`} className="inline-flex items-center gap-2 hover:underline underline-offset-4">
                                                    <Badge variant={getStatusVariant(verdict)} className="min-w-10 justify-center px-1.5 py-0.5 rounded text-xs font-semibold">
                                                        {verdict}
                                                    </Badge>
                                                    <span className="text-xs text-muted-foreground">
                                                        {s.case_total ? `${formatScore(s.case_points)}/${formatScore(s.case_total)} · ` : ''}
                                                        <span title={lang?.name}>{lang?.short_name || s.language}</span>
                                                    </span>
                                                </Link>
                                            </td>
                                            <td className="px-3 max-w-0 w-full">
                                                <Link href={`/problem/${s.problem}`} className="block truncate font-medium hover:text-primary hover:underline underline-offset-4" title={s.problem}>
                                                    {s.problem_name || s.problem}
                                                </Link>
                                            </td>
                                            <td className="px-3 whitespace-nowrap">
                                                <Link href={`/user/${s.user}`} className="font-medium hover:text-primary hover:underline underline-offset-4">{s.user}</Link>
                                            </td>
                                            <td className="hidden sm:table-cell px-3 text-right text-xs text-muted-foreground whitespace-nowrap">{submissionTime(s)}</td>
                                            <td className="hidden md:table-cell px-3 text-right text-xs text-muted-foreground whitespace-nowrap">{submissionMemory(s)}</td>
                                            <td className="px-3 text-right text-xs text-muted-foreground whitespace-nowrap" title={dayjs(s.date).format('YYYY-MM-DD HH:mm:ss')}>
                                                {dayjs(s.date).fromNow()}
                                            </td>
                                            {isAdmin && (
                                                <td className="px-3 text-right whitespace-nowrap">
                                                    <span className="inline-flex items-center gap-1">
                                                        <button
                                                            type="button"
                                                            onClick={() => rejudge(s.id)}
                                                            disabled={rejudging === s.id}
                                                            aria-label={`${t('rejudge')} #${s.id}`}
                                                            title={t('rejudge')}
                                                            className="p-1.5 rounded text-muted-foreground hover:text-foreground hover:bg-muted disabled:opacity-50"
                                                        >
                                                            <RefreshCw size={14} className={cn(rejudging === s.id && 'animate-spin')} />
                                                        </button>
                                                        <Link
                                                            href={`/admin/submissions/${s.id}`}
                                                            aria-label={`${t('adminActions')} #${s.id}`}
                                                            title={t('adminActions')}
                                                            className="p-1.5 rounded text-muted-foreground hover:text-foreground hover:bg-muted"
                                                        >
                                                            <Shield size={14} />
                                                        </Link>
                                                    </span>
                                                </td>
                                            )}
                                        </tr>
                                    );
                                })
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            <PaginationBar
                page={page}
                onPageChange={setPage}
                total={data?.total}
                hasMore={data?.has_more}
                pageSize={pageSize}
                onPageSizeChange={size => { setPageSize(size); setPage(1); }}
            />
        </div>
    );
}