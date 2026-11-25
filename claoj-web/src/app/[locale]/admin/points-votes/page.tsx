'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminPointsVoteApi } from '@/lib/adminApi';
import { formatDate } from '@/lib/date';
import { Link } from '@/navigation';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import { PaginationBar } from '@/components/ui/PaginationBar';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/Table';
import { Search, ThumbsUp } from 'lucide-react';

const FILTER_KEYS = ['search', 'problem'] as const;
const PAGE_SIZE = 20;

export default function AdminPointsVotesPage() {
    const t = useTranslations('Admin');
    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);
    const problem = useDebounce(filters.problem ?? '', 300);

    const { data, isLoading } = useQuery({
        queryKey: ['admin-points-votes', page, search, problem],
        queryFn: async () => (await adminPointsVoteApi.list(page, PAGE_SIZE, { search, problem })).data,
    });

    const votes = data?.data ?? [];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold flex items-center gap-3">
                    <ThumbsUp className="text-primary" size={30} />
                    {t('pointsVotes.title')}
                </h1>
                <p className="text-muted-foreground mt-1">{t('pointsVotes.subtitle')}</p>
            </div>

            <div className="flex flex-col gap-3 sm:flex-row">
                <div className="relative w-full sm:w-72">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <input
                        type="text"
                        value={filters.search ?? ''}
                        onChange={(e) => setFilters({ search: e.target.value })}
                        placeholder={t('pointsVotes.searchPlaceholder')}
                        aria-label={t('pointsVotes.searchPlaceholder')}
                        className="w-full h-10 pl-10 pr-4 rounded-md bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    />
                </div>
                <input
                    type="text"
                    value={filters.problem ?? ''}
                    onChange={(e) => setFilters({ problem: e.target.value })}
                    placeholder={t('pointsVotes.problemFilterPlaceholder')}
                    aria-label={t('pointsVotes.problemFilterPlaceholder')}
                    className="h-10 px-4 rounded-md bg-card border font-mono text-sm focus:ring-2 focus:ring-primary/20 outline-none sm:w-56"
                />
            </div>

            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t('pointsVotes.colPoints')}</TableHead>
                            <TableHead>{t('pointsVotes.colVoter')}</TableHead>
                            <TableHead>{t('pointsVotes.colProblem')}</TableHead>
                            <TableHead>{t('pointsVotes.colVoteTime')}</TableHead>
                            <TableHead>{t('pointsVotes.colNote')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8">{t('common.loading')}</TableCell>
                            </TableRow>
                        ) : votes.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                                    {t('pointsVotes.empty')}
                                </TableCell>
                            </TableRow>
                        ) : votes.map((vote) => (
                            <TableRow key={vote.id}>
                                <TableCell className="font-semibold tabular-nums">{vote.points}</TableCell>
                                <TableCell>
                                    <Link href={`/user/${vote.voter}`} className="hover:underline">{vote.voter}</Link>
                                </TableCell>
                                <TableCell>
                                    <Link href={`/problem/${vote.problem}`} className="font-mono text-sm hover:underline">
                                        {vote.problem}
                                    </Link>
                                    <div className="text-xs text-muted-foreground">{vote.problem_name}</div>
                                </TableCell>
                                <TableCell className="whitespace-nowrap text-sm">{formatDate(vote.vote_time)}</TableCell>
                                <TableCell className="text-sm text-muted-foreground max-w-md truncate">{vote.note}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>

            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />
        </div>
    );
}