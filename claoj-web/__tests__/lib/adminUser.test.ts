import type { AdminUserDetail } from '@/types';
import {
    visibleUserEditorTabs,
    userFlagGuard,
    userDetailToForm,
    formToPatch,
    filterPermissions,
    sameIds,
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