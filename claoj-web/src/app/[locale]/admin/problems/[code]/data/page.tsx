'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminProblemDataApi, adminProblemApi } from '@/lib/adminApi';
import { ArrowLeft, Database, Upload, Trash2, Edit2, FileText, Check, X, Loader2, FolderOpen, Eye } from 'lucide-react';
import { Link } from '@/navigation';
import { useState, useCallback } from 'react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence } from 'framer-motion';

interface TestCaseWithContent {
    id: number;
    order: number;
    input_file: string;
    output_file: string;
    input_data?: string;
    output_data?: string;
}

export default function ProblemDataPage() {
    const params = useParams();
    const router = useRouter();
    const code = params.code as string;
    const queryClient = useQueryClient();

    const [activeTab, setActiveTab] = useState<'testcases' | 'files' | 'config'>('testcases');
    const [selectedTestCase, setSelectedTestCase] = useState<TestCaseWithContent | null>(null);
    const [viewingContent, setViewingContent] = useState<{ type: 'input' | 'output'; data: string } | null>(null);
    const [uploadMode, setUploadMode] = useState<'single' | 'batch'>('single');
    const [newTestCase, setNewTestCase] = useState({ input: '', output: '' });

    // Fetch problem data
    const { data: problemData, isLoading: isLoadingData } = useQuery({
        queryKey: ['problem-data', code],
        queryFn: async () => {
            const res = await adminProblemDataApi.detail(code);
            return res.data;
        }
    });

    // Fetch problem info
    const { data: problem } = useQuery({
        queryKey: ['admin-problem-detail', code],
        queryFn: async () => {
            const res = await adminProblemApi.detail(code);
            return res.data;
        }
    });

    // Delete test case mutation
    const deleteMutation = useMutation({
        mutationFn: (testCaseId: number) =>
            adminProblemDataApi.deleteTestCase(code, testCaseId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
        }
    });

    // Reorder test cases mutation
    const reorderMutation = useMutation({
        mutationFn: (testCases: { id: number; order: number }[]) =>
            adminProblemDataApi.reorder(code, { test_cases: testCases }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
        }
    });

    // Upload test case mutation
    const uploadMutation = useMutation({
        mutationFn: (formData: FormData) =>
            adminProblemDataApi.upload(code, formData),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
            setNewTestCase({ input: '', output: '' });
        }
    });

    // Update test case mutation
    const updateTestCaseMutation = useMutation({
        mutationFn: ({ testCaseId, data }: { testCaseId: number; data: { input_data?: string; output_data?: string } }) =>
            adminProblemDataApi.updateTestCase(code, testCaseId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
            setViewingContent(null);
        }
    });

    // Fetch test case content for viewing/editing
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
        if (confirm('Are you sure you want to delete this test case? This action cannot be undone.')) {
            deleteMutation.mutate(testCaseId);
        }
    }, [deleteMutation]);

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
            alert('Please provide both input and output data');
            return;
        }

        const formData = new FormData();
        formData.append('input', newTestCase.input);
        formData.append('output', newTestCase.output);
        formData.append('type', 'single');

        uploadMutation.mutate(formData);
    }, [newTestCase, uploadMutation]);

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
    const maxOrder = Math.max(...testCases.map(tc => tc.order), 0);

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
                        Problem Data Management
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
                    Edit Problem
                </Link>
            </div>

            {/* Tabs */}
            <div className="flex gap-2 border-b">
                <button
                    onClick={() => setActiveTab('testcases')}
                    className={cn(
                        "px-4 py-2 font-medium transition-colors border-b-2",
                        activeTab === 'testcases'
                            ? "border-primary text-primary"
                            : "border-transparent text-muted-foreground hover:text-foreground"
                    )}
                >
                    Test Cases ({testCases.length})
                </button>
                <button
                    onClick={() => setActiveTab('files')}
                    className={cn(
                        "px-4 py-2 font-medium transition-colors border-b-2",
                        activeTab === 'files'
                            ? "border-primary text-primary"
                            : "border-transparent text-muted-foreground hover:text-foreground"
                    )}
                >
                    Files
                </button>
                <button
                    onClick={() => setActiveTab('config')}
                    className={cn(
                        "px-4 py-2 font-medium transition-colors border-b-2",
                        activeTab === 'config'
                            ? "border-primary text-primary"
                            : "border-transparent text-muted-foreground hover:text-foreground"
                    )}
                >
                    Configuration
                </button>
            </div>

            {/* Test Cases Tab */}
            {activeTab === 'testcases' && (
                <div className="space-y-4">
                    {/* Upload Section */}
                    <div className="p-6 border rounded-xl bg-muted/30">
                        <div className="flex items-center gap-2 mb-4">
                            <Upload size={20} className="text-primary" />
                            <h2 className="font-semibold text-lg">Upload Test Cases</h2>
                        </div>

                        <div className="flex gap-4 mb-4">
                            <button
                                onClick={() => setUploadMode('single')}
                                className={cn(
                                    "px-4 py-2 rounded-lg font-medium transition-colors",
                                    uploadMode === 'single'
                                        ? "bg-primary text-white"
                                        : "bg-muted hover:bg-muted/70"
                                )}
                            >
                                Single Test Case
                            </button>
                            <button
                                onClick={() => setUploadMode('batch')}
                                className={cn(
                                    "px-4 py-2 rounded-lg font-medium transition-colors",
                                    uploadMode === 'batch'
                                        ? "bg-primary text-white"
                                        : "bg-muted hover:bg-muted/70"
                                )}
                            >
                                Batch Upload (ZIP)
                            </button>
                        </div>

                        {uploadMode === 'single' ? (
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium mb-2">
                                        Input Data
                                    </label>
                                    <textarea
                                        value={newTestCase.input}
                                        onChange={(e) => setNewTestCase(prev => ({ ...prev, input: e.target.value }))}
                                        className="w-full min-h-[150px] p-3 border rounded-lg font-mono text-sm bg-background"
                                        placeholder="Enter input data..."
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium mb-2">
                                        Output Data
                                    </label>
                                    <textarea
                                        value={newTestCase.output}
                                        onChange={(e) => setNewTestCase(prev => ({ ...prev, output: e.target.value }))}
                                        className="w-full min-h-[150px] p-3 border rounded-lg font-mono text-sm bg-background"
                                        placeholder="Enter output data..."
                                    />
                                </div>
                                <button
                                    onClick={handleSingleUpload}
                                    disabled={uploadMutation.isPending}
                                    className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center gap-2"
                                >
                                    {uploadMutation.isPending && <Loader2 size={16} className="animate-spin" />}
                                    Upload Test Case
                                </button>
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <div className="border-2 border-dashed rounded-lg p-8 text-center">
                                    <input
                                        type="file"
                                        multiple
                                        accept=".zip,.in,.out"
                                        onChange={handleBatchUpload}
                                        className="hidden"
                                        id="batch-upload"
                                    />
                                    <label
                                        htmlFor="batch-upload"
                                        className="cursor-pointer flex flex-col items-center gap-2"
                                    >
                                        <Upload size={48} className="text-muted-foreground" />
                                        <span className="text-sm text-muted-foreground">
                                            Click to select files or drag and drop
                                        </span>
                                        <span className="text-xs text-muted-foreground/70">
                                            Supported formats: .zip (containing .in/.out pairs), individual .in/.out files
                                        </span>
                                    </label>
                                </div>
                                {uploadMutation.isPending && (
                                    <div className="flex items-center gap-2 text-muted-foreground">
                                        <Loader2 size={16} className="animate-spin" />
                                        Uploading files...
                                    </div>
                                )}
                            </div>
                        )}
                    </div>

                    {/* Test Cases List */}
                    <div className="border rounded-xl overflow-hidden">
                        <div className="bg-muted/30 px-4 py-3 border-b font-medium">
                            Test Cases ({testCases.length})
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
                                                <span className="font-medium">Test Case #{testCase.order}</span>
                                                <span className="text-xs text-muted-foreground">
                                                    ID: {testCase.id}
                                                </span>
                                            </div>
                                            <div className="text-sm text-muted-foreground mt-1">
                                                Input: {testCase.input_file} | Output: {testCase.output_file}
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <button
                                                onClick={() => {
                                                    setSelectedTestCase(testCase);
                                                    setViewingContent({ type: 'input', data: '' });
                                                }}
                                                className="p-2 hover:bg-muted rounded-lg transition-colors"
                                                title="View/Edit Input"
                                            >
                                                <Eye size={18} />
                                            </button>
                                            <button
                                                onClick={() => {
                                                    setSelectedTestCase(testCase);
                                                    setViewingContent({ type: 'output', data: '' });
                                                }}
                                                className="p-2 hover:bg-muted rounded-lg transition-colors"
                                                title="View/Edit Output"
                                            >
                                                <FileText size={18} />
                                            </button>
                                            <button
                                                onClick={() => handleDelete(testCase.id)}
                                                disabled={deleteMutation.isPending}
                                                className="p-2 hover:bg-destructive/10 rounded-lg transition-colors text-destructive"
                                                title="Delete"
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
                                    <p className="font-medium">No test cases yet</p>
                                    <p className="text-sm mt-1">Upload test cases using the form above</p>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* Files Tab */}
            {activeTab === 'files' && (
                <FilesTab code={code} />
            )}

            {/* Configuration Tab */}
            {activeTab === 'config' && (
                <ConfigTab code={code} data={problemData} />
            )}

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
                                    Test Case #{selectedTestCase?.order} - {viewingContent.type === 'input' ? 'Input' : 'Output'} Data
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
                                        Save Changes
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

// Files Tab Component
function FilesTab({ code }: { code: string }) {
    const { data: filesData, isLoading } = useQuery({
        queryKey: ['problem-data-files', code],
        queryFn: async () => {
            const res = await adminProblemDataApi.files(code);
            return res.data.files;
        }
    });

    const deleteFileMutation = useMutation({
        mutationFn: (path: string) =>
            adminProblemDataApi.deleteFile(code, path),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data-files', code] });
        }
    });

    const queryClient = useQueryClient();

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-12">
                <Loader2 size={32} className="animate-spin text-muted-foreground" />
            </div>
        );
    }

    return (
        <div className="border rounded-xl overflow-hidden">
            <div className="bg-muted/30 px-4 py-3 border-b font-medium flex items-center gap-2">
                <FolderOpen size={18} />
                Problem Files
            </div>
            <div className="divide-y">
                {filesData?.map((file) => (
                    <div key={file.path} className="flex items-center gap-4 p-4 hover:bg-muted/30 transition-colors">
                        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                            {file.type === 'directory' ? (
                                <FolderOpen size={20} className="text-primary" />
                            ) : (
                                <FileText size={20} className="text-primary" />
                            )}
                        </div>
                        <div className="flex-1">
                            <div className="font-medium">{file.name}</div>
                            <div className="text-sm text-muted-foreground">{file.path}</div>
                        </div>
                        {file.type === 'file' && (
                            <button
                                onClick={() => {
                                    if (confirm('Are you sure you want to delete this file?')) {
                                        deleteFileMutation.mutate(file.path);
                                    }
                                }}
                                disabled={deleteFileMutation.isPending}
                                className="p-2 hover:bg-destructive/10 rounded-lg transition-colors text-destructive"
                            >
                                <Trash2 size={18} />
                            </button>
                        )}
                    </div>
                ))}
                {filesData?.length === 0 && (
                    <div className="p-8 text-center text-muted-foreground">
                        <FolderOpen size={48} className="mx-auto mb-4 opacity-50" />
                        <p className="font-medium">No files found</p>
                        <p className="text-sm mt-1">Upload files using the test case upload feature</p>
                    </div>
                )}
            </div>
        </div>
    );
}

