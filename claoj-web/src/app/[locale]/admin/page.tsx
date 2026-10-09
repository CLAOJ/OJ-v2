'use client';

import { useQuery, useQueries } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { Link } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import {
    adminUserApi,
    adminProblemApi,
    adminContestApi,
    adminOrganizationApi,
    adminSubmissionApi,
    adminTicketApi,
    adminJudgeApi,
    adminAuditLogApi,
} from '@/lib/adminApi';
import { formatDate } from '@/lib/date';
import { cn, getStatusColor } from '@/lib/utils';
import { statusKey } from '@/lib/submissionStatus';
import { Skeleton } from '@/components/ui/Skeleton';
import { Code2, Globe, BookOpen, ChevronRight } from 'lucide-react';

// Live panels poll; inventory totals load once per visit.
const REFRESH_MS = 30_000;

function Panel({ title, href, linkLabel, children, className }: {
    title: string;
    href?: string;
    linkLabel?: string;
    children: React.ReactNode;
    className?: string;
}) {
    return (
        <section className={cn('rounded-md border bg-card overflow-hidden', className)}>
            <header className="flex items-center justify-between gap-2 px-4 py-2.5 border-b bg-secondary/60">
                <h2 className="text-sm font-semibold">{title}</h2>
                {href && (
                    <Link href={href} className="text-xs text-accent hover:underline flex items-center gap-0.5">
                        {linkLabel}
                        <ChevronRight size={14} />
                    </Link>
                )}
            </header>
            {children}
        </section>
    );
}

function Empty({ children }: { children: React.ReactNode }) {
    return <p className="px-4 py-6 text-sm text-muted-foreground text-center">{children}</p>;
}

function Rows({ n }: { n: number }) {
    return (
        <div className="p-4 space-y-2">
            {Array.from({ length: n }, (_, i) => <Skeleton key={i} className="h-5 w-full" />)}
        </div>
    );
}

