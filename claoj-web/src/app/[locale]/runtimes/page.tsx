'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Link } from '@/navigation';
import { formatRuntimeVersions } from '@/lib/runtimes';
import type { RuntimeLanguage } from '@/types';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/Table';

export default function RuntimesPage() {
    const t = useTranslations('Runtimes');
    const [languages, setLanguages] = useState<RuntimeLanguage[]>([]);
    const [loading, setLoading] = useState(true);
    const [failed, setFailed] = useState(false);

    useEffect(() => {
        api.get<{ data: RuntimeLanguage[] }>('/runtimes')
            .then((res) => setLanguages(res.data.data))
            .catch(() => setFailed(true))
            .finally(() => setLoading(false));
    }, []);

    return (
        <div className="container mx-auto px-4 py-8">
            <div className="flex flex-wrap items-end justify-between gap-4 mb-6">
                <div>
                    <h1 className="text-3xl font-bold">{t('title')}</h1>
                    <p className="text-muted-foreground mt-1">{t('subtitle')}</p>
                </div>
                <nav className="flex gap-1 border-b w-full sm:w-auto" aria-label={t('title')}>
                    <Link href="/stats" className="px-3 py-2 text-sm text-muted-foreground hover:text-foreground">
                        {t('tabJudges')}
                    </Link>
                    <span aria-current="page" className="px-3 py-2 text-sm font-semibold border-b-2 border-primary">
                        {t('tabRuntimes')}
                    </span>
                    <Link href="/runtimes/matrix" className="px-3 py-2 text-sm text-muted-foreground hover:text-foreground">
                        {t('tabVersions')}
                    </Link>
                </nav>
            </div>

            {loading ? (
                <div className="flex justify-center py-16">
                    <div className="animate-spin rounded-full h-10 w-10 border-2 border-muted border-t-primary" />
                </div>
            ) : failed ? (
                <p className="text-center py-16 text-muted-foreground">{t('loadFailed')}</p>
            ) : languages.length === 0 ? (
                <p className="text-center py-16 text-muted-foreground">{t('empty')}</p>
            ) : (
                <div className="border rounded-lg overflow-x-auto">
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>{t('colKey')}</TableHead>
                                <TableHead>{t('colName')}</TableHead>
                                <TableHead>{t('colInfo')}</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {languages.map((language) => (
                                <TableRow key={language.id}>
                                    <TableCell className="font-mono whitespace-nowrap">
                                        {language.short_name || language.key}
                                    </TableCell>
                                    <TableCell className="whitespace-nowrap">{language.name}</TableCell>
                                    <TableCell className="text-sm">
                                        <code>{formatRuntimeVersions(language.runtimes)}</code>
                                        {language.description && (
                                            <p className="mt-1 text-muted-foreground whitespace-pre-line">
                                                {language.description}
                                            </p>
                                        )}
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </div>
            )}

            <p className="mt-4 text-xs text-muted-foreground">{t('onlineOnlyNote')}</p>
        </div>
    );
}