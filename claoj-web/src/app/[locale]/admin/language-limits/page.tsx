'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
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
    const t = useTranslations('Admin');
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
            toast.error(t('languageLimits.loadProblemsFailed'));
        }
    };

    const loadLanguages = async () => {
        try {
            const response = await adminLanguageApi.list(1, 100);
            setLanguages(response.data.data);
        } catch (error) {
            toast.error(t('languageLimits.loadLanguagesFailed'));
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
            toast.error(t('languageLimits.loadFailed'));
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
            toast.error(t('languageLimits.loadDetailFailed'));
        }
    };

    const handleCreate = async () => {
        if (!formData.problem_id || !formData.language_id) {
            toast.error(t('languageLimits.selectBothError'));
            return;
        }
        try {
            await adminLanguageLimitApi.create(formData);
            toast.success(t('languageLimits.createSuccess'));
            setIsCreateDialogOpen(false);
            loadLimits();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('languageLimits.createError'));
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
            toast.success(t('languageLimits.updateSuccess'));
            setIsEditDialogOpen(false);
            loadLimits();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('languageLimits.updateError'));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminLanguageLimitApi.delete(id);
            toast.success(t('languageLimits.deleteSuccess'));
            setDeleteConfirmId(null);
            loadLimits();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('languageLimits.deleteError'));
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">{t('languageLimits.title')}</h1>
                    <p className="text-muted-foreground mt-1">
                        {t('languageLimits.subtitle')}
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('languageLimits.addButton')}
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
                    <option value="">{t('languageLimits.allProblemsOption')}</option>
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
                            <TableHead>{t('languageLimits.colProblem')}</TableHead>
                            <TableHead>{t('languageLimits.colLanguage')}</TableHead>
                            <TableHead>{t('languageLimits.colTimeLimit')}</TableHead>
                            <TableHead>{t('languageLimits.colMemoryLimit')}</TableHead>
                            <TableHead className="text-right">{t('common.actions')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8">
                                    {t('common.loading')}
                                </TableCell>
                            </TableRow>
                        ) : limits.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                                    {t('languageLimits.noLimitsFound')}
                                </TableCell>
                            </TableRow>
                        ) : (
                            limits.map((limit) => (
                                <TableRow key={limit.id}>
                                    <TableCell className="font-medium">
                                        {limit.problem?.code || t('languageLimits.unknownLabel')} - {limit.problem?.name || ''}
                                    </TableCell>
                                    <TableCell>
                                        <div className="flex items-center gap-2">
                                            <span className="font-medium">{limit.language?.name || limit.language?.key || t('languageLimits.unknownLabel')}</span>
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
                    {t('common.showingRange', { from: (page - 1) * pageSize + 1, to: Math.min(page * pageSize, total), total })}
                </div>
                <div className="flex gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        disabled={page === 1}
                        onClick={() => setPage(page - 1)}
                    >
                        {t('common.previous')}
                    </Button>
                    <Button
                        variant="outline"
                        size="sm"
                        disabled={page >= totalPages}
                        onClick={() => setPage(page + 1)}
                    >
                        {t('common.next')}
                    </Button>
                </div>
            </div>

            {/* Create Dialog */}
            <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
                <DialogContent className="max-w-lg">
                    <DialogHeader>
                        <DialogTitle>{t('languageLimits.createDialogTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('languageLimits.createDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="problem">{t('languageLimits.problemLabel')}</Label>
                            <Select
                                id="problem"
                                value={formData.problem_id?.toString() || ''}
                                onChange={(e) => setFormData({ ...formData, problem_id: parseInt(e.target.value) })}
                            >
                                <option value="">{t('languageLimits.selectProblemOption')}</option>
                                {problems.map((problem) => (
                                    <option key={problem.id} value={problem.id.toString()}>
                                        {problem.code} - {problem.name}
                                    </option>
                                ))}
                            </Select>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="language">{t('languageLimits.languageLabel')}</Label>
                            <Select
                                id="language"
                                value={formData.language_id?.toString() || ''}
                                onChange={(e) => setFormData({ ...formData, language_id: parseInt(e.target.value) })}
                            >
                                <option value="">{t('languageLimits.selectLanguageOption')}</option>
                                {languages.map((lang) => (
                                    <option key={lang.id} value={lang.id.toString()}>
                                        {lang.name} ({lang.key})
                                    </option>
                                ))}
                            </Select>
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="time_limit">{t('languageLimits.timeLimitLabel')}</Label>
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
                                <Label htmlFor="memory_limit">{t('languageLimits.memoryLimitLabel')}</Label>
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
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={handleCreate}>
                            {t('common.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Edit Dialog */}
            <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
                <DialogContent className="max-w-lg">
                    <DialogHeader>
                        <DialogTitle>{t('languageLimits.editDialogTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('languageLimits.editDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="edit_time_limit">{t('languageLimits.timeLimitLabel')}</Label>
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
                                <Label htmlFor="edit_memory_limit">{t('languageLimits.memoryLimitLabel')}</Label>
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
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={handleUpdate}>
                            {t('common.saveChanges')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('languageLimits.deleteDialogTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('languageLimits.deleteDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteConfirmId(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}
                        >
                            {t('common.delete')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
