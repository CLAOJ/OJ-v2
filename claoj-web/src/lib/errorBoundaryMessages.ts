import { routing } from '@/navigation';

/**
 * Translations for the two root error boundaries.
 *
 * `app/error.tsx` and `app/global-error.tsx` render *outside*
 * NextIntlClientProvider — global-error.tsx replaces the root layout entirely,
 * and app/error.tsx catches failures in the `[locale]` layout itself, i.e.
 * exactly the cases where the provider isn't there. Calling useTranslations in
 * either one throws, so they were left in English and were the last
 * user-visible strings with no Vietnamese at all.
 *
 * These messages are few and static, so they're inlined here and selected from
 * the same NEXT_LOCALE cookie next-intl's middleware sets.
 */
const MESSAGES = {
    en: {
        errorTitle: 'Something went wrong',
        errorDescription: 'An error occurred while processing your request. Please try again.',
        criticalTitle: 'Critical Error',
        criticalDescription: 'A critical error occurred. Please refresh the page or try again later.',
        tryAgain: 'Try again',
        goHome: 'Go home',
    },
    vi: {
        errorTitle: 'Đã có lỗi xảy ra',
        errorDescription: 'Đã xảy ra lỗi khi xử lý yêu cầu của bạn. Vui lòng thử lại.',
        criticalTitle: 'Lỗi nghiêm trọng',
        criticalDescription: 'Đã xảy ra lỗi nghiêm trọng. Vui lòng tải lại trang hoặc thử lại sau.',
        tryAgain: 'Thử lại',
        goHome: 'Về trang chủ',
    },
} as const;

export type ErrorMessageKey = keyof (typeof MESSAGES)['en'];

function readLocale(): keyof typeof MESSAGES {
    if (typeof document === 'undefined') return routing.defaultLocale as keyof typeof MESSAGES;
    const match = document.cookie.match(/(?:^|; )NEXT_LOCALE=([^;]*)/);
    const locale = match ? decodeURIComponent(match[1]) : undefined;
    return locale && locale in MESSAGES
        ? (locale as keyof typeof MESSAGES)
        : (routing.defaultLocale as keyof typeof MESSAGES);
}

/** Returns a `t`-shaped lookup usable without the next-intl provider. */
export function getErrorMessages() {
    const messages = MESSAGES[readLocale()];
    return (key: ErrorMessageKey) => messages[key];
}
