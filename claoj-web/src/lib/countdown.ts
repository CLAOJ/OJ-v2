import { formatUnit } from './date';

/**
 * Countdown formatting for the home page contest widgets. v1 shows days and
 * HH:MM:SS; a plain hours total ("890:49:45") reads as a bug once a contest is
 * more than a couple of days out.
 */
export function formatCountdown(ms: number): string {
    // An unparsable end time yields NaN; a NaN countdown must not reach the UI.
    const total = Number.isFinite(ms) ? Math.max(0, Math.floor(ms / 1000)) : 0;
    const days = Math.floor(total / 86400);
    const hms = [Math.floor((total % 86400) / 3600), Math.floor((total % 3600) / 60), total % 60]
        .map(n => String(n).padStart(2, '0'))
        .join(':');
    return days > 0 ? `${formatUnit(days, 'day')} ${hms}` : hms;
}