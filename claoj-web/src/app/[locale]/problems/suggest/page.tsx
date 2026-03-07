'use client';

import { useState, useEffect } from 'react';
import ProblemSuggestForm from '@/components/problems/ProblemSuggestForm';
import api from '@/lib/api';

export default function ProblemSuggestPage() {
    const [groupOptions, setGroupOptions] = useState<{ id: number; name: string }[]>([]);
    const [typeOptions, setTypeOptions] = useState<{ id: number; name: string }[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        // Fetch problem groups and types for the form
        const fetchOptions = async () => {
            try {
                // We need to add APIs for fetching groups and types
                // For now, using placeholder data
                setGroupOptions([
                    { id: 1, name: 'Uncategorized' },
                ]);
                setTypeOptions([
                    { id: 1, name: 'Standard' },
                ]);
            } catch (error) {
                // Failed to fetch options - using placeholder data
            } finally {
                setIsLoading(false);
            }
        };

        fetchOptions();
    }, []);

    if (isLoading) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-8">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                <div className="mb-8">
                    <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
                        Suggest a Problem
                    </h1>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">
                        Submit a problem suggestion for the platform. Admins will review your suggestion and approve it if it meets the quality standards.
                    </p>
                </div>

                <ProblemSuggestForm groupOptions={groupOptions} typeOptions={typeOptions} />
            </div>
        </div>
    );
}