export default function AdminDashboardPage() {
    const t = useTranslations('Admin');
    const tSub = useTranslations('Submissions');
    const { user } = useAuth();

    const sources = [
        { key: 'problems', href: '/admin/problems', fetch: () => adminProblemApi.list(1, 1) },
        { key: 'contests', href: '/admin/contests', fetch: () => adminContestApi.list(1, 1) },
        { key: 'users', href: '/admin/users', fetch: () => adminUserApi.list(1, 1) },
        { key: 'organizations', href: '/admin/organizations', fetch: () => adminOrganizationApi.list(1, 1) },
    ];
    const counts = useQueries({
        queries: sources.map(s => ({
            queryKey: ['admin-dashboard', s.key],
            queryFn: async () => (await s.fetch()).data.total,
        })),
    });

    const submissions = useQuery({
        queryKey: ['admin-dashboard', 'submissions'],
        queryFn: async () => (await adminSubmissionApi.list(1, 10)).data.data,
        refetchInterval: REFRESH_MS,
    });

    const tickets = useQuery({
        queryKey: ['admin-dashboard', 'tickets'],
        queryFn: async () => (await adminTicketApi.list(1, 5, { status: 'open' })).data,
        refetchInterval: REFRESH_MS,
    });

    const judges = useQuery({
        queryKey: ['admin-dashboard', 'judges'],
        queryFn: async () => (await adminJudgeApi.list(1, 100)).data.data,
        refetchInterval: REFRESH_MS,
    });

    // Every GET is audited too; the dashboard only cares about changes.
    const changes = useQuery({
        queryKey: ['admin-dashboard', 'audit'],
        queryFn: async () => (await adminAuditLogApi.list(1, 100)).data.results.filter(e => e.action !== 'VIEW').slice(0, 8),
        refetchInterval: REFRESH_MS,
    });

    const inventory = sources.map((s, i) => ({ label: t(`nav.${s.key}`), href: s.href, q: counts[i] }));

    const judgeList = judges.data ?? [];
    const onlineCount = judgeList.filter(j => j.online && !j.is_disabled && !j.is_blocked).length;

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold">{t('dashboard.title')}</h1>
                    <p className="text-sm text-muted-foreground mt-1">
                        {t('dashboard.welcome', { username: user?.username ?? '' })}
                    </p>
                </div>
                <div className="flex flex-wrap gap-2">
                    {[
                        { href: '/admin/problems/create', label: t('dashboard.newProblem'), icon: Code2 },
                        { href: '/admin/contests/create', label: t('dashboard.newContest'), icon: Globe },
                        { href: '/admin/blog-posts', label: t('dashboard.newPost'), icon: BookOpen },
                    ].map((a, i) => (
                        <Link
                            key={a.href}
                            href={a.href}
                            className={cn(
                                'inline-flex items-center gap-1.5 h-9 px-3 rounded-md text-sm font-medium transition-colors',
                                i === 0
                                    ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                                    : 'border bg-card hover:bg-muted'
                            )}
                        >
                            <a.icon size={16} />
                            {a.label}
                        </Link>
                    ))}
                </div>
            </div>

            {/* Inventory */}
            <div className="grid grid-cols-2 lg:grid-cols-4 rounded-md border bg-card divide-x divide-y lg:divide-y-0">
                {inventory.map(item => (
                    <Link key={item.href} href={item.href} className="group px-4 py-3 hover:bg-muted/60 transition-colors">
                        <div className="text-xs text-muted-foreground">{item.label}</div>
                        <div className="mt-0.5 text-2xl font-semibold tabular-nums">
                            {item.q.isLoading ? <Skeleton className="h-8 w-16" /> : item.q.isError ? '—' : item.q.data?.toLocaleString()}
                        </div>
                    </Link>
                ))}
            </div>

            <div className="grid gap-6 lg:grid-cols-3">
                {/* Recent submissions */}
                <Panel
                    title={t('dashboard.recentSubmissions')}
                    href="/admin/submissions"
                    linkLabel={t('dashboard.viewAll')}
                    className="lg:col-span-2"
                >
                    {submissions.isLoading ? <Rows n={8} /> : !submissions.data?.length ? (
                        <Empty>{t('dashboard.noSubmissions')}</Empty>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm">
                                <tbody className="divide-y">
                                    {submissions.data.map(s => {
                                        const code = s.result ?? s.status;
                                        return (
                                            <tr key={s.id} className="hover:bg-muted/40">
                                                <td className="px-4 py-2 tabular-nums">
                                                    <Link href={`/submission/${s.id}`} className="text-accent hover:underline">#{s.id}</Link>
                                                </td>
                                                <td className="px-2 py-2 font-medium truncate max-w-[10rem]">
                                                    <Link href={`/user/${s.user}`} className="hover:underline">{s.user}</Link>
                                                </td>
                                                <td className="px-2 py-2 truncate max-w-[12rem]">
                                                    <Link href={`/problem/${s.problem}`} className="hover:underline">{s.problem}</Link>
                                                </td>
                                                <td className={cn('px-2 py-2 font-semibold whitespace-nowrap', getStatusColor(code))}>
                                                    {tSub(`status.${statusKey(code)}`)}
                                                </td>
                                                <td className="px-4 py-2 text-right text-muted-foreground whitespace-nowrap">
                                                    {formatDate(s.date)}
                                                </td>
                                            </tr>
                                        );
                                    })}
                                </tbody>
                            </table>
                        </div>
                    )}
                </Panel>

                <div className="space-y-6">
                    {/* Judges */}
                    <Panel
                        title={judges.data ? t('dashboard.judgesOnline', { online: onlineCount, total: judgeList.length }) : t('nav.judges')}
                        href="/admin/judges"
                        linkLabel={t('dashboard.manage')}
                    >
                        {judges.isLoading ? <Rows n={3} /> : !judgeList.length ? (
                            <Empty>{t('dashboard.noJudges')}</Empty>
                        ) : (
                            <ul className="divide-y text-sm">
                                {judgeList.map(j => {
                                    const state = j.is_blocked ? 'blocked' : j.is_disabled ? 'disabled' : j.online ? 'online' : 'offline';
                                    return (
                                        <li key={j.id}>
                                            <Link href={`/admin/judges/${j.id}`} className="flex items-center gap-2.5 px-4 py-2 hover:bg-muted/40">
                                                <span
                                                    aria-hidden
                                                    className={cn('w-2 h-2 rounded-full shrink-0', {
                                                        'bg-success': state === 'online',
                                                        'bg-muted-foreground/40': state === 'offline',
                                                        'bg-warning': state === 'disabled',
                                                        'bg-destructive': state === 'blocked',
                                                    })}
                                                />
                                                <span className="font-medium truncate flex-1">{j.name}</span>
                                                <span className="text-xs text-muted-foreground tabular-nums">
                                                    {state === 'online' && j.load != null ? t('dashboard.load', { load: j.load.toFixed(2) }) : t(`judges.${state}`)}
                                                </span>
                                            </Link>
                                        </li>
                                    );
                                })}
                            </ul>
                        )}
                    </Panel>

                    {/* Open tickets */}
                    <Panel
                        title={tickets.data ? t('dashboard.openTickets', { count: tickets.data.total }) : t('nav.tickets')}
                        href="/admin/tickets"
                        linkLabel={t('dashboard.viewAll')}
                    >
                        {tickets.isLoading ? <Rows n={3} /> : !tickets.data?.items.length ? (
                            <Empty>{t('dashboard.noOpenTickets')}</Empty>
                        ) : (
                            <ul className="divide-y text-sm">
                                {tickets.data.items.map(tk => (
                                    <li key={tk.id}>
                                        <Link href={`/admin/tickets/${tk.id}`} className="block px-4 py-2 hover:bg-muted/40">
                                            <div className="font-medium truncate">{tk.title}</div>
                                            <div className="text-xs text-muted-foreground">
                                                {tk.user} · {formatDate(tk.created)}
                                                {!tk.assignees.length && <span className="text-warning"> · {t('dashboard.unassigned')}</span>}
                                            </div>
                                        </Link>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </Panel>
                </div>
            </div>

            {/* Recent changes */}
            <Panel title={t('dashboard.recentChanges')} href="/admin/audit-logs" linkLabel={t('dashboard.viewAll')}>
                {changes.isLoading ? <Rows n={4} /> : changes.isError ? (
                    <Empty>{t('auditLog.unavailable')}</Empty>
                ) : !changes.data?.length ? (
                    <Empty>{t('dashboard.noChanges')}</Empty>
                ) : (
                    <ul className="divide-y text-sm">
                        {changes.data.map(e => (
                            <li key={e.id} className="flex flex-wrap items-baseline gap-x-2 px-4 py-2">
                                <span className="font-medium">{e.username}</span>
                                <span className={cn('text-xs font-semibold', e.status === 'failure' ? 'text-destructive' : 'text-muted-foreground')}>
                                    {t(`auditLog.actions.${e.action}`)}
                                </span>
                                <span className="font-mono text-xs">{e.resource}{e.resource_id && ` #${e.resource_id}`}</span>
                                <span className="ml-auto text-xs text-muted-foreground">{formatDate(e.created_at)}</span>
                            </li>
                        ))}
                    </ul>
                )}
            </Panel>
        </div>
    );
}
