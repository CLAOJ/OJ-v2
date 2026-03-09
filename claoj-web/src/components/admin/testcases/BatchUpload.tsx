'use client';

import { Upload, FileText } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';

interface TestCaseFile {
    input?: File;
    output?: File;
    inputName: string;
    outputName: string;
}

interface BatchUploadProps {
    onFilesSelected: (files: TestCaseFile[]) => void;
}

export function BatchUpload({ onFilesSelected }: BatchUploadProps) {
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

        onFilesSelected(newTestFiles);
    };

    const testFiles: TestCaseFile[] = [];

    return (
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
    );
}
