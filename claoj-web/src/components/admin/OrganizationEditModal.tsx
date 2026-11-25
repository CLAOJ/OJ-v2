'use client';

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useConfirm } from '@/components/providers/ConfirmProvider';
import { toast } from 'sonner';
import {
    adminOrganizationApi,
    adminErrorMessage,
    type AdminOrganizationCreateRequest,
    type AdminOrganizationUpdateRequest,
} from '@/lib/adminApi';
import { searchUsers } from '@/lib/adminSearch';
import { EntityPicker } from '@/components/admin/EntityPicker';
import { useAuth } from '@/components/providers/AuthProvider';
import type { AdminOrganizationDetail } from '@/types';
import { Link } from '@/navigation';
import { X, Edit, Trash2, RefreshCw, ExternalLink } from 'lucide-react';
import dayjs from 'dayjs';

interface OrganizationEditModalProps {
    /** null creates a new organization; an id edits that organization. */
    organizationId: number | null;
    onClose: () => void;
}

interface OrgForm {
    name: string;
    slug: string;
    short_name: string;
    about: string;
    is_open: boolean;
    is_unlisted: boolean;
    slots: string;
    access_code: string;
    logo_override_image: string;
    admins: number[];
}

const EMPTY_FORM: OrgForm = {
    name: '',
    slug: '',
    short_name: '',
    about: '',
    is_open: false,
    is_unlisted: true,
    slots: '',
    access_code: '',
    logo_override_image: '',
    admins: [],
};

const inputClass = 'w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none';

export default function OrganizationEditModal({ organizationId, onClose }: OrganizationEditModalProps) {
    const t = useTranslations('Admin');
    const isEdit = organizationId !== null;
    const { data: detail, isLoading, isError } = useQuery({
        queryKey: ['admin-organization', organizationId],
        queryFn: async () => (await adminOrganizationApi.get(organizationId as number)).data,
        enabled: isEdit,
    });

    return (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
            <div className="bg-card rounded-md w-full max-w-2xl p-6 max-h-[90vh] overflow-y-auto">
                <div className="flex items-center justify-between mb-6">
                    <h2 className="text-xl font-semibold">
                        {isEdit ? t('organizations.editModalTitle') : t('organizations.createModalTitle')}
                    </h2>
                    <button onClick={onClose} className="p-2 hover:bg-muted rounded-lg transition-colors">
                        <X size={20} />
                    </button>
                </div>

                {isEdit && isLoading ? (
                    <p className="text-muted-foreground py-8 text-center">{t('common.loading')}</p>
                ) : isEdit && isError ? (
                    <p className="text-destructive py-8 text-center">{t('organizations.loadFailed')}</p>
                ) : (
                    <OrganizationForm organizationId={organizationId} detail={detail} onClose={onClose} />
                )}
            </div>
        </div>
    );
}

interface OrganizationFormProps {
    organizationId: number | null;
    detail?: AdminOrganizationDetail;
    onClose: () => void;
}

