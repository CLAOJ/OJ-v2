'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Link } from '@/navigation';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/Table';

interface JudgeStatus {
    name: string;
    ping: number | null; // seconds, as v1 stores it
    load: number | null;
    start_time: string | null;
    runtimes: { key: string; name: string }[];
}

// v1 timedelta: "H:MM:SS", "N days, H:MM:SS" past a day.
function formatUptime(start: string | null, now: number): string {
    if (!start) return '—';
    let s = Math.max(0, Math.floor((now - new Date(start).getTime()) / 1000));
    const days = Math.floor(s / 86400);
    s %= 86400;
    const pad = (n: number) => String(n).padStart(2, '0');
    const hms = `${pad(Math.floor(s / 3600))}:${pad(Math.floor((s % 3600) / 60))}:${pad(s % 60)}`;
    return days > 0 ? `${days}d ${hms}` : hms;
}

// v1 /status: online judges with uptime, ping, load and runtimes.
export default function StatusPage() {
    const t = useTranslations('Runtimes');
    const [judges, setJudges] = useState<JudgeStatus[] | null>(null);
    const [failed, setFailed] = useState(false);
    const [now, setNow] = useState(() => Date.now());

    useEffect(() => {
        const load = () =>
            api.get<{ data: JudgeStatus[] }>('/judges')
                .then((res) => { setJudges(res.data.data); setFailed(false); })
                .catch(() => setFailed(true));
        load();
        const poll = setInterval(load, 30_000);
        const tick = setInterval(() => setNow(Date.now()), 1000);
        return () => { clearInterval(poll); clearInterval(tick); };
    }, []);

    return (
        <div className="container mx-auto px-4 py-8">
            <div className="flex flex-wrap items-end justify-between gap-4 mb-6">
                <div>
                    <h1 className="text-3xl font-bold">{t('statusTitle')}</h1>
                    <p className="text-muted-foreground mt-1">{t('statusSubtitle')}</p>
                </div>
                <nav className="flex gap-1 border-b w-full sm:w-auto" aria-label={t('statusTitle')}>
                    <span aria-current="page" className="px-3 py-2 text-sm font-semibold border-b-2 border-primary">
                        {t('tabJudges')}
                    </span>
                    <Link href="/runtimes" className="px-3 py-2 text-sm text-muted-foreground hover:text-foreground">
                        {t('tabRuntimes')}
                    </Link>
                    <Link href="/runtimes/matrix" className="px-3 py-2 text-sm text-muted-foreground hover:text-foreground">
                        {t('tabVersions')}
                    </Link>
                </nav>
            </div>

            {judges === null && !failed ? (
                <div className="flex justify-center py-16">
                    <div className="animate-spin rounded-full h-10 w-10 border-2 border-muted border-t-primary" />
                </div>
            ) : failed && !judges ? (
                <p className="text-center py-16 text-muted-foreground">{t('statusLoadFailed')}</p>
            ) : !judges?.length ? (
                <p className="text-center py-16 text-muted-foreground">{t('statusEmpty')}</p>
            ) : (
                <div className="border rounded-lg overflow-x-auto">
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>{t('colJudge')}</TableHead>
                                <TableHead>{t('colUptime')}</TableHead>
                                <TableHead className="text-right">{t('colPing')}</TableHead>
                                <TableHead className="text-right">{t('colLoad')}</TableHead>
                                <TableHead>{t('colRuntimes')}</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {judges.map((j) => (
                                <TableRow key={j.name}>
                                    <TableCell className="whitespace-nowrap font-medium">{j.name}</TableCell>
                                    <TableCell className="whitespace-nowrap tabular-nums">{formatUptime(j.start_time, now)}</TableCell>
                                    <TableCell className="whitespace-nowrap tabular-nums text-right">
                                        {j.ping != null ? `${(j.ping * 1000).toFixed(3)} ms` : '—'}
                                    </TableCell>
                                    <TableCell className="whitespace-nowrap tabular-nums text-right">
                                        {j.load != null ? j.load.toFixed(3) : '—'}
                                    </TableCell>
                                    <TableCell className="text-sm">
                                        {j.runtimes.map((r, i) => (
                                            <span key={r.key}>
                                                {i > 0 && ', '}
                                                <Link href="/runtimes" className="text-primary hover:underline">{r.name}</Link>
                                            </span>
                                        ))}
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </div>
            )}
        </div>
    );
}
