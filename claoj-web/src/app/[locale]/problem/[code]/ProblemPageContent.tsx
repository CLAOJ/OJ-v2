'use client';

import { useQuery, useMutation } from '@tanstack/react-query';
import { useRouter, Link } from '@/navigation';
import { useTranslations } from 'next-intl';
import dynamic from 'next/dynamic';
import api, { problemClarificationApi, problemPdfApi, problemSubmitApi } from '@/lib/api';
import { ProblemDetail, ProblemClarification } from '@/types';
import MathRenderer from '@/components/ui/MathRenderer';
import CodeEditor from '@/components/ui/CodeEditor';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { useState, useEffect, useRef, use } from 'react';
import {
    Clock,
    HardDrive,
    User as UserIcon,
    Tag,
    Send,
    ChevronRight,
    Loader2,
    CheckCircle2,
    XCircle,
    Activity,
    ArrowUpRight,
    Trophy,
    Monitor,
    Code2,
    BookOpen,
    MessageSquare,
    FileText,
    Flag,
    Printer,
    Upload,
    Scale,
    X
} from 'lucide-react';
import { cn, formatMemoryLimit } from '@/lib/utils';
import { useSearchParams } from 'next/navigation';
import Comments from '@/components/common/Comments';
import { useRequireAuth } from '@/hooks/useRequireAuth';
import { useAuth } from '@/components/providers/AuthProvider';
import { toast } from 'sonner';
import {
    MAX_TEXT_SOURCE_BYTES,
    problemPrintPath,
    problemReportPath,
    problemTicketsPath,
    validateSubmissionFile,
    validateTextSourceFile,
} from '@/lib/submissionFile';
import dayjs from 'dayjs';

// react-pdf touches browser-only APIs — load it client-side only.
const PdfStatementViewer = dynamic(() => import('@/components/ui/PdfStatementViewer'), {
    ssr: false,
    loading: () => (
        <div className="flex items-center justify-center p-4 bg-card border rounded-md shadow-sm">
            <Loader2 className="animate-spin text-primary" size={28} />
        </div>
    ),
});

