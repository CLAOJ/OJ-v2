'use client';

import { useEffect } from 'react';

export default function GlobalError({
    error,
    reset,
}: {
    error: Error & { digest?: string };
    reset: () => void;
}) {
    useEffect(() => {
        // Log error to error reporting service (when configured)
        // console.error('Global error:', error);
    }, [error]);

    return (
        <html>
            <body>
                <div className="min-h-screen flex items-center justify-center p-8 bg-gray-50">
                    <div className="text-center space-y-6 max-w-md">
                        <div className="flex justify-center">
                            <div className="h-20 w-20 rounded-full bg-red-100 flex items-center justify-center">
                                <svg className="h-10 w-10 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                                </svg>
                            </div>
                        </div>

                        <div className="space-y-2">
                            <h2 className="text-2xl font-bold text-gray-900">Critical Error</h2>
                            <p className="text-gray-600">
                                A critical error occurred. Please refresh the page or try again later.
                            </p>
                        </div>

                        {process.env.NODE_ENV === 'development' && (
                            <div className="text-left p-4 bg-gray-100 rounded-lg text-sm font-mono overflow-auto text-gray-800">
                                {error.message}
                            </div>
                        )}

                        <div className="flex flex-col sm:flex-row gap-3 justify-center pt-4">
                            <button
                                onClick={reset}
                                className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
                            >
                                Try again
                            </button>
                            <a
                                href="/"
                                className="px-6 py-3 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors font-medium"
                            >
                                Go home
                            </a>
                        </div>
                    </div>
                </div>
            </body>
        </html>
    );
}
