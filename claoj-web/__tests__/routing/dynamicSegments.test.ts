import fs from 'node:fs';
import path from 'node:path';

// Next.js refuses to serve ANY page when two dynamic segments under the same
// parent use different names ("You cannot use different slug names for the
// same dynamic path"). `next build` does not catch it, so a /post/[id]/edit
// next to /post/[slug] took the whole site down in production. Guard it here.
const APP_DIR = path.join(__dirname, '..', '..', 'src', 'app', '[locale]');

function directories(dir: string, out: string[] = []): string[] {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        if (!entry.isDirectory()) continue;
        const full = path.join(dir, entry.name);
        out.push(full);
        directories(full, out);
    }
    return out;
}

describe('app route dynamic segments', () => {
    it('uses one dynamic name per parent path', () => {
        const byParent = new Map<string, Set<string>>();
        for (const dir of directories(APP_DIR)) {
            const name = path.basename(dir);
            if (!name.startsWith('[')) continue;
            const parent = path.dirname(dir);
            if (!byParent.has(parent)) byParent.set(parent, new Set());
            byParent.get(parent)!.add(name);
        }
        const conflicts = [...byParent.entries()]
            .filter(([, names]) => names.size > 1)
            .map(([parent, names]) => `${parent}: ${[...names].join(', ')}`);
        expect(conflicts).toEqual([]);
    });
});