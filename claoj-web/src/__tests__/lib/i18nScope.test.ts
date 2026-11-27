import fs from 'fs';
import path from 'path';
import { scopeMessages, SECTION_ONLY_NAMESPACES } from '@/i18n/scope';

jest.mock('next-intl/server', () => ({ getMessages: jest.fn() }));

const all = {
    Common: { save: 'Save' },
    Admin: { impersonation: { exit: 'Exit' }, banner: { label: 'Staff' }, nav: { tickets: 'Tickets', users: 'Users' }, big: 'x' },
    admin: { a: 'a' },
    Settings: { s: 's' },
};

test('drops section-only namespaces but keeps the always-included banner', () => {
    expect(scopeMessages(all)).toEqual({ Common: { save: 'Save' }, Admin: { impersonation: { exit: 'Exit' } } });
});

test('includes whole namespaces, subtrees and single messages without mutating the catalog', () => {
    const before = JSON.stringify(all);
    expect(scopeMessages(all, ['Admin.banner', 'Admin.nav.tickets', 'Settings'])).toEqual({
        Common: { save: 'Save' },
        Admin: { impersonation: { exit: 'Exit' }, banner: { label: 'Staff' }, nav: { tickets: 'Tickets' } },
        Settings: { s: 's' },
    });
    expect(scopeMessages(all, ['Admin', 'Admin.banner']).Admin).toEqual(all.Admin);
    expect(JSON.stringify(all)).toBe(before);
});

// Guard: a component outside /admin and /settings that starts using a
// section-only namespace would render raw keys, because its page's provider
// no longer ships that namespace. Wire the page's scope, then add it here.
const ALLOWED_OUTSIDE_SECTIONS: Record<string, string> = {
    'components/admin/ImpersonationBanner.tsx': 'root layout (ALWAYS_INCLUDED)',
    'components/admin/AdminWelcomeBanner.tsx': 'app/[locale]/page.tsx',
    'components/problems/SuggestProblemForm.tsx': 'problems/suggest/new/page.tsx',
    'components/admin/problem-form/BasicInfoSection.tsx': 'problems/suggest/new + organization/[id]/problem/new',
    'components/contests/ClarificationList.tsx': 'only imported by admin/contests/[key]/clarifications',
};
// Folders whose layout provides the full section namespaces.
const SCOPED_DIRS = [
    'app/[locale]/admin/',
    'app/[locale]/settings/',
    'components/settings/',
    'app/[locale]/organization/[id]/problem/new/',
    'app/[locale]/organization/[id]/contest/new/',
    'app/[locale]/organization/[id]/blog/new/',
];
// Admin-only components: imported only from SCOPED_DIRS pages (checked below).
const ADMIN_COMPONENTS = 'components/admin/';

test('section-only namespaces are only used where a provider ships them', () => {
    const src = path.join(__dirname, '..', '..');
    const files: string[] = [];
    const walk = (d: string) => fs.readdirSync(d, { withFileTypes: true }).forEach(e => {
        const p = path.join(d, e.name);
        if (e.isDirectory()) { if (e.name !== '__tests__') walk(p); } else if (/\.tsx?$/.test(e.name)) files.push(p);
    });
    walk(src);
    const rel = (f: string) => path.relative(src, f).split(path.sep).join('/');
    const ns = SECTION_ONLY_NAMESPACES.join('|');
    const uses = new RegExp(`useTranslations\\(['"](${ns})[.'"]`);
    const usesSectionNs = new Set(files.map(rel).filter(f => uses.test(fs.readFileSync(path.join(src, f), 'utf8'))));
    const scoped = (f: string) => SCOPED_DIRS.some(d => f.startsWith(d));

    // 1. Direct use outside the scoped folders and admin components.
    const offenders = [...usesSectionNs].filter(f => !scoped(f) && !ALLOWED_OUTSIDE_SECTIONS[f] && !f.startsWith(ADMIN_COMPONENTS));
    expect(offenders).toEqual([]);

    // 2. An unscoped file importing an admin component that uses those namespaces.
    // ponytail: one import level only; a page pulling in a form that pulls in such a
    // section is caught via the section itself (listed above) or needs a scoped folder.
    // `import type` renders nothing, so it cannot need translations.
    const importsAdmin = /^import (?!type\b)[^;]*?from ['"]@\/components\/admin\/([\w/-]+)['"]/gm;
    const badImports = files.map(rel).filter(f => !scoped(f) && !f.startsWith(ADMIN_COMPONENTS)).flatMap(f =>
        [...fs.readFileSync(path.join(src, f), 'utf8').matchAll(importsAdmin)]
            .map(m => `components/admin/${m[1]}.tsx`)
            .filter(c => usesSectionNs.has(c) && !ALLOWED_OUTSIDE_SECTIONS[c])
            .map(c => `${f} -> ${c}`));
    expect(badImports).toEqual([]);
});
