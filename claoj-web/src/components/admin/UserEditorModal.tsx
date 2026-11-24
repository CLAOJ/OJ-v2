'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
    adminUserApi,
    adminGroupsApi,
    adminLanguageApi,
    adminErrorMessage,
} from '@/lib/adminApi';
import {
    userDetailToForm,
    userFlagGuard,
    visibleUserEditorTabs,
    filterPermissions,
    isReadOnlyTarget,
    buildUserSavePlan,
    type UserEditorForm,
    type UserEditorTab,
} from '@/lib/adminUser';
import type { AdminUserDetail } from '@/types';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Select } from '@/components/ui/Select';
import { Skeleton } from '@/components/ui/Skeleton';
import { Switch } from '@/components/ui/Switch';
import { Textarea } from '@/components/ui/Textarea';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/Dialog';
import { cn } from '@/lib/utils';
import { Search, ShieldCheck, ShieldOff } from 'lucide-react';

interface UserEditorModalProps {
    userId: number;
    viewer: { username?: string; isSuperuser: boolean };
    onClose: () => void;
}

export function UserEditorModal({ userId, viewer, onClose }: UserEditorModalProps) {
    const t = useTranslations('Admin');
    const { data: detail, isLoading, isError } = useQuery({
        queryKey: ['admin-user-detail', userId],
        queryFn: async () => (await adminUserApi.detail(userId)).data,
    });

    return (
        <Dialog open onOpenChange={(open) => { if (!open) onClose(); }} className="max-w-3xl">
            <DialogHeader>
                <DialogTitle id="user-editor-title">
                    {t('users.editor.title', { username: detail?.username ?? '' })}
                </DialogTitle>
            </DialogHeader>
            {isLoading ? (
                <Skeleton className="h-96 rounded-md" />
            ) : isError || !detail ? (
                <p role="alert" className="text-sm text-destructive py-6">{t('users.saveError')}</p>
            ) : (
                <UserEditorBody key={detail.id} detail={detail} viewer={viewer} onClose={onClose} />
            )}
        </Dialog>
    );
}

interface UserEditorBodyProps {
    detail: AdminUserDetail;
    viewer: { username?: string; isSuperuser: boolean };
    onClose: () => void;
}

