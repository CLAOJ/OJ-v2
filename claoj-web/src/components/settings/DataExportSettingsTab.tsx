'use client';

import api from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Loader2, Info, Download } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';

export default function DataExportSettingsTab() {
    const { data: exportStatus, refetch } = useQuery<{
        last_export?: string;
        can_request: boolean;
        days_until_request?: number;
        download_url?: string;
    }>({
        queryKey: ['user-export', 'status'],
        queryFn: async () => {
            const res = await api.get('/user/export/status');
            return res.data;
        },
    });

    const { mutate: requestExport, isPending: isRequesting } = useMutation({
        mutationFn: async () => {
            const res = await api.post('/user/export/request');
            return res.data;
        },
        onSuccess: () => {
            refetch();
        },
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to request data export');
        },
    });

    return (
        <div className="space-y-6">
            <div className="space-y-2">
                <h2 className="text-2xl font-bold">Download Your Data</h2>
                <p className="text-muted-foreground text-sm">
                    Export all your personal data including submissions, comments, blog posts, and contest participations.
                </p>
            </div>

            <div className="p-6 rounded-2xl border bg-muted/50 space-y-4">
                <div className="flex items-start gap-3">
                    <Info className="w-5 h-5 text-primary mt-0.5" />
                    <div className="space-y-2 text-sm">
                        <p className="font-medium">What&apos;s included in your export:</p>
                        <ul className="list-disc list-inside space-y-1 text-muted-foreground ml-2">
                            <li>Profile information and preferences</li>
                            <li>All submissions with source code</li>
                            <li>Comments and blog posts</li>
                            <li>Support tickets</li>
                            <li>Contest participations and ratings</li>
                            <li>Organization memberships</li>
                        </ul>
                    </div>
                </div>
            </div>

            <div className="p-6 rounded-2xl border bg-card">
                <h3 className="text-lg font-bold mb-4">Export Status</h3>

                {exportStatus ? (
                    <div className="space-y-4">
                        <div className="flex items-center justify-between">
                            <span className="text-sm text-muted-foreground">Last export:</span>
                            <span className="text-sm font-medium">
                                {exportStatus.last_export
                                    ? new Date(exportStatus.last_export).toLocaleDateString()
                                    : 'Never'}
                            </span>
                        </div>

                        <div className="flex items-center justify-between">
                            <span className="text-sm text-muted-foreground">Can request export:</span>
                            <Badge variant={exportStatus.can_request ? 'success' : 'secondary'}>
                                {exportStatus.can_request ? 'Yes' : 'No'}
                            </Badge>
                        </div>

                        {!exportStatus.can_request && (
                            <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-700 text-sm">
                                <p className="font-medium flex items-center gap-2">
                                    <Info size={16} />
                                    Rate limit active
                                </p>
                                <p className="mt-1">
                                    You can request a new data export in {exportStatus.days_until_request} days.
                                </p>
                            </div>
                        )}

                        {exportStatus.download_url && (
                            <a
                                href={exportStatus.download_url}
                                className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold text-sm hover:bg-primary/90 transition-colors"
                            >
                                <Download size={16} />
                                Download Export
                            </a>
                        )}
                    </div>
                ) : (
                    <div className="text-sm text-muted-foreground">No export data available</div>
                )}
            </div>

            <button
                onClick={() => requestExport()}
                disabled={isRequesting || !exportStatus?.can_request}
                className="w-full px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold text-sm hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
            >
                {isRequesting && <Loader2 size={16} className="animate-spin" />}
                Request Data Export
            </button>
        </div>
    );
}
