import type { AdminUserDetail } from '@/types';
import {
    visibleUserEditorTabs,
    userFlagGuard,
    userDetailToForm,
    formToPatch,
    filterPermissions,
    sameIds,
    isReadOnlyTarget,
    buildUserSavePlan,
} from '@/lib/adminUser';

const detail: AdminUserDetail = {
    id: 7,
    username: 'alice',
    email: 'alice@example.com',
    first_name: 'Alice',
    last_name: 'A',
    display_name: '',
    about: 'hi',
    timezone: 'UTC',
    language_id: 3,
    ace_theme: 'auto',
    site_theme: 'auto',
    notes: '',
    user_script: '',
    username_display_override: '',
    points: 10,
    performance_points: 10,
    contribution_points: 0,
    rating: null,
    problem_count: 1,
    display_rank: 'user',
    is_staff: false,
    is_super_user: false,
    is_superuser: false,
    is_active: true,
    is_unlisted: false,
    mute: false,
    is_muted: false,
    ban_reason: null,
    is_totp_enabled: false,
    is_webauthn_enabled: false,
    webauthn_credential_count: 0,
    groups: [],
    group_ids: [],
    permissions: [],
    permission_ids: [],
    organizations: [],
    last_access: '2026-01-01T00:00:00Z',
    last_login: null,
    date_joined: '2025-01-01T00:00:00Z',
};

describe('visibleUserEditorTabs', () => {
    it('shows all four tabs to a superuser', () => {
        expect(visibleUserEditorTabs(true)).toEqual(['identity', 'access', 'profile', 'security']);
    });

    it('hides the superuser-only tabs from everyone else', () => {
        expect(visibleUserEditorTabs(false)).toEqual(['identity', 'profile']);
    });
});

describe('userFlagGuard', () => {
    it('locks every status control on your own account, even as superuser', () => {
        expect(userFlagGuard(true, true)).toEqual({
            is_active: false,
            is_staff: false,
            is_superuser: false,
            hint: 'self',
        });
    });

    it('locks status controls for a non-superuser editing someone else', () => {
        expect(userFlagGuard(false, false)).toMatchObject({
            is_active: false,
            is_staff: false,
            is_superuser: false,
            hint: 'superuser',
        });
    });

    it('unlocks status controls for a superuser editing someone else', () => {
        expect(userFlagGuard(false, true)).toEqual({
            is_active: true,
            is_staff: true,
            is_superuser: true,
            hint: null,
        });
    });
});

describe('userDetailToForm', () => {
    it('maps the detail response, treating a missing language and ban reason as empty', () => {
        const form = userDetailToForm({ ...detail, language_id: null, ban_reason: null });
        expect(form.language_id).toBe(0);
        expect(form.ban_reason).toBe('');
        expect(form.username).toBe('alice');
        expect(form.mute).toBe(false);
    });

    it('coalesces nullable text fields so textareas never receive null', () => {
        const form = userDetailToForm({
            ...detail,
            first_name: null as unknown as string,
            last_name: null as unknown as string,
            about: null as unknown as string,
            notes: null as unknown as string,
            user_script: null as unknown as string,
            username_display_override: null as unknown as string,
            timezone: null as unknown as string,
            ace_theme: null as unknown as string,
            site_theme: null as unknown as string,
            display_rank: null as unknown as string,
        });
        for (const value of Object.values(form)) {
            if (typeof value === 'string') expect(value).not.toBeNull();
        }
        expect(form.about).toBe('');
        expect(form.notes).toBe('');
    });
});

describe('formToPatch', () => {
    const form = userDetailToForm(detail);

    it('omits the flag fields the guard locked', () => {
        const patch = formToPatch(
            { ...form, is_active: false, is_staff: true, is_superuser: true },
            userFlagGuard(true, true)
        );
        expect(patch).not.toHaveProperty('is_active');
        expect(patch).not.toHaveProperty('is_staff');
        expect(patch).not.toHaveProperty('is_superuser');
    });

    it('sends the flag fields when the guard allows them', () => {
        const patch = formToPatch({ ...form, is_active: false }, userFlagGuard(false, true));
        expect(patch.is_active).toBe(false);
        expect(patch.is_staff).toBe(false);
        expect(patch.is_superuser).toBe(false);
    });

    it('trims identity fields and omits an unset language', () => {
        const patch = formToPatch(
            { ...form, username: '  bob ', email: ' b@x.io ', language_id: 0 },
            userFlagGuard(false, true)
        );
        expect(patch.username).toBe('bob');
        expect(patch.email).toBe('b@x.io');
        expect(patch).not.toHaveProperty('language_id');
    });

    it('sends a set language id', () => {
        expect(formToPatch(form, userFlagGuard(false, true)).language_id).toBe(3);
    });
});

