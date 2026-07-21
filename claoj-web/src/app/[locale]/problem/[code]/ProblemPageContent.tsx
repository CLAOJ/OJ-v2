'use client';

import { useQuery, useMutation } from '@tanstack/react-query';
import { useRouter, Link } from '@/navigation';
import { useTranslations } from 'next-intl';
import dynamic from 'next/dynamic';
import api, { problemClarificationApi, problemPdfApi } from '@/lib/api';
import { ProblemDetail, ProblemClarification } from '@/types';
import MathRenderer from '@/components/ui/MathRenderer';
import CodeEditor from '@/components/ui/CodeEditor';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { useState, useEffect, use } from 'react';
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
    HelpCircle,
    Activity,
    ArrowUpRight,
    Trophy,
    Monitor,
    Code2,
    BookOpen,
    MessageSquare,
    FileText,
    X
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useSearchParams } from 'next/navigation';
import Comments from '@/components/common/Comments';
import { useRequireAuth } from '@/hooks/useRequireAuth';
import { toast } from 'sonner';

// The API returns memory_limit in kilobytes (DMOJ convention). Render it in MB
// when it is at least 1 MB, otherwise keep KB. Previously the raw KB value was
// printed with an "MB" suffix, so 512 MB showed as "524288MB".
function formatMemoryLimit(kb?: number | null): string {
    if (kb == null) return '—';
    if (kb < 1024) return `${kb} KB`;
    const mb = kb / 1024;
    return `${Number.isInteger(mb) ? mb : mb.toFixed(1)} MB`;
}

