// Pure state/mapping helpers for the admin user editor. Kept free of React so
// the tab gating, self-edit guards and PATCH payload are unit-tested.
import type { AdminUserDetail, AdminUserUpdateRequest } from '@/types';

export type UserEditorTab = 'identity' | 'access' | 'profile' | 'security';

const ALL_TABS: readonly UserEditorTab[] = ['identity', 'access', 'profile', 'security'];

/**
 * Group/permission writes and the 2FA endpoint are superuser-only on the
 * backend, so those tabs are dropped for a non-superuser viewer.
 */
export function visibleUserEditorTabs(isSuperuser: boolean): UserEditorTab[] {
    return isSuperuser ? [...ALL_TABS] : ['identity', 'profile'];
}

export interface UserFlagGuard {
    is_active: boolean;
    is_staff: boolean;
    is_superuser: boolean;
    /** Why the three flag controls are locked; null when they are editable. */
    hint: 'self' | 'superuser' | null;
}

/**
 * Backend rules: nobody may deactivate or demote themselves, and only a
 * superuser may change staff/superuser/active on someone else.
 */
export function userFlagGuard(isSelf: boolean, viewerIsSuperuser: boolean): UserFlagGuard {
    if (isSelf) return { is_active: false, is_staff: false, is_superuser: false, hint: 'self' };
    if (!viewerIsSuperuser) return { is_active: false, is_staff: false, is_superuser: false, hint: 'superuser' };
    return { is_active: true, is_staff: true, is_superuser: true, hint: null };
}

/** Editable fields of the Identity and Profile tabs. */
export interface UserEditorForm {
    username: string;
    email: string;
    first_name: string;
    last_name: string;
    is_active: boolean;
    is_staff: boolean;
    is_superuser: boolean;
    username_display_override: string;
    about: string;
    notes: string;
    user_script: string;
    timezone: string;
    language_id: number;
    ace_theme: string;
    site_theme: string;
    mute: boolean;
    is_unlisted: boolean;
    display_rank: string;
    ban_reason: string;
}

export function userDetailToForm(d: AdminUserDetail): UserEditorForm {
    return {
        username: d.username,
        email: d.email,
        first_name: d.first_name,
        last_name: d.last_name,
        is_active: d.is_active,
        is_staff: d.is_staff,
        is_superuser: d.is_superuser,
        username_display_override: d.username_display_override,
        about: d.about,
        notes: d.notes,
        user_script: d.user_script,
        timezone: d.timezone,
        language_id: d.language_id ?? 0,
        ace_theme: d.ace_theme,
        site_theme: d.site_theme,
        mute: d.mute,
        is_unlisted: d.is_unlisted,
        display_rank: d.display_rank,
        ban_reason: d.ban_reason ?? '',
    };
}

/**
 * Form → PATCH body. Flags are only sent when the guard allows them, so the
 * disabled controls never trip the backend's self-demotion/403 guards.
 * `language_id` 0 means "unset", which PATCH cannot express (it validates the
 * id exists), so it is omitted rather than rejected.
 */
export function formToPatch(form: UserEditorForm, guard: UserFlagGuard): AdminUserUpdateRequest {
    const patch: AdminUserUpdateRequest = {
        username: form.username.trim(),
        email: form.email.trim(),
        first_name: form.first_name,
        last_name: form.last_name,
        username_display_override: form.username_display_override,
        about: form.about,
        notes: form.notes,
        user_script: form.user_script,
        timezone: form.timezone,
        ace_theme: form.ace_theme,
        site_theme: form.site_theme,
        mute: form.mute,
        is_unlisted: form.is_unlisted,
        display_rank: form.display_rank,
        ban_reason: form.ban_reason,
    };
    if (form.language_id > 0) patch.language_id = form.language_id;
    if (guard.is_active) patch.is_active = form.is_active;
    if (guard.is_staff) patch.is_staff = form.is_staff;
    if (guard.is_superuser) patch.is_superuser = form.is_superuser;
    return patch;
}

/** Case-insensitive search over a permission's name, codename and app label. */
export function filterPermissions<T extends { codename: string; name: string; app_label: string }>(
    permissions: T[],
    term: string
): T[] {
    const q = term.trim().toLowerCase();
    if (!q) return permissions;
    return permissions.filter(p =>
        [p.name, p.codename, p.app_label].some(v => v.toLowerCase().includes(q))
    );
}

/** Order-insensitive id-list comparison, for "is this tab dirty?". */
export function sameIds(a: number[], b: number[]): boolean {
    if (a.length !== b.length) return false;
    const set = new Set(a);
    return b.every(id => set.has(id));
}