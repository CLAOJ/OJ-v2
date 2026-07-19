'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
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
    const t = useTranslations('Admin');
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
            toast.error(t('blogPosts.loadFailed'));
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
            toast.error(t('blogPosts.loadDetailFailed'));
        }
    };

    const handleCreate = async () => {
        try {
            await adminBlogPostApi.create(formData);
            toast.success(t('blogPosts.createSuccess'));
            setIsCreateDialogOpen(false);
            loadPosts();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('blogPosts.createError'));
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
            toast.success(t('blogPosts.updateSuccess'));
            setIsEditDialogOpen(false);
            loadPosts();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('blogPosts.updateError'));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminBlogPostApi.delete(id);
            toast.success(t('blogPosts.deleteSuccess'));
            setDeleteConfirmId(null);
            loadPosts();
        } catch (error: any) {
            toast.error(error.response?.data?.error || t('blogPosts.deleteError'));
        }
    };

    const handleToggleVisibility = async (post: AdminBlogPost) => {
        try {
            await adminBlogPostApi.update(post.id, { visible: !post.visible });
            toast.success(post.visible ? t('blogPosts.hiddenToast') : t('blogPosts.visibleToast'));
            loadPosts();
        } catch (error) {
            toast.error(t('blogPosts.updateError'));
        }
    };

    const handleToggleSticky = async (post: AdminBlogPost) => {
        try {
            await adminBlogPostApi.update(post.id, { sticky: !post.sticky });
            toast.success(post.sticky ? t('blogPosts.unstickyToast') : t('blogPosts.stickyToast'));
            loadPosts();
        } catch (error) {
            toast.error(t('blogPosts.updateError'));
        }
    };

    const totalPages = Math.ceil(total / pageSize);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold">{t('blogPosts.title')}</h1>
                    <p className="text-muted-foreground mt-1">
                        {t('blogPosts.subtitle')}
                    </p>
                </div>
                <Button onClick={openCreateDialog}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('blogPosts.addButton')}
                </Button>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder={t('blogPosts.searchPlaceholder')}
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
                            <TableHead>{t('blogPosts.colTitle')}</TableHead>
                            <TableHead>{t('blogPosts.colAuthors')}</TableHead>
                            <TableHead>{t('blogPosts.colPublishDate')}</TableHead>
                            <TableHead>{t('blogPosts.colStatus')}</TableHead>
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
                        ) : posts.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                                    {t('blogPosts.noPostsFound')}
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
                                                        {t('blogPosts.stickyBadge')}
                                                    </Badge>
                                                )}
                                                {post.global_post && (
                                                    <Badge variant="outline" className="text-xs">
                                                        {t('blogPosts.globalBadge')}
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
                                                {post.visible ? t('contests.visible') : t('contests.hidden')}
                                            </Badge>
                                        </div>
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => handleToggleVisibility(post)}
                                                title={post.visible ? t('blogPosts.hideTitle') : t('blogPosts.showTitle')}
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
                                                title={post.sticky ? t('blogPosts.unstickyTitle') : t('blogPosts.stickyTitle')}
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

            {/* Create/Edit Dialog */}
            <Dialog open={isCreateDialogOpen || isEditDialogOpen} onOpenChange={(open) => {
                setIsCreateDialogOpen(open);
                setIsEditDialogOpen(open);
            }}>
                <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>
                            {editingPost ? t('blogPosts.editDialogTitle') : t('blogPosts.createDialogTitle')}
                        </DialogTitle>
                        <DialogDescription>
                            {editingPost ? t('blogPosts.editDialogDesc') : t('blogPosts.createDialogDesc')}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="title">{t('blogPosts.titleLabel')}</Label>
                                <Input
                                    id="title"
                                    value={formData.title}
                                    onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                                    placeholder={t('blogPosts.titlePlaceholder')}
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="slug">{t('blogPosts.slugLabel')}</Label>
                                <Input
                                    id="slug"
                                    value={formData.slug}
                                    onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                                    placeholder={t('blogPosts.slugPlaceholder')}
                                />
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="summary">{t('blogPosts.summaryLabel')}</Label>
                            <Textarea
                                id="summary"
                                value={formData.summary}
                                onChange={(e) => setFormData({ ...formData, summary: e.target.value })}
                                rows={3}
                                placeholder={t('blogPosts.summaryPlaceholder')}
                            />
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="content">{t('blogPosts.contentLabel')}</Label>
                            <Textarea
                                id="content"
                                value={formData.content}
                                onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                                rows={12}
                                className="font-mono text-sm"
                                placeholder={t('blogPosts.contentPlaceholder')}
                            />
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="publish_on">{t('blogPosts.publishDateLabel')}</Label>
                                <Input
                                    id="publish_on"
                                    type="datetime-local"
                                    value={formData.publish_on ? new Date(formData.publish_on).toISOString().slice(0, 16) : ''}
                                    onChange={(e) => setFormData({ ...formData, publish_on: new Date(e.target.value).toISOString() })}
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="og_image">{t('blogPosts.ogImageLabel')}</Label>
                                <Input
                                    id="og_image"
                                    value={formData.og_image}
                                    onChange={(e) => setFormData({ ...formData, og_image: e.target.value })}
                                    placeholder={t('blogPosts.ogImagePlaceholder')}
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
                                <Label htmlFor="visible">{t('blogPosts.visibleLabel')}</Label>
                            </div>
                            <div className="flex items-center gap-2">
                                <Switch
                                    id="sticky"
                                    checked={formData.sticky}
                                    onCheckedChange={(checked) => setFormData({ ...formData, sticky: checked })}
                                />
                                <Label htmlFor="sticky">{t('blogPosts.stickyLabel')}</Label>
                            </div>
                            <div className="flex items-center gap-2">
                                <Switch
                                    id="global_post"
                                    checked={formData.global_post}
                                    onCheckedChange={(checked) => setFormData({ ...formData, global_post: checked })}
                                />
                                <Label htmlFor="global_post">{t('blogPosts.globalPostLabel')}</Label>
                            </div>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setIsCreateDialogOpen(false);
                            setIsEditDialogOpen(false);
                        }}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={editingPost ? handleUpdate : handleCreate}>
                            {editingPost ? t('common.saveChanges') : t('common.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={deleteConfirmId !== null} onOpenChange={() => setDeleteConfirmId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('blogPosts.deleteDialogTitle')}</DialogTitle>
                        <DialogDescription>
                            {t('blogPosts.deleteDialogDesc')}
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
