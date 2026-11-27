import { getMessages } from 'next-intl/server';

type Messages = Record<string, unknown>;

/**
 * Namespaces only the admin and settings sections need: ~70KB of the ~150KB
 * catalog. Every client provider serialises its messages into each page's RSC
 * payload, so shipping them everywhere cost ~18% of a /problems render.
 * Sections that need them ask for them via `messagesFor(locale, [...])`.
 */
export const SECTION_ONLY_NAMESPACES = ['Admin', 'admin', 'Settings'] as const;

/** Bits of section-only namespaces that the root layout renders on every page. */
export const ALWAYS_INCLUDED = ['Admin.impersonation'] as const;

function getPath(obj: Messages, path: string[]): unknown {
    return path.reduce<unknown>((o, k) => (o && typeof o === 'object' ? (o as Messages)[k] : undefined), obj);
}

// Copies each object on the way down, so the shared (cached) catalog that
// getMessages() returns is never written to.
function setPath(obj: Messages, path: string[], value: unknown): void {
    let cur = obj;
    for (const k of path.slice(0, -1)) {
        const next = cur[k];
        cur[k] = next && typeof next === 'object' ? { ...(next as Messages) } : {};
        cur = cur[k] as Messages;
    }
    cur[path[path.length - 1]] = value;
}

/**
 * The full catalog minus SECTION_ONLY_NAMESPACES, plus each `include` path
 * ('Admin' for a whole namespace, 'Admin.banner' or 'Admin.nav.tickets' for a
 * subtree or a single message). A nested NextIntlClientProvider replaces its
 * parent's messages rather than merging, so every provider uses this.
 */
export function scopeMessages(all: Messages, include: readonly string[] = []): Messages {
    const out: Messages = { ...all };
    for (const ns of SECTION_ONLY_NAMESPACES) delete out[ns];
    for (const p of [...ALWAYS_INCLUDED, ...include]) {
        const path = p.split('.');
        const value = getPath(all, path);
        if (value !== undefined) setPath(out, path, value);
    }
    return out;
}

/** Messages for a client provider: shared catalog plus the given section paths. */
export async function messagesFor(locale: string, include: readonly string[] = []): Promise<Messages> {
    return scopeMessages((await getMessages({ locale })) as Messages, include);
}
