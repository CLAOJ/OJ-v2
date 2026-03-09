'use client';

import Link from 'next/link';
import { FileCode, User, Calendar } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { getStatusVariant } from '@/lib/utils';

interface SubmissionInfo {
    id: number;
    problem: string;
    user: string;
    date: string;
    language: string;
    result: string | null;
}

interface SubmissionInfoCardProps {
    title: string;
    submission: SubmissionInfo;
}

export function SubmissionInfoCard({ title, submission }: SubmissionInfoCardProps) {
    return (
        <div className="bg-card border rounded-2xl p-6 space-y-3">
            <div className="flex items-center justify-between">
                <h3 className="text-sm font-black uppercase tracking-widest text-muted-foreground">
                    {title}
                </h3>
                <Badge variant={getStatusVariant(submission.result || '')}>
                    {submission.result || 'Unknown'}
                </Badge>
            </div>
            <div className="space-y-2">
                <Link
                    href={`/problems/${submission.problem}`}
                    className="flex items-center gap-2 text-sm hover:text-primary transition-colors"
                >
                    <FileCode size={16} />
                    <span className="font-bold">{submission.problem}</span>
                </Link>
                <Link
                    href={`/user/${submission.user}`}
                    className="flex items-center gap-2 text-sm hover:text-primary transition-colors"
                >
                    <User size={16} />
                    <span className="font-medium">@{submission.user}</span>
                </Link>
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Calendar size={16} />
                    {new Date(submission.date).toLocaleString()}
                </div>
                <div className="text-sm font-mono text-muted-foreground">
                    {submission.language}
                </div>
            </div>
        </div>
    );
}
