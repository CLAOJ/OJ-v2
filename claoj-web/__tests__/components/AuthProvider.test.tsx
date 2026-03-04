import { describe, it, expect, beforeEach } from '@jest/globals';
import { renderHook, act } from '@testing-library/react';
import React from 'react';
import { AuthProvider, useAuth } from '@/components/providers/AuthProvider';

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
    },
}));

import api from '@/lib/api';

// Mock localStorage
const localStorageMock = {
    store: {} as Record<string, string>,
    getItem: jest.fn(function (key: string) {
        return this.store[key] || null;
    }),
    setItem: jest.fn(function (key: string, value: string) {
        this.store[key] = value;
    }),
    removeItem: jest.fn(function (key: string) {
        delete this.store[key];
    }),
    clear: jest.fn(function () {
        this.store = {};
    }),
};

Object.defineProperty(window, 'localStorage', {
    value: localStorageMock,
});

const wrapper = ({ children }: { children: React.ReactNode }) => (
    <AuthProvider>{children}</AuthProvider>
);

describe('AuthProvider', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        localStorageMock.store = {};
    });

    describe('login', () => {
        it('should store tokens and user on successful login', async () => {
            const mockResponse = {
                access_token: 'mock_access_token',
                refresh_token: 'mock_refresh_token',
                user: {
                    id: 1,
                    username: 'testuser',
                    is_admin: false,
                },
            };

            (api.post as jest.Mock).mockResolvedValue({ data: mockResponse });

            const { result } = renderHook(() => useAuth(), { wrapper });

            await act(async () => {
                await result.current.login('testuser', 'password123');
            });

            expect(localStorageMock.setItem).toHaveBeenCalledWith('access_token', 'mock_access_token');
            expect(localStorageMock.setItem).toHaveBeenCalledWith('refresh_token', 'mock_refresh_token');
            expect(result.current.user).toEqual(mockResponse.user);
        });

        it('should throw error on failed login', async () => {
            (api.post as jest.Mock).mockRejectedValue(new Error('Invalid credentials'));

            const { result } = renderHook(() => useAuth(), { wrapper });

            await expect(result.current.login('testuser', 'wrongpassword')).rejects.toThrow();
        });
    });

    describe('logout', () => {
        it('should clear tokens and user on logout', () => {
            const { result } = renderHook(() => useAuth(), { wrapper });

            act(() => {
                result.current.logout();
            });

            expect(localStorageMock.removeItem).toHaveBeenCalledWith('access_token');
            expect(localStorageMock.removeItem).toHaveBeenCalledWith('refresh_token');
            expect(result.current.user).toBeNull();
        });
    });
});
