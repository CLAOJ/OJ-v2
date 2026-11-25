'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminOrganizationRequestApi } from '@/lib/adminApi';
import { formatDate } from '@/lib/date';
import { Link } from '@/navigation';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import { PaginationBar } from '@/components/ui/PaginationBar';
import { Badge } from '@/components/ui/Badge';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/Table';
import { ClipboardList, Search } from 'lucide-react';

const FILTER_KEYS = ['search', 'state'] as const;
const PAGE_SIZE = 20;

const STATE_VARIANT: Record<string, 'success' | 'secondary' | 'destructive'> = {
    P: 'secondary',
    A: 'success',
    R: 'destructive',
};

export default function AdminOrganizationRequestsPage() {
    const t = useTranslations('Admin');
    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);
    const state = filters.state ?? '';

    const { data, isLoading } = useQuery({
        queryKey: ['admin-org-requests', page, search, state],
        queryFn: async () => (await adminOrganizationRequestApi.list(page, PAGE_SIZE, { search, state })).data,
    });

    const requests = data?.data ?? [];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold flex items-center gap-3">
                    <ClipboardList className="text-primary" size={30} />
                    {t('organizationRequests.title')}
                </h1>
                <p className="text-muted-foreground mt-1">{t('organizationRequests.subtitle')}</p>
            </div>

            <div className="flex flex-col gap-3 sm:flex-row">
                <div className="relative w-full sm:w-80">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <input
                        type="text"
                        value={filters.search ?? ''}
                        onChange={(e) => setFilters({ search: e.target.value })}
                        placeholder={t('organizationRequests.searchPlaceholder')}
                        aria-label={t('organizationRequests.searchPlaceholder')}
                        className="w-full h-10 pl-10 pr-4 rounded-md bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    />
                </div>
                <select
                    value={state}
                    onChange={(e) => setFilters({ state: e.target.value })}
                    aria-label={t('organizationRequests.stateFilter')}
                    className="h-10 px-3 rounded-md bg-card border text-sm focus:ring-2 focus:ring-primary/20 outline-none sm:w-56"
                >
                    <option value="">{t('organizationRequests.stateAll')}</option>
                    <option value="P">{t('organizationRequests.stateP')}</option>
                    <option value="A">{t('organizationRequests.stateA')}</option>
                    <option value="R">{t('organizationRequests.stateR')}</option>
                </select>
            </div>

            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t('organizationRequests.colUsername')}</TableHead>
                            <TableHead>{t('organizationRequests.colOrganization')}</TableHead>
                            <TableHead>{t('organizationRequests.colState')}</TableHead>
                            <TableHead>{t('organizationRequests.colTime')}</TableHead>
                            <TableHead>{t('organizationRequests.colReason')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8">{t('common.loading')}</TableCell>
                            </TableRow>
                        ) : requests.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                                    {t('organizationRequests.empty')}
                                </TableCell>
                            </TableRow>
                        ) : requests.map((req) => (
                            <TableRow key={req.id}>
                                <TableCell>
                                    <Link href={`/user/${req.username}`} className="hover:underline">{req.username}</Link>
                                </TableCell>
                                <TableCell>
                                    <Link href={`/organization/${req.organization_id}`} className="hover:underline">
                                        {req.organization}
                                    </Link>
                                </TableCell>
                                <TableCell>
                                    <Badge variant={STATE_VARIANT[req.state] ?? 'secondary'}>
                                        {t(`organizationRequests.state${req.state}`)}
                                    </Badge>
                                </TableCell>
                                <TableCell className="whitespace-nowrap text-sm">{formatDate(req.time)}</TableCell>
                                <TableCell className="text-sm text-muted-foreground max-w-md truncate">{req.reason}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>

            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />
        </div>
    );
}