describe('filterPermissions', () => {
    const permissions = [
        { id: 1, codename: 'change_profile', name: 'Can change profile', app_label: 'judge' },
        { id: 2, codename: 'add_contest', name: 'Can add contest', app_label: 'contest' },
    ];

    it('returns everything for a blank term', () => {
        expect(filterPermissions(permissions, '   ')).toHaveLength(2);
    });

    it('matches name, codename or app label, case-insensitively', () => {
        expect(filterPermissions(permissions, 'PROFILE').map(p => p.id)).toEqual([1]);
        expect(filterPermissions(permissions, 'add_contest').map(p => p.id)).toEqual([2]);
        expect(filterPermissions(permissions, 'contest').map(p => p.id)).toEqual([2]);
    });
});

describe('sameIds', () => {
    it('ignores order and rejects different sets', () => {
        expect(sameIds([1, 2], [2, 1])).toBe(true);
        expect(sameIds([1, 2], [1])).toBe(false);
        expect(sameIds([1, 2], [1, 3])).toBe(false);
    });
});

describe('buildUserSavePlan', () => {
    const guard = userFlagGuard(false, true);

    it('sends the PATCH and both PUTs when every tab changed', () => {
        const plan = buildUserSavePlan(
            detail,
            { ...userDetailToForm(detail), first_name: 'Alicia' },
            guard,
            [1, 2],
            [9]
        );
        expect(plan.patch).toMatchObject({ first_name: 'Alicia' });
        expect(plan.groupIds).toEqual([1, 2]);
        expect(plan.permissionIds).toEqual([9]);
    });

    it('skips the PATCH when only groups/permissions changed', () => {
        const plan = buildUserSavePlan(detail, userDetailToForm(detail), guard, [1], [9]);
        expect(plan.patch).toBeNull();
        expect(plan.groupIds).toEqual([1]);
        expect(plan.permissionIds).toEqual([9]);
    });

    it('skips the PUTs when only identity/profile changed', () => {
        const plan = buildUserSavePlan(
            detail,
            { ...userDetailToForm(detail), about: 'new bio' },
            guard,
            [...detail.group_ids],
            [...detail.permission_ids]
        );
        expect(plan.patch).toMatchObject({ about: 'new bio' });
        expect(plan.groupIds).toBeNull();
        expect(plan.permissionIds).toBeNull();
    });

    it('returns an empty plan when nothing changed', () => {
        const plan = buildUserSavePlan(
            detail,
            userDetailToForm(detail),
            guard,
            [...detail.group_ids],
            [...detail.permission_ids]
        );
        expect(plan).toEqual({ patch: null, groupIds: null, permissionIds: null });
    });
});

describe('isReadOnlyTarget', () => {
    const staffTarget = { username: 'mod', is_staff: true, is_superuser: false };

    it('locks another staff/superuser account for a non-superuser', () => {
        expect(isReadOnlyTarget(staffTarget, { username: 'alice', isSuperuser: false })).toBe(true);
        expect(isReadOnlyTarget({ ...staffTarget, is_staff: false, is_superuser: true }, { isSuperuser: false })).toBe(true);
    });

    it('allows a non-superuser to edit a regular account or their own', () => {
        expect(isReadOnlyTarget({ username: 'bob', is_staff: false, is_superuser: false }, { username: 'alice', isSuperuser: false })).toBe(false);
        expect(isReadOnlyTarget(staffTarget, { username: 'mod', isSuperuser: false })).toBe(false);
    });

    it('never locks anything for a superuser', () => {
        expect(isReadOnlyTarget(staffTarget, { username: 'root', isSuperuser: true })).toBe(false);
    });
});
