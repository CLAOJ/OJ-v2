'use client';

import { useState, useEffect } from 'react';
import { adminLanguageLimitApi, adminProblemApi, adminLanguageApi } from '@/lib/adminApi';
import { AdminLanguageLimit, AdminLanguageLimitCreateRequest, AdminLanguageLimitUpdateRequest } from '@/types';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/Table';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/Dialog';
import { Label } from '@/components/ui/Label';
import { Select } from '@/components/ui/Select';
import { toast } from 'sonner';
import { Plus, Edit, Trash2, Search, Settings2 } from 'lucide-react';

interface ProblemOption {
    id: number;
    code: string;
    name: string;
}

interface LanguageOption {
    id: number;
    key: string;
    name: string;
}

export default function LanguageLimitsAdminPage() {
    const [limits, setLimits] = useState<AdminLanguageLimit[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(20);
    const [loading, setLoading] = useState(false);
    const [searchProblemCode, setSearchProblemCode] = useState('');

    const [problems, setProblems] = useState<ProblemOption[]>([]);
    const [languages, setLanguages] = useState<LanguageOption[]>([]);

    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [editingLimit, setEditingLimit] = useState<AdminLanguageLimit | null>(null);
    const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);

    const [formData, setFormData] = useState<AdminLanguageLimitCreateRequest>({
        problem_id: 0,
        language_id: 0,
        time_limit: 1.0,
        memory_limit: 256,
    });

    const loadProblems = async () => {
        try {
            const response = await adminProblemApi.list(1, 1000);
            setProblems(response.data.data);
        } catch (error) {
            toast.error('Failed to load problems');
        }
    };

    const loadLanguages = async () => {
        try {
            const response = await adminLanguageApi.list(1, 100);
            setLanguages(response.data.data);
        } catch (error) {
            toast.error('Failed to load languages');
        }
    };

    const loadLimits = async () => {
        setLoading(true);
        try {
            const problem = problems.find(p => p.code === searchProblemCode);
            const response = await adminLanguageLimitApi.list(page, pageSize, problem?.id);
            setLimits(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error('Failed to load language limits');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadProblems();
        loadLanguages();
    }, []);

    useEffect(() => {
        loadLimits();
    }, [page, searchProblemCode]);

    const openCreateDialog = () => {
        setFormData({
            problem_id: 0,
            language_id: 0,
            time_limit: 1.0,
            memory_limit: 256,
        });
        setIsCreateDialogOpen(true);
    };

    const openEditDialog = async (limit: AdminLanguageLimit) => {
        try {
            const response = await adminLanguageLimitApi.detail(limit.id);
            setFormData({
                problem_id: response.data.problem_id,
                language_id: response.data.language_id,
                time_limit: response.data.time_limit,
                memory_limit: response.data.memory_limit,
            });
            setEditingLimit(limit);
            setIsEditDialogOpen(true);
        } catch (error) {
            toast.error('Failed to load language limit details');
        }
    };

    const handleCreate = async () => {
        if (!formData.problem_id || !formData.language_id) {
            toast.error('Please select both problem and language');
            return;
        }
        try {
            await adminLanguageLimitApi.create(formData);
            toast.success('Language limit created');
            setIsCreateDialogOpen(false);
            loadLimits();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to create language limit');
        }
    };

    const handleUpdate = async () => {
        if (!editingLimit) return;

        try {
            const updateData: AdminLanguageLimitUpdateRequest = {
                time_limit: formData.time_limit,
                memory_limit: formData.memory_limit,
            };
            await adminLanguageLimitApi.update(editingLimit.id, updateData);
            toast.success('Language limit updated');
            setIsEditDialogOpen(false);
            loadLimits();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to update language limit');
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminLanguageLimitApi.delete(id);
            toast.success('Language limit deleted');
            setDeleteConfirmId(null);
            loadLimits();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to delete language limit');
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">Language-Specific Limits</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage time and memory limits per language for each problem
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Language Limit
                </Button>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Select
                    value={searchProblemCode}
                    onChange={(e) => setSearchProblemCode(e.target.value)}
                    className="pl-10"
                >
                    <option value="">All Problems</option>
                    {problems.map((problem) => (
                        <option key={problem.id} value={problem.code}>
                            {problem.code} - {problem.name}
                        </option>
                    ))}
                </Select>
            </div>

            {/* Language Limits Table */}
            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Problem</TableHead>
                            <TableHead>Language</TableHead>
                            <TableHead>Time Limit (s)</TableHead>
                            <TableHead>Memory Limit (MB)</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8">
                                    Loading...
                                </TableCell>
                            </TableRow>
                        ) : limits.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                                    No language limits found
                                </TableCell>
                            </TableRow>
                        ) : (
                            limits.map((limit) => (
                                <TableRow key={limit.id}>
                                    <TableCell className="font-medium">
                                        {limit.problem?.code || 'Unknown'} - {limit.problem?.name || ''}
                                    </TableCell>
                                    <TableCell>
                                        <div className="flex items-center gap-2">
                                            <span className="font-medium">{limit.language?.name || limit.language?.key || 'Unknown'}</span>
                                        </div>
                                    </TableCell>
                                    <TableCell>{limit.time_limit.toFixed(2)}s</TableCell>
                                    <TableCell>{limit.memory_limit} MB</TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => openEditDialog(limit)}
                                            >
                                                <Edit className="h-4 w-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => setDeleteConfirmId(limit.id)}
                                                className="text-destructive hover:text-destructive"
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between">
                <div className="text-sm text-muted-foreground">
                    Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, total)} of {total}
                </div>
                <div className="flex gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        disabled={page === 1}
                        onClick={() => setPage(page - 1)}
                    >
                        Previous
                    </Button>
                    <Button
                        variant="outline"
                        size="sm"
                        disabled={page >= totalPages}
                        onClick={() => setPage(page + 1)}
                    >
                        Next
                    </Button>
                </div>
            </div>

            {/* Create Dialog */}
            <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
                <DialogContent className="max-w-lg">
                    <DialogHeader>
                        <DialogTitle>Create Language Limit</DialogTitle>
                        <DialogDescription>
                            Set custom time and memory limits for a specific language on a problem.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="problem">Problem</Label>
                            <Select
                                id="problem"
                                value={formData.problem_id?.toString() || ''}
                                onChange={(e) => setFormData({ ...formData, problem_id: parseInt(e.target.value) })}
                            >
                                <option value="">Select a problem</option>
                                {problems.map((problem) => (
                                    <option key={problem.id} value={problem.id.toString()}>
                                        {problem.code} - {problem.name}
                                    </option>
                                ))}
                            </Select>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="language">Language</Label>
                            <Select
                                id="language"
                                value={formData.language_id?.toString() || ''}
                                onChange={(e) => setFormData({ ...formData, language_id: parseInt(e.target.value) })}
                            >
                                <option value="">Select a language</option>
                                {languages.map((lang) => (
                                    <option key={lang.id} value={lang.id.toString()}>
                                        {lang.name} ({lang.key})
                                    </option>
                                ))}
                            </Select>
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="time_limit">Time Limit (seconds)</Label>
                                <Input
                                    id="time_limit"
                                    type="number"
                                    step="0.1"
                                    min="0.1"
                                    value={formData.time_limit}
                                    onChange={(e) => setFormData({ ...formData, time_limit: parseFloat(e.target.value) || 0 })}
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="memory_limit">Memory Limit (MB)</Label>
                                <Input
                                    id="memory_limit"
                                    type="number"
                                    min="1"
                                    value={formData.memory_limit}
                                    onChange={(e) => setFormData({ ...formData, memory_limit: parseInt(e.target.value) || 0 })}
                                />
                            </div>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsCreateDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleCreate}>
                            Create
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Edit Dialog */}
            <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
                <DialogContent className="max-w-lg">
                    <DialogHeader>
                        <DialogTitle>Edit Language Limit</DialogTitle>
                        <DialogDescription>
                            Update time and memory limits for this language.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="edit_time_limit">Time Limit (seconds)</Label>
                                <Input
                                    id="edit_time_limit"
                                    type="number"
                                    step="0.1"
                                    min="0.1"
                                    value={formData.time_limit}
                                    onChange={(e) => setFormData({ ...formData, time_limit: parseFloat(e.target.value) || 0 })}
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="edit_memory_limit">Memory Limit (MB)</Label>
                                <Input
                                    id="edit_memory_limit"
                                    type="number"
                                    min="1"
                                    value={formData.memory_limit}
                                    onChange={(e) => setFormData({ ...formData, memory_limit: parseInt(e.target.value) || 0 })}
                                />
                            </div>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsEditDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleUpdate}>
                            Save Changes
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Language Limit</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this language limit? The problem will fall back to default limits for this language.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteConfirmId(null)}>
                            Cancel
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}
                        >
                            Delete
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
