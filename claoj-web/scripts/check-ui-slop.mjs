import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

// Decorative patterns banned by docs/superpowers/specs/2026-10-09-v2-ui-deslop-design.md.
// A line containing `ui-check-allow` is skipped.
const RULES = [
    ['font-black', /\bfont-black\b/],
    ['tiny-text', /\btext-\[(8|9|10)px\]/],
    ['wide-tracking', /\btracking-(widest|\[0\.\d+em\])/],
    ['big-radius', /\brounded-(2xl|3xl|\[[\d.]+rem\])/],
    ['hover-motion', /\b(hover|active):-?(scale|translate)-/],
    ['gradient', /\bbg-(gradient|linear|radial|conic)-/],
    ['blur', /\bbackdrop-blur/],
    ['heavy-shadow', /\bshadow-(2xl|[a-z]+(-\d+)?\/\d+)/],
    ['page-fade', /\banimate-in\b/],
];

const walk = (p) => statSync(p).isDirectory()
    ? readdirSync(p).flatMap((f) => walk(join(p, f)))
    : /\.(tsx|ts)$/.test(p) ? [p] : [];

const targets = process.argv.slice(2);
const files = (targets.length ? targets : ['src']).flatMap(walk);
const counts = {};
let hits = 0;

for (const file of files) {
    readFileSync(file, 'utf8').split('\n').forEach((line, i) => {
        if (line.includes('ui-check-allow')) return;
        for (const [name, re] of RULES) {
            if (!re.test(line)) continue;
            console.error(`${relative('.', file)}:${i + 1}  [${name}]  ${line.trim().slice(0, 120)}`);
            counts[name] = (counts[name] || 0) + 1;
            hits++;
        }
    });
}

if (hits) {
    console.error(`\nui:check FAILED: ${hits} hits in ${files.length} files`, counts);
    process.exit(1);
}
console.log(`ui:check OK: ${files.length} files clean.`);
