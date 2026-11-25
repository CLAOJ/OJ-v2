'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { ChevronDown, ChevronUp, AlertCircle, CheckCircle, XCircle, Clock, HardDrive } from 'lucide-react';
import { cn, getStatusColor } from '@/lib/utils';
import { statusKey } from '@/lib/submissionStatus';

interface TestCase {
    case: number;
    status: string;
    time: number | null;
    memory: number | null;
    points: number | null;
    total: number | null;
    feedback: string;
    extended_feedback?: string;
    // Present only when the viewer may see test data (v1 is_testcase_accessible_by).
    input?: string;
    answer?: string;
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

// One labelled block of the Codeforces-style case view.
function CaseBlock({ label, value, emptyLabel }: { label: string; value: string; emptyLabel: string }) {
    return (
        <div>
            <h5 className="text-xs font-medium text-muted-foreground mb-1">{label}</h5>
            <pre className="text-sm bg-background border p-3 rounded-md overflow-x-auto max-h-60 font-mono whitespace-pre">
                {value === '' ? <span className="italic text-muted-foreground">{emptyLabel}</span> : value}
            </pre>
        </div>
    );
}

function hasCaseDetails(tc: TestCase) {
    return Boolean(tc.feedback || tc.extended_feedback || tc.input !== undefined || tc.output);
}

function TestCaseRow({ tc, isExpanded, onToggle, t }: { tc: TestCase; isExpanded: boolean; onToggle: () => void; t: ReturnType<typeof useTranslations> }) {
    const hasDetails = hasCaseDetails(tc);
    const hasData = tc.input !== undefined;

    return (
        <div className="border-b last:border-b-0">
            <button
                type="button"
                onClick={onToggle}
                disabled={!hasDetails}
                aria-expanded={hasDetails ? isExpanded : undefined}
                aria-label={t('toggleCase', { n: tc.case })}
                className={cn(
                    "grid w-full grid-cols-12 gap-2 px-4 py-3 items-center text-left text-sm transition-colors",
                    hasDetails ? "cursor-pointer hover:bg-muted/50" : "cursor-default"
                )}
            >
                {/* Case Number */}
                <div className="col-span-1 text-center">
                    <span className="font-mono text-muted-foreground">#{tc.case}</span>
                </div>

                {/* Status */}
                <div className="col-span-3 flex items-center gap-2">
                    {STATUS_ICONS[tc.status] || <AlertCircle size={16} className="text-zinc-500" />}
                    <span className={cn("font-medium", getStatusColor(tc.status))} title={tc.status}>
                        {t(`status.${statusKey(tc.status)}`)}
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
                        <span className="font-mono font-medium">
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
            </button>

            {/* Expanded Details */}
            {isExpanded && hasDetails && (
                <div className="px-4 py-4 bg-muted/30 border-t">
                    <div className="space-y-3">
                        {tc.feedback && (
                            <div>
                                <h5 className="text-xs font-medium text-muted-foreground mb-1">{t('feedback')}</h5>
                                <p className="text-sm whitespace-pre-wrap">{tc.feedback}</p>
                            </div>
                        )}
                        {hasData ? (
                            <>
                                <CaseBlock label={t('caseInput')} value={tc.input ?? ''} emptyLabel={t('emptyOutput')} />
                                <CaseBlock label={t('caseOutput')} value={tc.output ?? ''} emptyLabel={t('emptyOutput')} />
                                <CaseBlock label={t('caseAnswer')} value={tc.answer ?? ''} emptyLabel={t('emptyOutput')} />
                            </>
                        ) : tc.output ? (
                            <CaseBlock label={t('output')} value={tc.output} emptyLabel={t('emptyOutput')} />
                        ) : null}
                        {tc.extended_feedback && (
                            <CaseBlock label={t('checkerLog')} value={tc.extended_feedback} emptyLabel={t('emptyOutput')} />
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

export default function TestCaseResults({ testCases, className }: TestCaseResultsProps) {
    const t = useTranslations('Submissions');
    // Open the first failing case that has something to show, as Codeforces does.
    const [expandedCases, setExpandedCases] = useState<Set<number>>(() => {
        const first = testCases.find(tc => tc.status !== 'AC' && hasCaseDetails(tc));
        return new Set(first ? [first.case] : []);
    });

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
            <div className={cn("border rounded-md bg-card p-4 text-center", className)}>
                <p className="text-muted-foreground">{t('noTestCases')}</p>
            </div>
        );
    }

    // Calculate summary statistics
    const passedCount = testCases.filter(tc => tc.status === 'AC').length;
    const totalCount = testCases.length;
    const totalTime = testCases.reduce((sum, tc) => sum + (tc.time || 0), 0);
    const maxMemory = Math.max(...testCases.map(tc => tc.memory || 0));

    return (
        <div className={cn("border rounded-md bg-card overflow-hidden", className)}>
            {/* Summary Header */}
            <div className="px-4 py-3 bg-muted/50 border-b">
                <div className="flex flex-wrap items-center gap-4 text-sm">
                    <span className="font-medium">
                        {t('testCases')}:{' '}
                        <span className={passedCount === totalCount ? "text-emerald-500" : "text-amber-500"}>
                            {passedCount}/{totalCount} {t('passed')}
                        </span>
                    </span>
                    <span className="text-muted-foreground">|</span>
                    <span className="text-muted-foreground">
                        {t('totalTime')}: <span className="font-mono font-medium text-foreground">{totalTime.toFixed(3)}s</span>
                    </span>
                    <span className="text-muted-foreground">|</span>
                    <span className="text-muted-foreground">
                        {t('maxMemory')}: <span className="font-mono font-medium text-foreground">{(maxMemory / 1024).toFixed(2)} MB</span>
                    </span>
                </div>
            </div>

            {/* Table Header */}
            <div className="grid grid-cols-12 gap-2 px-4 py-3 bg-muted/30 border-b text-xs font-medium text-muted-foreground">
                <div className="col-span-1 text-center">#</div>
                <div className="col-span-3">{t('resultHeader')}</div>
                <div className="col-span-2 text-right">{t('timeHeader')}</div>
                <div className="col-span-2 text-right">{t('memoryHeader')}</div>
                <div className="col-span-3 text-right">{t('scoreHeader')}</div>
                <div className="col-span-1"></div>
            </div>

            {testCases.some(tc => tc.input !== undefined) && (
                <p className="px-4 py-2 border-b text-xs text-muted-foreground">{t('truncatedNote')}</p>
            )}

            {/* Test Cases */}
            <div className="divide-y">
                {testCases.map((tc) => (
                    <TestCaseRow
                        key={tc.case}
                        tc={tc}
                        isExpanded={expandedCases.has(tc.case)}
                        onToggle={() => toggleCase(tc.case)}
                        t={t}
                    />
                ))}
            </div>
        </div>
    );
}
