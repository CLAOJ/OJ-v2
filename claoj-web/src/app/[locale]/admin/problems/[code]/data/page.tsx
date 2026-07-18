'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminProblemDataApi, adminProblemApi } from '@/lib/adminApi';
import { ArrowLeft, Database, Upload, Trash2, Edit2, FileText, Check, X, Loader2, Eye } from 'lucide-react';
import { Link } from '@/navigation';
import { useState, useCallback } from 'react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence } from 'framer-motion';
import { FilesTab, ConfigTab, PdfTab } from '@/components/admin/problem-data';

interface TestCaseWithContent {
    id: number;
    order: number;
    input_file: string;
    output_file: string;
    input_data?: string;
    output_data?: string;
}

interface ProblemData {
    test_cases: TestCaseWithContent[];
    checker?: string;
    has_custom_checker?: boolean;
    grader?: string;
    has_custom_grader?: boolean;
    feedback?: string;
    has_generator_yml?: boolean;
    has_init_yml?: boolean;
}

interface NewTestCase {
    input: string;
    output: string;
}

export default function ProblemDataPage() {
    const t = useTranslations('Admin');
    const params = useParams();
    const code = params.code as string;
    const queryClient = useQueryClient();

    const [activeTab, setActiveTab] = useState<'testcases' | 'files' | 'pdf' | 'config'>('testcases');
    const [selectedTestCase, setSelectedTestCase] = useState<TestCaseWithContent | null>(null);
    const [viewingContent, setViewingContent] = useState<{ type: 'input' | 'output'; data: string } | null>(null);
    const [uploadMode, setUploadMode] = useState<'single' | 'batch'>('single');
    const [newTestCase, setNewTestCase] = useState<NewTestCase>({ input: '', output: '' });

    const { data: problemData, isLoading: isLoadingData } = useQuery<ProblemData>({
        queryKey: ['problem-data', code],
        queryFn: async () => {
            const res = await adminProblemDataApi.detail(code);
            return res.data;
        }
    });

    const { data: problem } = useQuery({
        queryKey: ['admin-problem-detail', code],
        queryFn: async () => {
            const res = await adminProblemApi.detail(code);
            return res.data;
        }
    });

    const deleteMutation = useMutation({
        mutationFn: (testCaseId: number) =>
            adminProblemDataApi.deleteTestCase(code, testCaseId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
        }
    });

    const reorderMutation = useMutation({
        mutationFn: (testCases: { id: number; order: number }[]) =>
            adminProblemDataApi.reorder(code, { test_cases: testCases }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
        }
    });

    const uploadMutation = useMutation({
        mutationFn: (formData: FormData) =>
            adminProblemDataApi.upload(code, formData),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
            setNewTestCase({ input: '', output: '' });
        }
    });

    const updateTestCaseMutation = useMutation({
        mutationFn: ({ testCaseId, data }: { testCaseId: number; data: { input_data?: string; output_data?: string } }) =>
            adminProblemDataApi.updateTestCase(code, testCaseId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
            setViewingContent(null);
        }
    });

    const { data: selectedTestCaseContent } = useQuery({
        queryKey: ['test-case-content', code, selectedTestCase?.id],
        queryFn: async () => {
            if (!selectedTestCase) return null;
            const res = await adminProblemDataApi.getTestCaseContent(code, selectedTestCase.id);
            return res.data;
        },
        enabled: !!selectedTestCase && !!viewingContent
    });

    const handleDelete = useCallback((testCaseId: number) => {
        if (confirm(t('problemData.deleteConfirm'))) {
            deleteMutation.mutate(testCaseId);
        }
    }, [deleteMutation, t]);

    const handleReorder = useCallback((newOrder: number[]) => {
        if (!problemData?.test_cases) return;

        const reordered = newOrder.map((id, index) => ({
            id,
            order: index + 1
        }));
        reorderMutation.mutate(reordered);
    }, [problemData?.test_cases, reorderMutation]);

    const handleSingleUpload = useCallback(() => {
        if (!newTestCase.input.trim() || !newTestCase.output.trim()) {
            alert(t('problemData.missingInputOutputAlert'));
            return;
        }

        const formData = new FormData();
        formData.append('input', newTestCase.input);
        formData.append('output', newTestCase.output);
        formData.append('type', 'single');

        uploadMutation.mutate(formData);
    }, [newTestCase, uploadMutation, t]);

    const handleBatchUpload = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const files = e.target.files;
        if (!files || files.length === 0) return;

        const formData = new FormData();
        formData.append('type', 'batch');

        for (let i = 0; i < files.length; i++) {
            formData.append(`file_${i}`, files[i]);
        }

        uploadMutation.mutate(formData);
    }, [uploadMutation]);

    const handleSaveTestCaseContent = useCallback((data: string) => {
        if (!selectedTestCase) return;

        updateTestCaseMutation.mutate({
            testCaseId: selectedTestCase.id,
            data: {
                [viewingContent?.type === 'input' ? 'input_data' : 'output_data']: data
            }
        });
    }, [selectedTestCase, viewingContent, updateTestCaseMutation]);

    if (isLoadingData) {
        return (
            <div className="flex items-center justify-center py-12">
                <Loader2 size={32} className="animate-spin text-muted-foreground" />
            </div>
        );
    }

    const testCases = problemData?.test_cases || [];

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Link
                    href="/admin/problems"
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                >
                    <ArrowLeft size={20} />
                </Link>
                <div className="flex-1">
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Database className="text-primary" size={32} />
                        {t('problemData.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {problem?.name} ({code})
                    </p>
                </div>
                <Link
                    href={`/admin/problems/${code}/edit`}
                    className="px-4 py-2 rounded-xl border hover:bg-muted transition-colors flex items-center gap-2"
                >
                    <Edit2 size={18} />
                    {t('problems.editProblemTitle')}
                </Link>
            </div>

            {/* Tabs */}
            <div className="flex gap-2 border-b">
                <TabButton
                    active={activeTab === 'testcases'}
                    onClick={() => setActiveTab('testcases')}
                    label={t('problemData.testCasesTab', { count: testCases.length })}
                />
                <TabButton
                    active={activeTab === 'files'}
                    onClick={() => setActiveTab('files')}
                    label={t('problemData.filesTab')}
                />
                <TabButton
                    active={activeTab === 'config'}
                    onClick={() => setActiveTab('config')}
                    label={t('problemData.configTab')}
                />
                <TabButton
                    active={activeTab === 'pdf'}
                    onClick={() => setActiveTab('pdf')}
                    label={t('problemData.pdfTab')}
                />
            </div>

            {/* Test Cases Tab */}
            {activeTab === 'testcases' && (
                <TestCasesTab
                    uploadMode={uploadMode}
                    setUploadMode={setUploadMode}
                    newTestCase={newTestCase}
                    setNewTestCase={setNewTestCase}
                    testCases={testCases}
                    onSingleUpload={handleSingleUpload}
                    onBatchUpload={handleBatchUpload}
                    onDelete={handleDelete}
                    onView={(testCase, type) => {
                        setSelectedTestCase(testCase);
                        setViewingContent({ type, data: '' });
                    }}
                    isUploading={uploadMutation.isPending}
                    isDeleting={deleteMutation.isPending}
                />
            )}

            {/* Files Tab */}
            {activeTab === 'files' && <FilesTab code={code} />}

            {/* Configuration Tab */}
            {activeTab === 'config' && <ConfigTab code={code} data={problemData} />}

            {/* PDF Statement Tab */}
            {activeTab === 'pdf' && <PdfTab code={code} />}

            {/* View/Edit Content Modal */}
            <AnimatePresence>
                {viewingContent && selectedTestCaseContent && (
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4"
                        onClick={() => setViewingContent(null)}
                    >
                        <motion.div
                            initial={{ scale: 0.95, y: 20 }}
                            animate={{ scale: 1, y: 0 }}
                            exit={{ scale: 0.95, y: 20 }}
                            onClick={(e) => e.stopPropagation()}
                            className="bg-card rounded-2xl shadow-2xl w-full max-w-4xl max-h-[80vh] overflow-hidden"
                        >
                            <div className="flex items-center justify-between p-4 border-b">
                                <h3 className="font-bold text-lg">
                                    {t('problemData.testCaseContentTitle', {
                                        order: selectedTestCase?.order ?? '',
                                        type: viewingContent.type === 'input' ? t('problemData.inputLabel') : t('problemData.outputLabel')
                                    })}
                                </h3>
                                <div className="flex items-center gap-2">
                                    <button
                                        onClick={() => handleSaveTestCaseContent(viewingContent.data)}
                                        disabled={updateTestCaseMutation.isPending}
                                        className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
                                    >
                                        {updateTestCaseMutation.isPending ? (
                                            <Loader2 size={16} className="animate-spin" />
                                        ) : (
                                            <Check size={16} />
                                        )}
                                        {t('common.saveChanges')}
                                    </button>
                                    <button
                                        onClick={() => setViewingContent(null)}
                                        className="p-2 hover:bg-muted rounded-lg transition-colors"
                                    >
                                        <X size={20} />
                                    </button>
                                </div>
                            </div>
                            <div className="p-4 overflow-auto max-h-[60vh]">
                                <textarea
                                    value={viewingContent.type === 'input'
                                        ? selectedTestCaseContent.input_data
                                        : selectedTestCaseContent.output_data}
                                    onChange={(e) => setViewingContent(prev =>
                                        prev ? { ...prev, data: e.target.value } : null
                                    )}
                                    className="w-full h-[500px] p-4 border rounded-lg font-mono text-sm bg-background whitespace-pre-wrap break-all"
                                />
                            </div>
                        </motion.div>
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
}

// Sub-components

interface TabButtonProps {
    active: boolean;
    onClick: () => void;
    label: string;
}

function TabButton({ active, onClick, label }: TabButtonProps) {
    return (
        <button
            onClick={onClick}
            className={cn(
                "px-4 py-2 font-medium transition-colors border-b-2",
                active
                    ? "border-primary text-primary"
                    : "border-transparent text-muted-foreground hover:text-foreground"
            )}
        >
            {label}
        </button>
    );
}

interface TestCasesTabProps {
    uploadMode: 'single' | 'batch';
    setUploadMode: (mode: 'single' | 'batch') => void;
    newTestCase: { input: string; output: string };
    setNewTestCase: (value: { input: string; output: string }) => void;
    testCases: TestCaseWithContent[];
    onSingleUpload: () => void;
    onBatchUpload: (e: React.ChangeEvent<HTMLInputElement>) => void;
    onDelete: (id: number) => void;
    onView: (testCase: TestCaseWithContent, type: 'input' | 'output') => void;
    isUploading: boolean;
    isDeleting: boolean;
}

function TestCasesTab({
    uploadMode,
    setUploadMode,
    newTestCase,
    setNewTestCase,
    testCases,
    onSingleUpload,
    onBatchUpload,
    onDelete,
    onView,
    isUploading,
    isDeleting
}: TestCasesTabProps) {
    const t = useTranslations('Admin');
    return (
        <div className="space-y-4">
            {/* Upload Section */}
            <div className="p-6 border rounded-xl bg-muted/30">
                <div className="flex items-center gap-2 mb-4">
                    <Upload size={20} className="text-primary" />
                    <h2 className="font-semibold text-lg">{t('problemData.uploadSectionTitle')}</h2>
                </div>

                <div className="flex gap-4 mb-4">
                    <UploadModeButton
                        active={uploadMode === 'single'}
                        onClick={() => setUploadMode('single')}
                        label={t('problemData.singleModeButton')}
                    />
                    <UploadModeButton
                        active={uploadMode === 'batch'}
                        onClick={() => setUploadMode('batch')}
                        label={t('problemData.batchModeButton')}
                    />
                </div>

                {uploadMode === 'single' ? (
                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium mb-2">{t('problemData.inputDataLabel')}</label>
                            <textarea
                                value={newTestCase.input}
                                onChange={(e) => setNewTestCase({ ...newTestCase, input: e.target.value })}
                                className="w-full min-h-[150px] p-3 border rounded-lg font-mono text-sm bg-background"
                                placeholder={t('problemData.inputDataPlaceholder')}
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-2">{t('problemData.outputDataLabel')}</label>
                            <textarea
                                value={newTestCase.output}
                                onChange={(e) => setNewTestCase({ ...newTestCase, output: e.target.value })}
                                className="w-full min-h-[150px] p-3 border rounded-lg font-mono text-sm bg-background"
                                placeholder={t('problemData.outputDataPlaceholder')}
                            />
                        </div>
                        <button
                            onClick={onSingleUpload}
                            disabled={isUploading}
                            className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
                        >
                            {isUploading && <Loader2 size={16} className="animate-spin" />}
                            {t('problemData.uploadTestCaseButton')}
                        </button>
                    </div>
                ) : (
                    <div className="space-y-4">
                        <div className="border-2 border-dashed rounded-lg p-8 text-center">
                            <input
                                type="file"
                                multiple
                                accept=".zip,.in,.out"
                                onChange={onBatchUpload}
                                className="hidden"
                                id="batch-upload"
                            />
                            <label
                                htmlFor="batch-upload"
                                className="cursor-pointer flex flex-col items-center gap-2"
                            >
                                <Upload size={48} className="text-muted-foreground" />
                                <span className="text-sm text-muted-foreground">
                                    {t('problemData.dragDropHint')}
                                </span>
                            </label>
                        </div>
                        {isUploading && (
                            <div className="flex items-center gap-2 text-muted-foreground">
                                <Loader2 size={16} className="animate-spin" />
                                {t('problemData.uploadingFilesMsg')}
                            </div>
                        )}
                    </div>
                )}
            </div>

            {/* Test Cases List */}
            <div className="border rounded-xl overflow-hidden">
                <div className="bg-muted/30 px-4 py-3 border-b font-medium">
                    {t('problemData.testCasesTab', { count: testCases.length })}
                </div>
                <div className="divide-y">
                    <AnimatePresence>
                        {testCases.map((testCase) => (
                            <motion.div
                                key={testCase.id}
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{ opacity: 0, y: -10 }}
                                className="flex items-center gap-4 p-4 hover:bg-muted/30 transition-colors"
                            >
                                <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary font-bold">
                                    {testCase.order}
                                </div>
                                <div className="flex-1">
                                    <div className="flex items-center gap-2">
                                        <span className="font-medium">{t('problemData.testCaseNumber', { order: testCase.order })}</span>
                                        <span className="text-xs text-muted-foreground">{t('problemData.idLabel', { id: testCase.id })}</span>
                                    </div>
                                    <div className="text-sm text-muted-foreground mt-1">
                                        {t('problemData.inputOutputFiles', { input: testCase.input_file, output: testCase.output_file })}
                                    </div>
                                </div>
                                <div className="flex items-center gap-2">
                                    <button
                                        onClick={() => onView(testCase, 'input')}
                                        className="p-2 hover:bg-muted rounded-lg transition-colors"
                                        title={t('problemData.viewEditInputTitle')}
                                    >
                                        <Eye size={18} />
                                    </button>
                                    <button
                                        onClick={() => onView(testCase, 'output')}
                                        className="p-2 hover:bg-muted rounded-lg transition-colors"
                                        title={t('problemData.viewEditOutputTitle')}
                                    >
                                        <FileText size={18} />
                                    </button>
                                    <button
                                        onClick={() => onDelete(testCase.id)}
                                        disabled={isDeleting}
                                        className="p-2 hover:bg-destructive/10 rounded-lg transition-colors text-destructive"
                                        title={t('common.delete')}
                                    >
                                        <Trash2 size={18} />
                                    </button>
                                </div>
                            </motion.div>
                        ))}
                    </AnimatePresence>
                    {testCases.length === 0 && (
                        <div className="p-8 text-center text-muted-foreground">
                            <Database size={48} className="mx-auto mb-4 opacity-50" />
                            <p className="font-medium">{t('problemData.noTestCases')}</p>
                            <p className="text-sm mt-1">{t('problemData.noTestCasesHint')}</p>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

interface UploadModeButtonProps {
    active: boolean;
    onClick: () => void;
    label: string;
}

function UploadModeButton({ active, onClick, label }: UploadModeButtonProps) {
    return (
        <button
            onClick={onClick}
            className={cn(
                "px-4 py-2 rounded-lg font-medium transition-colors",
                active
                    ? "bg-primary text-white"
                    : "bg-muted hover:bg-muted/70"
            )}
        >
            {label}
        </button>
    );
}
