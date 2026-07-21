'use client';

import { useRouter } from '@/navigation';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { useAuth } from '@/components/providers/AuthProvider';
import { rememberLoginRedirect } from '@/lib/loginRedirect';

/**
 * Gate an action behind being signed in.
 *
 * Most write endpoints (submit a solution, join a contest, vote on a comment)
 * sit behind the auth middleware and answer a signed-out caller with a 401.
 * Several call sites fired those requests unconditionally and registered no
 * onError, so React Query absorbed the rejection and the click did visibly
 * nothing. This gives them one place to ask "is anyone signed in?" and to send
 * the user to the login page with a path to come back to.
 *
 * Usage:
 *
 *   const { isAuthenticated, requireAuth } = useRequireAuth();
 *   <button onClick={() => requireAuth(() => submit())} />
 *
 * `requireAuth` returns true when the action ran.
 *
 * No useCallback here on purpose: this project builds with the React Compiler,
 * which memoizes automatically and rejects hand-written memoization it can't
 * verify (react-hooks/preserve-manual-memoization).
 */
export function useRequireAuth() {
    const { user, loading } = useAuth();
    const router = useRouter();
    const t = useTranslations('Auth');

    const isAuthenticated = !!user;

    const redirectToLogin = (message?: string) => {
        toast.error(message || t('loginRequired'));
        rememberLoginRedirect(window.location.pathname);
        router.push('/login');
    };

    const requireAuth = (action: () => void, message?: string): boolean => {
        // While the session is still being restored we genuinely can't tell yet,
        // and bouncing an already-signed-in user to /login would be worse than
        // waiting. Say so rather than dropping the click on the floor — a silent
        // no-op is the very failure mode this hook exists to remove.
        if (loading) {
            toast.info(t('checkingSession'));
            return false;
        }
        if (!user) {
            redirectToLogin(message);
            return false;
        }
        action();
        return true;
    };

    return { isAuthenticated, authLoading: loading, requireAuth, redirectToLogin };
}
