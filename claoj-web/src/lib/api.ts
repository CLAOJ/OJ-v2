import axios, { InternalAxiosRequestConfig } from 'axios';

// Extend Axios request config to include _retry and _skipAuthRedirect properties
interface AxiosRequestConfigWithRetry extends InternalAxiosRequestConfig {
    _retry?: boolean;
    _skipAuthRedirect?: boolean;
}

function getApiBaseUrl(): string {
    if (typeof window === 'undefined') {
        // Server-side (SSR metadata fetchers): reach the backend over the
        // internal network. Read at runtime — NOT inlined at build — so one
        // image works in every environment.
        return process.env.API_URL_INTERNAL || 'http://localhost:8081/api';
    }
    return `${window.location.origin}/api`;
}

const api = axios.create({
    baseURL: process.env.NEXT_PUBLIC_API_URL || getApiBaseUrl(),
    headers: {
        'Content-Type': 'application/json',
    },
    withCredentials: true, // Enable sending cookies with requests
});

// Get API URL - uses env var if set, otherwise derives from window.location.origin
export function getApiUrl(): string {
    return api.defaults.baseURL as string;
}

function getCookie(name: string): string | null {
    if (typeof document === 'undefined') return null;
    const match = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
    return match ? decodeURIComponent(match[1]) : null;
}

// Interceptor to add access token from httpOnly cookie
// NOTE: Tokens are stored in httpOnly cookies only - never in localStorage for security
api.interceptors.request.use((config) => {
    // Token is automatically sent via httpOnly cookie
    // No need to manually add Authorization header - backend reads from cookie

    // Double-submit CSRF: echo the csrf_token cookie (planted by the backend
    // on authenticated GETs) as a header on every mutating request.
    const method = (config.method || 'get').toLowerCase();
    if (method !== 'get' && method !== 'head' && method !== 'options') {
        const csrfToken = getCookie('csrf_token');
        if (csrfToken) {
            config.headers['X-CSRF-Token'] = csrfToken;
        }
    }
    return config;
});

// Refresh locking to prevent concurrent refresh attempts
let isRefreshing = false;
let failedQueue: Array<{ resolve: () => void; reject: (error: unknown) => void }> = [];

const processQueue = (error: unknown = null) => {
    failedQueue.forEach(prom => {
        if (error) {
            prom.reject(error);
        } else {
            prom.resolve();
        }
    });
    failedQueue = [];
};

// Interceptor to handle token refresh using httpOnly cookies
// NOTE: Refresh token is read from httpOnly cookie only - never from localStorage
api.interceptors.response.use(
    (response) => response,
    async (error) => {
        const originalRequest = error.config as AxiosRequestConfigWithRetry;

        // If refresh is in progress and this isn't a retried request, queue it
        if (isRefreshing && !originalRequest._retry) {
            return new Promise((resolve, reject) => {
                failedQueue.push({
                    resolve: () => {
                        resolve(api(originalRequest));
                    },
                    reject,
                });
            });
        }

        if (error.response?.status === 401 && !originalRequest._retry && !isRefreshing) {
            isRefreshing = true;
            originalRequest._retry = true;

            // Refresh using httpOnly cookie - no need to send refresh_token
            // Backend reads it from the cookie automatically
            try {
                const res = await axios.post(`${api.defaults.baseURL}/auth/refresh`, {}, {
                    withCredentials: true,
                });

                if (res.status === 200) {
                    // New tokens are set via httpOnly cookies by the server
                    processQueue();
                    return api(originalRequest);
                } else {
                    throw new Error('Refresh returned non-200 status');
                }
            } catch (refreshError) {
                processQueue(refreshError);
                // Refresh failed - don't redirect automatically
                // Let the AuthProvider handle the auth state clearing
                // Redirect should only happen on explicit login-required actions
                throw refreshError;
            } finally {
                isRefreshing = false;
            }
        }

        return Promise.reject(error);
    }
);

// ============================================================
// PUBLIC SOLUTION API
// ============================================================

import type { Solution } from '@/types';

export interface SolutionExistsResponse {
    exists: boolean;
}

export const solutionApi = {
    getSolution: (problemCode: string) =>
        api.get<Solution>(`/problem/${problemCode}/solution`),

    solutionExists: (problemCode: string) =>
        api.get<SolutionExistsResponse>(`/problem/${problemCode}/solution/exists`),
};

// ============================================================
// API TOKEN MANAGEMENT API
// ============================================================

import type { APITokenResponse, APITokenGenerateResponse } from '@/types';

export const apiTokenApi = {
    getAPIToken: () =>
        api.get<APITokenResponse>('/user/api-token'),

    generateAPIToken: () =>
        api.post<APITokenGenerateResponse>('/user/api-token'),

    revokeAPIToken: () =>
        api.delete<{ message: string }>('/user/api-token'),
};

// ============================================================
// USER DATA EXPORT API
// ============================================================

export interface ExportStatusResponse {
    can_request: boolean;
    last_export?: string;
    days_until_request: number;
    export_ready: boolean;
}

export const userExportApi = {
    requestExport: () =>
        api.post<{ message: string; estimated_time: string }>('/user/export/request'),

    getExportStatus: () =>
        api.get<ExportStatusResponse>('/user/export/status'),

    downloadExport: (exportId: string) =>
        api.get(`/user/export/download/${exportId}`, {
            responseType: 'blob',
        }),
};

