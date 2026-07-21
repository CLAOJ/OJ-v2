import {
    consumeLoginRedirect,
    hasStoredLoginRedirect,
    isReturnablePath,
    rememberLoginRedirect,
    stripLocalePrefix,
} from '@/lib/loginRedirect';

const KEY = 'loginRedirectUrl';

describe('loginRedirect', () => {
    beforeEach(() => {
        sessionStorage.clear();
    });

    describe('isReturnablePath', () => {
        it('accepts ordinary in-app paths', () => {
            expect(isReturnablePath('/problems')).toBe(true);
            expect(isReturnablePath('/problem/01_01')).toBe(true);
            expect(isReturnablePath('/')).toBe(true);
        });

        // The bug: the navbar renders on /register too, so clicking "Login"
        // from the sign-up page stashed /register and then bounced the user
        // back to sign-up *after* they successfully logged in.
        it('rejects the credential pages', () => {
            expect(isReturnablePath('/register')).toBe(false);
            expect(isReturnablePath('/login')).toBe(false);
            expect(isReturnablePath('/resend-verification')).toBe(false);
            expect(isReturnablePath('/reset-password')).toBe(false);
        });

        it('rejects credential sub-paths as well', () => {
            expect(isReturnablePath('/register/confirm')).toBe(false);
            expect(isReturnablePath('/auth/callback')).toBe(false);
        });

        it('rejects anything that is not a same-origin absolute path', () => {
            expect(isReturnablePath('//evil.example.com')).toBe(false);
            expect(isReturnablePath('https://evil.example.com/x')).toBe(false);
            expect(isReturnablePath('problems')).toBe(false);
            expect(isReturnablePath('')).toBe(false);
            expect(isReturnablePath(null)).toBe(false);
            expect(isReturnablePath(undefined)).toBe(false);
        });

        it('sees through a locale prefix', () => {
            expect(isReturnablePath('/vi/register')).toBe(false);
            expect(isReturnablePath('/en/register')).toBe(false);
            expect(isReturnablePath('/vi/problems')).toBe(true);
        });
    });

    describe('stripLocalePrefix', () => {
        it('removes a known locale segment', () => {
            expect(stripLocalePrefix('/vi/problems')).toBe('/problems');
            expect(stripLocalePrefix('/en/contest/abc')).toBe('/contest/abc');
        });

        it('maps a bare locale root to /', () => {
            expect(stripLocalePrefix('/vi')).toBe('/');
        });

        it('leaves unprefixed paths alone', () => {
            expect(stripLocalePrefix('/problems')).toBe('/problems');
            // "/entries" starts with "en" but is not a locale segment.
            expect(stripLocalePrefix('/entries')).toBe('/entries');
        });
    });

    describe('rememberLoginRedirect', () => {
        it('stores a returnable path', () => {
            rememberLoginRedirect('/problems');
            expect(sessionStorage.getItem(KEY)).toBe('/problems');
        });

        it('refuses to store a credential page', () => {
            rememberLoginRedirect('/register');
            expect(sessionStorage.getItem(KEY)).toBeNull();
        });

        // A path remembered an hour ago must not win over the current attempt.
        it('clears a previously stored value when the new path is not returnable', () => {
            sessionStorage.setItem(KEY, '/problems');
            rememberLoginRedirect('/register');
            expect(sessionStorage.getItem(KEY)).toBeNull();
        });
    });

    describe('consumeLoginRedirect', () => {
        it('returns and clears the stored path', () => {
            sessionStorage.setItem(KEY, '/contests');
            expect(consumeLoginRedirect()).toBe('/contests');
            expect(sessionStorage.getItem(KEY)).toBeNull();
        });

        it('falls back to / when nothing is stored', () => {
            expect(consumeLoginRedirect()).toBe('/');
        });

        // Defence in depth: even if a bad value was planted directly, the read
        // side refuses it rather than sending the user back to sign-up.
        it('falls back to / for a non-returnable stored value, and clears it', () => {
            sessionStorage.setItem(KEY, '/register');
            expect(consumeLoginRedirect()).toBe('/');
            expect(sessionStorage.getItem(KEY)).toBeNull();
        });

        it('strips the locale prefix before returning', () => {
            sessionStorage.setItem(KEY, '/vi/problems');
            expect(consumeLoginRedirect()).toBe('/problems');
        });
    });

    describe('hasStoredLoginRedirect', () => {
        it('is false when empty and true once a usable path is stored', () => {
            expect(hasStoredLoginRedirect()).toBe(false);
            rememberLoginRedirect('/problems');
            expect(hasStoredLoginRedirect()).toBe(true);
        });

        it('does not count a non-returnable stored value', () => {
            sessionStorage.setItem(KEY, '/login');
            expect(hasStoredLoginRedirect()).toBe(false);
        });

        it('does not consume the value', () => {
            rememberLoginRedirect('/problems');
            hasStoredLoginRedirect();
            expect(sessionStorage.getItem(KEY)).toBe('/problems');
        });
    });
});
