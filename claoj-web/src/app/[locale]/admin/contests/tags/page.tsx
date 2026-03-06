'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { AdminContestTag, AdminContestTagCreateRequest, AdminContestTagUpdateRequest } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import {
    Search,
    Tag,
    Plus,
    Edit,
    Trash2,
    Hash
} from 'lucide-react';

export default function AdminContestTagsPage() {
    const [search, setSearch] = useState('');
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [editingTag, setEditingTag] = useState<AdminContestTag | null>(null);
    const [selectedColor, setSelectedColor] = useState('#3b82f6');

    const queryClient = useQueryClient();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-contest-tags', search],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminContestTag[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/contest-tags?search=${search}`);
            return res.data;
        }
    });

    const createMutation = useMutation({
        mutationFn: (data: AdminContestTagCreateRequest) =>
            api.post('/admin/contest-tags', data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-contest-tags'] });
            setShowCreateModal(false);
        }
    });

    const updateMutation = useMutation({
        mutationFn: ({ id, data }: { id: number; data: AdminContestTagUpdateRequest }) =>
            api.patch(`/admin/contest-tag/${id}`, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-contest-tags'] });
            setEditingTag(null);
        }
    });

    const deleteMutation = useMutation({
        mutationFn: (id: number) =>
            api.delete(`/admin/contest-tag/${id}`),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-contest-tags'] });
        }
    });

    const tags = data?.data || [];

    const filteredTags = tags.filter(t =>
        t.name.toLowerCase().includes(search.toLowerCase()) ||
        t.description.toLowerCase().includes(search.toLowerCase())
    );

    const handleCreate = (tagData: AdminContestTagCreateRequest | AdminContestTagUpdateRequest) => {
        if ('name' in tagData && tagData.name !== undefined) {
            createMutation.mutate(tagData as AdminContestTagCreateRequest);
        }
    };

    const handleUpdate = (tagData: AdminContestTagUpdateRequest) => {
        if (editingTag) {
            updateMutation.mutate({ id: editingTag.id, data: tagData });
        }
    };

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Tag className="text-primary" size={32} />
                        Contest Tags
                    </h1>
                    <p className="text-muted-foreground mt-1">Manage contest categorization tags</p>
                </div>

                <div className="flex gap-3">
                    <div className="relative w-full md:w-80">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder="Search tags..."
                            className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                    <button
                        onClick={() => setShowCreateModal(true)}
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                    >
                        <Plus size={18} /> Create Tag
                    </button>
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-16 rounded-2xl" />)}
                </div>
            ) : (
                <div className="grid gap-4">
                    {filteredTags.length === 0 ? (
                        <div className="text-center py-12 rounded-2xl border border-dashed bg-muted/30">
                            <Tag size={48} className="mx-auto text-muted-foreground opacity-20" />
                            <p className="text-muted-foreground mt-4">No tags found</p>
                        </div>
                    ) : (
                        filteredTags.map((tag) => (
                            <div key={tag.id} className="bg-card rounded-2xl p-6 border hover:border-primary/30 hover:shadow-lg transition-all">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-4 flex-1">
                                        <div
                                            className="w-12 h-12 rounded-xl flex items-center justify-center"
                                            style={{ backgroundColor: tag.color + '20' }}
                                        >
                                            <Hash size={24} style={{ color: tag.color }} />
                                        </div>
                                        <div>
                                            <h3
                                                className="font-bold text-lg"
                                                style={{ color: tag.color }}
                                            >
                                                #{tag.name}
                                            </h3>
                                            <p className="text-sm text-muted-foreground mt-1">
                                                {tag.description || 'No description'}
                                            </p>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <Badge
                                            variant="secondary"
                                            className="text-xs"
                                            style={{ backgroundColor: tag.color + '20', color: tag.color }}
                                        >
                                            {tag.name}
                                        </Badge>
                                        <button
                                            onClick={() => setEditingTag(tag)}
                                            className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                            title="Edit tag"
                                        >
                                            <Edit size={18} />
                                        </button>
                                        <button
                                            onClick={() => {
                                                if (confirm(`Are you sure you want to delete tag "${tag.name}"?`)) {
                                                    deleteMutation.mutate(tag.id);
                                                }
                                            }}
                                            className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                            title="Delete tag"
                                        >
                                            <Trash2 size={18} />
                                        </button>
                                    </div>
                                </div>
                            </div>
                        ))
                    )}
                </div>
            )}

            {/* Create Modal */}
            {showCreateModal && (
                <CreateEditTagModal
                    tag={null}
                    onClose={() => setShowCreateModal(false)}
                    onSubmit={handleCreate}
                    isLoading={createMutation.isPending}
                />
            )}

            {/* Edit Modal */}
            {editingTag && (
                <CreateEditTagModal
                    tag={editingTag}
                    onClose={() => setEditingTag(null)}
                    onSubmit={handleUpdate}
                    isLoading={updateMutation.isPending}
                />
            )}
        </div>
    );
}

