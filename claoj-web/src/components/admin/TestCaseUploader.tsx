'use client';

import { useState, useRef } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { adminProblemDataApi, type ProblemTestCase } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Upload, Trash2, FileText, CheckCircle, XCircle, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

interface TestCaseUploaderProps {
    problemCode: string;
    existingTestCases?: ProblemTestCase[];
}

interface TestCaseFile {
    input?: File;
    output?: File;
    inputName: string;
    outputName: string;
}

export default function TestCaseUploader({ problemCode, existingTestCases = [] }: TestCaseUploaderProps) {
    const [testFiles, setTestFiles] = useState<TestCaseFile[]>([]);
    const [uploadMode, setUploadMode] = useState<'single' | 'batch'>('single');
    const [currentInput, setCurrentInput] = useState<File | null>(null);
    const [currentOutput, setCurrentOutput] = useState<File | null>(null);
    const fileInputRef = useRef<HTMLInputElement>(null);

    const queryClient = useQueryClient();

    const { data: testData, isLoading: isLoadingTestCases } = useQueryClient().getQueryState(['problem-data', problemCode])
        ? { data: { test_cases: existingTestCases } }
        : { data: null, isLoading: false };

    const testCases = existingTestCases;

    const uploadMutation = useMutation({
        mutationFn: (formData: FormData) => adminProblemDataApi.upload(problemCode, formData),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-problems'] });
            setTestFiles([]);
            setCurrentInput(null);
            setCurrentOutput(null);
        }
    });

    const deleteMutation = useMutation({
        mutationFn: (testCaseId: number) => adminProblemDataApi.deleteTestCase(problemCode, testCaseId),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', problemCode] });
        }
    });

    const handleSingleFileChange = (type: 'input' | 'output', file: File | null) => {
        if (type === 'input') {
            setCurrentInput(file);
        } else {
            setCurrentOutput(file);
        }
    };

    const handleBatchFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const files = Array.from(e.target.files || []);
        const newTestFiles: TestCaseFile[] = [];

        // Group files by base name
        const inputFiles = new Map<string, File>();
        const outputFiles = new Map<string, File>();

        files.forEach(file => {
            const name = file.name;
            if (name.endsWith('.in') || name.endsWith('.input') || name.endsWith('.txt')) {
                const baseName = name.replace(/\.(in|input|txt)$/, '');
                inputFiles.set(baseName, file);
            } else if (name.endsWith('.out') || name.endsWith('.output') || name.endsWith('.ans')) {
                const baseName = name.replace(/\.(out|output|ans)$/, '');
                outputFiles.set(baseName, file);
            }
        });

        // Match input and output files
        const allBaseNames = new Set([...inputFiles.keys(), ...outputFiles.keys()]);
        allBaseNames.forEach(baseName => {
            newTestFiles.push({
                input: inputFiles.get(baseName),
                output: outputFiles.get(baseName),
                inputName: inputFiles.get(baseName)?.name || '',
                outputName: outputFiles.get(baseName)?.name || ''
            });
        });

        setTestFiles(newTestFiles);
    };

    const handleUpload = () => {
        if (uploadMode === 'single') {
            if (!currentInput || !currentOutput) return;

            const formData = new FormData();
            formData.append('input', currentInput);
            formData.append('output', currentOutput);

            uploadMutation.mutate(formData);
        } else {
            const validFiles = testFiles.filter(f => f.input && f.output);
            if (validFiles.length === 0) return;

            const formData = new FormData();
            validFiles.forEach((file, index) => {
                if (file.input && file.output) {
                    formData.append(`testcase_${index}_input`, file.input);
                    formData.append(`testcase_${index}_output`, file.output);
                }
            });

            uploadMutation.mutate(formData);
        }
    };

    const canUpload = uploadMode === 'single'
        ? currentInput && currentOutput && !uploadMutation.isPending
        : testFiles.some(f => f.input && f.output) && !uploadMutation.isPending;

    return (
        <div className="space-y-6">
            {/* Upload Mode Toggle */}
            <div className="flex gap-2">
                <button
                    type="button"
                    onClick={() => setUploadMode('single')}
                    className={cn(
                        "px-4 py-2 rounded-xl font-medium transition-colors",
                        uploadMode === 'single'
                            ? "bg-primary text-white"
                            : "bg-card border hover:bg-muted"
                    )}
                >
                    Single Test Case
                </button>
                <button
                    type="button"
                    onClick={() => setUploadMode('batch')}
                    className={cn(
                        "px-4 py-2 rounded-xl font-medium transition-colors",
                        uploadMode === 'batch'
                            ? "bg-primary text-white"
                            : "bg-card border hover:bg-muted"
                    )}
                >
                    Batch Upload
                </button>
            </div>

            {/* Single File Upload */}
            {uploadMode === 'single' && (
                <div className="bg-card rounded-2xl border p-6 space-y-4">
                    <h4 className="font-bold">Upload Single Test Case</h4>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <label className="text-sm font-medium text-muted-foreground block mb-2">
                                Input File
                            </label>
                            <div className="border-2 border-dashed rounded-xl p-4 text-center hover:border-primary/50 transition-colors">
                                <input
                                    type="file"
                                    accept=".in,.input,.txt"
                                    onChange={(e) => handleSingleFileChange('input', e.target.files?.[0] || null)}
                                    className="hidden"
                                    id="single-input"
                                />
                                <label htmlFor="single-input" className="cursor-pointer">
                                    {currentInput ? (
                                        <div className="flex items-center justify-center gap-2 text-success">
                                            <CheckCircle size={20} />
                                            <span className="text-sm">{currentInput.name}</span>
                                        </div>
                                    ) : (
                                        <div className="flex flex-col items-center gap-2 text-muted-foreground">
                                            <Upload size={24} />
                                            <span className="text-sm">Click to select input file</span>
                                        </div>
                                    )}
                                </label>
                            </div>
                        </div>

                        <div>
                            <label className="text-sm font-medium text-muted-foreground block mb-2">
                                Output File
                            </label>
                            <div className="border-2 border-dashed rounded-xl p-4 text-center hover:border-primary/50 transition-colors">
                                <input
                                    type="file"
                                    accept=".out,.output,.ans"
                                    onChange={(e) => handleSingleFileChange('output', e.target.files?.[0] || null)}
                                    className="hidden"
                                    id="single-output"
                                />
                                <label htmlFor="single-output" className="cursor-pointer">
                                    {currentOutput ? (
                                        <div className="flex items-center justify-center gap-2 text-success">
                                            <CheckCircle size={20} />
                                            <span className="text-sm">{currentOutput.name}</span>
                                        </div>
                                    ) : (
                                        <div className="flex flex-col items-center gap-2 text-muted-foreground">
                                            <Upload size={24} />
                                            <span className="text-sm">Click to select output file</span>
                                        </div>
                                    )}
                                </label>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* Batch Upload */}
            {uploadMode === 'batch' && (
                <div className="bg-card rounded-2xl border p-6 space-y-4">
                    <h4 className="font-bold">Batch Upload Test Cases</h4>
                    <p className="text-sm text-muted-foreground">
                        Select multiple input (.in, .input, .txt) and output (.out, .output, .ans) files.
                        Files with matching base names will be paired automatically.
                    </p>
                    <div className="border-2 border-dashed rounded-xl p-8 text-center hover:border-primary/50 transition-colors">
                        <input
                            type="file"
                            multiple
                            accept=".in,.input,.txt,.out,.output,.ans"
                            onChange={handleBatchFileChange}
                            className="hidden"
                            id="batch-files"
                        />
                        <label htmlFor="batch-files" className="cursor-pointer">
                            <div className="flex flex-col items-center gap-2 text-muted-foreground">
                                <Upload size={32} />
                                <span>Click to select files or drag and drop</span>
                            </div>
                        </label>
                    </div>

                    {testFiles.length > 0 && (
                        <div className="space-y-2">
                            <h5 className="font-medium text-sm">Selected Files ({testFiles.length} test cases)</h5>
                            <div className="max-h-64 overflow-y-auto space-y-2">
                                {testFiles.map((file, idx) => (
                                    <div key={idx} className="flex items-center gap-3 p-3 bg-muted/30 rounded-xl">
                                        <FileText size={16} className="text-muted-foreground" />
                                        <div className="flex-1 min-w-0">
                                            <div className="text-sm truncate">
                                                {file.inputName || 'No input'} → {file.outputName || 'No output'}
                                            </div>
                                        </div>
                                        {!file.input && (
                                            <Badge variant="destructive">Missing input</Badge>
                                        )}
                                        {!file.output && (
                                            <Badge variant="destructive">Missing output</Badge>
                                        )}
                                        {file.input && file.output && (
                                            <Badge variant="success">Ready</Badge>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            )}

            {/* Upload Button */}
            <button
                type="button"
                onClick={handleUpload}
                disabled={!canUpload}
                className={cn(
                    "w-full py-3 rounded-xl font-medium transition-colors flex items-center justify-center gap-2",
                    canUpload
                        ? "bg-primary text-white hover:bg-primary/90"
                        : "bg-muted text-muted-foreground cursor-not-allowed"
                )}
            >
                {uploadMutation.isPending ? (
                    <>
                        <Loader2 className="animate-spin" size={20} />
                        Uploading...
                    </>
                ) : (
                    <>
                        <Upload size={20} />
                        Upload Test Case{uploadMode === 'batch' ? 's' : ''}
                    </>
                )}
            </button>

            {/* Existing Test Cases */}
            {testCases.length > 0 && (
                <div className="bg-card rounded-2xl border p-6 space-y-4">
                    <h4 className="font-bold flex items-center gap-2">
                        <FileText size={20} />
                        Existing Test Cases ({testCases.length})
                    </h4>
                    <div className="space-y-2">
                        {testCases.map((tc) => (
                            <div key={tc.id} className="flex items-center justify-between p-3 bg-muted/30 rounded-xl">
                                <div className="flex items-center gap-3">
                                    <Badge variant="secondary">#{tc.order}</Badge>
                                    <div className="text-sm">
                                        <span className="text-muted-foreground">Input:</span>{' '}
                                        <span className="font-mono">{tc.input_file}</span>
                                        {' → '}
                                        <span className="text-muted-foreground">Output:</span>{' '}
                                        <span className="font-mono">{tc.output_file}</span>
                                    </div>
                                </div>
                                <button
                                    type="button"
                                    onClick={() => deleteMutation.mutate(tc.id)}
                                    disabled={deleteMutation.isPending}
                                    className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors disabled:opacity-50"
                                >
                                    <Trash2 size={18} />
                                </button>
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
