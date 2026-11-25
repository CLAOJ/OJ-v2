import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import localizedFormat from 'dayjs/plugin/localizedFormat';
import 'dayjs/locale/vi';

dayjs.extend(relativeTime);
dayjs.extend(localizedFormat);

export type DateFormat = 'relative' | 'short' | 'long' | 'datetime' | 'time';

/**
 * Format a date string or Date object using dayjs
 * @param date - The date to format (string or Date object)
 * @param format - The format style: 'relative' (from now), 'short' (MMM D, YYYY), 'long' (full date with time), 'datetime' (YYYY-MM-DD HH:mm:ss), 'time' (HH:mm:ss)
 * @param locale - Optional locale; omitted, the app-wide locale set by DayjsLocale applies
 * @returns Formatted date string
 */
export function formatDate(date: string | Date | null | undefined, format: DateFormat = 'relative', locale?: string): string {
    if (!date) return '';

    const d = locale ? dayjs(date).locale(locale) : dayjs(date);

    switch (format) {
        case 'relative':
            return d.fromNow();
        case 'short':
            return d.format('ll');
        case 'long':
            return d.format('LLL');
        case 'datetime':
            return d.format('YYYY-MM-DD HH:mm:ss');
        case 'time':
            return d.format('HH:mm:ss');
        default:
            return d.fromNow();
    }
}

/**
 * Format a date as a relative time (e.g., "2 hours ago", "in 3 days")
 * @param date - The date to format
 * @returns Relative time string
 */
export function formatRelativeTime(date: string | Date | null | undefined): string {
    return formatDate(date, 'relative');
}

/**
 * Format a date as a short string (e.g., "Jan 15, 2024")
 * @param date - The date to format
 * @returns Short date string
 */
export function formatShortDate(date: string | Date | null | undefined): string {
    return formatDate(date, 'short');
}

/**
 * Format a date as a time string (e.g., "14:30:00")
 * @param date - The date to format
 * @returns Time string
 */
export function formatTime(date: string | Date | null | undefined): string {
    return formatDate(date, 'time');
}

/**
 * Format a date as a datetime string (e.g., "Jan 15, 2024 14:30")
 * @param date - The date to format
 * @returns DateTime string
 */
export function formatDateTime(date: string | Date | null | undefined): string {
    if (!date) return '';
    return dayjs(date).format('ll HH:mm');
}

/**
 * Check if a date is today
 * @param date - The date to check
 * @returns true if the date is today
 */
export function isToday(date: string | Date | null | undefined): boolean {
    if (!date) return false;
    return dayjs(date).isSame(dayjs(), 'day');
}

/**
 * Check if a date is in the past
 * @param date - The date to check
 * @returns true if the date is in the past
 */
export function isPast(date: string | Date | null | undefined): boolean {
    if (!date) return false;
    return dayjs(date).isBefore(dayjs());
}

/**
 * Check if a date is in the future
 * @param date - The date to check
 * @returns true if the date is in the future
 */
export function isFuture(date: string | Date | null | undefined): boolean {
    if (!date) return false;
    return dayjs(date).isAfter(dayjs());
}

/**
 * Get the time difference between two dates in milliseconds
 * @param date1 - First date
 * @param date2 - Second date (defaults to now)
 * @returns Difference in milliseconds
 */
export function getTimeDifference(date1: string | Date, date2: string | Date = new Date()): number {
    return dayjs(date1).diff(dayjs(date2));
}

/**
 * Get the time difference in a human-readable format
 * @param date - The date to compare with now
 * @returns Object with days, hours, minutes, seconds
 */
export function getTimeDifferenceBreakdown(date: string | Date): {
    days: number;
    hours: number;
    minutes: number;
    seconds: number;
    total: number;
} {
    const diff = dayjs(date).diff(dayjs(), 'millisecond');
    const absDiff = Math.abs(diff);

    const days = Math.floor(absDiff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((absDiff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((absDiff % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((absDiff % (1000 * 60)) / 1000);

    return {
        days,
        hours,
        minutes,
        seconds,
        total: diff
    };
}

type DurationUnit = 'day' | 'hour' | 'minute' | 'second';

/** "3d" in English, "3 ngày" in Vietnamese: a unit count in the app-wide dayjs locale. */
export function formatUnit(n: number, unit: DurationUnit): string {
    return new Intl.NumberFormat(dayjs.locale(), { style: 'unit', unit, unitDisplay: 'narrow' }).format(n);
}

/** Length of a span in its `parts` largest units, from the first non-zero one: "2d 3h", "45m". */
export function formatDuration(ms: number, parts = 2): string {
    const total = Number.isFinite(ms) ? Math.max(0, Math.floor(ms / 1000)) : 0;
    const units: [number, DurationUnit][] = [
        [Math.floor(total / 86400), 'day'],
        [Math.floor((total % 86400) / 3600), 'hour'],
        [Math.floor((total % 3600) / 60), 'minute'],
        [total % 60, 'second'],
    ];
    const first = units.findIndex(([n]) => n > 0);
    const start = first < 0 ? units.length - 1 : first;
    return units.slice(start, start + parts).map(([n, u]) => formatUnit(n, u)).join(' ');
}
