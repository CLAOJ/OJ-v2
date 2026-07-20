'use client';

import { Link } from '@/navigation';
import { useTranslations } from 'next-intl';
import { Heart } from 'lucide-react';
import { useState, useEffect } from 'react';

export default function Footer() {
    const currentYear = new Date().getFullYear();
    const t = useTranslations('Footer');

    // Donation QR widget. Collapsed by default to a small floating button so it
    // never covers page content; expands to the QR panel on tap. `dismissed`
    // starts true to avoid a flash before the localStorage check runs.
    const [dismissed, setDismissed] = useState(true);
    const [expanded, setExpanded] = useState(false);

    useEffect(() => {
        const hideUntil = localStorage.getItem('qrWidgetClosed');
        const now = new Date().getTime();
        if (!hideUntil || now >= parseInt(hideUntil)) {
            setDismissed(false);
        }
    }, []);

    const handleCloseQr = () => {
        setDismissed(true);
        setExpanded(false);
        const DAYS_TO_WAIT = 1;
        const expiryTime = new Date().getTime() + (DAYS_TO_WAIT * 24 * 60 * 60 * 1000);
        localStorage.setItem('qrWidgetClosed', expiryTime.toString());
    };

    return (
        <>
            <footer className="border-t bg-card pt-12 pb-6">
                <div className="container mx-auto px-4">
                    <div className="grid grid-cols-1 md:grid-cols-4 gap-8 mb-8">
                        <div className="col-span-1 md:col-span-2">
                            <h3 className="text-xl font-bold mb-4">CLAOJ</h3>
                            <p className="text-muted-foreground max-w-sm mb-4">
                                {t('description')}
                            </p>
                            <p className="text-sm text-muted-foreground">
                                <strong>Long An HSGS Online Judge</strong><br />
                                Powered by{' '}
                                <a href="https://dmoj.ca" target="_blank" rel="noreferrer" className="text-primary hover:underline">
                                    <b>DMOJ</b>
                                </a>{' '}
                                and{' '}
                                <a href="https://oj.vnoi.info" target="_blank" rel="noreferrer" className="text-primary hover:underline">
                                    <b>VNOI</b>
                                </a>
                            </p>
                        </div>
                        <div>
                            <h4 className="font-semibold mb-4">{t('platform')}</h4>
                            <ul className="space-y-2 text-sm text-muted-foreground">
                                <li><Link href="/problems" className="hover:text-primary transition-colors">{t('problems')}</Link></li>
                                <li><Link href="/contests" className="hover:text-primary transition-colors">{t('contests')}</Link></li>
                                <li><Link href="/submissions" className="hover:text-primary transition-colors">{t('submissions')}</Link></li>
                                <li><Link href="/organizations" className="hover:text-primary transition-colors">{t('organizations')}</Link></li>
                            </ul>
                        </div>
                        <div>
                            <h4 className="font-semibold mb-4">{t('support')}</h4>
                            <ul className="space-y-2 text-sm text-muted-foreground">
                                <li><Link href="/tickets" className="hover:text-primary transition-colors">{t('helpTickets')}</Link></li>
                                <li><Link href="/about" className="hover:text-primary transition-colors">{t('aboutUs')}</Link></li>
                                <li><a href="https://www.facebook.com/itclaoj" target="_blank" rel="noreferrer" className="hover:text-primary transition-colors">IT-CLA Productions</a></li>
                                <li><Link href="/donate" className="hover:text-primary transition-colors">{t('donate')}</Link></li>
                            </ul>
                        </div>
                    </div>

                    <div className="pt-8 border-t flex flex-col md:flex-row justify-between items-center gap-4 text-sm text-muted-foreground">
                        <div className="flex items-center gap-1 flex-wrap justify-center">
                            <span>&copy; {currentYear} {t('copyright')}</span>
                            <Heart size={14} className="text-destructive fill-destructive" />
                            <span>{t('builtBy')}</span>
                        </div>
                        <div className="flex items-center gap-4">
                            <a href="https://github.com/CLAOJ" target="_blank" rel="noreferrer" className="hover:text-primary transition-colors">{t('github')}</a>
                        </div>
                    </div>
                </div>
            </footer>

            {/* Donation QR widget — collapsed button by default, expands on tap */}
            {!dismissed && !expanded && (
                <button
                    onClick={() => setExpanded(true)}
                    aria-label={t('supportTitle')}
                    title={t('supportTitle')}
                    className="fixed bottom-5 right-5 z-40 flex h-12 w-12 items-center justify-center rounded-full bg-primary text-white shadow-lg transition-transform hover:scale-105 active:scale-95"
                >
                    <Heart size={20} className="fill-white" />
                </button>
            )}

            {!dismissed && expanded && (
                <div className="fixed bottom-5 right-5 z-40 w-[180px] rounded-lg border border-border bg-background shadow-xl">
                    {/* Header */}
                    <div className="flex items-center justify-between border-b border-border bg-muted/50 px-3 py-2">
                        <span className="text-xs font-semibold text-foreground">{t('supportTitle')}</span>
                        <div className="flex items-center gap-1">
                            <button
                                onClick={() => setExpanded(false)}
                                className="px-1 text-sm text-muted-foreground hover:text-foreground"
                                title={t('supportMinimize')}
                                aria-label={t('supportMinimize')}
                            >
                                −
                            </button>
                            <button
                                onClick={handleCloseQr}
                                className="px-1 text-sm text-muted-foreground hover:text-foreground"
                                title={t('supportClose')}
                                aria-label={t('supportClose')}
                            >
                                ×
                            </button>
                        </div>
                    </div>

                    {/* Body with QR Code */}
                    <div className="p-3 text-center">
                        <a href="https://quy.momo.vn/v2/MsvmMoip7r?e14fe" target="_blank" rel="noreferrer">
                            <img
                                src="/static/qr-code.jpg"
                                alt={t('supportTitle')}
                                className="h-auto w-full rounded"
                                onError={(e) => {
                                    // Fallback if QR image not found
                                    (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect fill="none" stroke="%23333" stroke-width="2" width="100" height="100"/><rect fill="%23333" x="10" y="10" width="30" height="30"/><rect fill="%23333" x="60" y="10" width="30" height="30"/><rect fill="%23333" x="10" y="60" width="30" height="30"/><rect fill="%23333" x="15" y="15" width="20" height="20"/><rect fill="%23333" x="65" y="15" width="20" height="20"/><rect fill="%23333" x="15" y="65" width="20" height="20"/></svg>';
                                }}
                            />
                        </a>
                    </div>
                </div>
            )}
        </>
    );
}