// react-pdf touches browser-only APIs — load it client-side only.
const PdfStatementViewer = dynamic(() => import('@/components/ui/PdfStatementViewer'), {
    ssr: false,
    loading: () => (
        <div className="flex items-center justify-center p-10 bg-card border rounded-3xl shadow-sm">
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
    const [showPdfViewer, setShowPdfViewer] = useState(false);

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

    const { mutate: submitCode, isPending: isSubmitting } = useMutation({
        mutationFn: async () => {
            const url = contestKey ? `/problem/${code}/submit?contest=${contestKey}` : `/problem/${code}/submit`;
            const res = await api.post<{ submission_id: number }>(url, {
                language,
                source: codeValue
            });
            return res.data;
        },
        onSuccess: (data) => {
            router.push(`/submission/${data.submission_id}`);
        },
        onError: (err: any) => {
            toast.error(err.response?.data?.error || t('submitFailed'));
        }
    });

    if (isLoading) {
        return (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 h-[calc(100vh-12rem)] animate-in fade-in">
                <div className="space-y-6 overflow-y-auto pr-4">
                    <Skeleton className="h-10 w-3/4" />
                    <div className="flex gap-4">
                        <Skeleton className="h-6 w-24" />
                        <Skeleton className="h-6 w-24" />
                    </div>
                    <Skeleton className="h-[60vh] w-full" />
                </div>
                <Skeleton className="h-full w-full rounded-2xl" />
            </div>
        );
    }

    if (!problem) return <div className="p-8">{t('notFound')}</div>;

    return (
        <div className="flex flex-col lg:flex-row gap-8 min-h-[calc(100vh-12rem)] animate-in fade-in duration-500 mt-4">
            {/* Left Sidebar: Problem Metadata & Status */}
            <aside className="w-full lg:w-72 flex flex-col gap-6 shrink-0 h-fit lg:sticky lg:top-4">
                {/* Status Card */}
                <div className={cn(
                    "relative overflow-hidden rounded-3xl border p-6 flex flex-col items-center gap-4 transition-all duration-700 shadow-sm",
                    problem.is_solved ? "bg-emerald-50/50 border-emerald-200" :
                        problem.is_attempted ? "bg-amber-50/50 border-amber-200" : "bg-card shadow-sm"
                )}>
                    {problem.is_solved ? (
                        <div className="p-4 rounded-full bg-emerald-500 text-white shadow-lg shadow-emerald-500/30 animate-in zoom-in duration-500">
                            <CheckCircle2 size={40} />
                        </div>
                    ) : problem.is_attempted ? (
                        <div className="p-4 rounded-full bg-amber-500 text-white shadow-lg shadow-amber-500/30">
                            <XCircle size={40} />
                        </div>
                    ) : (
                        <div className="p-4 rounded-full bg-muted text-muted-foreground/30">
                            <HelpCircle size={40} />
                        </div>
                    )}

                    <div className="text-center space-y-1">
                        <p className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">
                            {problem.is_solved ? t('solved') : problem.is_attempted ? t('attempted') : t('unsolved')}
                        </p>
                        <h2 className="text-xl font-black tracking-tight">{problem.code}</h2>
                    </div>

                    <div className="absolute top-2 right-2 text-[8px] font-mono text-muted-foreground opacity-30">
                        {t('idLabel', { id: String(problem.id) })}
                    </div>
                </div>

                {/* Submissions Links */}
                <div className="bg-card border rounded-3xl p-6 shadow-sm space-y-3">
                    <h3 className="text-xs font-black uppercase tracking-widest text-muted-foreground mb-4">{t('activity')}</h3>
                    <Link
                        href={`/submissions?problem=${problem.code}`}
                        className="flex items-center justify-between group p-3 rounded-2xl hover:bg-primary/5 transition-all outline-none"
                    >
                        <div className="flex items-center gap-3">
                            <Activity size={18} className="text-primary group-hover:scale-110 transition-transform" />
                            <span className="text-sm font-bold">{t('allSubmissions')}</span>
                        </div>
                        <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all" />
                    </Link>
                    <Link
                        href={`/submissions?problem=${problem.code}&best=1`}
                        className="flex items-center justify-between group p-3 rounded-2xl hover:bg-amber-500/5 transition-all outline-none"
                    >
                        <div className="flex items-center gap-3">
                            <Trophy size={18} className="text-amber-500 group-hover:scale-110 transition-transform" />
                            <span className="text-sm font-bold">{t('bestSolutions')}</span>
                        </div>
                        <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all" />
                    </Link>
                </div>

                {/* Editorial Link */}
                {editorialData?.exists && (
                    <div className="bg-card border rounded-3xl p-6 shadow-sm">
                        <Link
                            href={`/problem/${code}/editorial`}
                            className="flex items-center gap-3 p-3 rounded-2xl hover:bg-primary/5 transition-all outline-none"
                        >
                            <BookOpen size={18} className="text-emerald-500 group-hover:scale-110 transition-transform" />
                            <span className="text-sm font-bold">{t('editorial')}</span>
                            <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all ml-auto" />
                        </Link>
                    </div>
                )}

                {/* PDF Statement Link */}
                {problem.pdf_url && (
                    <div className="bg-card border rounded-3xl p-6 shadow-sm">
                        <button
                            onClick={() => setShowPdfViewer(true)}
                            className="flex items-center gap-3 p-3 rounded-2xl hover:bg-primary/5 transition-all outline-none w-full"
                        >
                            <FileText size={18} className="text-red-500 group-hover:scale-110 transition-transform" />
                            <span className="text-sm font-bold">{t('pdfViewer.sidebarLabel')}</span>
                            <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all ml-auto" />
                        </button>
                    </div>
                )}

                {/* Problem Clarifications Link */}
                {clarificationsData && clarificationsData.length > 0 && (
                    <div className="bg-card border rounded-3xl p-6 shadow-sm">
                        <Link
                            href={`#clarifications`}
                            className="flex items-center gap-3 p-3 rounded-2xl hover:bg-primary/5 transition-all outline-none"
                        >
                            <MessageSquare size={18} className="text-amber-500 group-hover:scale-110 transition-transform" />
                            <span className="text-sm font-bold">{t('clarificationsCount', { count: clarificationsData.length })}</span>
                            <ArrowUpRight size={16} className="text-muted-foreground opacity-0 group-hover:opacity-100 transition-all ml-auto" />
                        </Link>
                    </div>
                )}

                {/* Technical Details */}
                <div className="bg-card border rounded-3xl p-6 shadow-sm space-y-6">
                    <h3 className="text-xs font-black uppercase tracking-widest text-muted-foreground">{t('details')}</h3>

                    <div className="space-y-4">
                        <div className="flex flex-col gap-1.5">
                            <div className="flex items-center gap-2 text-muted-foreground text-[10px] font-bold uppercase tracking-wider">
                                <Monitor size={14} className="text-blue-500" />
                                <span>{t('ioMethod')}</span>
                            </div>
                            <span className="text-sm font-black">{t('standardConsole')}</span>
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <div className="flex items-center gap-2 text-muted-foreground text-[10px] font-bold uppercase tracking-wider">
                                <Clock size={14} className="text-purple-500" />
                                <span>{t('timeLimit')}</span>
                            </div>
                            <span className="text-sm font-black">{problem.time_limit}s</span>
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <div className="flex items-center gap-2 text-muted-foreground text-[10px] font-bold uppercase tracking-wider">
                                <HardDrive size={14} className="text-emerald-500" />
                                <span>{t('memoryLimit')}</span>
                            </div>
                            <span className="text-sm font-black">{formatMemoryLimit(problem.memory_limit)}</span>
                        </div>
                    </div>

                    <div className="pt-4 border-t space-y-2">
                        <div className="flex justify-between items-center text-xs">
                            <span className="text-muted-foreground font-medium">{t('acRate')}</span>
                            <span className="font-black text-primary">{Math.round(problem.ac_rate)}%</span>
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
            <div className="flex-1 flex flex-col gap-8 min-w-0">
                <div className="grid grid-cols-1 xl:grid-cols-2 gap-8 items-start">
                    {/* Problem Statement */}
                    <div className="space-y-8 min-w-0">
                        <header className="space-y-2">
                            <h1 className="text-4xl lg:text-5xl font-black tracking-tight leading-tight">
                                {problem.name}
                            </h1>
                            <div className="flex flex-wrap gap-2 pt-2">
                                {problem.types.map(type => (
                                    <Badge key={type.name} variant="secondary" className="px-3 py-1 bg-muted/50 font-bold hover:bg-primary hover:text-primary-foreground transition-all cursor-default">
                                        {type.name}
                                    </Badge>
                                ))}
                                <Badge variant="outline" className="border-primary/20 bg-primary/5 text-primary font-black px-3 py-1">
                                    {problem.points} {t('points')}
                                </Badge>
                            </div>
                        </header>

                        {(problem.description?.trim() || !problem.pdf_url) && (
                            <div className="prose prose-sm dark:prose-invert max-w-none bg-card border rounded-3xl p-8 lg:p-10 shadow-sm leading-relaxed">
                                <MathRenderer content={problem.description} fullMarkup={problem.is_full_markup} />
                            </div>
                        )}
                        {problem.pdf_url && <PdfStatementViewer code={code} />}

                        {/* Authors & Group */}
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                            <div className="bg-muted/30 border border-dashed rounded-3xl p-6 flex flex-col gap-2">
                                <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                                    <UserIcon size={12} /> {t('authors')}
                                </span>
                                <div className="flex flex-wrap gap-2 text-sm font-bold">
                                    {problem.authors.map(a => (
                                        <Link key={a.username} href={`/user/${a.username}`} className="hover:text-primary transition-colors">
                                            @{a.username}
                                        </Link>
                                    ))}
                                </div>
                            </div>
                            <div className="bg-muted/30 border border-dashed rounded-3xl p-6 flex flex-col gap-2">
                                <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                                    <Tag size={12} /> {t('group')}
                                </span>
                                <span className="text-sm font-bold truncate">{problem.group}</span>
                            </div>
                        </div>

                        {/* Problem Clarifications */}
                        {clarificationsData && clarificationsData.length > 0 && (
                            <section id="clarifications" className="space-y-4">
                                <h2 className="text-2xl font-bold flex items-center gap-2">
                                    <MessageSquare className="w-6 h-6" />
                                    {t('problemClarifications')}
                                </h2>
                                <div className="space-y-3">
                                    {clarificationsData.map((clar) => (
                                        <div key={clar.id} className="bg-card border rounded-2xl p-6">
                                            <div className="flex items-center gap-2 text-sm text-muted-foreground mb-3">
                                                <Clock className="w-4 h-4" />
                                                {new Date(clar.date).toLocaleString()}
                                            </div>
                                            <p className="text-sm whitespace-pre-wrap">{clar.description}</p>
                                        </div>
                                    ))}
                                </div>
                            </section>
                        )}

                        <section className="pt-10 border-t">
                            <Comments page={`p/${problem.code}`} problemName={problem.name} contextType="problem" />
                        </section>
                    </div>

                    {/* Editor Section */}
                    <div className="flex flex-col gap-4 lg:sticky lg:top-4 h-[75vh] xl:h-[85vh]">
                        <div className="flex justify-between items-center bg-card border p-2 rounded-2xl shadow-sm">
                            <div className="flex items-center gap-2 pl-2">
                                <Code2 size={16} className="text-primary" />
                                <select
                                    className="bg-transparent font-black outline-none text-xs cursor-pointer hover:text-primary transition-colors"
                                    value={language}
                                    onChange={(e) => setLanguage(e.target.value)}
                                >
                                    {problem.languages.map(l => (
                                        <option key={l.key} value={l.key}>{l.name}</option>
                                    ))}
                                </select>
                            </div>

                            <button
                                onClick={() => requireAuth(() => submitCode(), tAuth('loginToSubmit'))}
                                disabled={isSubmitting}
                                className="flex items-center gap-2 px-8 py-2 rounded-xl bg-primary text-primary-foreground text-sm font-black shadow-lg shadow-primary/20 hover:scale-[1.02] active:scale-95 transition-all disabled:opacity-50"
                            >
                                {isSubmitting ? <Loader2 size={16} className="animate-spin" /> : <Send size={16} />}
                                {t('submit')}
                            </button>
                        </div>

                        <div className="flex-1 rounded-3xl border overflow-hidden shadow-inner bg-[#1e1e1e]">
                            <CodeEditor
                                value={codeValue}
                                onChange={(val) => setCodeValue(val || '')}
                                language={language}
                            />
                        </div>
                    </div>
                </div>
            </div>

            {/* PDF Viewer Modal */}
            {showPdfViewer && problem.pdf_url && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm animate-in fade-in">
                    <div className="relative w-full h-full max-w-7xl mx-auto p-4">
                        {/* Modal header */}
                        <div className="absolute top-4 right-4 z-10 flex items-center gap-2">
                            <a
                                href={problemPdfApi.getPdfUrl(code)}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:scale-105 transition-all font-bold shadow-lg"
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
                        <div className="w-full h-full overflow-hidden rounded-lg shadow-2xl">
                            <PdfStatementViewer code={code} variant="modal" />
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
