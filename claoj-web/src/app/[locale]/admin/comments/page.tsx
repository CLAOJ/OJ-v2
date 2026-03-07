'use client';

import { useState, useEffect } from 'react';
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

export default function CommentAdminPage() {
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
    const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);
    const [pageTypeFilter, setPageTypeFilter] = useState<'all' | 'problem' | 'editorial' | 'blog'>('all');

    const loadComments = async () => {
        setLoading(true);
        try {
            const filters: { search?: string; hidden?: 'true' | 'false'; page_type?: 'problem' | 'editorial' | 'blog' } = {};
            if (search) filters.search = search;
            if (hiddenFilter !== 'all') filters.hidden = hiddenFilter;
            if (pageTypeFilter !== 'all') filters.page_type = pageTypeFilter;

            const response = await adminCommentApi.list(page, pageSize, filters);
            setComments(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error('Failed to load comments');
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
        setIsEditDialogOpen(true);
    };

    const handleUpdateComment = async () => {
        if (!editingComment) return;

        try {
            await adminCommentApi.update(editingComment.id, {
                body: editBody,
                hidden: editHidden,
                reason: editReason,
            });
            toast.success('Comment updated');
            setIsEditDialogOpen(false);
            loadComments();
        } catch (error) {
            toast.error('Failed to update comment');
        }
    };

    const handleToggleHidden = async (comment: AdminComment) => {
        try {
            await adminCommentApi.update(comment.id, {
                hidden: !comment.hidden,
            });
            toast.success(comment.hidden ? 'Comment unhidden' : 'Comment hidden');
            loadComments();
        } catch (error) {
            toast.error('Failed to update comment');
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminCommentApi.delete(id);
            toast.success('Comment deleted');
            setDeleteConfirmId(null);
            loadComments();
        } catch (error) {
            toast.error('Failed to delete comment');
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">Comments</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage and moderate user comments
                    </p>
                </div>
                <Badge variant="secondary">{total} total</Badge>
            </div>

            {/* Filters */}
            <form onSubmit={handleSearch} className="flex gap-4 flex-wrap">
                <div className="flex-1 relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                        placeholder="Search by body, username, or page..."
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
                    <option value="all">All Types</option>
                    <option value="problem">Problem Comments</option>
                    <option value="editorial">Editorial Comments</option>
                    <option value="blog">Blog Comments</option>
                </select>
                <select
                    value={hiddenFilter}
                    onChange={(e) => {
                        setHiddenFilter(e.target.value as typeof hiddenFilter);
                        setPage(1);
                    }}
                    className="px-4 py-2 border rounded-lg bg-background"
                >
                    <option value="all">All Comments</option>
                    <option value="false">Visible</option>
                    <option value="true">Hidden</option>
                </select>
                <Button type="submit">
                    <Filter className="h-4 w-4 mr-2" />
                    Filter
                </Button>
            </form>

            {/* Comments Table */}
            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>ID</TableHead>
                            <TableHead>Author</TableHead>
                            <TableHead>Page</TableHead>
                            <TableHead className="max-w-md">Body</TableHead>
                            <TableHead>Score</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Time</TableHead>
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
                        ) : comments.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                                    No comments found
                                </TableCell>
                            </TableRow>
                        ) : (
                            comments.map((comment) => (
                                <TableRow key={comment.id}>
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
                                    <TableCell className="max-w-md">
                                        <div className={`truncate ${comment.hidden ? 'text-muted-foreground italic' : ''}`}>
                                            {comment.body}
                                        </div>
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
                                                Hidden
                                            </Badge>
                                        ) : (
                                            <Badge variant="outline" className="gap-1">
                                                <Eye className="h-3 w-3" />
                                                Visible
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
                                                title={comment.hidden ? 'Unhide' : 'Hide'}
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
                                            >
                                                <Edit className="h-4 w-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => setDeleteConfirmId(comment.id)}
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

            {/* Edit Dialog */}
            <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
                <DialogContent className="max-w-2xl">
                    <DialogHeader>
                        <DialogTitle>Edit Comment</DialogTitle>
                        <DialogDescription>
                            Modify comment content and visibility. Changes will be logged with a reason.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="grid gap-2">
                            <Label htmlFor="body">Comment Body</Label>
                            <Textarea
                                id="body"
                                value={editBody}
                                onChange={(e) => setEditBody(e.target.value)}
                                rows={8}
                                className="font-mono text-sm"
                            />
                        </div>
                        <div className="grid gap-2">
                            <Label htmlFor="reason">Edit Reason (optional)</Label>
                            <Input
                                id="reason"
                                value={editReason}
                                onChange={(e) => setEditReason(e.target.value)}
                                placeholder="Why are you editing this comment?"
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
                                Hide comment (only visible to admins)
                            </Label>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsEditDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleUpdateComment}>
                            Save Changes
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Comment</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this comment? This action cannot be undone.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteConfirmId(null)}>
                            Cancel
                        </Button>
                        <Button variant="destructive" onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}>
                            Delete
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