function UserEditorBody({ detail, viewer, onClose }: UserEditorBodyProps) {
    const t = useTranslations('Admin');
    const queryClient = useQueryClient();
    const [tab, setTab] = useState<UserEditorTab>('identity');
    const [form, setForm] = useState<UserEditorForm>(() => userDetailToForm(detail));
    const [groupIds, setGroupIds] = useState<number[]>(detail.group_ids);
    const [permissionIds, setPermissionIds] = useState<number[]>(detail.permission_ids);
    const [permissionSearch, setPermissionSearch] = useState('');

    const isSelf = !!viewer.username && viewer.username === detail.username;
    const guard = userFlagGuard(isSelf, viewer.isSuperuser);
    const readOnly = isReadOnlyTarget(detail, viewer);
    const tabs = visibleUserEditorTabs(viewer.isSuperuser);
    const activeTab = tabs.includes(tab) ? tab : 'identity';

    const { data: groups = [] } = useQuery({
        queryKey: ['admin-groups'],
        queryFn: async () => (await adminGroupsApi.list()).data.data,
        enabled: viewer.isSuperuser,
    });
    const { data: permissions = [] } = useQuery({
        queryKey: ['admin-permissions'],
        queryFn: async () => (await adminGroupsApi.permissions()).data.data,
        enabled: viewer.isSuperuser,
    });
    const { data: languages = [] } = useQuery({
        queryKey: ['admin-languages'],
        queryFn: async () => (await adminLanguageApi.list(1, 100)).data.data,
    });

    const saveMutation = useMutation({
        mutationFn: async () => {
            const plan = buildUserSavePlan(detail, form, guard, groupIds, permissionIds);
            await Promise.all([
                plan.patch ? adminUserApi.update(detail.id, plan.patch) : Promise.resolve(),
                plan.groupIds ? adminUserApi.replaceGroups(detail.id, plan.groupIds) : Promise.resolve(),
                plan.permissionIds ? adminUserApi.replacePermissions(detail.id, plan.permissionIds) : Promise.resolve(),
            ]);
        },
        onSuccess: () => {
            toast.success(t('users.editor.saved'));
            queryClient.invalidateQueries({ queryKey: ['admin-users'] });
            queryClient.invalidateQueries({ queryKey: ['admin-user-detail', detail.id] });
            onClose();
        },
        onError: (error) => {
            // A parallel write may have landed; refresh the detail either way.
            queryClient.invalidateQueries({ queryKey: ['admin-users'] });
            queryClient.invalidateQueries({ queryKey: ['admin-user-detail', detail.id] });
            toast.error(adminErrorMessage(error, t('users.saveError')));
        },
    });

    const securityMutation = useMutation({
        mutationFn: (flags: { totp_enabled?: false; webauthn_enabled?: false }) =>
            adminUserApi.setSecurity(detail.id, flags),
        onSuccess: () => {
            toast.success(t('users.editor.saved'));
            queryClient.invalidateQueries({ queryKey: ['admin-user-detail', detail.id] });
        },
        onError: (error) => toast.error(adminErrorMessage(error, t('users.saveError'))),
    });

    const toggleGroup = (id: number) =>
        setGroupIds(prev => (prev.includes(id) ? prev.filter(g => g !== id) : [...prev, id]));
    const togglePermission = (id: number) =>
        setPermissionIds(prev => (prev.includes(id) ? prev.filter(p => p !== id) : [...prev, id]));

    const tabItems: { id: UserEditorTab; label: string }[] = [
        { id: 'identity', label: t('users.editor.tabIdentity') },
        { id: 'access', label: t('users.editor.tabAccess') },
        { id: 'profile', label: t('users.editor.tabProfile') },
        { id: 'security', label: t('users.editor.tabSecurity') },
    ];
    const tabLabels = tabItems.filter(item => tabs.includes(item.id));

    return (
        <>
            <div role="tablist" className="flex gap-1 border-b overflow-x-auto">
                {tabLabels.map(item => (
                    <button
                        key={item.id}
                        id={`user-tab-${item.id}`}
                        role="tab"
                        aria-selected={activeTab === item.id}
                        aria-controls={`user-panel-${item.id}`}
                        onClick={() => setTab(item.id)}
                        className={cn(
                            '-mb-px border-b-2 px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors',
                            activeTab === item.id
                                ? 'border-primary text-primary'
                                : 'border-transparent text-muted-foreground hover:text-foreground'
                        )}
                    >
                        {item.label}
                    </button>
                ))}
            </div>

            {readOnly && (
                <p role="status" className="mt-4 rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-600 dark:text-amber-400">
                    {t('users.editor.readOnlyNotice')}
                </p>
            )}

            <DialogContent className="min-h-[20rem]">
                <fieldset disabled={readOnly} className="min-w-0">
                    {activeTab === 'identity' && (
                        <div role="tabpanel" id="user-panel-identity" aria-labelledby="user-tab-identity" className="space-y-4">
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <Field id="user-username" label={t('users.usernameLabel')}>
                                    <Input
                                        id="user-username"
                                        value={form.username}
                                        onChange={(e) => setForm({ ...form, username: e.target.value })}
                                    />
                                </Field>
                                <Field id="user-email" label={t('users.emailLabel')}>
                                    <Input
                                        id="user-email"
                                        type="email"
                                        value={form.email}
                                        onChange={(e) => setForm({ ...form, email: e.target.value })}
                                    />
                                </Field>
                                <Field id="user-first-name" label={t('users.editor.firstNameLabel')}>
                                    <Input
                                        id="user-first-name"
                                        value={form.first_name}
                                        onChange={(e) => setForm({ ...form, first_name: e.target.value })}
                                    />
                                </Field>
                                <Field id="user-last-name" label={t('users.editor.lastNameLabel')}>
                                    <Input
                                        id="user-last-name"
                                        value={form.last_name}
                                        onChange={(e) => setForm({ ...form, last_name: e.target.value })}
                                    />
                                </Field>
                            </div>

                            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                                <FlagSwitch
                                    id="user-active"
                                    label={t('users.activeLabel')}
                                    checked={form.is_active}
                                    disabled={!guard.is_active}
                                    onChange={(v) => setForm({ ...form, is_active: v })}
                                />
                                <FlagSwitch
                                    id="user-staff"
                                    label={t('users.staff')}
                                    checked={form.is_staff}
                                    disabled={!guard.is_staff}
                                    onChange={(v) => setForm({ ...form, is_staff: v })}
                                />
                                <FlagSwitch
                                    id="user-superuser"
                                    label={t('users.superuser')}
                                    checked={form.is_superuser}
                                    disabled={!guard.is_superuser}
                                    onChange={(v) => setForm({ ...form, is_superuser: v })}
                                />
                            </div>
                            {!readOnly && guard.hint === 'self' && (
                                <p className="text-xs text-muted-foreground">{t('users.editor.selfGuardHint')}</p>
                            )}
                            {!readOnly && guard.hint === 'superuser' && (
                                <p className="text-xs text-muted-foreground">{t('users.editor.superuserGuardHint')}</p>
                            )}
                        </div>
                    )}

                    {activeTab === 'access' && (
                        <div role="tabpanel" id="user-panel-access" aria-labelledby="user-tab-access" className="space-y-6">
                            <section className="space-y-2">
                                <h3 className="text-sm font-semibold">{t('users.editor.groupsLabel')}</h3>
                                {groups.length === 0 ? (
                                    <p className="text-sm text-muted-foreground">{t('users.editor.noGroups')}</p>
                                ) : (
                                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-56 overflow-y-auto">
                                        {groups.map(group => (
                                            <label
                                                key={group.id}
                                                className={cn(
                                                    'flex items-center gap-2 p-2 rounded-md border text-sm cursor-pointer transition-colors',
                                                    groupIds.includes(group.id) ? 'bg-primary/10 border-primary/30' : 'hover:bg-muted/30'
                                                )}
                                            >
                                                <input
                                                    type="checkbox"
                                                    className="rounded w-4 h-4"
                                                    checked={groupIds.includes(group.id)}
                                                    onChange={() => toggleGroup(group.id)}
                                                />
                                                <span>{group.name}</span>
                                            </label>
                                        ))}
                                    </div>
                                )}
                            </section>

                            <section className="space-y-2">
                                <h3 className="text-sm font-semibold">
                                    {t('users.editor.permissionsLabel', { count: permissionIds.length })}
                                </h3>
                                <div className="relative">
                                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} />
                                    <Input
                                        aria-label={t('users.editor.permissionSearchPlaceholder')}
                                        placeholder={t('users.editor.permissionSearchPlaceholder')}
                                        className="pl-9"
                                        value={permissionSearch}
                                        onChange={(e) => setPermissionSearch(e.target.value)}
                                    />
                                </div>
                                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-56 overflow-y-auto">
                                    {filterPermissions(permissions, permissionSearch).map(perm => (
                                        <label
                                            key={perm.id}
                                            className={cn(
                                                'flex items-center gap-2 p-2 rounded-md border text-sm cursor-pointer transition-colors',
                                                permissionIds.includes(perm.id) ? 'bg-primary/10 border-primary/30' : 'hover:bg-muted/30'
                                            )}
                                        >
                                            <input
                                                type="checkbox"
                                                className="rounded w-4 h-4"
                                                checked={permissionIds.includes(perm.id)}
                                                onChange={() => togglePermission(perm.id)}
                                            />
                                            <span className="truncate" title={perm.codename}>{perm.name}</span>
                                            <span className="ml-auto text-[10px] text-muted-foreground shrink-0">{perm.app_label}</span>
                                        </label>
                                    ))}
                                </div>
                            </section>
                        </div>
                    )}

                    {activeTab === 'profile' && (
                        <div role="tabpanel" id="user-panel-profile" aria-labelledby="user-tab-profile" className="space-y-4">
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <Field id="user-display-name" label={t('users.editor.displayNameLabel')}>
                                    <Input
                                        id="user-display-name"
                                        value={form.username_display_override}
                                        onChange={(e) => setForm({ ...form, username_display_override: e.target.value })}
                                    />
                                </Field>
                                <Field id="user-timezone" label={t('users.editor.timezoneLabel')}>
                                    <Input
                                        id="user-timezone"
                                        placeholder={t('users.editor.timezonePlaceholder')}
                                        value={form.timezone}
                                        onChange={(e) => setForm({ ...form, timezone: e.target.value })}
                                    />
                                </Field>
                                <Field id="user-language" label={t('users.editor.languageLabel')}>
                                    <Select
                                        id="user-language"
                                        value={form.language_id || ''}
                                        onChange={(e) => setForm({ ...form, language_id: Number(e.target.value) || 0 })}
                                    >
                                        <option value="">{t('users.editor.languageUnset')}</option>
                                        {languages.map(lang => (
                                            <option key={lang.id} value={lang.id}>{lang.name}</option>
                                        ))}
                                    </Select>
                                </Field>
                                <Field id="user-display-rank" label={t('users.editor.displayRankLabel')}>
                                    <Input
                                        id="user-display-rank"
                                        value={form.display_rank}
                                        onChange={(e) => setForm({ ...form, display_rank: e.target.value })}
                                    />
                                </Field>
                                <Field id="user-ace-theme" label={t('users.editor.aceThemeLabel')}>
                                    <Input
                                        id="user-ace-theme"
                                        value={form.ace_theme}
                                        onChange={(e) => setForm({ ...form, ace_theme: e.target.value })}
                                    />
                                </Field>
                                <Field id="user-site-theme" label={t('users.editor.siteThemeLabel')}>
                                    <Input
                                        id="user-site-theme"
                                        value={form.site_theme}
                                        onChange={(e) => setForm({ ...form, site_theme: e.target.value })}
                                    />
                                </Field>
                            </div>

                            <Field id="user-about" label={t('users.editor.aboutLabel')}>
                                <Textarea
                                    id="user-about"
                                    rows={3}
                                    value={form.about}
                                    onChange={(e) => setForm({ ...form, about: e.target.value })}
                                />
                            </Field>
                            <Field id="user-notes" label={t('users.editor.notesLabel')}>
                                <Textarea
                                    id="user-notes"
                                    rows={2}
                                    value={form.notes}
                                    onChange={(e) => setForm({ ...form, notes: e.target.value })}
                                />
                            </Field>
                            <Field id="user-user-script" label={t('users.editor.userScriptLabel')}>
                                <Textarea
                                    id="user-user-script"
                                    rows={3}
                                    value={form.user_script}
                                    onChange={(e) => setForm({ ...form, user_script: e.target.value })}
                                />
                            </Field>
                            <Field id="user-ban-reason" label={t('users.banReasonLabel')}>
                                <Input
                                    id="user-ban-reason"
                                    value={form.ban_reason}
                                    onChange={(e) => setForm({ ...form, ban_reason: e.target.value })}
                                />
                            </Field>

                            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                                <FlagSwitch
                                    id="user-mute"
                                    label={t('users.editor.muteLabel')}
                                    checked={form.mute}
                                    onChange={(v) => setForm({ ...form, mute: v })}
                                />
                                <FlagSwitch
                                    id="user-unlisted"
                                    label={t('users.hidden')}
                                    checked={form.is_unlisted}
                                    onChange={(v) => setForm({ ...form, is_unlisted: v })}
                                />
                            </div>
                        </div>
                    )}

                    {activeTab === 'security' && (
                        <div role="tabpanel" id="user-panel-security" aria-labelledby="user-tab-security" className="space-y-4">
                            <SecurityRow
                                title={t('users.editor.totpLabel')}
                                enabled={detail.is_totp_enabled}
                                enabledText={t('users.editor.enabled')}
                                disabledText={t('users.editor.disabled')}
                                actionLabel={t('users.editor.disable2fa')}
                                busy={securityMutation.isPending}
                                onDisable={() => securityMutation.mutate({ totp_enabled: false })}
                            />
                            <SecurityRow
                                title={t('users.editor.webauthnLabel', { count: detail.webauthn_credential_count })}
                                enabled={detail.is_webauthn_enabled}
                                enabledText={t('users.editor.enabled')}
                                disabledText={t('users.editor.disabled')}
                                actionLabel={t('users.editor.disable2fa')}
                                busy={securityMutation.isPending}
                                onDisable={() => securityMutation.mutate({ webauthn_enabled: false })}
                            />
                            <p className="text-xs text-muted-foreground">{t('users.editor.securityHint')}</p>
                        </div>
                    )}
                </fieldset>
            </DialogContent>

            <DialogFooter>
                <Button variant="outline" onClick={onClose} disabled={saveMutation.isPending}>
                    {activeTab === 'security' ? t('common.close') : t('common.cancel')}
                </Button>
                {activeTab !== 'security' && (
                    <Button
                        onClick={() => saveMutation.mutate()}
                        loading={saveMutation.isPending}
                        disabled={readOnly}
                    >
                        {t('common.saveChanges')}
                    </Button>
                )}
            </DialogFooter>
        </>
    );
}