export default function ProblemPageContent({ params }: { params: Promise<{ code: string }> }) {
    const { code } = use(params);
    const searchParams = useSearchParams();
    const contestKey = searchParams.get('contest');
    const router = useRouter();
    const t = useTranslations('Problems');
    const tAuth = useTranslations('Auth');
    const { requireAuth } = useRequireAuth();
    const [codeValue, setCodeValue] = useState('// Your code here\n#include <iostream>\n\nint main() {\n    return 0;\n}');
    // Empty until the problem's allowed languages arrive — see the effect below.
    // This used to default to the literal 'cpp', which is not a language key the
    // backend knows (the keys are C, CPP17, PY3, ...). No <option> matched it, so
    // the browser displayed the first option while React still held 'cpp', and
    // submitting without touching the dropdown always failed with
    // {"error":"invalid language"}.
    const [language, setLanguage] = useState('');
    const [langOpen, setLangOpen] = useState(false);
    const [showPdfViewer, setShowPdfViewer] = useState(false);
    const [submissionFile, setSubmissionFile] = useState<File | null>(null);
    const [isReadingFile, setIsReadingFile] = useState(false);
    const submissionFileInput = useRef<HTMLInputElement>(null);
    const textFileInput = useRef<HTMLInputElement>(null);

    const { data: problem, isLoading } = useQuery({
        queryKey: ['problem', code, contestKey],
        queryFn: async () => {
            const url = contestKey ? `/problem/${code}?contest=${contestKey}` : `/problem/${code}`;
            const res = await api.get<ProblemDetail>(url);
            return res.data;
        }
    });

    // Keep the selected language pinned to something the problem actually
    // allows: on first load, and again if the user navigates to a problem whose
    // allowed set doesn't include the current pick. The starter snippet is C++,
    // so prefer a C++ variant when one is permitted.
    useEffect(() => {
        const allowed = problem?.languages;
        if (!allowed?.length) return;
        if (allowed.some(l => l.key === language)) return;
        const preferred = allowed.find(l => l.key === 'CPP17')
            ?? allowed.find(l => l.key.startsWith('CPP'))
            ?? allowed[0];
        setLanguage(preferred.key);
    }, [problem, language]);

    // v1 /problem/<p>/resubmit/<submission>: prefill the editor from an earlier
    // submission. The API decides whether this viewer may read it (owner, or
    // judge.resubmit_other), so a copied link never leaks someone's source.
    const { user } = useAuth();
    const resubmitId = searchParams.get('resubmit');
    const { data: resubmitData } = useQuery({
        queryKey: ['resubmit', resubmitId],
        queryFn: async () => {
            const res = await api.get<{ language: string; source: string }>(
                `/submission/${resubmitId}/resubmit`
            );
            return res.data;
        },
        enabled: !!resubmitId && !!user,
        retry: false
    });
    // Applied once per id: after that the editor belongs to the user.
    const appliedResubmit = useRef<string | null>(null);
    useEffect(() => {
        if (!resubmitId || !resubmitData || !problem || appliedResubmit.current === resubmitId) return;
        appliedResubmit.current = resubmitId;
        if (!problem.languages.some(l => l.key === resubmitData.language)) {
            toast.error(t('resubmitFailed'));
            return;
        }
        setLanguage(resubmitData.language);
        setCodeValue(resubmitData.source);
        toast.success(t('resubmitLoaded', { id: resubmitId }));
    }, [resubmitId, resubmitData, problem, t]);

    const { data: editorialData } = useQuery({
        queryKey: ['editorial', code],
        queryFn: async () => {
            const res = await api.get<{ exists: boolean }>(`/problem/${code}/solution/exists`);
            return res.data;
        },
        enabled: !!problem
    });

    const { data: clarificationsData } = useQuery({
        queryKey: ['problem-clarifications', code],
        queryFn: async () => {
            const res = await problemClarificationApi.getClarifications(code);
            return res.data.data;
        },
        enabled: !!problem
    });

    // The selected language decides the submit shape: file-only languages take
    // an uploaded file, everything else keeps the JSON source path.
    const selectedLanguage = problem?.languages.find(l => l.key === language);
    const isFileOnly = !!selectedLanguage?.file_only;
    // DMOJ stores the extension without a dot ("zip"); tolerate either shape.
    const fileExtension = (selectedLanguage?.extension ?? '').replace(/^\./, '');

    const { mutate: submitSolution, isPending: isSubmitting } = useMutation({
        mutationFn: async () => {
            const body = isFileOnly
                ? { language, file: submissionFile ?? undefined }
                : { language, source: codeValue };
            const res = await problemSubmitApi.submit(code, body, contestKey);
            return res.data;
        },
        onSuccess: (data) => {
            router.push(`/submission/${data.submission_id}`);
        },
        onError: (err) => {
            const message = (err as { response?: { data?: { error?: string } } }).response?.data?.error;
            toast.error(message || t('submitFailed'));
        }
    });

    const pickSubmissionFile = (file: File | null | undefined) => {
        if (!file || !selectedLanguage) return;
        const reason = validateSubmissionFile(file, selectedLanguage);
        if (reason) {
            setSubmissionFile(null);
            toast.error(t(`fileSubmit.errors.${reason}`));
            return;
        }
        setSubmissionFile(file);
    };

    const loadTextFile = async (file: File | null | undefined) => {
        if (!file) return;
        const reason = validateTextSourceFile(file);
        if (reason) {
            toast.error(t(`fileSubmit.errors.${reason}`));
            return;
        }
        setIsReadingFile(true);
        try {
            setCodeValue(await file.text());
        } catch {
            toast.error(t('fileSubmit.errors.unreadable'));
        } finally {
            setIsReadingFile(false);
        }
    };

    const handleSubmitClick = () => requireAuth(() => {
        if (isFileOnly && !submissionFile) {
            toast.error(t('fileSubmit.errors.missing'));
            return;
        }
        submitSolution();
    }, tAuth('loginToSubmit'));

    if (isLoading) {
        return (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 h-[calc(100vh-12rem)] ">
                <div className="space-y-6 overflow-y-auto pr-4">
                    <Skeleton className="h-10 w-3/4" />
                    <div className="flex gap-4">
                        <Skeleton className="h-6 w-24" />
                        <Skeleton className="h-6 w-24" />
                    </div>
                    <Skeleton className="h-[60vh] w-full" />
                </div>
                <Skeleton className="h-full w-full rounded-md" />
            </div>
        );
    }

    if (!problem) return <div className="p-4">{t('notFound')}</div>;

    return (
        <div className="flex flex-col lg:flex-row gap-4 min-h-[calc(100vh-12rem)] mt-4">
            {/* Left Sidebar: Problem Metadata & Status */}
            <aside className="order-last lg:order-none w-full lg:w-72 flex flex-col gap-6 shrink-0 h-fit lg:sticky lg:top-4">
                {/* Status Card */}
                <div className={cn(
                    "relative overflow-hidden rounded-md border p-6 flex flex-col items-center gap-4 transition-all shadow-sm",
                    problem.is_solved ? "bg-emerald-50/50 border-emerald-200" :
                        problem.is_attempted ? "bg-amber-50/50 border-amber-200" : "bg-card shadow-sm"
                )}>
                    {problem.is_solved ? (
                        <div className="p-4 rounded-full bg-emerald-700 text-white">
                            <CheckCircle2 size={40} />
                        </div>
                    ) : problem.is_attempted ? (
                        <div className="p-4 rounded-full bg-amber-700 text-white">
                            <XCircle size={40} />
                        </div>
                    ) : null}

                    <div className="text-center space-y-1">
                        <p className="text-xs font-medium text-muted-foreground">
                            {problem.is_solved ? t('solved') : problem.is_attempted ? t('attempted') : t('unsolved')}
                        </p>
                        <h2 className="text-xl font-semibold tracking-tight">{problem.code}</h2>
                    </div>

                    <div className="absolute top-2 right-2 text-xs font-mono text-muted-foreground opacity-30">
                        {t('idLabel', { id: String(problem.id) })}
                    </div>
                </div>

                {/* Submissions Links */}
                <div className="bg-card border rounded-md p-6 shadow-sm space-y-3">
                    <h3 className="text-xs font-semibold text-muted-foreground mb-4">{t('activity')}</h3>
                    <Link
                        href={`/submissions?problem=${problem.code}`}
                        className="flex items-center justify-between group p-3 rounded-md hover:bg-primary/5 transition-all outline-none"
                    >
                        <div className="flex items-center gap-3">
                            <Activity size={18} className="text-muted-foreground" />
                            <span className="text-sm font-medium">{t('allSubmissions')}</span>
                        </div>
                        <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all" />
                    </Link>
                    <Link
                        href={`/submissions?problem=${problem.code}&best=1`}
                        className="flex items-center justify-between group p-3 rounded-md hover:bg-amber-500/5 transition-all outline-none"
                    >
                        <div className="flex items-center gap-3">
                            <Trophy size={18} className="text-muted-foreground" />
                            <span className="text-sm font-medium">{t('bestSolutions')}</span>
                        </div>
                        <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all" />
                    </Link>
                </div>

                {/* Editorial Link */}
                {editorialData?.exists && (
                    <div className="bg-card border rounded-md p-6 shadow-sm">
                        <Link
                            href={`/problem/${code}/editorial`}
                            className="flex items-center gap-3 p-3 rounded-md hover:bg-primary/5 transition-all outline-none"
                        >
                            <BookOpen size={18} className="text-muted-foreground" />
                            <span className="text-sm font-medium">{t('editorial')}</span>
                            <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all ml-auto" />
                        </Link>
                    </div>
                )}

                {/* PDF Statement Link */}
                {problem.pdf_url && (
                    <div className="bg-card border rounded-md p-6 shadow-sm">
                        <button
                            onClick={() => setShowPdfViewer(true)}
                            className="flex items-center gap-3 p-3 rounded-md hover:bg-primary/5 transition-all outline-none w-full"
                        >
                            <FileText size={18} className="text-muted-foreground" />
                            <span className="text-sm font-medium">{t('pdfViewer.sidebarLabel')}</span>
                            <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all ml-auto" />
                        </button>
                    </div>
                )}

                {/* Print / Save as PDF. v2 has no server-side HTML to PDF engine, so
                    problems without an uploaded PDF open a print-optimized view and
                    hand off to the browser's print dialog. */}
                {!problem.pdf_url && (
                    <div className="bg-card border rounded-md p-6 shadow-sm">
                        <Link
                            href={problemPrintPath(problem.code)}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="flex items-center gap-3 p-3 rounded-md hover:bg-primary/5 transition-all outline-none"
                        >
                            <Printer size={18} className="text-muted-foreground" />
                            <span className="text-sm font-medium">{t('downloadPdf')}</span>
                            <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all ml-auto" />
                        </Link>
                    </div>
                )}

                {/* Problem-scoped report / clarification ticket */}
                <div className="bg-card border rounded-md p-6 shadow-sm">
                    <Link
                        href={problemReportPath(problem.code)}
                        className="flex items-center gap-3 p-3 rounded-md hover:bg-primary/5 transition-all outline-none"
                    >
                        <Flag size={18} className="text-muted-foreground" />
                        <span className="text-sm font-medium">{t('reportProblem')}</span>
                        <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all ml-auto" />
                    </Link>
                </div>

                {/* v1 problem.html: "Manage tickets" for editors, "My tickets"
                    for users who already have one on this problem. */}
                {(problem.can_edit || problem.has_tickets) && (
                    <div className="bg-card border rounded-md p-6 shadow-sm">
                        <Link
                            href={problemTicketsPath(problem.code)}
                            className="flex items-center gap-3 p-3 rounded-md hover:bg-primary/5 transition-all outline-none"
                        >
                            <MessageSquare size={18} className="text-muted-foreground" />
                            <span className="text-sm font-medium">
                                {problem.can_edit ? t('manageTickets') : t('myTickets')}
                            </span>
                            <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all ml-auto" />
                        </Link>
                    </div>
                )}

                {/* Problem Clarifications Link */}
                {clarificationsData && clarificationsData.length > 0 && (
                    <div className="bg-card border rounded-md p-6 shadow-sm">
                        <Link
                            href={`#clarifications`}
                            className="flex items-center gap-3 p-3 rounded-md hover:bg-primary/5 transition-all outline-none"
                        >
                            <MessageSquare size={18} className="text-muted-foreground" />
                            <span className="text-sm font-medium">{t('clarificationsCount', { count: clarificationsData.length })}</span>
                            <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all ml-auto" />
                        </Link>
                    </div>
                )}

                {/* Technical Details */}
                <div className="bg-card border rounded-md p-6 shadow-sm space-y-6">
                    <h3 className="text-xs font-semibold text-muted-foreground">{t('details')}</h3>

                    <div className="space-y-4">
                        <div className="flex flex-col gap-1.5">
                            <div className="flex items-center gap-2 text-muted-foreground text-xs font-medium ">
                                <Monitor size={14} className="text-muted-foreground" />
                                <span>{t('ioMethod')}</span>
                            </div>
                            <span className="text-sm font-medium">{t('standardConsole')}</span>
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <div className="flex items-center gap-2 text-muted-foreground text-xs font-medium ">
                                <Clock size={14} className="text-muted-foreground" />
                                <span>{t('timeLimit')}</span>
                            </div>
                            <span className="text-sm font-medium">{problem.time_limit}s</span>
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <div className="flex items-center gap-2 text-muted-foreground text-xs font-medium ">
                                <HardDrive size={14} className="text-muted-foreground" />
                                <span>{t('memoryLimit')}</span>
                            </div>
                            <span className="text-sm font-medium">{formatMemoryLimit(problem.memory_limit)}</span>
                        </div>
                    </div>

                    <div className="pt-4 border-t space-y-2">
                        <div className="flex justify-between items-center text-xs">
                            <span className="text-muted-foreground font-medium">{t('acRate')}</span>
                            <span className="font-medium text-primary">{Math.round(problem.ac_rate)}%</span>
                        </div>
                        <div className="w-full h-1.5 bg-muted rounded-full overflow-hidden" role="presentation">
                            <div
                                className="h-full bg-primary transition-all duration-500"
                                style={{ width: `${Math.min(100, Math.max(0, problem.ac_rate))}%` }}
                                role="progressbar"
                                aria-valuenow={Math.round(problem.ac_rate)}
                                aria-valuemin={0}
                                aria-valuemax={100}
                                aria-label={t('acRateAria')}
                            />
                        </div>
                    </div>
                </div>
            </aside>

            {/* Main Content: Statement & Editor */}
            <div className="flex-1 flex flex-col gap-4 min-w-0">
                <div className="grid grid-cols-1 xl:grid-cols-2 gap-4 items-start">
                    {/* Problem Statement */}
                    <div className="space-y-4 min-w-0">
                        <header className="space-y-2">
                            <h1 className="text-3xl lg:text-4xl font-semibold tracking-tight leading-tight text-balance">
                                {problem.name}
                            </h1>
                            <div className="flex flex-wrap gap-2 pt-2">
                                {problem.types.map(type => (
                                    <Badge key={type.name} variant="secondary" className="px-3 py-1 bg-muted/50 font-medium hover:bg-primary hover:text-primary-foreground transition-all cursor-default">
                                        {type.name}
                                    </Badge>
                                ))}
                                <Badge variant="outline" className="border-primary/20 bg-primary/5 text-primary font-medium px-3 py-1">
                                    {problem.points} {t('points')}
                                </Badge>
                                {problem.partial && (
                                    <Badge variant="outline" className="border-amber-500/30 bg-amber-500/5 text-amber-600 font-medium px-3 py-1">
                                        {t('partialScoring')}
                                    </Badge>
                                )}
                            </div>
                        </header>

                        {(problem.description?.trim() || !problem.pdf_url) && (
                            <div className="prose prose-sm dark:prose-invert max-w-none bg-card border rounded-md p-4 shadow-sm leading-relaxed">
                                <MathRenderer content={problem.description} copyable />
                            </div>
                        )}
                        {problem.pdf_url && <PdfStatementViewer code={code} />}

                        {/* Authors, group and license: one metadata line, not three cards. */}
                        <dl className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
                            <div className="flex items-center gap-2">
                                <dt className="flex items-center gap-1.5 text-muted-foreground"><UserIcon size={14} aria-hidden /> {t('authors')}</dt>
                                <dd className="flex flex-wrap gap-x-2 font-medium">
                                    {problem.authors.length ? problem.authors.map(a => (
                                        <Link key={a.username} href={`/user/${a.username}`} className="hover:text-primary hover:underline underline-offset-4">
                                            {a.username}
                                        </Link>
                                    )) : '—'}
                                </dd>
                            </div>
                            <div className="flex items-center gap-2 min-w-0">
                                <dt className="flex items-center gap-1.5 text-muted-foreground"><Tag size={14} aria-hidden /> {t('group')}</dt>
                                <dd className="font-medium truncate">{problem.group}</dd>
                            </div>
                            {/* v1 problem.html: license link labelled display or name. */}
                            {problem.license && (
                                <div className="flex items-center gap-2">
                                    <dt className="flex items-center gap-1.5 text-muted-foreground"><Scale size={14} aria-hidden /> {t('license')}</dt>
                                    <dd>
                                        <Link href={`/license/${problem.license.key}`} className="font-medium hover:text-primary hover:underline underline-offset-4">
                                            {problem.license.display || problem.license.name}
                                        </Link>
                                    </dd>
                                </div>
                            )}
                        </dl>

                        {/* Problem Clarifications */}
                        {clarificationsData && clarificationsData.length > 0 && (
                            <section id="clarifications" className="space-y-4">
                                <h2 className="text-2xl font-semibold flex items-center gap-2">
                                    <MessageSquare className="w-6 h-6" />
                                    {t('problemClarifications')}
                                </h2>
                                <div className="space-y-3">
                                    {clarificationsData.map((clar) => (
                                        <div key={clar.id} className="bg-card border rounded-md p-6">
                                            <div className="flex items-center gap-2 text-sm text-muted-foreground mb-3">
                                                <Clock className="w-4 h-4" />
                                                {dayjs(clar.date).format('L HH:mm')}
                                            </div>
                                            <p className="text-sm whitespace-pre-wrap">{clar.description}</p>
                                        </div>
                                    ))}
                                </div>
                            </section>
                        )}

                        <section className="pt-10 border-t">
                            <Comments page={`p:${problem.code}`} problemName={problem.name} contextType="problem" />
                        </section>
                    </div>

                    {/* Editor Section */}
                    <div className="flex flex-col gap-4 lg:sticky lg:top-4 h-[75vh] xl:h-[85vh]">
                        <div className="bg-card border rounded-md shadow-sm">
                            <div className="flex justify-between items-center p-2">
                                <div className="flex items-center gap-2 pl-2">
                                    <Code2 size={16} className="text-muted-foreground" />
                                    <div
                                        className="relative"
                                        onBlur={(e) => { if (!e.currentTarget.contains(e.relatedTarget)) setLangOpen(false); }}
                                    >
                                        <button
                                            type="button"
                                            aria-haspopup="listbox"
                                            aria-expanded={langOpen}
                                            onClick={() => setLangOpen(o => !o)}
                                            className="bg-transparent font-medium outline-none text-xs cursor-pointer hover:text-primary transition-colors"
                                        >
                                            {problem.languages.find(l => l.key === language)?.name ?? ''}
                                        </button>
                                        {langOpen && (
                                            <ul role="listbox" className="absolute left-0 z-20 mt-1 max-h-72 min-w-[10rem] overflow-y-auto rounded-md border bg-card py-1 shadow-md">
                                                {problem.languages.map(l => (
                                                    <li
                                                        key={l.key}
                                                        role="option"
                                                        aria-selected={l.key === language}
                                                        tabIndex={0}
                                                        onClick={() => { setLanguage(l.key); setLangOpen(false); setSubmissionFile(null); }}
                                                        onKeyDown={(e) => {
                                                            if (e.key === 'Enter' || e.key === ' ') {
                                                                e.preventDefault();
                                                                setLanguage(l.key);
                                                                setLangOpen(false);
                                                                setSubmissionFile(null);
                                                            }
                                                        }}
                                                        className="cursor-pointer px-3 py-1 text-xs font-medium text-primary hover:bg-muted hover:text-muted-foreground outline-none"
                                                    >
                                                        {l.name}
                                                    </li>
                                                ))}
                                            </ul>
                                        )}
                                    </div>
                                </div>

                                <button
                                    onClick={handleSubmitClick}
                                    disabled={isSubmitting}
                                    className="flex items-center gap-2 px-4 py-2 rounded-xl bg-primary text-primary-foreground text-sm font-medium transition-all disabled:opacity-50"
                                >
                                    {isSubmitting ? <Loader2 size={16} className="animate-spin" /> : <Send size={16} />}
                                    {t('submit')}
                                </button>
                            </div>

                            {/* Allowed languages: the exact set the submit dropdown offers. */}
                            <div className="flex flex-wrap items-center gap-1.5 border-t px-3 py-2 text-xs text-muted-foreground">
                                <span className="font-medium">{t('allowedLanguages')}:</span>
                                {problem.languages.map(l => (
                                    <span key={l.key} className="rounded bg-muted px-2 py-0.5 font-medium">
                                        {l.name}
                                    </span>
                                ))}
                            </div>
                        </div>

                        {isFileOnly ? (
                            <div className="flex-1 rounded-md border bg-card flex flex-col items-center justify-center gap-3 p-6 text-center">
                                <Upload size={32} className="text-muted-foreground" />
                                <p className="text-sm font-medium">{t('fileSubmit.label')}</p>
                                <input
                                    ref={submissionFileInput}
                                    type="file"
                                    accept={fileExtension ? `.${fileExtension}` : undefined}
                                    className="sr-only"
                                    onChange={(e) => {
                                        pickSubmissionFile(e.target.files?.[0]);
                                        e.target.value = '';
                                    }}
                                />
                                <button
                                    type="button"
                                    onClick={() => submissionFileInput.current?.click()}
                                    className="max-w-full truncate rounded-md border bg-muted/30 px-4 py-2 text-sm font-medium hover:border-primary/40 hover:text-primary transition-all"
                                >
                                    {submissionFile ? submissionFile.name : t('fileSubmit.choose')}
                                </button>
                                <p className="text-xs text-muted-foreground">
                                    {t('fileSubmit.hint', {
                                        ext: fileExtension,
                                        size: selectedLanguage?.file_size_limit ?? 0,
                                    })}
                                </p>
                            </div>
                        ) : (
                            <>
                                <div className="flex items-center justify-end gap-2 text-xs">
                                    <input
                                        ref={textFileInput}
                                        type="file"
                                        className="sr-only"
                                        onChange={(e) => {
                                            void loadTextFile(e.target.files?.[0]);
                                            e.target.value = '';
                                        }}
                                    />
                                    <button
                                        type="button"
                                        onClick={() => textFileInput.current?.click()}
                                        disabled={isReadingFile}
                                        title={t('loadFromFileHint', { size: MAX_TEXT_SOURCE_BYTES / 1024 })}
                                        className="flex items-center gap-1.5 font-medium text-muted-foreground hover:text-primary transition-colors disabled:opacity-50"
                                    >
                                        {isReadingFile ? <Loader2 size={14} className="animate-spin" /> : <FileText size={14} />}
                                        {t('loadFromFile')}
                                    </button>
                                </div>
                                <div className="flex-1 rounded-md border overflow-hidden shadow-inner bg-[#1e1e1e]">
                                    <CodeEditor
                                        value={codeValue}
                                        onChange={(val) => setCodeValue(val || '')}
                                        language={language}
                                    />
                                </div>
                            </>
                        )}
                    </div>
                </div>
            </div>

            {/* PDF Viewer Modal */}
            {showPdfViewer && problem.pdf_url && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
                    <div className="relative w-full h-full max-w-7xl mx-auto p-4">
                        {/* Modal header */}
                        <div className="absolute top-4 right-4 z-10 flex items-center gap-2">
                            <a
                                href={problemPdfApi.getPdfUrl(code)}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg transition-all font-medium"
                            >
                                <ArrowUpRight size={16} />
                                {t('pdfViewer.openNewTab')}
                            </a>
                            <button
                                onClick={() => setShowPdfViewer(false)}
                                className="p-2 bg-white/10 hover:bg-white/20 rounded-lg transition-all text-white"
                                aria-label={t('pdfViewer.close')}
                            >
                                <X size={24} />
                            </button>
                        </div>

                        {/* PDF viewer */}
                        <div className="w-full h-full overflow-hidden rounded-lg shadow-2xl">{/* ui-check-allow: floating modal surface */}
                            <PdfStatementViewer code={code} variant="modal" />
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
