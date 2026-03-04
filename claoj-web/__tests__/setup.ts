import '@testing-library/jest-dom';

// Mock next-intl
jest.mock('next-intl', () => ({
    useTranslations: () => {
        return function (key: string) {
            return key;
        };
    },
}));

// Mock next-intl/navigation
jest.mock('@/navigation', () => ({
    routing: {
        locales: ['en', 'vi'],
        defaultLocale: 'en',
        localePrefix: 'as-needed',
    },
    Link: jest.fn(),
    redirect: jest.fn(),
    usePathname: jest.fn(() => '/'),
    useRouter: () => ({
        push: jest.fn(),
        replace: jest.fn(),
        prefetch: jest.fn(),
    }),
}));

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: jest.fn().mockImplementation((query) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        dispatchEvent: jest.fn(),
    })),
});

// Mock fetch
global.fetch = jest.fn();
