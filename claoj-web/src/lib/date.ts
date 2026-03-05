import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import localizedFormat from 'dayjs/plugin/localizedFormat';

dayjs.extend(relativeTime);
dayjs.extend(localizedFormat);

export type DateFormat = 'relative' | 'short' | 'long' | 'datetime' | 'time';

/**
 * Format a date string or Date object using dayjs
 * @param date - The date to format (string or Date object)
 * @param format - The format style: 'relative' (from now), 'short' (MMM D, YYYY), 'long' (full date with time), 'datetime' (YYYY-MM-DD HH:mm:ss), 'time' (HH:mm:ss)
 * @param locale - Optional locale (default: 'en')
 * @returns Formatted date string
 */
export function formatDate(date: string | Date | null | undefined, format: DateFormat = 'relative', locale = 'en'): string {
    if (!date) return '';

    dayjs.locale(locale);
    const d = dayjs(date);

    switch (format) {
        case 'relative':
            return d.fromNow();
        case 'short':
            return d.format('MMM D, YYYY');
        case 'long':
            return d.format('MMMM D, YYYY [at] h:mm A');
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
