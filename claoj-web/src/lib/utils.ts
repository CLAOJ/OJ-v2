import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}

/**
 * Reusable focus ring utility for consistent accessibility focus states
 */
export function focusRing(className?: string) {
    return cn(
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        className
    );
}

export function formatTime(seconds: number): string {
    if (seconds < 60) return `${Math.round(seconds)}s`;
    const mins = Math.floor(seconds / 60);
    const secs = Math.round(seconds % 60);
    return `${mins}m ${secs}s`;
}

/** Time limit in seconds → "1.50s"; null when the value is missing. */
export function formatLimitSeconds(seconds: number | null | undefined): string | null {
    if (typeof seconds !== 'number' || !Number.isFinite(seconds)) return null;
    return `${seconds.toFixed(2)}s`;
}

/** Language-limit memory is stored in KB → "256 MB"; null when missing. */
export function formatLimitMemory(kilobytes: number | null | undefined): string | null {
    if (typeof kilobytes !== 'number' || !Number.isFinite(kilobytes)) return null;
    return `${Math.round(kilobytes / 1024)} MB`;
}

/**
 * Problem memory limit in KB → "512 MB" / "512 KB". Kept separate from
 * formatLimitMemory because sub-megabyte values stay in KB (v1 shows them so)
 * and non-integers keep one decimal.
 */
export function formatMemoryLimit(kilobytes?: number | null): string {
    if (kilobytes == null) return '—';
    if (kilobytes < 1024) return `${kilobytes} KB`;
    const mb = kilobytes / 1024;
    return `${Number.isInteger(mb) ? mb : mb.toFixed(1)} MB`;
}

export function getStatusColor(status: string): string {
    switch (status) {
        case 'AC': return 'text-emerald-500';
        case 'WA': return 'text-destructive';
        case 'TLE': return 'text-amber-500';
        case 'MLE': return 'text-amber-500';
        case 'OLE': return 'text-amber-500';
        case 'RE': return 'text-purple-500';
        case 'CE': return 'text-blue-500';
        case 'IE': return 'text-zinc-500';
        case 'QU': return 'text-zinc-400 animate-pulse';
        case 'SC': return 'text-zinc-400 animate-pulse';
        default: return 'text-zinc-500';
    }
}

export function getStatusVariant(status: string): 'success' | 'destructive' | 'warning' | 'default' | 'outline' {
    switch (status) {
        case 'AC': return 'success';
        case 'WA': return 'destructive';
        case 'TLE':
        case 'MLE':
        case 'OLE': return 'warning';
        case 'CE': return 'default';
        default: return 'outline';
    }
}

// Rating tiers exactly as v1 (OJ/judge/ratings.py RATING_VALUES/RATING_LEVELS),
// so a student's colour is the same on both sites. Hues follow v1's
// vars-default.scss / vars-dark.scss, darkened or lightened only as far as
// needed for 4.5:1 text contrast (light on #fff, dark on #263238 cards).
const RATING_VALUES = [1200, 1400, 1600, 1900, 2200, 2300, 2400, 2600, 2900];
const RATING_TIERS = [
    { title: 'Newbie', text: 'text-[#737373] dark:text-[#a89f92]', bg: 'bg-[#737373]' },
    { title: 'Pupil', text: 'text-[#008000] dark:text-[#72ff72]', bg: 'bg-[#008000]' },
    { title: 'Specialist', text: 'text-[#037d75] dark:text-[#57fcf2]', bg: 'bg-[#037d75]' },
    { title: 'Expert', text: 'text-[#0000ff] dark:text-[#7aa7ff]', bg: 'bg-[#0000ff]' },
    { title: 'Candidate Master', text: 'text-[#aa00aa] dark:text-[#ff55ff]', bg: 'bg-[#aa00aa]' },
    { title: 'Master', text: 'text-[#b85600] dark:text-[#ff981a]', bg: 'bg-[#b85600]' },
    { title: 'International Master', text: 'text-[#b85600] dark:text-[#ff981a]', bg: 'bg-[#b85600]' },
    { title: 'Grandmaster', text: 'text-[#d10000] dark:text-[#ff6b6b]', bg: 'bg-[#d10000]' },
    { title: 'International Grandmaster', text: 'text-[#d10000] dark:text-[#ff6b6b]', bg: 'bg-[#d10000]' },
    { title: 'Legendary Grandmaster', text: 'text-[#d10000] dark:text-[#ff6b6b]', bg: 'bg-[#d10000]' },
];
const UNRATED = { title: 'Unrated', text: 'text-zinc-500 dark:text-zinc-400', bg: 'bg-zinc-500' };

/** v1 rating_level: bisect_right over RATING_VALUES. */
export function ratingLevel(rating: number): number {
    let level = 0;
    while (level < RATING_VALUES.length && rating >= RATING_VALUES[level]) level++;
    return level;
}

function ratingTier(rating: number | null | undefined) {
    return rating == null ? UNRATED : RATING_TIERS[ratingLevel(rating)];
}

/** v1 rating_progress: how far through the current tier (0..1), for the rating circle. */
export function ratingProgress(rating: number): number {
    const level = ratingLevel(rating);
    if (level === RATING_VALUES.length) return 1;
    const prev = level ? RATING_VALUES[level - 1] : 0;
    return (rating - prev) / (RATING_VALUES[level] - prev);
}

export function getRankColor(rating: number | null | undefined): string {
    return ratingTier(rating).text;
}

export function getRankBadgeColor(rating: number | null | undefined): string {
    return ratingTier(rating).bg;
}

export function getRatingChangeColor(change: number): string {
    if (change > 0) return 'text-emerald-500';
    if (change < 0) return 'text-rose-500';
    return 'text-zinc-400';
}

export function formatRatingChange(change: number): string {
    if (change > 0) return `+${change}`;
    return `${change}`;
}

/** v1 rating_name (English, as v1 shows it). */
export function getRankTitle(rating: number | null | undefined): string {
    return ratingTier(rating).title;
}

// Readable contest format names (keys from claoj/contest_format/registry.go).
// "default" is DMOJ's scoring, the format almost every CLAOJ contest uses.
const CONTEST_FORMAT_LABELS: Record<string, string> = {
    default: 'DMOJ', ioi: 'IOI', ioi16: 'IOI', icpc: 'ICPC', atcoder: 'AtCoder', ecoo: 'ECOO',
};

export function contestFormatLabel(format: string | null | undefined): string {
    if (!format) return CONTEST_FORMAT_LABELS.default;
    return CONTEST_FORMAT_LABELS[format] ?? format;
}