function Field({ id, label, children }: { id: string; label: string; children: React.ReactNode }) {
    return (
        <div className="space-y-1.5">
            <Label htmlFor={id}>{label}</Label>
            {children}
        </div>
    );
}

function FlagSwitch({ id, label, checked, disabled, onChange }: {
    id: string;
    label: string;
    checked: boolean;
    disabled?: boolean;
    onChange: (value: boolean) => void;
}) {
    return (
        <div className="flex items-center justify-between gap-3 rounded-md border px-3 py-2">
            <Label htmlFor={id} className={cn(disabled && 'text-muted-foreground')}>{label}</Label>
            <Switch id={id} checked={checked} disabled={disabled} onCheckedChange={onChange} />
        </div>
    );
}

function SecurityRow({ title, enabled, enabledText, disabledText, actionLabel, busy, onDisable }: {
    title: string;
    enabled: boolean;
    enabledText: string;
    disabledText: string;
    actionLabel: string;
    busy: boolean;
    onDisable: () => void;
}) {
    return (
        <div className="flex items-center justify-between gap-3 rounded-md border p-3">
            <div className="flex items-center gap-2 text-sm">
                {enabled ? <ShieldCheck size={16} className="text-emerald-500" /> : <ShieldOff size={16} className="text-muted-foreground" />}
                <span className="font-medium">{title}</span>
                <Badge variant={enabled ? 'success' : 'secondary'} className="text-[10px] py-0 px-1.5">
                    {enabled ? enabledText : disabledText}
                </Badge>
            </div>
            {enabled && (
                <Button variant="destructive" size="sm" disabled={busy} onClick={onDisable}>
                    {actionLabel}
                </Button>
            )}
        </div>
    );
}
