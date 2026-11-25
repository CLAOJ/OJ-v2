'use client';

import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';
import { ArrowLeft, GitCompare, User } from 'lucide-react';
import dayjs from 'dayjs';
import { adminProblemApi } from '@/lib/adminApi';
import { buildDiffMatrix, diffSelectionUrl, toggleDiffSelection } from '@/lib/diffMatrix';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import { cn, getStatusVariant } from '@/lib/utils';

/**
 * Setter diff matrix (v1 ProblemSubmissionDiff): rows are the problem's
 * submissions, columns its test cases. Pick two rows and the existing source
 * diff viewer opens on them. The API behind it is gated by RequireProblemEdit.
 */
export default function ProblemDiffMatrixPage() {
    const t = useTranslations('Admin');
    const params = useParams();
    const code = params.code as string;
    const [userFilter, setUserFilter] = useState('');
    const [selected, setSelected] = useState<number[]>([]);

    const { data, isLoading, isError } = useQuery({
        queryKey: ['admin-problem-diff', code, userFilter],
        queryFn: async () => (await adminProblemApi.diff(code, userFilter.trim() || undefined)).data,
    });

    const matrix = useMemo(() => buildDiffMatrix(data?.submissions ?? []), [data]);
    const diffUrl = diffSelectionUrl(selected);

    return (
        <div className="space-y-6">
            <div className="flex flex-wrap items-center gap-4">
                <Link
                    href={`/admin/problems/${code}/data`}
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                    title={t('problems.manageDataLink')}
                >
                    <ArrowLeft size={20} />
                </Link>
                <div className="flex-1">
                    <h1 className="text-3xl font-semibold flex items-center gap-3">
                        <GitCompare className="text-primary" size={32} />
                        {t('problems.diffTitle', { name: data?.problem_name || code })}
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {t('problems.diffSubtitle', { code })}
                    </p>
                </div>
                <div className="relative">
                    <User className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                    <input
                        type="text"
                        value={userFilter}
                        onChange={(e) => { setUserFilter(e.target.value); setSelected([]); }}
                        placeholder={t('problems.diffUserPlaceholder')}
                        className="h-10 w-56 rounded-xl border bg-background pl-9 pr-3 text-sm outline-none focus:ring-2 focus:ring-primary/20"
                    />
                </div>
                {diffUrl ? (
                    <Link
                        href={diffUrl}
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                    >
                        <GitCompare size={18} />
                        {t('problems.diffCompare')}
                    </Link>
                ) : (
                    <span
                        className="px-4 py-2 rounded-xl bg-muted text-muted-foreground flex items-center gap-2 font-medium cursor-not-allowed"
                        title={t('problems.diffSelectTwo')}
                    >
                        <GitCompare size={18} />
                        {t('problems.diffCompare')}
                    </span>
                )}
            </div>

            {isError ? (
                <div className="p-8 text-center text-muted-foreground border rounded-xl">
                    {t('problems.diffForbidden')}
                </div>
            ) : (
                <div className="border rounded-xl overflow-x-auto">
                    <table className="w-full text-left border-collapse text-sm">
                        <thead>
                            <tr className="bg-muted/30 border-b text-xs text-muted-foreground">
                                <th className="px-3 py-3 w-10" />
                                <th className="px-3 py-3">{t('problems.diffIdColumn')}</th>
                                <th className="px-3 py-3">{t('problems.diffUserColumn')}</th>
                                <th className="px-3 py-3">{t('problems.diffResultColumn')}</th>
                                <th className="px-3 py-3 text-center">{t('problems.diffPointsColumn')}</th>
                                <th className="px-3 py-3">{t('problems.diffLanguageColumn')}</th>
                                <th className="px-3 py-3">{t('problems.diffDateColumn')}</th>
                                {matrix.caseNumbers.map(c => (
                                    <th key={c} className="px-2 py-3 text-center whitespace-nowrap">
                                        {t('problems.diffCaseColumn', { case: c })}
                                    </th>
                                ))}
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {isLoading ? (
                                Array.from({ length: 8 }).map((_, i) => (
                                    <tr key={i}>
                                        <td colSpan={7 + matrix.caseNumbers.length} className="px-3 py-3">
                                            <Skeleton className="h-8 w-full rounded-md" />
                                        </td>
                                    </tr>
                                ))
                            ) : matrix.rows.length === 0 ? (
                                <tr>
                                    <td colSpan={7 + matrix.caseNumbers.length} className="px-3 py-8 text-center text-muted-foreground">
                                        {t('problems.diffEmpty')}
                                    </td>
                                </tr>
                            ) : (
                                matrix.rows.map(row => (
                                    <tr
                                        key={row.id}
                                        onClick={() => setSelected(prev => toggleDiffSelection(prev, row.id))}
                                        className={cn(
                                            'cursor-pointer transition-colors',
                                            selected.includes(row.id) ? 'bg-primary/5' : 'hover:bg-muted/20'
                                        )}
                                    >
                                        <td className="px-3 py-2">
                                            <input
                                                type="checkbox"
                                                checked={selected.includes(row.id)}
                                                onChange={() => setSelected(prev => toggleDiffSelection(prev, row.id))}
                                                onClick={(e) => e.stopPropagation()}
                                                className="rounded border-muted-foreground/30"
                                                aria-label={`${t('problems.diffCompare')} #${row.id}`}
                                            />
                                        </td>
                                        <td className="px-3 py-2 font-mono">
                                            <Link href={`/submission/${row.id}`} className="hover:text-primary">
                                                {row.id}
                                            </Link>
                                        </td>
                                        <td className="px-3 py-2">
                                            <Link href={`/user/${row.user}`} className="hover:text-primary">
                                                {row.user}
                                            </Link>
                                        </td>
                                        <td className="px-3 py-2">
                                            {row.result ? (
                                                <Badge variant={getStatusVariant(row.result)}>{row.result}</Badge>
                                            ) : (
                                                <span className="text-xs text-muted-foreground">{row.status}</span>
                                            )}
                                        </td>
                                        <td className="px-3 py-2 text-center font-mono">
                                            {row.points === null ? '—' : row.points}
                                        </td>
                                        <td className="px-3 py-2 text-muted-foreground">{row.language}</td>
                                        <td className="px-3 py-2 text-muted-foreground whitespace-nowrap">
                                            {dayjs(row.date).format('YYYY-MM-DD HH:mm')}
                                        </td>
                                        {matrix.caseNumbers.map(c => {
                                            const cell = row.cells.get(c);
                                            return (
                                                <td key={c} className="px-2 py-2 text-center">
                                                    {cell ? (
                                                        <div className="flex flex-col items-center gap-0.5">
                                                            <Badge variant={getStatusVariant(cell.status)} className="text-[10px]">
                                                                {cell.status}
                                                            </Badge>
                                                            {cell.points !== null && cell.total !== null && (
                                                                <span className="text-[10px] font-mono text-muted-foreground">
                                                                    {cell.points}/{cell.total}
                                                                </span>
                                                            )}
                                                        </div>
                                                    ) : (
                                                        <span className="text-muted-foreground/40">—</span>
                                                    )}
                                                </td>
                                            );
                                        })}
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}