interface CreateEditTagModalProps {
    tag: AdminContestTag | null;
    onClose: () => void;
    onSubmit: (data: AdminContestTagCreateRequest | AdminContestTagUpdateRequest) => void;
    isLoading: boolean;
}

const PRESET_COLORS = [
    '#ef4444', // red
    '#f97316', // orange
    '#f59e0b', // amber
    '#84cc16', // lime
    '#22c55e', // green
    '#10b981', // emerald
    '#14b8a6', // teal
    '#06b6d4', // cyan
    '#3b82f6', // blue
    '#6366f1', // indigo
    '#8b5cf6', // violet
    '#a855f7', // purple
    '#d946ef', // fuchsia
    '#ec4899', // pink
    '#f43f5e', // rose
];

function CreateEditTagModal({ tag, onClose, onSubmit, isLoading }: CreateEditTagModalProps) {
    const [name, setName] = useState(tag?.name || '');
    const [color, setColor] = useState(tag?.color || '#3b82f6');
    const [description, setDescription] = useState(tag?.description || '');

    const handleSubmit = () => {
        if (!name.trim()) {
            alert('Tag name is required');
            return;
        }
        onSubmit({
            name: name.trim(),
            color,
            description: description.trim(),
        });
    };

    return (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
            <div className="bg-card rounded-2xl w-full max-w-md p-6">
                <h2 className="text-xl font-bold mb-4">
                    {tag ? 'Edit Tag' : 'Create Tag'}
                </h2>

                <div className="space-y-4">
                    <div>
                        <label className="text-sm font-medium mb-2 block">Tag Name *</label>
                        <input
                            type="text"
                            className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            placeholder="e.g., icpc, monthly, rated"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            maxLength={20}
                        />
                        <p className="text-xs text-muted-foreground mt-1">
                            Alphanumeric, underscore, and hyphen only. Max 20 characters.
                        </p>
                    </div>

                    <div>
                        <label className="text-sm font-medium mb-2 block">Color</label>
                        <div className="flex items-center gap-3 mb-3">
                            <input
                                type="color"
                                value={color}
                                onChange={(e) => setColor(e.target.value)}
                                className="w-12 h-10 rounded-lg border cursor-pointer"
                            />
                            <input
                                type="text"
                                value={color}
                                onChange={(e) => setColor(e.target.value)}
                                className="flex-1 px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none font-mono"
                                placeholder="#3b82f6"
                            />
                        </div>
                        <div className="flex flex-wrap gap-2">
                            {PRESET_COLORS.map((c) => (
                                <button
                                    key={c}
                                    onClick={() => setColor(c)}
                                    className={`w-8 h-8 rounded-lg border-2 transition-transform hover:scale-110 ${
                                        color === c ? 'border-primary scale-110' : 'border-transparent'
                                    }`}
                                    style={{ backgroundColor: c }}
                                />
                            ))}
                        </div>
                    </div>

                    <div>
                        <label className="text-sm font-medium mb-2 block">Description</label>
                        <textarea
                            className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none min-h-[80px]"
                            placeholder="Optional description for this tag..."
                            value={description}
                            onChange={(e) => setDescription(e.target.value)}
                        />
                    </div>

                    {/* Preview */}
                    <div className="pt-2">
                        <label className="text-sm font-medium mb-2 block">Preview</label>
                        <div className="flex items-center gap-3 p-3 rounded-xl border bg-muted/30">
                            <div
                                className="w-10 h-10 rounded-lg flex items-center justify-center"
                                style={{ backgroundColor: color + '20' }}
                            >
                                <Hash size={20} style={{ color: color }} />
                            </div>
                            <div>
                                <div className="font-bold" style={{ color: color }}>
                                    #{name || 'tag-name'}
                                </div>
                                <div className="text-xs text-muted-foreground">
                                    {description || 'Tag description'}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="flex justify-end gap-3 mt-6">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 rounded-xl hover:bg-muted transition-colors"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={handleSubmit}
                        disabled={isLoading || !name.trim()}
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors disabled:opacity-50"
                    >
                        {isLoading ? 'Saving...' : (tag ? 'Update' : 'Create')}
                    </button>
                </div>
            </div>
        </div>
    );
}
