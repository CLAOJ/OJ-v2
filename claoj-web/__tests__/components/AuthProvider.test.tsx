import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { renderHook, act, waitFor } from '@testing-library/react';
import React from 'react';
import { AuthProvider, useAuth } from '@/components/providers/AuthProvider';
import { User } from '@/types';

// Mock the api module
jest.mock('@/lib/api', () => ({
    __esModule: true,
    default: {
        post: jest.fn(),
        get: jest.fn(),
        interceptors: {
            request: {
                use: jest.fn(),
            },
            response: {
                use: jest.fn(),
            },
        },
        defaults: {
            baseURL: '/api/v2',
        },
    },
    webauthnApi: {
        beginLogin: jest.fn(),
        finishLogin: jest.fn(),
    },
}));

import api from '@/lib/api';

const wrapper = ({ children }: { children: React.ReactNode }) => (
    <AuthProvider>{children}</AuthProvider>
);

describe('AuthProvider', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        jest.useFakeTimers();
    });

    afterEach(() => {
        jest.useRealTimers();
    });

    describe('login', () => {
        it('should set user on successful login', async () => {
            const mockUser: User = {
                id: 1,
                username: 'testuser',
                is_admin: false,
                is_staff: false,
            };
            const mockResponse = {
                access_token: 'mock_access_token',
                refresh_token: 'mock_refresh_token',
                user: mockUser,
            };

            (api.post as jest.Mock).mockResolvedValue({ data: mockResponse });

            const { result } = renderHook(() => useAuth(), { wrapper });

            await act(async () => {
                await result.current.login('testuser', 'password123');
            });

            await waitFor(() => {
                expect(result.current.user).toEqual(mockUser);
            });
        });

        it('should pass rememberMe flag to login API', async () => {
            const mockUser: User = {
                id: 1,
                username: 'testuser',
                is_admin: false,
                is_staff: false,
            };
            const mockResponse = {
                access_token: 'mock_access_token',
                refresh_token: 'mock_refresh_token',
                user: mockUser,
            };

            (api.post as jest.Mock).mockResolvedValue({ data: mockResponse });

            const { result } = renderHook(() => useAuth(), { wrapper });

            await act(async () => {
                await result.current.login('testuser', 'password123', true);
            });

            expect(api.post).toHaveBeenCalledWith('/auth/login', {
                username: 'testuser',
                password: 'password123',
                remember_me: true,
            });
        });

        it('should default rememberMe to false when not provided', async () => {
            const mockUser: User = {
                id: 1,
                username: 'testuser',
                is_admin: false,
                is_staff: false,
            };
            const mockResponse = {
                access_token: 'mock_access_token',
                refresh_token: 'mock_refresh_token',
                user: mockUser,
            };

            (api.post as jest.Mock).mockResolvedValue({ data: mockResponse });

            const { result } = renderHook(() => useAuth(), { wrapper });

            await act(async () => {
                await result.current.login('testuser', 'password123');
            });

            expect(api.post).toHaveBeenCalledWith('/auth/login', {
                username: 'testuser',
                password: 'password123',
                remember_me: undefined,
            });
        });

        it('should handle TOTP requirement response', async () => {
            const mockResponse = {
                requires_totp: true,
                username: 'testuser',
                message: 'Please enter your TOTP code',
            };

            (api.post as jest.Mock).mockResolvedValue({ data: mockResponse });

            const { result } = renderHook(() => useAuth(), { wrapper });

            let loginResult;
            await act(async () => {
                loginResult = await result.current.login('testuser', 'password123');
            });

            expect(loginResult).toEqual({
                requiresTotp: true,
                username: 'testuser',
            });
            expect(result.current.user).toBeNull();
        });

        it('should throw error on failed login', async () => {
            (api.post as jest.Mock).mockRejectedValue(new Error('Invalid credentials'));

            const { result } = renderHook(() => useAuth(), { wrapper });

            await expect(result.current.login('testuser', 'wrongpassword')).rejects.toThrow();
        });
    });

    describe('loginTotp', () => {
        it('should set user on successful TOTP verification', async () => {
            const mockUser: User = {
                id: 1,
                username: 'testuser',
                is_admin: false,
                is_staff: false,
            };
            const mockResponse = {
                access_token: 'mock_access_token',
                refresh_token: 'mock_refresh_token',
                user: mockUser,
            };

            (api.post as jest.Mock).mockResolvedValue({ data: mockResponse });

            const { result } = renderHook(() => useAuth(), { wrapper });

            await act(async () => {
                const user = await result.current.loginTotp('testuser', '123456');
                expect(user).toEqual(mockUser);
            });

            await waitFor(() => {
                expect(result.current.user).toEqual(mockUser);
            });
        });
    });

    describe('logout', () => {
        it('should clear user on logout', async () => {
            const mockUser: User = {
                id: 1,
                username: 'testuser',
                is_admin: false,
                is_staff: false,
            };
            const mockResponse = {
                access_token: 'mock_access_token',
                refresh_token: 'mock_refresh_token',
                user: mockUser,
            };

            (api.post as jest.Mock).mockResolvedValue({ data: mockResponse });

            const { result } = renderHook(() => useAuth(), { wrapper });

            // First login
            await act(async () => {
                await result.current.login('testuser', 'password123');
            });

            await waitFor(() => {
                expect(result.current.user).toEqual(mockUser);
            });

            // Mock logout API call
            (api.post as jest.Mock).mockResolvedValue({ data: {} });

            // Then logout
            await act(async () => {
                await result.current.logout();
            });

            await waitFor(() => {
                expect(result.current.user).toBeNull();
            });
        });

        it('should clear user even if logout API fails', async () => {
            const mockUser: User = {
                id: 1,
                username: 'testuser',
                is_admin: false,
                is_staff: false,
            };
            const mockResponse = {
                access_token: 'mock_access_token',
                refresh_token: 'mock_refresh_token',
                user: mockUser,
            };

            (api.post as jest.Mock).mockResolvedValueOnce({ data: mockResponse });

            const { result } = renderHook(() => useAuth(), { wrapper });

            // First login
            await act(async () => {
                await result.current.login('testuser', 'password123');
            });

            await waitFor(() => {
                expect(result.current.user).toEqual(mockUser);
            });

            // Mock logout API failure
            (api.post as jest.Mock).mockRejectedValue(new Error('Network error'));

            // Then logout - should still clear user
            await act(async () => {
                await result.current.logout();
            });

            await waitFor(() => {
                expect(result.current.user).toBeNull();
            });
        });
    });

    describe('checkAuth (initial load)', () => {
        it('should set user from /user/me response (user data in res.data)', async () => {
            const mockUser: User = {
                id: 1,
                username: 'testuser',
                is_admin: false,
                is_staff: false,
            };

            // /user/me returns user directly in res.data, not res.data.user
            (api.get as jest.Mock).mockResolvedValue({
                status: 200,
                data: mockUser,
            });

            const { result } = renderHook(() => useAuth(), { wrapper });

            await waitFor(() => {
                expect(result.current.loading).toBe(false);
            });

            expect(result.current.user).toEqual(mockUser);
        });

        it('should handle 401 response gracefully', async () => {
            (api.get as jest.Mock).mockRejectedValue({
                response: { status: 401 },
                isAxiosError: true,
            });

            const { result } = renderHook(() => useAuth(), { wrapper });

            await waitFor(() => {
                expect(result.current.loading).toBe(false);
            });

            expect(result.current.user).toBeNull();
        });

        it('should handle network error gracefully', async () => {
            (api.get as jest.Mock).mockRejectedValue({
                code: 'ERR_NETWORK',
                isAxiosError: true,
            });

            const { result } = renderHook(() => useAuth(), { wrapper });

            await waitFor(() => {
                expect(result.current.loading).toBe(false);
            });

            expect(result.current.user).toBeNull();
        });
    });

    describe('periodic token refresh', () => {
        it('should call /auth/refresh every 10 minutes when user is logged in', async () => {
            const mockUser: User = {
                id: 1,
                username: 'testuser',
                is_admin: false,
                is_staff: false,
            };

            (api.get as jest.Mock).mockResolvedValue({
                status: 200,
                data: mockUser,
            });

            (api.post as jest.Mock).mockResolvedValue({ data: {} });

            const { result } = renderHook(() => useAuth(), { wrapper });

            // Wait for initial auth check to complete and user to be set
            await waitFor(() => {
                expect(result.current.user).toEqual(mockUser);
            });

            // Fast-forward 10 minutes
            act(() => {
                jest.advanceTimersByTime(10 * 60 * 1000);
            });

            // Wait for the interval callback to execute
            await waitFor(() => {
                expect(api.post).toHaveBeenCalledWith(
                    '/auth/refresh',
                    {},
                    { withCredentials: true }
                );
            });
        });

        it('should not call /auth/refresh when user is not logged in', async () => {
            (api.get as jest.Mock).mockRejectedValue({
                response: { status: 401 },
                isAxiosError: true,
            });

            (api.post as jest.Mock).mockResolvedValue({ data: {} });

            renderHook(() => useAuth(), { wrapper });

            await waitFor(() => {
                expect(api.get).toHaveBeenCalled();
            });

            // Fast-forward 10 minutes
            act(() => {
                jest.advanceTimersByTime(10 * 60 * 1000);
            });

            // Should not call refresh since user is null
            expect(api.post).not.toHaveBeenCalledWith('/auth/refresh', expect.anything(), expect.anything());
        });

        it('should handle refresh errors silently', async () => {
            const mockUser: User = {
                id: 1,
                username: 'testuser',
                is_admin: false,
                is_staff: false,
            };

            (api.get as jest.Mock).mockResolvedValue({
                status: 200,
                data: mockUser,
            });

            (api.post as jest.Mock).mockRejectedValue(new Error('Refresh failed'));

            const { result } = renderHook(() => useAuth(), { wrapper });

            await waitFor(() => {
                expect(result.current.user).toEqual(mockUser);
            });

            // Fast-forward 10 minutes - should not throw
            await act(async () => {
                jest.advanceTimersByTime(10 * 60 * 1000);
            });

            // User should still be set (error is silent)
            expect(result.current.user).toEqual(mockUser);
        });
    });
});
