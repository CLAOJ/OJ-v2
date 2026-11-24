'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { adminCommentApi } from '@/lib/adminApi';
import { AdminComment } from '@/types';
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
import { toast } from 'sonner';
import { Eye, EyeOff, Trash2, Edit, Search, Filter, Link as LinkIcon } from 'lucide-react';
import { Link } from '@/navigation';
import { formatDateTime } from '@/lib/date';
import { EntityPicker } from '@/components/admin/EntityPicker';
import { searchUsers } from '@/lib/adminSearch';

export default function CommentAdminPage() {
    const t = useTranslations('Admin');
    const [comments, setComments] = useState<AdminComment[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(20);
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');
    const [hiddenFilter, setHiddenFilter] = useState<'all' | 'true' | 'false'>('all');
    const [editingComment, setEditingComment] = useState<AdminComment | null>(null);
    const [editBody, setEditBody] = useState('');
    const [editHidden, setEditHidden] = useState(false);
    const [editReason, setEditReason] = useState('');
    const [editAuthorIds, setEditAuthorIds] = useState<number[]>([]);
    const [editPage, setEditPage] = useState('');
    const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);
    const [pageTypeFilter, setPageTypeFilter] = useState<'all' | 'problem' | 'editorial' | 'blog'>('all');
    const [selectedIds, setSelectedIds] = useState<number[]>([]);
    // null = bulk confirmation closed; true = hide, false = unhide.
    const [bulkHidden, setBulkHidden] = useState<boolean | null>(null);
    const [expandedIds, setExpandedIds] = useState<number[]>([]);

    const loadComments = async () => {
        setLoading(true);
        setSelectedIds([]);
        try {
            const filters: { search?: string; hidden?: 'true' | 'false'; page_type?: 'problem' | 'editorial' | 'blog' } = {};
            if (search) filters.search = search;
            if (hiddenFilter !== 'all') filters.hidden = hiddenFilter;
            if (pageTypeFilter !== 'all') filters.page_type = pageTypeFilter;

            const response = await adminCommentApi.list(page, pageSize, filters);
            setComments(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error(t('comments.loadFailed'));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadComments();
    }, [page, search, hiddenFilter, pageTypeFilter]);

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault();
        setPage(1);
        loadComments();
    };

    const openEditDialog = (comment: AdminComment) => {
        setEditingComment(comment);
        setEditBody(comment.body);
        setEditHidden(comment.hidden);
        setEditReason('');
        setEditAuthorIds([comment.author_id]);
        setEditPage(comment.page);
        setIsEditDialogOpen(true);
    };

    const handleUpdateComment = async () => {
        if (!editingComment) return;

        try {
            await adminCommentApi.update(editingComment.id, {
                body: editBody,
                hidden: editHidden,
                author_id: editAuthorIds[0],
                page: editPage,
                reason: editReason,
            });
            toast.success(t('comments.updateSuccess'));
            setIsEditDialogOpen(false);
            loadComments();
        } catch (error) {
            toast.error(t('comments.updateError'));
        }
    };

    const toggleSelected = (id: number) => {
        setSelectedIds((prev) => prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]);
    };

    const toggleAllOnPage = () => {
        setSelectedIds((prev) => prev.length === comments.length ? [] : comments.map((c) => c.id));
    };

    const toggleExpanded = (id: number) => {
        setExpandedIds((prev) => prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]);
    };

    const handleBulkHide = async () => {
        if (bulkHidden === null) return;
        const ids = [...selectedIds];
        try {
            await adminCommentApi.bulkHide(ids, bulkHidden);
            toast.success(t('comments.bulkSuccess', { count: ids.length }));
            setBulkHidden(null);
            loadComments();
        } catch (error) {
            toast.error(t('comments.bulkError'));
        }
    };

    const handleToggleHidden = async (comment: AdminComment) => {
        try {
            await adminCommentApi.update(comment.id, {
                hidden: !comment.hidden,
            });
            toast.success(comment.hidden ? t('comments.unhiddenToast') : t('comments.hiddenToast'));
            loadComments();
        } catch (error) {
            toast.error(t('comments.updateError'));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminCommentApi.delete(id);
            toast.success(t('comments.deleteSuccess'));
            setDeleteConfirmId(null);
            loadComments();
        } catch (error) {
            toast.error(t('comments.deleteError'));
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">{t('comments.title')}</h1>
                    <p className="text-muted-foreground mt-1">
                        {t('comments.subtitle')}
                    </p>
                </div>
                <Badge variant="secondary">{t('comments.totalBadge', { total })}</Badge>
            </div>

            {/* Filters */}
            <form onSubmit={handleSearch} className="flex gap-4 flex-wrap">
                <div className="flex-1 relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                        placeholder={t('comments.searchPlaceholder')}
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        className="pl-10"
                    />
                </div>
                <select
                    value={pageTypeFilter}
                    onChange={(e) => {
                        setPageTypeFilter(e.target.value as typeof pageTypeFilter);
                        setPage(1);
                    }}
                    className="px-4 py-2 border rounded-lg bg-background"
                >
                    <option value="all">{t('comments.allTypesOption')}</option>
                    <option value="problem">{t('comments.problemCommentsOption')}</option>
                    <option value="editorial">{t('comments.editorialCommentsOption')}</option>
                    <option value="blog">{t('comments.blogCommentsOption')}</option>
                </select>
                <select
                    value={hiddenFilter}
                    onChange={(e) => {
                        setHiddenFilter(e.target.value as typeof hiddenFilter);
                        setPage(1);
                    }}
                    className="px-4 py-2 border rounded-lg bg-background"
                >
                    <option value="all">{t('comments.allCommentsOption')}</option>
                    <option value="false">{t('comments.visibleOption')}</option>
                    <option value="true">{t('comments.hiddenOption')}</option>
                </select>
                <Button type="submit">
                    <Filter className="h-4 w-4 mr-2" />
                    {t('comments.filterButton')}
                </Button>
            </form>

            {/* Bulk actions */}
            {selectedIds.length > 0 && (
                <div className="flex items-center gap-3 flex-wrap rounded-lg border bg-muted/40 px-3 py-2">
                    <span className="text-sm">{t('comments.selectedCount', { count: selectedIds.length })}</span>
                    <Button size="sm" variant="outline" onClick={() => setBulkHidden(true)}>
                        <EyeOff className="h-4 w-4 mr-1" />
                        {t('comments.bulkHideButton')}
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => setBulkHidden(false)}>
                        <Eye className="h-4 w-4 mr-1" />
                        {t('comments.bulkUnhideButton')}
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => setSelectedIds([])}>
                        {t('comments.clearSelection')}
                    </Button>
                </div>
            )}

            {/* Comments Table */}
            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-10">
                                <input
                                    type="checkbox"
                                    className="h-4 w-4"
                                    checked={comments.length > 0 && selectedIds.length === comments.length}
                                    onChange={toggleAllOnPage}
                                    aria-label={t('comments.selectAll')}
                                />
                            </TableHead>
                            <TableHead>{t('comments.colId')}</TableHead>
                            <TableHead>{t('comments.colAuthor')}</TableHead>
                            <TableHead>{t('comments.colPage')}</TableHead>
                            <TableHead className="max-w-md">{t('comments.colBody')}</TableHead>
                            <TableHead>{t('comments.colScore')}</TableHead>
                            <TableHead>{t('comments.colStatus')}</TableHead>
                            <TableHead>{t('comments.colTime')}</TableHead>
                            <TableHead className="text-right">{t('common.actions')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={9} className="text-center py-8">
                                    {t('common.loading')}
                                </TableCell>
                            </TableRow>
                        ) : comments.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={9} className="text-center py-8 text-muted-foreground">
                                    {t('comments.noCommentsFound')}
                                </TableCell>
                            </TableRow>
                        ) : (
                            comments.map((comment) => (
                                <TableRow key={comment.id}>
                                    <TableCell>
                                        <input
                                            type="checkbox"
                                            className="h-4 w-4"
                                            checked={selectedIds.includes(comment.id)}
                                            onChange={() => toggleSelected(comment.id)}
                                            aria-label={t('comments.selectRow', { id: comment.id })}
                                        />
                                    </TableCell>
                                    <TableCell className="font-mono text-sm">{comment.id}</TableCell>
                                    <TableCell>
                                        <Link
                                            href={`/user/${comment.username}`}
                                            className="text-primary hover:underline font-medium"
                                        >
                                            {comment.username}
                                        </Link>
                                    </TableCell>
                                    <TableCell>
                                        <div className="flex items-center gap-1 text-muted-foreground">
                                            <LinkIcon className="h-3 w-3" />
                                            <code className="text-xs">{comment.page}</code>
                                        </div>
                                    </TableCell>
                                    <TableCell className="max-w-md align-top">
                                        <div className={`${expandedIds.includes(comment.id) ? '' : 'line-clamp-2'} ${comment.hidden ? 'text-muted-foreground italic' : ''}`}>
                                            {comment.body}
                                        </div>
                                        {comment.body.length > 150 && (
                                            <button
                                                type="button"
                                                onClick={() => toggleExpanded(comment.id)}
                                                className="mt-1 text-xs text-primary hover:underline"
                                            >
                                                {expandedIds.includes(comment.id) ? t('comments.showLess') : t('comments.showMore')}
                                            </button>
                                        )}
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant={comment.score > 0 ? 'default' : comment.score < 0 ? 'destructive' : 'secondary'}>
                                            {comment.score > 0 ? '+' : ''}{comment.score}
                                        </Badge>
                                    </TableCell>
                                    <TableCell>
                                        {comment.hidden ? (
                                            <Badge variant="secondary" className="gap-1">
                                                <EyeOff className="h-3 w-3" />
                                                {t('comments.hiddenBadge')}
                                            </Badge>
                                        ) : (
                                            <Badge variant="outline" className="gap-1">
                                                <Eye className="h-3 w-3" />
                                                {t('comments.visibleBadge')}
                                            </Badge>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-sm text-muted-foreground">
                                        {formatDateTime(new Date(comment.time))}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => handleToggleHidden(comment)}
                                                title={comment.hidden ? t('comments.unhideTitle') : t('comments.hideTitle')}
                                                aria-label={comment.hidden ? t('comments.unhideTitle') : t('comments.hideTitle')}
                                            >
                                                {comment.hidden ? (
                                                    <Eye className="h-4 w-4" />
                                                ) : (
                                                    <EyeOff className="h-4 w-4" />
                                                )}
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => openEditDialog(comment)}
                                                title={t('common.edit')}
                                                aria-label={t('common.edit')}
                                            >
                                                <Edit className="h-4 w-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => setDeleteConfirmId(comment.id)}
                                                className="text-destructive hover:text-destructive"
                                                title={t('common.delete')}
                                                aria-label={t('common.delete')}
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

            {/* Edit Dialog */}
            <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
                <DialogContent className="max-w-2xl">
                    <DialogHeader>
                        <DialogTitle>{t('comments.editDialogTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('comments.editDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <EntityPicker
                                id="comment_author"
                                label={t('comments.authorLabel')}
                                value={editAuthorIds}
                                onChange={setEditAuthorIds}
                                search={searchUsers}
                                queryKey="users"
                                single
                                clearable={false}
                                initialOptions={editingComment ? [{ id: editingComment.author_id, label: editingComment.username }] : []}
                            />
                            <div className="grid gap-2">
                                <Label htmlFor="page">{t('comments.pageLabel')}</Label>
                                <Input
                                    id="page"
                                    value={editPage}
                                    maxLength={30}
                                    onChange={(e) => setEditPage(e.target.value)}
                                    placeholder={t('comments.pagePlaceholder')}
                                />
                            </div>
                        </div>
                        <div className="grid grid-cols-2 gap-4 text-sm">
                            <div>
                                <span className="font-medium">{t('comments.revisionsLabel')}: </span>
                                <span className="text-muted-foreground">{editingComment?.revisions}</span>
                            </div>
                            <div>
                                <span className="font-medium">{t('comments.parentLabel')}: </span>
                                <span className="text-muted-foreground">
                                    {editingComment?.parent_id ? `#${editingComment.parent_id}` : t('comments.noParent')}
                                </span>
                            </div>
                        </div>
                        {editingComment?.parent_id && (
                            <p className="text-xs text-muted-foreground">{t('comments.parentReadOnlyHint')}</p>
                        )}
                        <div className="grid gap-2">
                            <Label htmlFor="body">{t('comments.bodyLabel')}</Label>
                            <Textarea
                                id="body"
                                value={editBody}
                                onChange={(e) => setEditBody(e.target.value)}
                                rows={8}
                                className="font-mono text-sm"
                            />
                        </div>
                        <div className="grid gap-2">
                            <Label htmlFor="reason">{t('comments.reasonLabel')}</Label>
                            <Input
                                id="reason"
                                value={editReason}
                                onChange={(e) => setEditReason(e.target.value)}
                                placeholder={t('comments.reasonPlaceholder')}
                            />
                        </div>
                        <div className="flex items-center gap-2">
                            <input
                                type="checkbox"
                                id="hidden"
                                checked={editHidden}
                                onChange={(e) => setEditHidden(e.target.checked)}
                                className="h-4 w-4"
                            />
                            <Label htmlFor="hidden" className="cursor-pointer">
                                {t('comments.hideCheckboxLabel')}
                            </Label>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsEditDialogOpen(false)}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={handleUpdateComment}>
                            {t('common.saveChanges')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Bulk hide/unhide confirmation */}
            <Dialog open={bulkHidden !== null} onOpenChange={() => setBulkHidden(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>
                            {bulkHidden ? t('comments.bulkHideTitle') : t('comments.bulkUnhideTitle')}
                        </DialogTitle>
                        <DialogDescription>
                            {bulkHidden
                                ? t('comments.bulkHideDesc', { count: selectedIds.length })
                                : t('comments.bulkUnhideDesc', { count: selectedIds.length })}
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setBulkHidden(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button variant={bulkHidden ? 'destructive' : 'default'} onClick={handleBulkHide}>
                            {bulkHidden ? t('comments.bulkHideButton') : t('comments.bulkUnhideButton')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('comments.deleteDialogTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('comments.deleteDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteConfirmId(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button variant="destructive" onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}>
                            {t('common.delete')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
