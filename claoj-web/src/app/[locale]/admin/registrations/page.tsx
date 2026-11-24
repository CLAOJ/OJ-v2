'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useAuth } from '@/components/providers/AuthProvider';
import { adminRegistrationApi, adminErrorMessage } from '@/lib/adminApi';
import { toggleSort, type SortOrder } from '@/lib/listSort';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import type { AdminRegistration } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Alert } from '@/components/ui/Alert';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/Dialog';
import { Button } from '@/components/ui/Button';
import { PaginationBar } from '@/components/ui/PaginationBar';
import { SortableHeader } from '@/components/ui/SortableHeader';
import { Search, UserCheck, Send, Mail, KeyRound, UserX } from 'lucide-react';

const FILTER_KEYS = ['search', 'status', 'order'] as const;
const PAGE_SIZE = 50;

type ActionKind = 'activate' | 'resend';

export default function AdminRegistrationsPage() {
    const t = useTranslations('Admin');
    const { user: viewer } = useAuth();
    const queryClient = useQueryClient();

    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);
    const status = (filters.status ?? 'pending') as 'pending' | 'active' | 'all';
    const order: SortOrder = filters.order === 'asc' ? 'asc' : 'desc';

    const [target, setTarget] = useState<AdminRegistration | null>(null);
    const [action, setAction] = useState<ActionKind>('activate');
    const [notice, setNotice] = useState<{ kind: 'success' | 'error'; text: string } | null>(null);

    const isSuperuser = !!viewer?.is_admin;

    const { data, isLoading, isError } = useQuery({
        queryKey: ['admin-registrations', page, search, status, order],
        queryFn: async () =>
            (await adminRegistrationApi.list(page, PAGE_SIZE, { search, status, order })).data,
    });

    const finish = (kind: 'success' | 'error', text: string) => {
        setTarget(null);
        setNotice({ kind, text });
    };

    const activateMutation = useMutation({
        mutationFn: (userId: number) => adminRegistrationApi.activate(userId),
        onSuccess: (res, userId) => {
            queryClient.invalidateQueries({ queryKey: ['admin-registrations'] });
            const already = res.data.already_active;
            finish('success', t(already ? 'registrations.activateAlready' : 'registrations.activateSuccess',
                { username: target?.username ?? String(userId) }));
        },
        onError: (err) => finish('error', adminErrorMessage(err, t('registrations.actionError'))),
    });

    const resendMutation = useMutation({
        mutationFn: (userId: number) => adminRegistrationApi.resend(userId),
        onSuccess: (_, userId) => {
            queryClient.invalidateQueries({ queryKey: ['admin-registrations'] });
            finish('success', t('registrations.resendSuccess', { username: target?.username ?? String(userId) }));
        },
        onError: (err) => finish('error', adminErrorMessage(err, t('registrations.actionError'))),
    });

    const isPending = activateMutation.isPending || resendMutation.isPending;

    const openAction = (kind: ActionKind, row: AdminRegistration) => {
        activateMutation.reset();
        resendMutation.reset();
        setNotice(null);
        setAction(kind);
        setTarget(row);
    };

    const confirm = () => {
        if (!target) return;
        if (action === 'activate') activateMutation.mutate(target.user_id);
        else resendMutation.mutate(target.user_id);
    };

    const handleSort = () => {
        const next = toggleSort({ sort: 'date_joined', order }, 'date_joined', 'desc');
        setFilters({ order: next.order });
    };

    const rows = data?.data ?? [];

    const registrationState = (r: AdminRegistration) => {
        if (!r.registration) return t('registrations.v1None');
        if (r.registration.activated) return t('registrations.v1Activated');
        return r.registration.has_activation_key
            ? t('registrations.v1KeyPending')
            : t('registrations.v1NoKey');
    };

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-semibold flex items-center gap-3">
                        <Mail className="text-primary" size={32} />
                        {t('registrations.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('registrations.subtitle')}</p>
                </div>
                <div className="relative w-full md:w-72">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                    <input
                        type="text"
                        placeholder={t('registrations.searchPlaceholder')}
                        aria-label={t('registrations.searchPlaceholder')}
                        className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={filters.search ?? ''}
                        onChange={(e) => setFilters({ search: e.target.value })}
                    />
                </div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
                <select
                    aria-label={t('registrations.statusFilter')}
                    className="h-10 px-3 rounded-xl bg-card border"
                    value={status}
                    onChange={(e) => setFilters({ status: e.target.value })}
                >
                    <option value="pending">{t('registrations.statusPending')}</option>
                    <option value="active">{t('registrations.statusActive')}</option>
                    <option value="all">{t('registrations.statusAll')}</option>
                </select>
                {!isSuperuser && (
                    <span className="text-xs text-muted-foreground">{t('registrations.superuserOnly')}</span>
                )}
            </div>

            {notice && (
                <Alert variant={notice.kind === 'success' ? 'success' : 'destructive'}>{notice.text}</Alert>
            )}

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-20 rounded-md" />)}
                </div>
            ) : isError ? (
                <div role="alert" className="bg-card rounded-md border p-12 text-center text-destructive">
                    {t('registrations.loadError')}
                </div>
            ) : (
                <div className="bg-card rounded-md border overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left text-sm">
                            <thead className="bg-muted/50 border-b">
                                <tr>
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground">{t('registrations.colUser')}</th>
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground">{t('registrations.colStatus')}</th>
                                    <SortableHeader
                                        label={t('registrations.colJoined')}
                                        active
                                        order={order}
                                        onClick={handleSort}
                                    />
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground">{t('registrations.colRegistration')}</th>
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground text-right">{t('common.actions')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {rows.length === 0 ? (
                                    <tr>
                                        <td colSpan={5} className="px-4 py-12 text-center text-muted-foreground">
                                            {t('registrations.noRows')}
                                        </td>
                                    </tr>
                                ) : (
                                    rows.map((row) => (
                                        <tr key={row.user_id} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-4 py-2">
                                                <div className="flex items-center gap-2">
                                                    <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary text-sm font-medium shrink-0">
                                                        {row.username[0]?.toUpperCase()}
                                                    </div>
                                                    <div className="min-w-0">
                                                        <div className="font-medium truncate">{row.username}</div>
                                                        <div className="text-xs text-muted-foreground truncate">{row.email}</div>
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-4 py-2">
                                                <div className="flex flex-wrap items-center gap-1">
                                                    {row.is_active ? (
                                                        <Badge variant="success" className="flex items-center gap-1 text-[10px] py-0 px-1.5">
                                                            <UserCheck size={10} /> {t('registrations.active')}
                                                        </Badge>
                                                    ) : (
                                                        <Badge variant="warning" className="flex items-center gap-1 text-[10px] py-0 px-1.5">
                                                            <UserX size={10} /> {t('registrations.pending')}
                                                        </Badge>
                                                    )}
                                                    {row.has_pending_token && (
                                                        <Badge variant="outline" className="flex items-center gap-1 text-[10px] py-0 px-1.5">
                                                            <KeyRound size={10} /> {t('registrations.tokenSent')}
                                                        </Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-4 py-2 text-xs text-muted-foreground whitespace-nowrap">
                                                {row.date_joined ? new Date(row.date_joined).toLocaleDateString() : t('common.notAvailable')}
                                            </td>
                                            <td className="px-4 py-2 text-xs text-muted-foreground">
                                                {registrationState(row)}
                                            </td>
                                            <td className="px-4 py-2 text-right">
                                                <div className="flex justify-end gap-1">
                                                    {!row.is_active && (
                                                        <>
                                                            <button
                                                                onClick={() => openAction('activate', row)}
                                                                disabled={!isSuperuser}
                                                                className="p-1.5 hover:bg-success/10 text-success rounded-lg transition-colors disabled:opacity-40 disabled:hover:bg-transparent"
                                                                title={isSuperuser ? t('registrations.activateButton') : t('registrations.superuserOnly')}
                                                                aria-label={t('registrations.activateButton')}
                                                            >
                                                                <UserCheck size={16} />
                                                            </button>
                                                            <button
                                                                onClick={() => openAction('resend', row)}
                                                                disabled={!isSuperuser}
                                                                className="p-1.5 hover:bg-primary/10 text-primary rounded-lg transition-colors disabled:opacity-40 disabled:hover:bg-transparent"
                                                                title={isSuperuser ? t('registrations.resendButton') : t('registrations.superuserOnly')}
                                                                aria-label={t('registrations.resendButton')}
                                                            >
                                                                <Send size={16} />
                                                            </button>
                                                        </>
                                                    )}
                                                </div>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}

            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />

            <Dialog open={target !== null} onOpenChange={(open) => { if (!open) setTarget(null); }} className="max-w-md">
                <DialogHeader>
                    <DialogTitle id="registration-action-title">
                        {t(action === 'activate' ? 'registrations.activateTitle' : 'registrations.resendTitle',
                            { username: target?.username ?? '' })}
                    </DialogTitle>
                </DialogHeader>
                <DialogContent>
                    <p className="text-sm text-muted-foreground">
                        {t(action === 'activate' ? 'registrations.activateDescription' : 'registrations.resendDescription')}
                    </p>
                </DialogContent>
                <DialogFooter>
                    <Button type="button" variant="outline" onClick={() => setTarget(null)}>{t('common.cancel')}</Button>
                    <Button
                        type="button"
                        disabled={isPending}
                        onClick={confirm}
                    >
                        {t(action === 'activate' ? 'registrations.activateButton' : 'registrations.resendButton')}
                    </Button>
                </DialogFooter>
            </Dialog>
        </div>
    );
}