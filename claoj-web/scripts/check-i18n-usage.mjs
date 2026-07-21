// Verifies that every key the code passes to a next-intl translation function
// actually exists, as a string, in both locales.
//
// check-i18n.mjs compares en.json against vi.json — it can only catch a key
// present in one locale and absent from the other. It cannot see a `t('foo')`
// whose key was never added anywhere, or a namespace typo (`t('common.x')`
// where the namespace is really `Common`). Both shipped in this codebase and
// rendered as raw key text to users.
import { readFileSync, readdirSync } from 'node:fs';
import { join, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
const load = (p) => JSON.parse(readFileSync(join(root, p), 'utf8'));

const en = load('src/i18n/en.json');
const vi = load('src/i18n/vi.json');
const lookup = (tree, dotted) =>
    dotted.split('.').reduce((node, part) => (node == null ? node : node[part]), tree);

const files = [];
(function walk(dir) {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
        const full = join(dir, entry.name);
        if (entry.isDirectory()) walk(full);
        else if (/\.(tsx|ts)$/.test(entry.name)) files.push(full);
    }
})(join(root, 'src'));

let checked = 0;
const missing = [];
const notAString = [];

for (const file of files) {
    const src = readFileSync(file, 'utf8');
    const rel = file.slice(root.length).split(sep).join('/');

    // Map each translation-function binding to the namespace it was created
    // with: `const tAuth = useTranslations('Auth')` -> tAuth resolves 'Auth.*'.
    // A namespace-less useTranslations() means its keys are already absolute.
    const bindings = new Map();
    const decl = /const\s+(\w+)\s*=\s*(?:await\s+)?(?:useTranslations|getTranslations)\(\s*(?:['"`]([^'"`]+)['"`])?\s*\)/g;
    for (const m of src.matchAll(decl)) bindings.set(m[1], m[2] ?? '');
    if (bindings.size === 0) continue;

    for (const [fn, namespace] of bindings) {
        const call = new RegExp(`\\b${fn}(?:\\.rich|\\.raw)?\\(\\s*['"\`]([A-Za-z0-9_.-]+)['"\`]`, 'g');
        for (const m of src.matchAll(call)) {
            const key = namespace ? `${namespace}.${m[1]}` : m[1];
            checked++;
            const e = lookup(en, key);
            const v = lookup(vi, key);
            if (e === undefined || v === undefined) {
                missing.push({ key, rel, en: e !== undefined, vi: v !== undefined });
            } else if (typeof e !== 'string' || typeof v !== 'string') {
                notAString.push({ key, rel });
            }
        }
    }
}

if (missing.length || notAString.length) {
    for (const m of missing) {
        console.error(`missing key: ${m.key}  (en:${m.en ? 'ok' : 'MISSING'} vi:${m.vi ? 'ok' : 'MISSING'})  <- ${m.rel}`);
    }
    for (const m of notAString) {
        console.error(`key resolves to an object, not a string: ${m.key}  <- ${m.rel}`);
    }
    process.exit(1);
}

console.log(`i18n usage OK: ${checked} t() references all resolve in both locales.`);
