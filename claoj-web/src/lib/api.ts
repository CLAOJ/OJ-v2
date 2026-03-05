import axios, { InternalAxiosRequestConfig } from 'axios';

// Extend Axios request config to include _retry property
interface AxiosRequestConfigWithRetry extends InternalAxiosRequestConfig {
    _retry?: boolean;
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
                // Refresh failed, redirect to login
                if (typeof window !== 'undefined') {
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
