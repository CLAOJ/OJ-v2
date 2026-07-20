'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Document, Page } from 'react-pdf';
import { useTranslations } from 'next-intl';
import { Loader2, FileText, Download, ArrowUpRight, ZoomIn, ZoomOut } from 'lucide-react';
import api, { problemPdfApi } from '@/lib/api';
import '@/components/ui/pdfSetup';

interface PdfStatementViewerProps {
    code: string;
    heightClass?: string;
}

export default function PdfStatementViewer({ code, heightClass = 'max-h-[80vh]' }: PdfStatementViewerProps) {
    const t = useTranslations('Problems');
    const [objectUrl, setObjectUrl] = useState<string | null>(null);
    const [status, setStatus] = useState<'loading' | 'error' | 'ready'>('loading');
    const [numPages, setNumPages] = useState(0);
    const [zoom, setZoom] = useState(1);
    const [width, setWidth] = useState(0);
    const containerRef = useRef<HTMLDivElement>(null);

    const downloadUrl = problemPdfApi.getPdfUrl(code);

    // Fetch through the authenticated axios client (cookie session + CSRF/refresh
    // interceptors); the backend enforces CanViewProblem and caps at 10 MB.
    useEffect(() => {
        let created: string | null = null;
        let cancelled = false;
        setStatus('loading');
        setNumPages(0);
        setObjectUrl(null);
        api.get(`/problem/${code}/pdf`, { responseType: 'blob' })
            .then((res) => {
                if (cancelled) return;
                created = URL.createObjectURL(res.data as Blob);
                setObjectUrl(created);
            })
            .catch(() => {
                if (!cancelled) setStatus('error');
            });
        return () => {
            cancelled = true;
            if (created) URL.revokeObjectURL(created);
        };
    }, [code]);

    // Measure container width so pages fit; re-measure on window resize.
    useEffect(() => {
        const measure = () => setWidth(containerRef.current?.clientWidth ?? 0);
        measure();
        window.addEventListener('resize', measure);
        return () => window.removeEventListener('resize', measure);
    }, []);

    const file = useMemo(() => (objectUrl ? { url: objectUrl } : null), [objectUrl]);
    const pageWidth = width > 0 ? Math.max(200, (width - 32) * zoom) : undefined;

    if (status === 'error') {
        return (
            <div className="bg-card border rounded-3xl shadow-sm">
                <div className="flex flex-col items-center justify-center gap-4 p-10 text-center">
                    <FileText size={40} className="text-red-500" />
                    <p className="text-sm text-muted-foreground">{t('pdfViewer.error')}</p>
                    <div className="flex items-center gap-2">
                        <a href={downloadUrl} download className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg font-bold text-sm">
                            <Download size={16} /> {t('pdfViewer.download')}
                        </a>
                        <a href={downloadUrl} target="_blank" rel="noopener noreferrer" className="flex items-center gap-2 px-4 py-2 border rounded-lg font-bold text-sm">
                            <ArrowUpRight size={16} /> {t('pdfViewer.openNewTab')}
                        </a>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="bg-card border rounded-3xl shadow-sm overflow-hidden flex flex-col">
            <div className="flex items-center justify-between gap-2 border-b p-3">
                <span className="text-xs font-bold text-muted-foreground">
                    {status === 'ready' ? t('pdfViewer.pageCount', { count: numPages }) : t('pdfViewer.loading')}
                </span>
                <div className="flex items-center gap-1">
                    <button onClick={() => setZoom((z) => Math.max(0.5, z - 0.1))} aria-label={t('pdfViewer.zoomOut')} className="p-2 rounded-lg hover:bg-muted transition-colors"><ZoomOut size={16} /></button>
                    <button onClick={() => setZoom((z) => Math.min(2.5, z + 0.1))} aria-label={t('pdfViewer.zoomIn')} className="p-2 rounded-lg hover:bg-muted transition-colors"><ZoomIn size={16} /></button>
                    <a href={downloadUrl} download aria-label={t('pdfViewer.download')} className="p-2 rounded-lg hover:bg-muted transition-colors"><Download size={16} /></a>
                    <a href={downloadUrl} target="_blank" rel="noopener noreferrer" aria-label={t('pdfViewer.openNewTab')} className="p-2 rounded-lg hover:bg-muted transition-colors"><ArrowUpRight size={16} /></a>
                </div>
            </div>

            <div ref={containerRef} className={`overflow-auto bg-muted/30 ${heightClass}`}>
                {status === 'loading' && !file && (
                    <div className="flex items-center justify-center p-10"><Loader2 className="animate-spin text-primary" size={28} /></div>
                )}
                {file && (
                    <Document
                        file={file}
                        onLoadSuccess={({ numPages: n }) => { setNumPages(n); setStatus('ready'); }}
                        onLoadError={() => setStatus('error')}
                        loading={<div className="flex items-center justify-center p-10"><Loader2 className="animate-spin text-primary" size={28} /></div>}
                    >
                        <div className="flex flex-col items-center gap-4 py-4">
                            {Array.from({ length: numPages }, (_, i) => (
                                <Page key={i} pageNumber={i + 1} width={pageWidth} className="shadow-lg" />
                            ))}
                        </div>
                    </Document>
                )}
            </div>
        </div>
    );
}
