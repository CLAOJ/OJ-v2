'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminClassApi } from '@/lib/adminApi';
import { apiErrorMessage } from '@/lib/apiError';
import { searchOrganizations, searchUsers } from '@/lib/adminSearch';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import { useAuth } from '@/components/providers/AuthProvider';
import { hasPerm } from '@/lib/permissions';
import { EntityPicker, EntityOption } from '@/components/admin/EntityPicker';
import { PaginationBar } from '@/components/ui/PaginationBar';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Switch } from '@/components/ui/Switch';
import { Textarea } from '@/components/ui/Textarea';
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
import { Edit, GraduationCap, Plus, Search, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

const FILTER_KEYS = ['search'] as const;
const PAGE_SIZE = 20;

interface ClassForm {
    name: string;
    slug: string;
    description: string;
    is_active: boolean;
    access_code: string;
    organization_ids: number[];
    admin_ids: number[];
    member_ids: number[];
}

const EMPTY_FORM: ClassForm = {
    name: '',
    slug: '',
    description: '',
    is_active: true,
    access_code: '',
    organization_ids: [],
    admin_ids: [],
    member_ids: [],
};

export default function AdminClassesPage() {
    const t = useTranslations('Admin');
    const { user } = useAuth();
    // v1 ClassAdmin gates add/change/delete with Django model codenames.
    const canCreate = hasPerm(user, 'judge.add_class');
    const canEdit = hasPerm(user, 'judge.change_class');
    const canDelete = hasPerm(user, 'judge.delete_class');

    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);

    const [dialogOpen, setDialogOpen] = useState(false);
    const [editingId, setEditingId] = useState<number | null>(null);
    const [deletingId, setDeletingId] = useState<number | null>(null);
    const [form, setForm] = useState<ClassForm>(EMPTY_FORM);
    const [orgOptions, setOrgOptions] = useState<EntityOption[]>([]);
    const [adminOptions, setAdminOptions] = useState<EntityOption[]>([]);
    const [memberOptions, setMemberOptions] = useState<EntityOption[]>([]);
    const [saving, setSaving] = useState(false);

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-classes', page, search],
        queryFn: async () => (await adminClassApi.list(page, PAGE_SIZE, search)).data,
    });

    const classes = data?.data ?? [];

    const openCreate = () => {
        setForm(EMPTY_FORM);
        setOrgOptions([]);
        setAdminOptions([]);
        setMemberOptions([]);
        setEditingId(null);
        setDialogOpen(true);
    };

    const openEdit = async (id: number) => {
        try {
            const classDetail = (await adminClassApi.detail(id)).data;
            setForm({
                name: classDetail.name,
                slug: classDetail.slug,
                description: classDetail.description,
                is_active: classDetail.is_active,
                access_code: classDetail.access_code ?? '',
                organization_ids: [classDetail.organization_id],
                admin_ids: classDetail.admins.map((p) => p.id),
                member_ids: classDetail.members.map((p) => p.id),
            });
            setOrgOptions([{ id: classDetail.organization_id, label: classDetail.organization_name }]);
            setAdminOptions(classDetail.admins.map((p) => ({ id: p.id, label: p.username })));
            setMemberOptions(classDetail.members.map((p) => ({ id: p.id, label: p.username })));
            setEditingId(id);
            setDialogOpen(true);
        } catch {
            toast.error(t('classes.loadDetailFailed'));
        }
    };

    const save = async () => {
        if (!form.name.trim() || form.organization_ids.length === 0) {
            toast.error(t('classes.nameOrgRequired'));
            return;
        }
        setSaving(true);
        const payload = {
            name: form.name.trim(),
            slug: form.slug.trim(),
            description: form.description,
            is_active: form.is_active,
            access_code: form.access_code.trim() || null,
            admin_ids: form.admin_ids,
            member_ids: form.member_ids,
        };
        try {
            if (editingId === null) {
                await adminClassApi.create({ ...payload, organization_id: form.organization_ids[0] });
                toast.success(t('classes.createSuccess'));
            } else {
                await adminClassApi.update(editingId, payload);
                toast.success(t('classes.updateSuccess'));
            }
            setDialogOpen(false);
            refetch();
        } catch (error) {
            toast.error(apiErrorMessage(error, t('classes.saveError')));
        } finally {
            setSaving(false);
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await adminClassApi.delete(id);
            toast.success(t('classes.deleteSuccess'));
            setDeletingId(null);
            refetch();
        } catch (error) {
            toast.error(apiErrorMessage(error, t('classes.deleteError')));
        }
    };

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <GraduationCap className="text-primary" size={30} />
                        {t('classes.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('classes.subtitle')}</p>
                </div>
                <Button onClick={openCreate} disabled={!canCreate} title={!canCreate ? t('classes.createHint') : undefined}>
                    <Plus className="h-4 w-4 mr-2" />
                    {t('classes.addButton')}
                </Button>
            </div>

            <div className="relative w-full sm:w-80">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                    value={filters.search ?? ''}
                    onChange={(e) => setFilters({ search: e.target.value })}
                    placeholder={t('classes.searchPlaceholder')}
                    aria-label={t('classes.searchPlaceholder')}
                    className="pl-9"
                />
            </div>

            <div className="border rounded-lg overflow-hidden">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t('classes.colName')}</TableHead>
                            <TableHead>{t('classes.colOrganization')}</TableHead>
                            <TableHead>{t('classes.colActive')}</TableHead>
                            <TableHead>{t('classes.colAccessCode')}</TableHead>
                            <TableHead>{t('classes.colAdmins')}</TableHead>
                            <TableHead>{t('classes.colMembers')}</TableHead>
                            <TableHead className="text-right">{t('common.actions')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={7} className="text-center py-8">{t('common.loading')}</TableCell>
                            </TableRow>
                        ) : classes.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                                    {t('classes.empty')}
                                </TableCell>
                            </TableRow>
                        ) : classes.map((c) => (
                            <TableRow key={c.id}>
                                <TableCell>
                                    <div className="font-medium">{c.name}</div>
                                    <div className="font-mono text-xs text-muted-foreground">{c.slug}</div>
                                </TableCell>
                                <TableCell>{c.organization_name}</TableCell>
                                <TableCell>
                                    <Badge variant={c.is_active ? 'success' : 'secondary'}>
                                        {c.is_active ? t('common.yes') : t('common.no')}
                                    </Badge>
                                </TableCell>
                                <TableCell className="font-mono text-sm">{c.access_code ?? '—'}</TableCell>
                                <TableCell className="tabular-nums">{c.admin_count}</TableCell>
                                <TableCell className="tabular-nums">{c.member_count}</TableCell>
                                <TableCell className="text-right">
                                    <div className="flex justify-end gap-2">
                                        {canEdit && (
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => openEdit(c.id)}
                                                title={t('common.edit')}
                                                aria-label={t('common.edit')}
                                            >
                                                <Edit className="h-4 w-4" />
                                            </Button>
                                        )}
                                        {canDelete && (
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => setDeletingId(c.id)}
                                                className="text-destructive hover:text-destructive"
                                                title={t('common.delete')}
                                                aria-label={t('common.delete')}
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </Button>
                                        )}
                                    </div>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>

            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />

            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>{editingId === null ? t('classes.createTitle') : t('classes.editTitle')}</DialogTitle>
                        <DialogDescription>{t('classes.formDesc')}</DialogDescription>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="class-name">{t('classes.nameLabel')}</Label>
                                <Input
                                    id="class-name"
                                    value={form.name}
                                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                                />
                            </div>
                            <div className="grid gap-2">
                                <Label htmlFor="class-slug">{t('classes.slugLabel')}</Label>
                                <Input
                                    id="class-slug"
                                    value={form.slug}
                                    onChange={(e) => setForm({ ...form, slug: e.target.value })}
                                    placeholder={t('classes.slugPlaceholder')}
                                />
                            </div>
                        </div>

                        <EntityPicker
                            id="class-organization"
                            label={t('classes.organizationLabel')}
                            value={form.organization_ids}
                            onChange={(ids) => setForm({ ...form, organization_ids: ids })}
                            search={searchOrganizations}
                            queryKey="organizations"
                            initialOptions={orgOptions}
                            single
                            clearable={false}
                            disabled={editingId !== null}
                        />

                        <div className="grid grid-cols-2 gap-4">
                            <div className="grid gap-2">
                                <Label htmlFor="class-access-code">{t('classes.accessCodeLabel')}</Label>
                                <Input
                                    id="class-access-code"
                                    value={form.access_code}
                                    onChange={(e) => setForm({ ...form, access_code: e.target.value })}
                                    maxLength={7}
                                    className="font-mono"
                                />
                            </div>
                            <div className="flex items-center gap-2 pt-7">
                                <Switch
                                    id="class-active"
                                    checked={form.is_active}
                                    onCheckedChange={(checked) => setForm({ ...form, is_active: checked })}
                                />
                                <Label htmlFor="class-active">{t('classes.activeLabel')}</Label>
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="class-description">{t('classes.descriptionLabel')}</Label>
                            <Textarea
                                id="class-description"
                                value={form.description}
                                onChange={(e) => setForm({ ...form, description: e.target.value })}
                                rows={3}
                            />
                        </div>

                        <EntityPicker
                            id="class-admins"
                            label={t('classes.adminsLabel')}
                            value={form.admin_ids}
                            onChange={(ids) => setForm({ ...form, admin_ids: ids })}
                            search={searchUsers}
                            queryKey="users"
                            initialOptions={adminOptions}
                        />

                        <EntityPicker
                            id="class-members"
                            label={t('classes.membersLabel')}
                            value={form.member_ids}
                            onChange={(ids) => setForm({ ...form, member_ids: ids })}
                            search={searchUsers}
                            queryKey="users"
                            initialOptions={memberOptions}
                        />
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDialogOpen(false)}>
                            {t('common.cancel')}
                        </Button>
                        <Button onClick={save} disabled={saving}>
                            {editingId === null ? t('common.create') : t('common.saveChanges')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={deletingId !== null} onOpenChange={() => setDeletingId(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('classes.deleteTitle')}</DialogTitle>
                        <DialogDescription>{t('classes.deleteDesc')}</DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeletingId(null)}>
                            {t('common.cancel')}
                        </Button>
                        <Button variant="destructive" onClick={() => deletingId && handleDelete(deletingId)}>
                            {t('common.delete')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}