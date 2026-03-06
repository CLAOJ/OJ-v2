'use client';

import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { ContestClarificationList } from '@/components/contests/ClarificationList';
import { Badge } from '@/components/ui/Badge';
import { MessageSquare } from 'lucide-react';

export default function AdminContestClarificationsPage() {
    const params = useParams();
    const t = useTranslations();
    const contestKey = params.key as string;

    return (
        <div className="container mx-auto py-8 space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">{t('admin.contest_clarifications')}</h1>
                    <p className="text-muted-foreground mt-1">
                        Answer clarification questions from contestants
                    </p>
                </div>
                <Badge variant="default">
                    <MessageSquare className="w-4 h-4 mr-2" />
                    Staff View
                </Badge>
            </div>

            <div className="bg-card border rounded-xl p-6">
                <ContestClarificationList
                    contestKey={contestKey}
                    canCreate={false}
                    isAdmin={true}
                />
            </div>
        </div>
    );
}
