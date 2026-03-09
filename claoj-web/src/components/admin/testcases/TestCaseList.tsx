'use client';

import { useMutation } from '@tanstack/react-query';
import { adminProblemDataApi, type ProblemTestCase } from '@/lib/adminApi';
import { FileText, Trash2 } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';

interface TestCaseListProps {
    problemCode: string;
    testCases: ProblemTestCase[];
    onTestCaseDeleted: () => void;
}

export function TestCaseList({ problemCode, testCases, onTestCaseDeleted }: TestCaseListProps) {
    const deleteMutation = useMutation({
        mutationFn: (testCaseId: number) => adminProblemDataApi.deleteTestCase(problemCode, testCaseId),
        onSuccess: onTestCaseDeleted
    });

    if (testCases.length === 0) {
        return null;
    }

    return (
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
    );
}
