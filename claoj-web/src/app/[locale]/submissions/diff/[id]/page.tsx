'use client';

import { useSearchParams } from 'next/navigation';
import { use } from 'react';
import SubmissionDiffViewer from '@/components/submission/SubmissionDiffViewer';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

export default function SubmissionDiffPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = use(params);
    const searchParams = useSearchParams();
    const compareId = searchParams.get('compare');

    if (!compareId) {
        return (
            <div className="max-w-4xl mx-auto p-8 space-y-4">
                <div className="flex items-center gap-2 text-muted-foreground">
                    <Link href="/submissions" className="hover:text-primary transition-colors flex items-center gap-2">
                        <ArrowLeft size={16} />
                        Back to Submissions
                    </Link>
                </div>
                <div className="p-8 text-center space-y-4">
                    <h2 className="text-2xl font-bold">Missing comparison ID</h2>
                    <p className="text-muted-foreground">
                        Please provide a submission ID to compare with using the <code className="bg-muted px-2 py-1 rounded">compare</code> query parameter.
                    </p>
                    <p className="text-sm text-muted-foreground">
                        Example: <code className="bg-muted px-2 py-1 rounded">/submissions/diff/123?compare=456</code>
                    </p>
                </div>
            </div>
        );
    }

    return (
        <div className="max-w-7xl mx-auto p-4 space-y-4">
            <div className="flex items-center gap-2 text-muted-foreground mb-6">
                <Link href="/submissions" className="hover:text-primary transition-colors flex items-center gap-2">
                    <ArrowLeft size={16} />
                    Back to Submissions
                </Link>
            </div>
            <SubmissionDiffViewer
                submission1Id={parseInt(id)}
                submission2Id={parseInt(compareId)}
            />
        </div>
    );
}
