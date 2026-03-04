'use client';

import { useState } from 'react';
import { ChevronDown, ChevronUp, AlertCircle, CheckCircle, XCircle, Clock, HardDrive } from 'lucide-react';
import { cn, getStatusColor } from '@/lib/utils';

interface TestCase {
    case: number;
    status: string;
    time: number | null;
    memory: number | null;
    points: number | null;
    total: number | null;
    feedback: string;
    extended_feedback?: string;
    output?: string;
}

interface TestCaseResultsProps {
    testCases: TestCase[];
    className?: string;
}

const STATUS_ICONS: Record<string, React.ReactNode> = {
    'AC': <CheckCircle size={16} className="text-emerald-500" />,
    'WA': <XCircle size={16} className="text-destructive" />,
    'TLE': <Clock size={16} className="text-amber-500" />,
    'MLE': <HardDrive size={16} className="text-amber-500" />,
    'OLE': <AlertCircle size={16} className="text-amber-500" />,
    'RE': <AlertCircle size={16} className="text-purple-500" />,
    'RTE': <AlertCircle size={16} className="text-purple-500" />,
    'CE': <AlertCircle size={16} className="text-blue-500" />,
    'IE': <AlertCircle size={16} className="text-zinc-500" />,
};

const STATUS_NAMES: Record<string, string> = {
    'AC': 'Accepted',
    'WA': 'Wrong Answer',
    'TLE': 'Time Limit Exceeded',
    'MLE': 'Memory Limit Exceeded',
    'OLE': 'Output Limit Exceeded',
    'RE': 'Runtime Error',
    'RTE': 'Runtime Error',
    'CE': 'Compile Error',
    'IE': 'Internal Error',
    'SC': 'Short Circuited',
    'QU': 'Queued',
};

function TestCaseRow({ tc, isExpanded, onToggle }: { tc: TestCase; isExpanded: boolean; onToggle: () => void }) {
    const hasDetails = tc.feedback || tc.extended_feedback || tc.output;

    return (
        <div className="border-b last:border-b-0">
            <div
                onClick={() => hasDetails && onToggle()}
                className={cn(
                    "grid grid-cols-12 gap-2 px-4 py-3 items-center text-sm transition-colors",
                    hasDetails && "cursor-pointer hover:bg-muted/50",
                    !hasDetails && "cursor-default"
                )}
            >
                {/* Case Number */}
                <div className="col-span-1 text-center">
                    <span className="font-mono text-muted-foreground">#{tc.case}</span>
                </div>

                {/* Status */}
                <div className="col-span-3 flex items-center gap-2">
                    {STATUS_ICONS[tc.status] || <AlertCircle size={16} className="text-zinc-500" />}
                    <span className={cn("font-bold", getStatusColor(tc.status))}>
                        {tc.status}
                    </span>
                </div>

                {/* Time */}
                <div className="col-span-2 text-right font-mono text-muted-foreground">
                    {tc.time !== null ? `${tc.time.toFixed(3)}s` : '-'}
                </div>

                {/* Memory */}
                <div className="col-span-2 text-right font-mono text-muted-foreground">
                    {tc.memory !== null ? `${(tc.memory / 1024).toFixed(2)} MB` : '-'}
                </div>

                {/* Score */}
                <div className="col-span-3 text-right">
                    {tc.points !== null && tc.total !== null ? (
                        <span className="font-mono font-bold">
                            <span className={tc.points > 0 ? "text-emerald-500" : "text-destructive"}>
                                {tc.points}
                            </span>
                            <span className="text-muted-foreground"> / {tc.total}</span>
                        </span>
                    ) : (
                        <span className="font-mono text-muted-foreground">-</span>
                    )}
                </div>

                {/* Expand Icon */}
                <div className="col-span-1 text-center">
                    {hasDetails && (
                        isExpanded ? <ChevronUp size={16} className="text-muted-foreground" /> : <ChevronDown size={16} className="text-muted-foreground" />
                    )}
                </div>
            </div>

            {/* Expanded Details */}
            {isExpanded && hasDetails && (
                <div className="px-4 py-4 bg-muted/30 border-t">
                    <div className="space-y-3">
                        {tc.feedback && (
                            <div>
                                <h5 className="text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">Feedback</h5>
                                <p className="text-sm whitespace-pre-wrap">{tc.feedback}</p>
                            </div>
                        )}
                        {tc.extended_feedback && (
                            <div>
                                <h5 className="text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">Extended Feedback</h5>
                                <pre className="text-sm bg-background p-3 rounded-lg overflow-x-auto">{tc.extended_feedback}</pre>
                            </div>
                        )}
                        {tc.output && (
                            <div>
                                <h5 className="text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">Output</h5>
                                <pre className="text-sm bg-background p-3 rounded-lg overflow-x-auto font-mono">{tc.output}</pre>
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

export default function TestCaseResults({ testCases, className }: TestCaseResultsProps) {
    const [expandedCases, setExpandedCases] = useState<Set<number>>(new Set());

    const toggleCase = (caseNum: number) => {
        const newExpanded = new Set(expandedCases);
        if (newExpanded.has(caseNum)) {
            newExpanded.delete(caseNum);
        } else {
            newExpanded.add(caseNum);
        }
        setExpandedCases(newExpanded);
    };

    if (testCases.length === 0) {
        return (
            <div className={cn("border rounded-2xl bg-card p-8 text-center", className)}>
                <p className="text-muted-foreground">No test cases available yet.</p>
            </div>
        );
    }

    // Calculate summary statistics
    const passedCount = testCases.filter(tc => tc.status === 'AC').length;
    const totalCount = testCases.length;
    const totalTime = testCases.reduce((sum, tc) => sum + (tc.time || 0), 0);
    const maxMemory = Math.max(...testCases.map(tc => tc.memory || 0));

    return (
        <div className={cn("border rounded-2xl bg-card overflow-hidden", className)}>
            {/* Summary Header */}
            <div className="px-4 py-3 bg-muted/50 border-b">
                <div className="flex flex-wrap items-center gap-4 text-sm">
                    <span className="font-bold">
                        Test Cases:{' '}
                        <span className={passedCount === totalCount ? "text-emerald-500" : "text-amber-500"}>
                            {passedCount}/{totalCount} passed
                        </span>
                    </span>
                    <span className="text-muted-foreground">|</span>
                    <span className="text-muted-foreground">
                        Total Time: <span className="font-mono font-medium text-foreground">{totalTime.toFixed(3)}s</span>
                    </span>
                    <span className="text-muted-foreground">|</span>
                    <span className="text-muted-foreground">
                        Max Memory: <span className="font-mono font-medium text-foreground">{(maxMemory / 1024).toFixed(2)} MB</span>
                    </span>
                </div>
            </div>

            {/* Table Header */}
            <div className="grid grid-cols-12 gap-2 px-4 py-3 bg-muted/30 border-b text-xs font-bold uppercase tracking-wider text-muted-foreground">
                <div className="col-span-1 text-center">#</div>
                <div className="col-span-3">Result</div>
                <div className="col-span-2 text-right">Time</div>
                <div className="col-span-2 text-right">Memory</div>
                <div className="col-span-3 text-right">Score</div>
                <div className="col-span-1"></div>
            </div>

            {/* Test Cases */}
            <div className="divide-y">
                {testCases.map((tc) => (
                    <TestCaseRow
                        key={tc.case}
                        tc={tc}
                        isExpanded={expandedCases.has(tc.case)}
                        onToggle={() => toggleCase(tc.case)}
                    />
                ))}
            </div>
        </div>
    );
}
