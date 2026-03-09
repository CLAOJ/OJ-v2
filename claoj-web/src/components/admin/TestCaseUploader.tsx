'use client';

import { useState, useRef } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { adminProblemDataApi, type ProblemTestCase } from '@/lib/adminApi';
import { Upload, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { UploadModeToggle } from './testcases/UploadModeToggle';
import { SingleUpload } from './testcases/SingleUpload';
import { BatchUpload } from './testcases/BatchUpload';
import { TestCaseList } from './testcases/TestCaseList';

interface TestCaseFile {
    input?: File;
    output?: File;
    inputName: string;
    outputName: string;
}

interface TestCaseUploaderProps {
    problemCode: string;
    existingTestCases?: ProblemTestCase[];
}

export default function TestCaseUploader({ problemCode, existingTestCases = [] }: TestCaseUploaderProps) {
    const [testFiles, setTestFiles] = useState<TestCaseFile[]>([]);
    const [uploadMode, setUploadMode] = useState<'single' | 'batch'>('single');
    const [currentInput, setCurrentInput] = useState<File | null>(null);
    const [currentOutput, setCurrentOutput] = useState<File | null>(null);

    const queryClient = useQueryClient();
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

    const handleSingleFilesSelected = (input: File | null, output: File | null) => {
        setCurrentInput(input);
        setCurrentOutput(output);
    };

    const handleBatchFilesSelected = (files: TestCaseFile[]) => {
        setTestFiles(files);
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
            <UploadModeToggle mode={uploadMode} onModeChange={setUploadMode} />

            {uploadMode === 'single' && (
                <SingleUpload onFilesSelected={handleSingleFilesSelected} />
            )}

            {uploadMode === 'batch' && (
                <BatchUpload onFilesSelected={handleBatchFilesSelected} />
            )}

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

            {testCases.length > 0 && (
                <TestCaseList
                    problemCode={problemCode}
                    testCases={testCases}
                    onTestCaseDeleted={() => queryClient.invalidateQueries({ queryKey: ['problem-data', problemCode] })}
                />
            )}
        </div>
    );
}
