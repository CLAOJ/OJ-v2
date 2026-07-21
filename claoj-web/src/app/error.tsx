'use client';

import { useEffect } from 'react';
import { Button } from '@/components/ui/Button';
import { AlertTriangle, RefreshCw, Home } from 'lucide-react';
import { Link } from '@/navigation';
import { getErrorMessages } from '@/lib/errorBoundaryMessages';

interface ErrorProps {
    error: Error & { digest?: string };
    reset: () => void;
}

export default function Error({ error, reset }: ErrorProps) {
    // This boundary can catch a failure in the [locale] layout itself, so the
    // next-intl provider may not exist here — see lib/errorBoundaryMessages.
    const t = getErrorMessages();

    useEffect(() => {
        // Log error to error reporting service (when configured)
        // console.error('Application error:', error);
    }, [error]);

    return (
        <div className="min-h-[400px] flex flex-col items-center justify-center p-8">
            <div className="text-center space-y-6 max-w-md">
                <div className="flex justify-center">
                    <div className="h-20 w-20 rounded-full bg-destructive/10 flex items-center justify-center">
                        <AlertTriangle className="h-10 w-10 text-destructive" />
                    </div>
                </div>

                <div className="space-y-2">
                    <h2 className="text-2xl font-bold">{t('errorTitle')}</h2>
                    <p className="text-muted-foreground">
                        {t('errorDescription')}
                    </p>
                </div>

                {process.env.NODE_ENV === 'development' && (
                    <div className="text-left p-4 bg-muted rounded-lg text-sm font-mono overflow-auto">
                        {error.message}
                    </div>
                )}

                <div className="flex flex-col sm:flex-row gap-3 justify-center pt-4">
                    <Button
                        onClick={reset}
                        className="flex items-center gap-2"
                    >
                        <RefreshCw size={18} />
                        {t('tryAgain')}
                    </Button>
                    <Link href="/">
                        <Button variant="outline" className="flex items-center gap-2">
                            <Home size={18} />
                            {t('goHome')}
                        </Button>
                    </Link>
                </div>
            </div>
        </div>
    );
}