// ============================================================
// CONTEST CALENDAR API
// ============================================================

import type { ContestCalendarResponse } from '@/types';

export const contestCalendarApi = {
    getCalendar: (year: number, month: number) =>
        api.get<ContestCalendarResponse>('/contests/calendar', {
            params: { year, month },
        }),
};

// ============================================================
// RANDOM PROBLEM API
// ============================================================

export const randomProblemApi = {
    getRandomProblem: () =>
        api.get<{ code: string }>('/problems/random'),
};

// ============================================================
// WEBAUTHN API
// ============================================================

import type { WebAuthnStatus, WebAuthnSetupResponse, WebAuthnLoginResponse, WebAuthnCredential } from '@/types';

export const webauthnApi = {
    getStatus: () =>
        api.get<WebAuthnStatus>('/auth/webauthn/status'),

    beginRegistration: (password: string) =>
        api.post<WebAuthnSetupResponse>('/auth/webauthn/register/begin', { password }),

    finishRegistration: (response: unknown, name: string) =>
        api.post<{ message: string; credential: WebAuthnCredential }>('/auth/webauthn/register/finish', {
            response,
            name,
        }),

    beginLogin: (username: string) =>
        api.post<WebAuthnLoginResponse>('/auth/webauthn/login/begin', { username }),

    finishLogin: (response: unknown) =>
        api.post<{ access_token: string; refresh_token: string; user: unknown }>('/auth/webauthn/login/finish', {
            response,
        }),

    getCredentials: () =>
        api.get<{ credentials: WebAuthnCredential[] }>('/auth/webauthn/credentials'),

    updateCredential: (id: string, name: string) =>
        api.patch<{ message: string }>(`/auth/webauthn/credentials/${id}`, { name }),

    deleteCredential: (id: string, password: string) =>
        api.delete<{ message: string }>(`/auth/webauthn/credentials/${id}`, {
            data: { password },
        }),
};

// ============================================================
// PROBLEM CLARIFICATION API (Task #33)
// ============================================================

import type { ProblemClarification } from '@/types';

export const problemClarificationApi = {
    // Public: List clarifications for a problem (backend returns { data } with no total)
    getClarifications: (problemCode: string) =>
        api.get<{ data: ProblemClarification[] }>(`/problem/${problemCode}/clarifications`),

    // Admin: Create a new clarification
    createClarification: (problemCode: string, description: string) =>
        api.post<{ id: number; problem_id: number; description: string; date: string; problem_code: string }>(
            `/admin/problem/${problemCode}/clarification`,
            { description }
        ),

    // Admin: Delete a clarification
    deleteClarification: (id: number) =>
        api.delete<{ message: string }>(`/admin/problem/clarification/${id}`),
};

// ============================================================
// BLOG VOTING API (Task #37)
// ============================================================

import type { BlogVoteRequest } from '@/types';

export const blogVoteApi = {
    // Vote on a blog post (delta: 1 for upvote, -1 for downvote)
    vote: (blogId: number, delta: 1 | -1) =>
        api.post<{ message: string }>(`/blog/${blogId}/vote`, { delta } as BlogVoteRequest),
};

// ============================================================
// PERFORMANCE POINTS BREAKDOWN API (Task #42)
// ============================================================

import type { PPBreakdown } from '@/types';

export const ppBreakdownApi = {
    getPPBreakdown: (username: string) =>
        api.get<PPBreakdown>(`/user/${username}/pp-breakdown`),
};

// ============================================================
// CONTEST STATISTICS API (Task #24)
// ============================================================

import type { ContestStats } from '@/types';

export const contestStatsApi = {
    getContestStats: (contestKey: string) =>
        api.get<ContestStats>(`/contest/${contestKey}/stats`),
};

// ============================================================
// BLOG RSS/ATOM FEEDS API (Task #40)
// ============================================================

export const blogFeedApi = {
    // Get RSS feed URL
    getRssUrl: () => `${api.defaults.baseURL}/blogs/feed/rss`,

    // Get Atom feed URL
    getAtomUrl: () => `${api.defaults.baseURL}/blogs/feed/atom`,
};

// ============================================================
// PROBLEM PDF STATEMENT API (Task #29)
// ============================================================

export const problemPdfApi = {
    // Get PDF statement URL
    getPdfUrl: (problemCode: string) => `${api.defaults.baseURL}/problem/${problemCode}/pdf`,

    // Check if PDF exists (by checking if pdf_url is non-empty in problem detail)
    hasPdf: (problem: { pdf_url?: string }) => !!problem.pdf_url,
};

// ============================================================
// SUBMISSION DIFF API (Task #58)
// ============================================================

export interface SubmissionDiffResponse {
    submission1: {
        id: number;
        problem: string;
        user: string;
        date: string;
        language: string;
        result: string | null;
        points: number | null;
    };
    submission2: {
        id: number;
        problem: string;
        user: string;
        date: string;
        language: string;
        result: string | null;
        points: number | null;
    };
    unified_diff: string;
    diff_lines: Array<{
        type: 'add' | 'delete' | 'context';
        line: number;
        content: string;
    }>;
    stats: {
        additions: number;
        deletions: number;
    };
}

export const submissionDiffApi = {
    // Get diff between two submissions (backend route: /submissions/:id1/diff/:id2)
    getDiff: (id1: number, id2: number) =>
        api.get<SubmissionDiffResponse>(`/submissions/${id1}/diff/${id2}`),
};

export default api;
