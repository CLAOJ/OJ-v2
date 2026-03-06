import axios, { InternalAxiosRequestConfig } from 'axios';

// Extend Axios request config to include _retry and _skipAuthRedirect properties
interface AxiosRequestConfigWithRetry extends InternalAxiosRequestConfig {
    _retry?: boolean;
    _skipAuthRedirect?: boolean;
}

function getApiBaseUrl(): string {
    if (typeof window === 'undefined') return 'http://localhost:8080/api/v2';
    return `${window.location.origin}/api/v2`;
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

// Interceptor to add access token from httpOnly cookie
// NOTE: Tokens are stored in httpOnly cookies only - never in localStorage for security
api.interceptors.request.use((config) => {
    // Token is automatically sent via httpOnly cookie
    // No need to manually add Authorization header - backend reads from cookie
    return config;
});

// Interceptor to handle token refresh using httpOnly cookies
// NOTE: Refresh token is read from httpOnly cookie only - never from localStorage
api.interceptors.response.use(
    (response) => response,
    async (error) => {
        const originalRequest = error.config as AxiosRequestConfigWithRetry;
        if (error.response?.status === 401 && !originalRequest._retry) {
            originalRequest._retry = true;

            // Refresh using httpOnly cookie - no need to send refresh_token
            // Backend reads it from the cookie automatically
            try {
                const res = await axios.post(`${api.defaults.baseURL}/auth/refresh`, {}, {
                    withCredentials: true,
                });

                if (res.status === 200) {
                    // New tokens are set via httpOnly cookies by the server
                    return api(originalRequest);
                }
            } catch (refreshError) {
                // Refresh failed, redirect to login unless _skipAuthRedirect is set
                if (typeof window !== 'undefined' && !originalRequest._skipAuthRedirect) {
                    window.location.href = '/login';
                }
            }
        }
        return Promise.reject(error);
    }
);

export default api;

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

    finishRegistration: (response: any, name: string) =>
        api.post<{ message: string; credential: WebAuthnCredential }>('/auth/webauthn/register/finish', {
            response,
            name,
        }),

    beginLogin: (username: string) =>
        api.post<WebAuthnLoginResponse>('/auth/webauthn/login/begin', { username }),

    finishLogin: (response: any) =>
        api.post<{ access_token: string; refresh_token: string; user: any }>('/auth/webauthn/login/finish', {
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
// PROBLEM SUGGESTION API (Task #31)
// ============================================================

import type { 
    ProblemSuggestion, 
    ProblemSuggestionAdmin, 
    ProblemSuggestionDetail,
    ProblemSuggestRequest,
    ApproveSuggestionRequest,
    RejectSuggestionRequest 
} from '@/types';

export const problemSuggestionApi = {
    // Submit a new problem suggestion
    suggestProblem: (data: ProblemSuggestRequest) =>
        api.post<{ success: boolean; message: string; suggestion: ProblemSuggestion }>('/problems/suggest', data),

    // Get current user's suggestions
    getUserSuggestions: (page: number = 1, pageSize: number = 20) =>
        api.get<{ data: ProblemSuggestion[]; total: number }>('/my-suggestions', {
            params: { page, page_size: pageSize },
        }),

    // Admin: List all suggestions
    listSuggestions: (page: number = 1, pageSize: number = 20, status?: string) =>
        api.get<{ data: ProblemSuggestionAdmin[]; total: number }>('/admin/problem-suggestions', {
            params: { page, page_size: pageSize, status },
        }),

    // Admin: Get suggestion detail
    getSuggestion: (id: number) =>
        api.get<ProblemSuggestionDetail>(`/admin/problem-suggestion/${id}`),

    // Admin: Approve suggestion
    approveSuggestion: (id: number, data: ApproveSuggestionRequest) =>
        api.post<{ success: boolean; message: string; problem: { id: number; code: string; name: string } }>(
            `/admin/problem-suggestion/${id}/approve`,
            data
        ),

    // Admin: Reject suggestion
    rejectSuggestion: (id: number, data: RejectSuggestionRequest) =>
        api.post<{ success: boolean; message: string }>(
            `/admin/problem-suggestion/${id}/reject`,
            data
        ),

    // Admin: Delete suggestion
    deleteSuggestion: (id: number) =>
        api.delete<{ success: boolean; message: string }>(`/admin/problem-suggestion/${id}`),
};
