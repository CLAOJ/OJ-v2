'use client';

import { Fragment, useState } from 'react';
import { useQuery, keepPreviousData } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { History, ChevronDown, ChevronRight } from 'lucide-react';
import { adminAuditLogApi, type AuditLogFilters } from '@/lib/adminApi';
import { formatDate } from '@/lib/date';
import { cn } from '@/lib/utils';
import { Skeleton } from '@/components/ui/Skeleton';
import { PaginationBar } from '@/components/ui/PaginationBar';

const ACTIONS = ['CREATE', 'UPDATE', 'DELETE', 'VIEW'] as const;
const field = 'h-9 px-3 rounded-md border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-ring/40';

function prettyDetails(raw: string): string {
    try {
        return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
        return raw;
    }
}

export default function AdminAuditLogPage() {
    const t = useTranslations('Admin');
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(50);
    // Changes are what staff audit; reads are opt-in.
    const [filters, setFilters] = useState<AuditLogFilters>({ action: 'UPDATE' });
    const [open, setOpen] = useState<string | null>(null);

    const setFilter = (k: keyof AuditLogFilters, v: string) => {
        setFilters(f => ({ ...f, [k]: v || undefined }));
        setPage(1);
    };

    const { data, isLoading, isError, isFetching } = useQuery({
        queryKey: ['admin-audit-logs', page, pageSize, filters],
        queryFn: async () => (await adminAuditLogApi.list(page, pageSize, filters)).data,
        placeholderData: keepPreviousData,
    });

    const entries = data?.results ?? [];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold flex items-center gap-2.5">
                    <History className="text-primary" size={26} />
                    {t('auditLog.title')}
                </h1>
                <p className="text-sm text-muted-foreground mt-1">{t('auditLog.subtitle')}</p>
            </div>

            <div className="flex flex-wrap items-end gap-3">
                <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                    {t('auditLog.columns.action')}
                    <select className={field} value={filters.action ?? ''} onChange={e => setFilter('action', e.target.value)}>
                        <option value="">{t('auditLog.all')}</option>
                        {ACTIONS.map(a => <option key={a} value={a}>{t(`auditLog.actions.${a}`)}</option>)}
                    </select>
                </label>
                <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                    {t('auditLog.columns.status')}
                    <select className={field} value={filters.status ?? ''} onChange={e => setFilter('status', e.target.value)}>
                        <option value="">{t('auditLog.all')}</option>
                        <option value="success">{t('auditLog.statuses.success')}</option>
                        <option value="failure">{t('auditLog.statuses.failure')}</option>
                    </select>
                </label>
                <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                    {t('auditLog.columns.resource')}
                    <input
                        className={cn(field, 'w-40')}
                        placeholder={t('auditLog.resourcePlaceholder')}
                        value={filters.resource ?? ''}
                        onChange={e => setFilter('resource', e.target.value.trim())}
                    />
                </label>
                <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                    {t('auditLog.from')}
                    <input type="date" className={field} value={filters.start_date ?? ''} onChange={e => setFilter('start_date', e.target.value)} />
                </label>
                <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                    {t('auditLog.to')}
                    <input type="date" className={field} value={filters.end_date ?? ''} onChange={e => setFilter('end_date', e.target.value)} />
                </label>
            </div>

            <div className={cn('rounded-md border bg-card overflow-x-auto transition-opacity', isFetching && !isLoading && 'opacity-60')}>
                <table className="w-full text-sm">
                    <thead className="bg-secondary/60 text-left text-xs text-muted-foreground">
                        <tr>
                            <th className="w-8" />
                            <th className="px-3 py-2.5 font-semibold">{t('auditLog.columns.time')}</th>
                            <th className="px-3 py-2.5 font-semibold">{t('auditLog.columns.user')}</th>
                            <th className="px-3 py-2.5 font-semibold">{t('auditLog.columns.action')}</th>
                            <th className="px-3 py-2.5 font-semibold">{t('auditLog.columns.resource')}</th>
                            <th className="px-3 py-2.5 font-semibold">{t('auditLog.columns.status')}</th>
                            <th className="px-3 py-2.5 font-semibold">{t('auditLog.columns.ip')}</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y">
                        {isLoading ? (
                            Array.from({ length: 8 }, (_, i) => (
                                <tr key={i}><td colSpan={7} className="px-3 py-2"><Skeleton className="h-5 w-full" /></td></tr>
                            ))
                        ) : isError ? (
                            <tr><td colSpan={7} className="px-3 py-10 text-center text-muted-foreground">{t('auditLog.unavailable')}</td></tr>
                        ) : !entries.length ? (
                            <tr><td colSpan={7} className="px-3 py-10 text-center text-muted-foreground">{t('auditLog.empty')}</td></tr>
                        ) : entries.map(e => (
                            <Fragment key={e.id}>
                                <tr
                                    className="hover:bg-muted/40 cursor-pointer"
                                    onClick={() => setOpen(o => (o === e.id ? null : e.id))}
                                >
                                    <td className="pl-3 text-muted-foreground">
                                        <button
                                            aria-expanded={open === e.id}
                                            aria-label={t('auditLog.toggleDetails')}
                                            className="p-0.5 rounded hover:bg-muted"
                                        >
                                            {open === e.id ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                                        </button>
                                    </td>
                                    <td className="px-3 py-2 whitespace-nowrap tabular-nums" title={formatDate(e.created_at, 'datetime')}>
                                        {formatDate(e.created_at)}
                                    </td>
                                    <td className="px-3 py-2 font-medium">{e.username}</td>
                                    <td className="px-3 py-2 text-xs font-semibold">{t(`auditLog.actions.${e.action}`)}</td>
                                    <td className="px-3 py-2 font-mono text-xs">{e.resource}{e.resource_id && ` #${e.resource_id}`}</td>
                                    <td className={cn('px-3 py-2 text-xs font-semibold', e.status === 'failure' ? 'text-destructive' : 'text-success')}>
                                        {t(`auditLog.statuses.${e.status === 'failure' ? 'failure' : 'success'}`)}
                                    </td>
                                    <td className="px-3 py-2 font-mono text-xs text-muted-foreground">{e.ip_address}</td>
                                </tr>
                                {open === e.id && (
                                    <tr className="bg-muted/30">
                                        <td />
                                        <td colSpan={6} className="px-3 py-3">
                                            <div className="text-xs text-muted-foreground mb-2 break-all">{e.user_agent}</div>
                                            <pre className="text-xs font-mono whitespace-pre-wrap break-all max-h-80 overflow-auto rounded border bg-card p-3">
                                                {prettyDetails(e.details)}
                                            </pre>
                                        </td>
                                    </tr>
                                )}
                            </Fragment>
                        ))}
                    </tbody>
                </table>
            </div>

            <PaginationBar
                page={page}
                onPageChange={setPage}
                total={data?.total}
                pageSize={pageSize}
                onPageSizeChange={s => { setPageSize(s); setPage(1); }}
                pageSizeOptions={[20, 50, 100]}
            />
        </div>
    );
}
