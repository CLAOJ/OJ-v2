import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeEach, jest } from '@jest/globals';

// Re-export common testing utilities
export { render, screen, fireEvent, waitFor };

// Custom renderers with providers
export function renderWithProviders(ui: React.ReactElement, options?: any) {
  // TODO: Add providers (QueryClient, WebSocket, Theme, Intl)
  return render(ui, options);
}

// Test utilities
export function createMockRouter(overrides: any = {}) {
  return {
    route: '/',
    pathname: '',
    query: {},
    asPath: '',
    push: jest.fn(),
    events: {
      on: jest.fn(),
      off: jest.fn(),
    },
    beforePopState: jest.fn(() => null),
    prefetch: jest.fn(() => null),
    ...overrides,
  };
}

export function createMockFetchResponse(data: any, status: number = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: jest.fn().mockResolvedValue(data),
    text: jest.fn().mockResolvedValue(JSON.stringify(data)),
  };
}
