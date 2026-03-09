'use client';

import { useState } from 'react';
import { Upload, CheckCircle } from 'lucide-react';

interface SingleUploadProps {
    onFilesSelected: (input: File | null, output: File | null) => void;
}

export function SingleUpload({ onFilesSelected }: SingleUploadProps) {
    const [currentInput, setCurrentInput] = useState<File | null>(null);
    const [currentOutput, setCurrentOutput] = useState<File | null>(null);

    const handleFileChange = (type: 'input' | 'output', file: File | null) => {
        if (type === 'input') {
            setCurrentInput(file);
        } else {
            setCurrentOutput(file);
        }
        onFilesSelected(
            type === 'input' ? file : currentInput,
            type === 'output' ? file : currentOutput
        );
    };

    return (
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
                            onChange={(e) => handleFileChange('input', e.target.files?.[0] || null)}
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
                            onChange={(e) => handleFileChange('output', e.target.files?.[0] || null)}
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
    );
}
