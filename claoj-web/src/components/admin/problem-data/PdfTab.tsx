'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminProblemDataApi, adminProblemApi } from '@/lib/adminApi';
import { FileText, Check, Trash2, Eye, Upload, Loader2 } from 'lucide-react';

interface PdfTabProps {
    code: string;
}

interface ProblemData {
    pdf_url?: string;
}

export function PdfTab({ code }: PdfTabProps) {
    const queryClient = useQueryClient();

    const [selectedFile, setSelectedFile] = useState<File | null>(null);
    const [isUploading, setIsUploading] = useState(false);

    // Fetch problem to get current pdf_url
    const { data: problemData } = useQuery<ProblemData>({
        queryKey: ['admin-problem-detail', code],
        queryFn: async () => {
            const res = await adminProblemApi.detail(code);
            return res.data;
        }
    });

    const uploadMutation = useMutation({
        mutationFn: async (file: File) => {
            setIsUploading(true);
            const res = await adminProblemDataApi.uploadPdf(code, file);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-problem-detail', code] });
            setSelectedFile(null);
            alert('PDF uploaded successfully');
        },
        onError: (err: Error) => {
            alert(err.message || 'Failed to upload PDF');
        },
        onSettled: () => {
            setIsUploading(false);
        }
    });

    const deleteMutation = useMutation({
        mutationFn: () => adminProblemDataApi.deletePdf(code),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-problem-detail', code] });
            alert('PDF deleted successfully');
        },
        onError: (err: Error) => {
            alert(err.message || 'Failed to delete PDF');
        }
    });

    const handleUpload = () => {
        if (!selectedFile) return;
        uploadMutation.mutate(selectedFile);
    };

    const hasPdf = !!problemData?.pdf_url;

    return (
        <div className="space-y-6">
            <div className="p-6 border rounded-xl bg-muted/30">
                <h2 className="font-semibold text-lg mb-4 flex items-center gap-2">
                    <FileText size={20} className="text-primary" />
                    PDF Statement Upload
                </h2>

                {hasPdf ? (
                    <div className="space-y-4">
                        <div className="p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
                            <div className="flex items-center gap-2 text-green-700 dark:text-green-400">
                                <Check size={18} />
                                <span className="font-medium">PDF statement is configured</span>
                            </div>
                            <p className="text-sm mt-2 text-green-600 dark:text-green-500">
                                Current file: {problemData.pdf_url}
                            </p>
                        </div>

                        <div className="flex gap-4">
                            <a
                                href={`/api/v2/problem/${code}/pdf`}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors inline-flex items-center gap-2"
                            >
                                <Eye size={16} />
                                View PDF
                            </a>
                            <button
                                onClick={() => deleteMutation.mutate()}
                                disabled={deleteMutation.isPending}
                                className="px-4 py-2 bg-destructive text-destructive-foreground rounded-lg hover:bg-destructive/90 transition-colors disabled:opacity-50 inline-flex items-center gap-2"
                            >
                                <Trash2 size={16} />
                                {deleteMutation.isPending ? 'Deleting...' : 'Delete PDF'}
                            </button>
                        </div>
                    </div>
                ) : (
                    <div className="space-y-4">
                        <div className="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
                            <p className="text-amber-700 dark:text-amber-400">
                                No PDF statement configured. Upload a PDF file to enable PDF statement viewing.
                            </p>
                        </div>

                        <div className="flex items-center gap-4">
                            <input
                                type="file"
                                accept=".pdf"
                                onChange={(e) => setSelectedFile(e.target.files?.[0] || null)}
                                className="block w-full text-sm text-muted-foreground file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-semibold file:bg-primary/10 file:text-primary hover:file:bg-primary/20"
                            />
                            <button
                                onClick={handleUpload}
                                disabled={!selectedFile || isUploading}
                                className="px-6 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 inline-flex items-center gap-2"
                            >
                                {isUploading ? (
                                    <Loader2 size={16} className="animate-spin" />
                                ) : (
                                    <Upload size={16} />
                                )}
                                {isUploading ? 'Uploading...' : 'Upload PDF'}
                            </button>
                        </div>
                    </div>
                )}
            </div>

            {selectedFile && (
                <div className="p-4 border rounded-xl bg-card">
                    <h3 className="font-medium mb-2">Selected file:</h3>
                    <p className="text-sm text-muted-foreground">{selectedFile.name}</p>
                    <p className="text-xs text-muted-foreground mt-1">
                        {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                    </p>
                </div>
            )}
        </div>
    );
}
