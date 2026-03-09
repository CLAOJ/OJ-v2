'use client';

import { Link } from '@/navigation';
import { useTranslations } from 'next-intl';
import { Heart } from 'lucide-react';
import { useState, useEffect } from 'react';

export default function Footer() {
    const currentYear = new Date().getFullYear();
    const t = useTranslations('Footer');

    // QR Popup state
    const [showQr, setShowQr] = useState(false);
    const [minimized, setMinimized] = useState(false);

    useEffect(() => {
        // Check if popup should be shown (based on localStorage)
        const hideUntil = localStorage.getItem('qrWidgetClosed');
        const now = new Date().getTime();
        const DAYS_TO_WAIT = 1;

        if (!hideUntil || now >= parseInt(hideUntil)) {
            setShowQr(true);
        }
    }, []);

    const handleCloseQr = () => {
        setShowQr(false);
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

            {/* QR Popup Widget - Original CLAOJ Feature */}
            {showQr && (
                <div
                    className={`fixed bottom-5 right-5 bg-background border border-border rounded-lg shadow-lg z-50 transition-all duration-300 ${minimized ? 'w-[120px]' : 'w-[160px]'}`}
                    style={{ display: showQr ? 'block' : 'none' }}
                >
                    {/* Header */}
                    <div className="bg-muted/50 px-3 py-2 flex items-center justify-between border-b border-border">
                        <span className="text-xs font-semibold text-foreground">Hỗ Trợ CLAOJ</span>
                        <div className="flex items-center gap-1">
                            <button
                                onClick={() => setMinimized(!minimized)}
                                className="text-muted-foreground hover:text-foreground text-sm"
                                title={minimized ? 'Expand' : 'Minimize'}
                            >
                                {minimized ? '+' : '−'}
                            </button>
                            <button
                                onClick={handleCloseQr}
                                className="text-muted-foreground hover:text-foreground text-sm"
                                title="Close"
                            >
                                ×
                            </button>
                        </div>
                    </div>

                    {/* Body with QR Code */}
                    {!minimized && (
                        <div className="p-3 text-center">
                            <a href="https://quy.momo.vn/v2/MsvmMoip7r?e14fe" target="_blank" rel="noreferrer">
                                <img
                                    src="/static/qr-code.jpg"
                                    alt="QR Code"
                                    className="w-full h-auto rounded"
                                    onError={(e) => {
                                        // Fallback if QR image not found
                                        (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect fill="none" stroke="%23333" stroke-width="2" width="100" height="100"/><rect fill="%23333" x="10" y="10" width="30" height="30"/><rect fill="%23333" x="60" y="10" width="30" height="30"/><rect fill="%23333" x="10" y="60" width="30" height="30"/><rect fill="%23333" x="15" y="15" width="20" height="20"/><rect fill="%23333" x="65" y="15" width="20" height="20"/><rect fill="%23333" x="15" y="65" width="20" height="20"/></svg>';
                                    }}
                                />
                            </a>
                        </div>
                    )}
                </div>
            )}
        </>
    );
}