// Configuration Tab Component
function ConfigTab({ code, data }: { code: string; data: any }) {
    const queryClient = useQueryClient();

    const updateMutation = useMutation({
        mutationFn: async (formData: FormData) => {
            // Use the upload endpoint with config type
            return adminProblemDataApi.upload(code, formData);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
            alert('Configuration updated successfully');
        }
    });

    return (
        <div className="space-y-6">
            {/* Checker Configuration */}
            <div className="p-6 border rounded-xl">
                <h3 className="font-semibold text-lg mb-4">Checker Configuration</h3>
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2">
                            Checker Type
                        </label>
                        <div className="text-muted-foreground">{data?.checker || 'default'}</div>
                    </div>
                    {data?.has_custom_checker && (
                        <div className="p-3 bg-muted rounded-lg">
                            <div className="flex items-center gap-2 text-green-600">
                                <Check size={16} />
                                <span className="font-medium">Custom checker configured</span>
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* Grader Configuration */}
            <div className="p-6 border rounded-xl">
                <h3 className="font-semibold text-lg mb-4">Grader Configuration</h3>
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2">
                            Grader Type
                        </label>
                        <div className="text-muted-foreground">{data?.grader || 'default'}</div>
                    </div>
                    {data?.has_custom_grader && (
                        <div className="p-3 bg-muted rounded-lg">
                            <div className="flex items-center gap-2 text-green-600">
                                <Check size={16} />
                                <span className="font-medium">Custom grader configured</span>
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* Other Settings */}
            <div className="p-6 border rounded-xl">
                <h3 className="font-semibold text-lg mb-4">Other Settings</h3>
                <div className="grid grid-cols-2 gap-4">
                    <div className="p-3 bg-muted rounded-lg">
                        <div className="text-sm text-muted-foreground">Feedback Level</div>
                        <div className="font-medium">{data?.feedback || 'default'}</div>
                    </div>
                    <div className={cn(
                        "p-3 rounded-lg flex items-center gap-2",
                        data?.has_generator_yml ? "bg-muted" : "bg-muted/50"
                    )}>
                        {data?.has_generator_yml ? (
                            <>
                                <Check size={16} className="text-green-600" />
                                <span className="font-medium">Generator configured</span>
                            </>
                        ) : (
                            <span className="text-muted-foreground">No generator</span>
                        )}
                    </div>
                    <div className={cn(
                        "p-3 rounded-lg flex items-center gap-2",
                        data?.has_init_yml ? "bg-muted" : "bg-muted/50"
                    )}>
                        {data?.has_init_yml ? (
                            <>
                                <Check size={16} className="text-green-600" />
                                <span className="font-medium">init.yml present</span>
                            </>
                        ) : (
                            <span className="text-muted-foreground">No init.yml</span>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
