'use client';

import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import localizedFormat from 'dayjs/plugin/localizedFormat';
import 'dayjs/locale/vi';

dayjs.extend(relativeTime);
dayjs.extend(localizedFormat);

/**
 * Points dayjs's global locale at the site language, so every `fromNow()`,
 * month name and `ll`/`LLL` format follows /vi or /en instead of dayjs's
 * English default. Set during render (not in an effect) because it is
 * mounted before the page content and must apply to that first render.
 */
export function DayjsLocale({ locale }: { locale: string }) {
    if (dayjs.locale() !== locale) dayjs.locale(locale);
    return null;
}
