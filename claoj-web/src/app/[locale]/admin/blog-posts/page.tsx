'use client';

import { useState, useEffect } from 'react';
import { adminBlogPostApi } from '@/lib/adminApi';
import { AdminBlogPost, AdminBlogPostCreateRequest, AdminBlogPostUpdateRequest } from '@/types';
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
import { Plus, Edit, Trash2, Search, Eye, EyeOff, Pin } from 'lucide-react';
import { formatDateTime } from '@/lib/date';

export default function BlogPostAdminPage() {
    const [posts, setPosts] = useState<AdminBlogPost[]>([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize] = useState(20);
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');

    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [editingPost, setEditingPost] = useState<AdminBlogPost | null>(null);
    const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
    const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);

    const [formData, setFormData] = useState<AdminBlogPostCreateRequest>({
        title: '',
        slug: '',
        content: '',
        summary: '',
        publish_on: new Date().toISOString(),
        visible: true,
        sticky: false,
        global_post: false,
        og_image: '',
        organization_id: undefined,
    });

    const loadPosts = async () => {
        setLoading(true);
        try {
            const response = await adminBlogPostApi.list(page, pageSize);
            setPosts(response.data.data);
            setTotal(response.data.total);
        } catch (error) {
            toast.error('Failed to load blog posts');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadPosts();
    }, [page]);

    const openCreateDialog = () => {
        setFormData({
            title: '',
            slug: '',
            content: '',
            summary: '',
            publish_on: new Date().toISOString(),
            visible: true,
            sticky: false,
            global_post: false,
            og_image: '',
            organization_id: undefined,
        });
        setIsCreateDialogOpen(true);
    };

    const openEditDialog = async (post: AdminBlogPost) => {
        try {
            const response = await adminBlogPostApi.detail(post.id);
            setFormData({
                title: response.data.title,
                slug: response.data.slug,
                content: response.data.content,
                summary: response.data.summary,
                publish_on: response.data.publish_on,
                visible: response.data.visible,
                sticky: response.data.sticky,
                global_post: response.data.global_post,
                og_image: response.data.og_image,
                organization_id: response.data.organization_id,
            });
            setEditingPost(post);
            setIsEditDialogOpen(true);
        } catch (error) {
            toast.error('Failed to load blog post details');
        }
    };

    const handleCreate = async () => {
        try {
            await adminBlogPostApi.create(formData);
            toast.success('Blog post created');
            setIsCreateDialogOpen(false);
            loadPosts();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to create blog post');
        }
    };

    const handleUpdate = async () => {
        if (!editingPost) return;

        try {
            const updateData: AdminBlogPostUpdateRequest = {
                title: formData.title,
                slug: formData.slug,
                content: formData.content,
                summary: formData.summary,
                publish_on: formData.publish_on,
                visible: formData.visible,
                sticky: formData.sticky,
                global_post: formData.global_post,
                og_image: formData.og_image,
                organization_id: formData.organization_id,
            };
            await adminBlogPostApi.update(editingPost.id, updateData);
            toast.success('Blog post updated');
            setIsEditDialogOpen(false);
            loadPosts();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to update blog post');
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminBlogPostApi.delete(id);
            toast.success('Blog post deleted');
            setDeleteConfirmId(null);
            loadPosts();
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to delete blog post');
        }
    };

    const handleToggleVisibility = async (post: AdminBlogPost) => {
        try {
            await adminBlogPostApi.update(post.id, { visible: !post.visible });
            toast.success(post.visible ? 'Blog post hidden' : 'Blog post made visible');
            loadPosts();
        } catch (error) {
            toast.error('Failed to update blog post');
        }
    };

    const handleToggleSticky = async (post: AdminBlogPost) => {
        try {
            await adminBlogPostApi.update(post.id, { sticky: !post.sticky });
            toast.success(post.sticky ? 'Blog post unsticky' : 'Blog post made sticky');
            loadPosts();
        } catch (error) {
            toast.error('Failed to update blog post');
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">Blog Posts</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage blog posts and announcements
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Post
                </Button>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder="Search posts..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-10"
                />
            </div>

            {/* Blog Posts Table */}
            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Title</TableHead>
                            <TableHead>Authors</TableHead>
                            <TableHead>Publish Date</TableHead>
                            <TableHead>Status</TableHead>
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
                        ) : posts.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                                    No blog posts found
                                </TableCell>
                            </TableRow>
                        ) : (
                            posts.map((post) => (
                                <TableRow key={post.id}>
                                    <TableCell>
                                        <div>
                                            <div className="font-medium">{post.title}</div>
                                            <code className="text-xs text-muted-foreground">{post.slug}</code>
                                            <div className="flex gap-2 mt-1">
                                                {post.sticky && (
                                                    <Badge variant="default" className="text-xs">
                                                        <Pin className="h-3 w-3 mr-1" />
                                                        Sticky
                                                    </Badge>
                                                )}
                                                {post.global_post && (
                                                    <Badge variant="outline" className="text-xs">
                                                        Global
                                                    </Badge>
                                                )}
                                                {post.organization && (
                                                    <Badge variant="secondary" className="text-xs">
                                                        {post.organization}
                                                    </Badge>
                                                )}
                                            </div>
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <div className="text-sm">
                                            {post.author_names.join(', ')}
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <div className="text-sm text-muted-foreground">
                                            {formatDateTime(new Date(post.publish_on))}
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <div className="flex gap-2">
                                            <Badge variant={post.visible ? 'default' : 'secondary'}>
                                                {post.visible ? 'Visible' : 'Hidden'}
                                            </Badge>
                                        </div>
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => handleToggleVisibility(post)}
                                                title={post.visible ? 'Hide' : 'Show'}
                                            >
                                                {post.visible ? (
                                                    <Eye className="h-4 w-4" />
                                                ) : (
                                                    <EyeOff className="h-4 w-4" />
                                                )}
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => handleToggleSticky(post)}
                                                title={post.sticky ? 'Unsticky' : 'Sticky'}
                                            >
                                                <Pin className={`h-4 w-4 ${post.sticky ? 'fill-current' : ''}`} />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => openEditDialog(post)}
                                            >
                                                <Edit className="h-4 w-4" />
                                            </Button>
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => setDeleteConfirmId(post.id)}
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

            {/* Create/Edit Dialog */}
            <Dialog open={isCreateDialogOpen || isEditDialogOpen} onOpenChange={(open) => {
                setIsCreateDialogOpen(open);
                setIsEditDialogOpen(open);
            }}>
                <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>
                            {editingPost ? 'Edit Blog Post' : 'Create Blog Post'}
                        </DialogTitle>
                        <DialogDescription>
                            {editingPost ? 'Update blog post details.' : 'Create a new blog post.'}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="title">Title</Label>
                                <Input
                                    id="title"
                                    value={formData.title}
                                    onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                                    placeholder="Post title"
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="slug">Slug</Label>
                                <Input
                                    id="slug"
                                    value={formData.slug}
                                    onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                                    placeholder="post-slug"
                                />
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="summary">Summary</Label>
                            <Textarea
                                id="summary"
                                value={formData.summary}
                                onChange={(e) => setFormData({ ...formData, summary: e.target.value })}
                                rows={3}
                                placeholder="Brief summary"
                            />
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="content">Content</Label>
                            <Textarea
                                id="content"
                                value={formData.content}
                                onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                                rows={12}
                                className="font-mono text-sm"
                                placeholder="Markdown content"
                            />
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="publish_on">Publish Date</Label>
                                <Input
                                    id="publish_on"
                                    type="datetime-local"
                                    value={formData.publish_on ? new Date(formData.publish_on).toISOString().slice(0, 16) : ''}
                                    onChange={(e) => setFormData({ ...formData, publish_on: new Date(e.target.value).toISOString() })}
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="og_image">OG Image URL</Label>
                                <Input
                                    id="og_image"
                                    value={formData.og_image}
                                    onChange={(e) => setFormData({ ...formData, og_image: e.target.value })}
                                    placeholder="https://..."
                                />
                            </div>
                        </div>

                        <div className="flex flex-wrap gap-4 pt-2">
                            <div className="flex items-center gap-2">
                                <Switch
                                    id="visible"
                                    checked={formData.visible}
                                    onCheckedChange={(checked) => setFormData({ ...formData, visible: checked })}
                                />
                                <Label htmlFor="visible">Visible</Label>
                            </div>
                            <div className="flex items-center gap-2">
                                <Switch
                                    id="sticky"
                                    checked={formData.sticky}
                                    onCheckedChange={(checked) => setFormData({ ...formData, sticky: checked })}
                                />
                                <Label htmlFor="sticky">Sticky</Label>
                            </div>
                            <div className="flex items-center gap-2">
                                <Switch
                                    id="global_post"
                                    checked={formData.global_post}
                                    onCheckedChange={(checked) => setFormData({ ...formData, global_post: checked })}
                                />
                                <Label htmlFor="global_post">Global Post</Label>
                            </div>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateDialogOpen(false);
                            setIsEditDialogOpen(false);
                        }}>
                            Cancel
                        </Button>
                        <Button onClick={editingPost ? handleUpdate : handleCreate}>
                            {editingPost ? 'Save Changes' : 'Create'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Blog Post</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this blog post? This action cannot be undone.
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
