import { readFileSync } from 'node:fs';

const load = (p) => JSON.parse(readFileSync(new URL(p, import.meta.url), 'utf8'));
const flatten = (obj, prefix = '') =>
    Object.entries(obj).flatMap(([k, v]) =>
        typeof v === 'object' && v !== null ? flatten(v, `${prefix}${k}.`) : [`${prefix}${k}`]);

const en = new Set(flatten(load('../src/i18n/en.json')));
const vi = new Set(flatten(load('../src/i18n/vi.json')));
const missingInEn = [...vi].filter(k => !en.has(k));
const missingInVi = [...en].filter(k => !vi.has(k));

if (missingInEn.length || missingInVi.length) {
    if (missingInEn.length) console.error(`Missing in en.json (${missingInEn.length}):\n  ` + missingInEn.join('\n  '));
    if (missingInVi.length) console.error(`Missing in vi.json (${missingInVi.length}):\n  ` + missingInVi.join('\n  '));
    process.exit(1);
}
console.log(`i18n OK: ${en.size} keys in both locales.`);