function OrganizationForm({ organizationId, detail, onClose }: OrganizationFormProps) {
    const t = useTranslations('Admin');
    const confirm = useConfirm();
    const queryClient = useQueryClient();
    const isEdit = organizationId !== null;
    // Mirrors the judge.organization_admin gate on the API, which rejects
    // changes to admins/is_open/slots without it.
    const canManage = useAuth().user?.can_organization_admin === true;

    // Mounted once with the loaded detail, so initial state needs no effect.
    const [form, setForm] = useState<OrgForm>(() => detail ? {
        name: detail.name,
        slug: detail.slug,
        short_name: detail.short_name,
        about: detail.about ?? '',
        is_open: detail.is_open,
        is_unlisted: detail.is_unlisted,
        slots: detail.slots == null ? '' : String(detail.slots),
        access_code: detail.access_code ?? '',
        logo_override_image: detail.logo_override_image ?? '',
        admins: (detail.admins ?? []).map(a => a.id),
    } : EMPTY_FORM);

    const invalidate = () => {
        queryClient.invalidateQueries({ queryKey: ['admin-organizations'] });
        queryClient.invalidateQueries({ queryKey: ['admin-organization', organizationId] });
    };

    const saveMutation = useMutation({
        mutationFn: async () => {
            const slots = form.slots.trim() === '' ? undefined : Number(form.slots);
            if (isEdit) {
                const payload: AdminOrganizationUpdateRequest = {
                    name: form.name,
                    slug: form.slug,
                    short_name: form.short_name,
                    about: form.about,
                    is_open: form.is_open,
                    is_unlisted: form.is_unlisted,
                    access_code: form.access_code,
                    logo_override_image: form.logo_override_image,
                    admins: form.admins,
                };
                if (slots !== undefined) payload.slots = slots;
                else payload.clear_slots = true;
                return adminOrganizationApi.update(organizationId as number, payload);
            }
            const create: AdminOrganizationCreateRequest = {
                name: form.name,
                slug: form.slug,
                short_name: form.short_name,
                about: form.about,
                is_open: form.is_open,
                is_unlisted: form.is_unlisted,
                access_code: form.access_code,
                logo_override_image: form.logo_override_image,
                admins: form.admins,
                slots,
            };
            return adminOrganizationApi.create(create);
        },
        onSuccess: () => {
            invalidate();
            toast.success(t('organizations.saved'));
            onClose();
        },
        onError: (error) => toast.error(adminErrorMessage(error, t('organizations.saveFailed'))),
    });

    const recalcMutation = useMutation({
        mutationFn: () => adminOrganizationApi.recalculate(organizationId as number),
        onSuccess: (res) => {
            invalidate();
            toast.success(t('organizations.recalculated', { points: res.data.performance_points }));
        },
        onError: (error) => toast.error(adminErrorMessage(error, t('organizations.recalcFailed'))),
    });

    const deleteMutation = useMutation({
        mutationFn: () => adminOrganizationApi.delete(organizationId as number),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-organizations'] });
            onClose();
        },
        onError: (error) => toast.error(adminErrorMessage(error, t('organizations.deleteFailed'))),
    });

    const handleDelete = async () => {
        if (await confirm(t('organizations.deleteConfirm'), { variant: 'danger' })) {
            deleteMutation.mutate();
        }
    };

    const isValid = form.name.trim() !== '' && form.slug.trim() !== '' && form.short_name.trim() !== '';

    return (
        <>
            <div className="space-y-4">
                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('organizations.nameLabel')}
                    </label>
                    <input
                        type="text"
                        className={inputClass}
                        value={form.name}
                        onChange={(e) => setForm({ ...form, name: e.target.value })}
                    />
                </div>

                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            {t('organizations.slugLabel')}
                        </label>
                        <input
                            type="text"
                            className={inputClass}
                            value={form.slug}
                            onChange={(e) => setForm({ ...form, slug: e.target.value })}
                        />
                    </div>
                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            {t('organizations.shortNameLabel')}
                        </label>
                        <input
                            type="text"
                            className={inputClass}
                            value={form.short_name}
                            onChange={(e) => setForm({ ...form, short_name: e.target.value })}
                        />
                    </div>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('organizations.aboutLabel')}
                    </label>
                    <textarea
                        className={`${inputClass} min-h-[100px] text-sm`}
                        value={form.about}
                        onChange={(e) => setForm({ ...form, about: e.target.value })}
                    />
                </div>

                <div className="grid grid-cols-2 gap-4">
                    <div className="flex items-center gap-4">
                        <label className="flex items-center gap-2 cursor-pointer">
                            <input
                                type="checkbox"
                                className="rounded w-5 h-5"
                                checked={form.is_open}
                                disabled={!canManage}
                                onChange={(e) => setForm({ ...form, is_open: e.target.checked })}
                            />
                            <span className="text-sm font-medium">{t('organizations.openLabel')}</span>
                        </label>
                        <label className="flex items-center gap-2 cursor-pointer">
                            <input
                                type="checkbox"
                                className="rounded w-5 h-5"
                                checked={form.is_unlisted}
                                onChange={(e) => setForm({ ...form, is_unlisted: e.target.checked })}
                            />
                            <span className="text-sm font-medium">{t('organizations.unlistedLabel')}</span>
                        </label>
                    </div>
                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            {t('organizations.slotsLabel')}
                        </label>
                        <input
                            type="number"
                            min={0}
                            className={inputClass}
                            value={form.slots}
                            disabled={!canManage}
                            onChange={(e) => setForm({ ...form, slots: e.target.value })}
                        />
                    </div>
                </div>
                {!canManage && (
                    <p className="text-xs text-muted-foreground">{t('organizations.fieldGateHint')}</p>
                )}

                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            {t('organizations.accessCodeLabel')}
                        </label>
                        <input
                            type="text"
                            maxLength={7}
                            className={inputClass}
                            value={form.access_code}
                            onChange={(e) => setForm({ ...form, access_code: e.target.value })}
                        />
                    </div>
                    <div>
                        <label className="text-sm font-medium text-muted-foreground block mb-2">
                            {t('organizations.logoLabel')}
                        </label>
                        <input
                            type="text"
                            placeholder={t('organizations.logoPlaceholder')}
                            className={inputClass}
                            value={form.logo_override_image}
                            onChange={(e) => setForm({ ...form, logo_override_image: e.target.value })}
                        />
                    </div>
                </div>

                <EntityPicker
                    id="org-admins"
                    label={t('organizations.adminsLabel')}
                    value={form.admins}
                    onChange={(admins) => setForm({ ...form, admins })}
                    search={searchUsers}
                    queryKey="users"
                    initialOptions={(detail?.admins ?? []).map(a => ({ id: a.id, label: a.username }))}
                    disabled={!canManage}
                />

                {isEdit && detail && (
                    <div className="bg-muted/30 rounded-xl p-4 space-y-2 text-sm">
                        <div className="flex justify-between">
                            <span className="text-muted-foreground">{t('organizations.membersLabel')}</span>
                            <span className="font-medium">{detail.member_count}</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-muted-foreground">{t('organizations.performancePointsLabel')}</span>
                            <span className="font-medium flex items-center gap-3">
                                {detail.performance_points}
                                <button
                                    type="button"
                                    onClick={() => recalcMutation.mutate()}
                                    disabled={recalcMutation.isPending}
                                    className="px-2 py-1 rounded-lg border text-xs hover:bg-muted transition-colors flex items-center gap-1 disabled:opacity-50"
                                >
                                    <RefreshCw size={12} />
                                    {t('organizations.recalculate')}
                                </button>
                            </span>
                        </div>
                        <div className="flex justify-between">
                            <span className="text-muted-foreground">{t('organizations.creationDateLabel')}</span>
                            <span className="font-medium">{dayjs(detail.creation_date).format('L HH:mm')}</span>
                        </div>
                        <Link
                            href={`/organization/${detail.id}`}
                            target="_blank"
                            className="inline-flex items-center gap-1 text-primary hover:underline"
                        >
                            <ExternalLink size={14} />
                            {t('organizations.viewOnSite')}
                        </Link>
                    </div>
                )}
            </div>

            <div className="flex justify-between items-center mt-6 pt-6 border-t">
                {isEdit ? (
                    <button
                        type="button"
                        onClick={handleDelete}
                        disabled={deleteMutation.isPending}
                        className="px-4 py-2 rounded-xl bg-destructive/10 text-destructive hover:bg-destructive/20 flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <Trash2 size={18} />
                        {t('common.delete')}
                    </button>
                ) : <span />}

                <div className="flex gap-3">
                    <button type="button" onClick={onClose} className="px-4 py-2 rounded-xl hover:bg-muted transition-colors">
                        {t('common.cancel')}
                    </button>
                    <button
                        type="button"
                        onClick={() => saveMutation.mutate()}
                        disabled={saveMutation.isPending || !isValid}
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium disabled:opacity-50"
                    >
                        <Edit size={18} />
                        {saveMutation.isPending
                            ? t('common.saving')
                            : isEdit ? t('common.saveChanges') : t('common.create')}
                    </button>
                </div>
            </div>
        </>
    );
}