'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminProblemDataApi } from '@/lib/adminApi';
import { FolderOpen, FileText, Trash2, Loader2 } from 'lucide-react';

interface FilesTabProps {
    code: string;
}

interface ProblemFile {
    name: string;
    path: string;
    type: 'file' | 'directory';
}

export function FilesTab({ code }: FilesTabProps) {
    const queryClient = useQueryClient();

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
                {filesData?.map((file: ProblemFile) => (
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
