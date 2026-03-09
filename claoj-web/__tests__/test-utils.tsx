import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from 'next-themes';
import { NextIntlClientProvider } from 'next-intl';

// Re-export common testing utilities
export { render, screen, fireEvent, waitFor };

// Custom renderers with providers
interface RenderWithProvidersOptions {
  initialRoute?: string;
  locale?: 'en' | 'vi';
  messages?: Record<string, string>;
  queryClient?: QueryClient;
  preloadedState?: any;
  [key: string]: any;
}

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        staleTime: Infinity,
      },
    },
  });
}

function createTestMessages() {
  const messages: Record<string, string> = {};
  const handler = {
    get: (target: Record<string, string>, prop: string) => {
      if (!target[prop]) {
        target[prop] = prop; // Return key as fallback for missing translations
      }
      return target[prop];
    },
  };
  return new Proxy(messages, handler);
}

export function renderWithProviders(
  ui: React.ReactElement,
  options: RenderWithProvidersOptions = {}
) {
  const {
    initialRoute = '/',
    locale = 'en',
    messages = createTestMessages(),
    queryClient = createTestQueryClient(),
    ...restOptions
  } = options;

  const wrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <ThemeProvider defaultTheme="light" storageKey={undefined}>
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      </ThemeProvider>
    </NextIntlClientProvider>
  );

  return render(ui, {
    wrapper,
    ...restOptions,
  });
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
