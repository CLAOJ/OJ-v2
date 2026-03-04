'use client';

import { useState, useEffect } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { adminOrganizationApi, type AdminOrganizationUpdateRequest } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { X, Edit, Trash2 } from 'lucide-react';

interface OrganizationEditModalProps {
    organization: {
        id: number;
        name: string;
        slug: string;
        short_name: string;
        about?: string;
        is_open: boolean;
        is_unlisted: boolean;
        member_count: number;
    } | null;
    onClose: () => void;
}

export default function OrganizationEditModal({ organization, onClose }: OrganizationEditModalProps) {
    const queryClient = useQueryClient();
    const [formData, setFormData] = useState<AdminOrganizationUpdateRequest>({});

    useEffect(() => {
        if (organization) {
            setFormData({
                name: organization.name,
                slug: organization.slug,
                short_name: organization.short_name,
                about: organization.about || '',
                is_open: organization.is_open,
                is_unlisted: organization.is_unlisted,
            });
        }
    }, [organization]);

    const updateMutation = useMutation({
        mutationFn: (data: AdminOrganizationUpdateRequest) =>
            adminOrganizationApi.update(organization!.id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-organizations'] });
            onClose();
        }
    });

    const deleteMutation = useMutation({
        mutationFn: () => adminOrganizationApi.delete(organization!.id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-organizations'] });
            onClose();
        }
    });

    const handleSubmit = () => {
        updateMutation.mutate(formData);
    };

    const handleDelete = () => {
        if (confirm('Are you sure you want to delete this organization? This action cannot be undone.')) {
            deleteMutation.mutate();
        }
    };

    if (!organization) return null;

    return (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
            <div className="bg-card rounded-2xl w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto">
                {/* Header */}
                <div className="flex items-center justify-between mb-6">
                    <h2 className="text-xl font-bold">Edit Organization</h2>
                    <button
                        onClick={onClose}
                        className="p-2 hover:bg-muted rounded-lg transition-colors"
                    >
                        <X size={20} />
                    </button>
                </div>

                {/* Content */}
                <div className="space-y-4">
                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            Name
                        </label>
                        <input
                            type="text"
                            className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={formData.name || ''}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                        />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="text-sm font-medium text-muted-foreground block mb-2">
                                Slug
                            </label>
                            <input
                                type="text"
                                className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                value={formData.slug || ''}
                                onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                            />
                        </div>

                        <div>
                            <label className="text-sm font-medium text-muted-foreground block mb-2">
                                Short Name
                            </label>
                            <input
                                type="text"
                                className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                value={formData.short_name || ''}
                                onChange={(e) => setFormData({ ...formData, short_name: e.target.value })}
                            />
                        </div>
                    </div>

                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            About
                        </label>
                        <textarea
                            className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none min-h-[100px] text-sm"
                            value={formData.about || ''}
                            onChange={(e) => setFormData({ ...formData, about: e.target.value })}
                        />
                    </div>

                    <div className="flex items-center gap-4">
                        <label className="flex items-center gap-2 cursor-pointer">
                            <input
                                type="checkbox"
                                className="rounded w-5 h-5"
                                checked={formData.is_open ?? false}
                                onChange={(e) => setFormData({ ...formData, is_open: e.target.checked })}
                            />
                            <span className="text-sm font-medium">Open (users can join)</span>
                        </label>

                        <label className="flex items-center gap-2 cursor-pointer">
                            <input
                                type="checkbox"
                                className="rounded w-5 h-5"
                                checked={formData.is_unlisted ?? false}
                                onChange={(e) => setFormData({ ...formData, is_unlisted: e.target.checked })}
                            />
                            <span className="text-sm font-medium">Unlisted (hidden from public)</span>
                        </label>
                    </div>

                    {/* Stats */}
                    <div className="bg-muted/30 rounded-xl p-4">
                        <div className="text-sm text-muted-foreground">
                            <span className="font-medium">Members:</span> {organization.member_count}
                        </div>
                    </div>
                </div>

                {/* Actions */}
                <div className="flex justify-between items-center mt-6 pt-6 border-t">
                    <button
                        type="button"
                        onClick={handleDelete}
                        disabled={deleteMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-destructive/10 text-destructive hover:bg-destructive/20 flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <Trash2 size={18} />
                        Delete
                    </button>

                    <div className="flex gap-3">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 rounded-xl hover:bg-muted transition-colors"
                        >
                            Cancel
                        </button>
                        <button
                            type="button"
                            onClick={handleSubmit}
                            disabled={updateMutation.isPending}
                            className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium disabled:opacity-50"
                        >
                            <Edit size={18} />
                            {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}
