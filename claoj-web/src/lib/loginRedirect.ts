import { routing } from '@/navigation';

/**
 * Where to send the user back to once they finish logging in.
 *
 * The path is stashed in sessionStorage rather than a `?next=` query param
 * because the navbar's login button navigates with `window.location.href`
 * (a full document load), and several callers reach /login via next-intl's
 * client router. sessionStorage survives both.
 */
const STORAGE_KEY = 'loginRedirectUrl';

/**
 * Pages that must never be used as a post-login destination.
 *
 * `/register` is the important one: the navbar renders on the register page
 * too, so clicking "Login" from there used to stash `/register` and then bounce
 * the user straight back to the sign-up form *after a successful login*. The
 * other entries are the same trap for the remaining credential flows.
 */
const NON_RETURNABLE = [
    '/login',
    '/register',
    '/resend-verification',
    '/reset-password',
    '/verify',
    '/auth',
];

/** Strip a leading locale segment, if the URL happens to carry one. */
export function stripLocalePrefix(path: string): string {
    for (const l of routing.locales) {
        if (path === `/${l}`) return '/';
        if (path.startsWith(`/${l}/`)) return path.slice(l.length + 1);
    }
    return path;
}

/** True when `path` is a real destination worth returning the user to. */
export function isReturnablePath(path: string | null | undefined): boolean {
    if (!path) return false;
    // Only same-origin absolute paths — never an attacker-supplied full URL.
    if (!path.startsWith('/') || path.startsWith('//')) return false;
    const normalized = stripLocalePrefix(path);
    return !NON_RETURNABLE.some(p => normalized === p || normalized.startsWith(`${p}/`));
}

/**
 * Remember where the user was before being sent to /login.
 * No-ops for pages we must not return to, so a stale value can never be planted.
 */
export function rememberLoginRedirect(path: string | null | undefined): void {
    if (typeof window === 'undefined') return;
    if (!isReturnablePath(path)) {
        // Actively drop a previously stored value rather than leaving a stale
        // one behind — otherwise a path remembered an hour ago still wins.
        sessionStorage.removeItem(STORAGE_KEY);
        return;
    }
    sessionStorage.setItem(STORAGE_KEY, path as string);
}

/** True when a usable destination is already stashed. */
export function hasStoredLoginRedirect(): boolean {
    if (typeof window === 'undefined') return false;
    return isReturnablePath(sessionStorage.getItem(STORAGE_KEY));
}

/** Read and clear the stored destination. Returns '/' when there isn't a usable one. */
export function consumeLoginRedirect(): string {
    if (typeof window === 'undefined') return '/';
    const stored = sessionStorage.getItem(STORAGE_KEY);
    sessionStorage.removeItem(STORAGE_KEY);
    if (!isReturnablePath(stored)) return '/';
    const path = stripLocalePrefix(stored as string);
    return path === '' ? '/' : path;
}
