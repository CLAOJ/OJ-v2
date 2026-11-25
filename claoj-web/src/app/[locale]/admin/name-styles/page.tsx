'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import api, { type ApiError } from '@/lib/api';
import { useNameStyles, type NameStyle, type NameStylesData } from '@/components/providers/NameStyles';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Plus, Save, Trash2 } from 'lucide-react';

type Row = NameStyle & { rank: string };

const blank: Row = { rank: '', color: '', bold: false, strike: false, prefix: '' };

// Admin editor for how names look per display rank (v1 hardcoded this in
// ranks.scss). Rank keys are judge_profile.display_rank values.
export default function NameStylesAdminPage() {
    const t = useTranslations('Admin.nameStyles');
    const { data } = useNameStyles();
    if (!data) return <p className="text-sm text-muted-foreground">{t('loading')}</p>;
    return <NameStylesEditor data={data} />;
}

function NameStylesEditor({ data }: { data: NameStylesData }) {
    const t = useTranslations('Admin.nameStyles');
    const queryClient = useQueryClient();
    const [rows, setRows] = useState<Row[]>(() => Object.entries(data.styles).map(([rank, s]) => ({ ...s, rank })));

    const counts: Record<string, number> = {};
    for (const rank of Object.values(data.users)) counts[rank] = (counts[rank] ?? 0) + 1;

    const save = useMutation({
        mutationFn: () =>
            api.put('/admin/name-styles', Object.fromEntries(
                rows.filter(r => r.rank.trim()).map(({ rank, ...s }) => [rank.trim(), s]),
            )),
        onSuccess: () => {
            toast.success(t('saved'));
            queryClient.invalidateQueries({ queryKey: ['name-styles'] });
        },
        onError: (err) => toast.error((err as ApiError).response?.data?.error || t('saveFailed')),
    });

    const update = (i: number, patch: Partial<Row>) =>
        setRows(rs => rs.map((r, j) => (j === i ? { ...r, ...patch } : r)));

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold">{t('title')}</h1>
                <p className="text-sm text-muted-foreground">{t('description')}</p>
            </div>

            <div className="overflow-x-auto rounded-lg border">
                <table className="w-full text-sm">
                    <thead className="bg-muted/50 text-left">
                        <tr>
                            <th className="p-3 font-medium">{t('rank')}</th>
                            <th className="p-3 font-medium">{t('color')}</th>
                            <th className="p-3 font-medium">{t('bold')}</th>
                            <th className="p-3 font-medium">{t('strike')}</th>
                            <th className="p-3 font-medium">{t('prefix')}</th>
                            <th className="p-3 font-medium">{t('preview')}</th>
                            <th className="p-3" />
                        </tr>
                    </thead>
                    <tbody>
                        {rows.map((r, i) => (
                            <tr key={i} className="border-t">
                                <td className="p-3">
                                    <Input
                                        value={r.rank}
                                        maxLength={10}
                                        aria-label={t('rank')}
                                        onChange={e => update(i, { rank: e.target.value.toLowerCase() })}
                                        className="h-9 w-28 font-mono"
                                    />
                                    <span className="mt-1 block text-xs text-muted-foreground">
                                        {t('userCount', { count: counts[r.rank] ?? 0 })}
                                    </span>
                                </td>
                                <td className="p-3">
                                    <div className="flex items-center gap-2">
                                        <input
                                            type="color"
                                            value={r.color || '#000000'}
                                            aria-label={t('color')}
                                            onChange={e => update(i, { color: e.target.value })}
                                            className={`h-9 w-10 cursor-pointer rounded border bg-transparent ${r.color ? '' : 'opacity-25'}`}
                                        />
                                        {r.color ? (
                                            <button type="button" onClick={() => update(i, { color: '' })} className="text-xs text-muted-foreground hover:underline">
                                                {t('clearColor')}
                                            </button>
                                        ) : (
                                            <span className="text-xs text-muted-foreground">{t('noColor')}</span>
                                        )}
                                    </div>
                                </td>
                                <td className="p-3">
                                    <input type="checkbox" checked={r.bold} aria-label={t('bold')} onChange={e => update(i, { bold: e.target.checked })} className="h-4 w-4" />
                                </td>
                                <td className="p-3">
                                    <input type="checkbox" checked={r.strike} aria-label={t('strike')} onChange={e => update(i, { strike: e.target.checked })} className="h-4 w-4" />
                                </td>
                                <td className="p-3">
                                    <Input value={r.prefix} maxLength={8} aria-label={t('prefix')} onChange={e => update(i, { prefix: e.target.value })} className="h-9 w-20" />
                                </td>
                                <td className="p-3 whitespace-nowrap">
                                    <span
                                        className="text-primary"
                                        style={{
                                            color: r.color || undefined,
                                            fontWeight: r.bold ? 700 : undefined,
                                            textDecoration: r.strike ? 'line-through' : undefined,
                                        }}
                                    >
                                        {r.prefix && <span className="mr-[.2em]">{r.prefix}</span>}
                                        username
                                    </span>
                                </td>
                                <td className="p-3">
                                    <Button variant="ghost" size="sm" aria-label={t('remove')} onClick={() => setRows(rs => rs.filter((_, j) => j !== i))}>
                                        <Trash2 className="h-4 w-4" />
                                    </Button>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>

            <div className="flex flex-wrap gap-2">
                <Button variant="outline" onClick={() => setRows(rs => [...rs, { ...blank }])}>
                    <Plus className="mr-2 h-4 w-4" />
                    {t('addRank')}
                </Button>
                <Button onClick={() => save.mutate()} disabled={save.isPending}>
                    <Save className="mr-2 h-4 w-4" />
                    {t('save')}
                </Button>
            </div>
            <p className="text-xs text-muted-foreground">{t('hint')}</p>
        </div>
    );
}
