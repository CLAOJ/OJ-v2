'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { problemSuggestionApi } from '@/lib/api';
import type { ProblemSuggestRequest } from '@/types';

interface ProblemSuggestFormProps {
    groupOptions?: { id: number; name: string }[];
    typeOptions?: { id: number; name: string }[];
}

export default function ProblemSuggestForm({ groupOptions, typeOptions }: ProblemSuggestFormProps) {
    const router = useRouter();
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const [formData, setFormData] = useState<ProblemSuggestRequest>({
        name: '',
        description: '',
        points: 10,
        partial: false,
        time_limit: 1.0,
        memory_limit: 256,
        group_id: 1,
        type_ids: [],
        source: '',
        summary: '',
        pdf_url: '',
        is_full_markup: false,
        short_circuit: false,
        additional_notes: '',
    });

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsSubmitting(true);
        setError(null);

        try {
            const response = await problemSuggestionApi.suggestProblem(formData);
            if (response.data.success) {
                alert('Problem suggestion submitted successfully! It will be reviewed by admins.');
                router.push('/problems');
            }
        } catch (err: any) {
            setError(err.response?.data?.message || 'Failed to submit suggestion');
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleChange = (
        e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
    ) => {
        const { name, value, type } = e.target;
        setFormData((prev) => ({
            ...prev,
            [name]: type === 'checkbox' ? (e.target as HTMLInputElement).checked :
                    type === 'number' ? parseFloat(value) : value,
        }));
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-6 max-w-4xl mx-auto">
            {error && (
                <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
                    {error}
                </div>
            )}

            {/* Basic Information */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                <h3 className="text-lg font-semibold mb-4">Basic Information</h3>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                        <label className="block text-sm font-medium mb-1">Problem Name *</label>
                        <input
                            type="text"
                            name="name"
                            value={formData.name}
                            onChange={handleChange}
                            required
                            className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                            placeholder="Enter problem name"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1">Source</label>
                        <input
                            type="text"
                            name="source"
                            value={formData.source}
                            onChange={handleChange}
                            className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                            placeholder="e.g., IOI 2024, Original, etc."
                        />
                    </div>
                </div>

                <div className="mt-4">
                    <label className="block text-sm font-medium mb-1">Summary</label>
                    <input
                        type="text"
                        name="summary"
                        value={formData.summary}
                        onChange={handleChange}
                        className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                        placeholder="Brief summary of the problem"
                    />
                </div>

                <div className="mt-4">
                    <label className="block text-sm font-medium mb-1">Description *</label>
                    <textarea
                        name="description"
                        value={formData.description}
                        onChange={handleChange}
                        required
                        rows={10}
                        className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 font-mono"
                        placeholder="Problem description (Markdown supported)"
                    />
                </div>
            </div>

            {/* Problem Settings */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                <h3 className="text-lg font-semibold mb-4">Problem Settings</h3>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                        <label className="block text-sm font-medium mb-1">Points *</label>
                        <input
                            type="number"
                            name="points"
                            value={formData.points}
                            onChange={handleChange}
                            required
                            step="0.01"
                            min="0"
                            className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1">Time Limit (seconds) *</label>
                        <input
                            type="number"
                            name="time_limit"
                            value={formData.time_limit}
                            onChange={handleChange}
                            required
                            step="0.1"
                            min="0.1"
                            className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1">Memory Limit (MB) *</label>
                        <input
                            type="number"
                            name="memory_limit"
                            value={formData.memory_limit}
                            onChange={handleChange}
                            required
                            min="1"
                            className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                        />
                    </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                    <div>
                        <label className="block text-sm font-medium mb-1">Problem Group *</label>
                        <select
                            name="group_id"
                            value={formData.group_id}
                            onChange={handleChange}
                            required
                            className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                        >
                            {groupOptions?.map((group) => (
                                <option key={group.id} value={group.id}>
                                    {group.name}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1">Problem Types</label>
                        <select
                            multiple
                            name="type_ids"
                            value={formData.type_ids?.map(String) || []}
                            onChange={(e) => {
                                const values = Array.from(e.target.selectedOptions, (option) =>
                                    parseInt(option.value)
                                );
                                setFormData((prev) => ({ ...prev, type_ids: values }));
                            }}
                            className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                        >
                            {typeOptions?.map((type) => (
                                <option key={type.id} value={type.id}>
                                    {type.name}
                                </option>
                            ))}
                        </select>
                        <p className="text-xs text-gray-500 mt-1">Hold Ctrl/Cmd to select multiple</p>
                    </div>
                </div>

                <div className="mt-4 space-y-2">
                    <label className="flex items-center space-x-2">
                        <input
                            type="checkbox"
                            name="partial"
                            checked={formData.partial}
                            onChange={handleChange}
                            className="rounded"
                        />
                        <span className="text-sm font-medium">Enable partial scoring</span>
                    </label>

                    <label className="flex items-center space-x-2">
                        <input
                            type="checkbox"
                            name="is_full_markup"
                            checked={formData.is_full_markup}
                            onChange={handleChange}
                            className="rounded"
                        />
                        <span className="text-sm font-medium">Use full markup for output</span>
                    </label>

                    <label className="flex items-center space-x-2">
                        <input
                            type="checkbox"
                            name="short_circuit"
                            checked={formData.short_circuit}
                            onChange={handleChange}
                            className="rounded"
                        />
                        <span className="text-sm font-medium">Short-circuit evaluation (stop on first failure)</span>
                    </label>
                </div>
            </div>

            {/* Additional Options */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                <h3 className="text-lg font-semibold mb-4">Additional Options</h3>

                <div>
                    <label className="block text-sm font-medium mb-1">PDF URL (optional)</label>
                    <input
                        type="url"
                        name="pdf_url"
                        value={formData.pdf_url}
                        onChange={handleChange}
                        className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                        placeholder="https://example.com/problem.pdf"
                    />
                </div>

                <div className="mt-4">
                    <label className="block text-sm font-medium mb-1">Notes to Admins</label>
                    <textarea
                        name="additional_notes"
                        value={formData.additional_notes}
                        onChange={handleChange}
                        rows={3}
                        className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                        placeholder="Any additional notes or context for the admins reviewing your suggestion"
                    />
                </div>
            </div>

            <div className="flex items-center justify-between pt-4">
                <p className="text-sm text-gray-500">
                    Your suggestion will be reviewed by admins. If approved, you will receive 20 contribution points.
                </p>
                <button
                    type="submit"
                    disabled={isSubmitting}
                    className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    {isSubmitting ? 'Submitting...' : 'Submit Suggestion'}
                </button>
            </div>
        </form>
    );
}
