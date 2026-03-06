'use client';

import { useState, useEffect } from 'react';
import { adminLanguageApi } from '@/lib/adminApi';
import { AdminLanguage, AdminLanguageCreateRequest, AdminLanguageUpdateRequest } from '@/types';
import { Badge } from '@/components/ui/Badge';
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
import { Textarea } from '@/components/ui/Textarea';
import { Switch } from '@/components/ui/Switch';
import { toast } from 'sonner';
import { Plus, Edit, Trash2, Search } from 'lucide-react';

export default function LanguageAdminPage() {
    const [languages, setLanguages] = useState<AdminLanguage[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(20);
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');

    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [editingLanguage, setEditingLanguage] = useState<AdminLanguage | null>(null);
    const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);

    // Form state
    const [formData, setFormData] = useState<AdminLanguageCreateRequest>({
        key: '',
        name: '',
        short_name: '',
        common_name: '',
        ace: '',
        pygments: '',
        extension: '',
        file_only: false,
        file_size_limit: 0,
        include_in_problem: false,
        info: '',
        template: '',
        description: '',
    });

    const loadLanguages = async () => {
        setLoading(true);
        try {
            const response = await adminLanguageApi.list(page, pageSize);
            setLanguages(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error('Failed to load languages');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadLanguages();
    }, [page]);

    const openCreateDialog = () => {
        setFormData({
            key: '',
            name: '',
            short_name: '',
            common_name: '',
            ace: '',
            pygments: '',
            extension: '',
            file_only: false,
            file_size_limit: 0,
            include_in_problem: false,
            info: '',
            template: '',
            description: '',
        });
        setIsCreateDialogOpen(true);
    };

    const openEditDialog = async (lang: AdminLanguage) => {
        try {
            const response = await adminLanguageApi.detail(lang.id);
            setFormData({
                key: response.data.key,
                name: response.data.name,
                short_name: response.data.short_name || '',
                common_name: response.data.common_name,
                ace: response.data.ace,
                pygments: response.data.pygments,
                extension: response.data.extension,
                file_only: response.data.file_only,
                file_size_limit: response.data.file_size_limit,
                include_in_problem: response.data.include_in_problem,
                info: response.data.info,
                template: response.data.template,
                description: response.data.description,
            });
            setEditingLanguage(lang);
            setIsEditDialogOpen(true);
        } catch (error) {
            toast.error('Failed to load language details');
        }
    };

    const handleCreate = async () => {
        try {
            await adminLanguageApi.create(formData);
            toast.success('Language created');
            setIsCreateDialogOpen(false);
            loadLanguages();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to create language');
        }
    };

    const handleUpdate = async () => {
        if (!editingLanguage) return;

        try {
            const updateData: AdminLanguageUpdateRequest = {
                name: formData.name,
                short_name: formData.short_name || undefined,
                common_name: formData.common_name,
                ace: formData.ace,
                pygments: formData.pygments,
                extension: formData.extension,
                file_only: formData.file_only,
                file_size_limit: formData.file_size_limit,
                include_in_problem: formData.include_in_problem,
                info: formData.info,
                template: formData.template,
                description: formData.description,
            };
            await adminLanguageApi.update(editingLanguage.id, updateData);
            toast.success('Language updated');
            setIsEditDialogOpen(false);
            loadLanguages();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to update language');
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminLanguageApi.delete(id);
            toast.success('Language deleted');
            setDeleteConfirmId(null);
            loadLanguages();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to delete language');
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    const renderFormFields = () => (
        <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                    <Label htmlFor="key">Language Key</Label>
                    <Input
                        id="key"
                        value={formData.key}
                        onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                        placeholder="e.g., CPP17"
                        disabled={editingLanguage !== null}
                    />
                </div>
                <div className="grid gap-2">
                    <Label htmlFor="name">Display Name</Label>
                    <Input
                        id="name"
                        value={formData.name}
                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                        placeholder="e.g., C++17"
                    />
                </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                    <Label htmlFor="short_name">Short Name</Label>
                    <Input
                        id="short_name"
                        value={formData.short_name}
                        onChange={(e) => setFormData({ ...formData, short_name: e.target.value })}
                        placeholder="e.g., C++17"
                    />
                </div>
                <div className="grid gap-2">
                    <Label htmlFor="common_name">Common Name</Label>
                    <Input
                        id="common_name"
                        value={formData.common_name}
                        onChange={(e) => setFormData({ ...formData, common_name: e.target.value })}
                        placeholder="e.g., C++"
                    />
                </div>
            </div>

            <div className="grid grid-cols-3 gap-4">
                <div className="grid gap-2">
                    <Label htmlFor="ace">ACE Mode</Label>
                    <Input
                        id="ace"
                        value={formData.ace}
                        onChange={(e) => setFormData({ ...formData, ace: e.target.value })}
                        placeholder="e.g., c_cpp"
                    />
                </div>
                <div className="grid gap-2">
                    <Label htmlFor="pygments">Pygments Lexer</Label>
                    <Input
                        id="pygments"
                        value={formData.pygments}
                        onChange={(e) => setFormData({ ...formData, pygments: e.target.value })}
                        placeholder="e.g., cpp"
                    />
                </div>
                <div className="grid gap-2">
                    <Label htmlFor="extension">File Extension</Label>
                    <Input
                        id="extension"
                        value={formData.extension}
                        onChange={(e) => setFormData({ ...formData, extension: e.target.value })}
                        placeholder="e.g., cpp"
                    />
                </div>
            </div>

            <div className="grid gap-2">
                <Label htmlFor="info">Info</Label>
                <Input
                    id="info"
                    value={formData.info}
                    onChange={(e) => setFormData({ ...formData, info: e.target.value })}
                    placeholder="Version info, etc."
                />
            </div>

            <div className="grid gap-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                    id="description"
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    rows={3}
                    placeholder="Language description"
                />
            </div>

            <div className="grid gap-2">
                <Label htmlFor="template">Default Template</Label>
                <Textarea
                    id="template"
                    value={formData.template}
                    onChange={(e) => setFormData({ ...formData, template: e.target.value })}
                    rows={6}
                    className="font-mono text-sm"
                    placeholder="// Default code template"
                />
            </div>

            <div className="flex gap-6 pt-2">
                <div className="flex items-center gap-2">
                    <Switch
                        id="file_only"
                        checked={formData.file_only}
                        onCheckedChange={(checked) => setFormData({ ...formData, file_only: checked })}
                    />
                    <Label htmlFor="file_only">File Only</Label>
                </div>
                <div className="flex items-center gap-2">
                    <Switch
                        id="include_in_problem"
                        checked={formData.include_in_problem}
                        onCheckedChange={(checked) => setFormData({ ...formData, include_in_problem: checked })}
                    />
                    <Label htmlFor="include_in_problem">Include in Problem</Label>
                </div>
            </div>

            <div className="grid gap-2">
                <Label htmlFor="file_size_limit">File Size Limit (bytes)</Label>
                <Input
                    id="file_size_limit"
                    type="number"
                    value={formData.file_size_limit}
                    onChange={(e) => setFormData({ ...formData, file_size_limit: parseInt(e.target.value) || 0 })}
                />
            </div>
        </div>
    );

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">Languages</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage programming languages
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Language
                </Button>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder="Search languages..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-10"
                />
            </div>

            {/* Languages Table */}
            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Key</TableHead>
                            <TableHead>Name</TableHead>
                            <TableHead>Common Name</TableHead>
                            <TableHead>ACE Mode</TableHead>
                            <TableHead>Extension</TableHead>
                            <TableHead>File Only</TableHead>
                            <TableHead>Inc. Problem</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={8} className="text-center py-8">
                                    Loading...
                                </TableCell>
                            </TableRow>
                        ) : languages.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                                    No languages found
                                </TableCell>
                            </TableRow>
                        ) : (
                            languages.map((lang) => (
                                <TableRow key={lang.id}>
                                    <TableCell className="font-mono font-bold">{lang.key}</TableCell>
                                    <TableCell>
                                        <div>
                                            <div className="font-medium">{lang.name}</div>
                                            {lang.short_name && (
                                                <div className="text-xs text-muted-foreground">{lang.short_name}</div>
                                            )}
                                        </div>
                                    </TableCell>
                                    <TableCell>{lang.common_name}</TableCell>
                                    <TableCell><code className="text-xs">{lang.ace}</code></TableCell>
                                    <TableCell><code className="text-xs">.{lang.extension}</code></TableCell>
                                    <TableCell>
                                        <Badge variant={lang.file_only ? 'default' : 'secondary'}>
                                            {lang.file_only ? 'Yes' : 'No'}
                                        </Badge>
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant={lang.include_in_problem ? 'default' : 'secondary'}>
                                            {lang.include_in_problem ? 'Yes' : 'No'}
                                        </Badge>
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => openEditDialog(lang)}
                                            >
                                                <Edit className="h-4 w-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => setDeleteConfirmId(lang.id)}
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
                <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>Create Language</DialogTitle>
                        <DialogDescription>
                            Add a new programming language to the system.
                        </DialogDescription>
                    </DialogHeader>
                    {renderFormFields()}
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
                <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>Edit Language</DialogTitle>
                        <DialogDescription>
                            Update language configuration.
                        </DialogDescription>
                    </DialogHeader>
                    {renderFormFields()}
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
                        <DialogTitle>Delete Language</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this language? This cannot be undone if the language has no submissions.
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
