import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}

export function formatTime(seconds: number): string {
    if (seconds < 60) return `${Math.round(seconds)}s`;
    const mins = Math.floor(seconds / 60);
    const secs = Math.round(seconds % 60);
    return `${mins}m ${secs}s`;
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

export function getStatusVariant(status: string): any {
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

export function getRankColor(rating: number | null): string {
    if (rating === null) return 'text-zinc-500';
    if (rating < 1200) return 'text-zinc-500';
    if (rating < 1400) return 'text-emerald-500';
    if (rating < 1600) return 'text-blue-500';
    if (rating < 1900) return 'text-purple-500';
    if (rating < 2100) return 'text-amber-500';
    if (rating < 2400) return 'text-orange-500';
    return 'text-red-600';
}

export function getRankBadgeColor(rating: number | null): string {
    if (rating === null) return 'bg-zinc-500';
    if (rating < 1200) return 'bg-zinc-500';
    if (rating < 1400) return 'bg-emerald-500';
    if (rating < 1600) return 'bg-blue-500';
    if (rating < 1900) return 'bg-purple-500';
    if (rating < 2100) return 'bg-amber-500';
    if (rating < 2400) return 'bg-orange-500';
    return 'bg-red-600';
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

export function getRankTitle(rating: number | null): string {
    if (rating === null) return 'Newbie';
    if (rating < 1200) return 'Newbie';
    if (rating < 1400) return 'Pupil';
    if (rating < 1600) return 'Specialist';
    if (rating < 1900) return 'Expert';
    if (rating < 2100) return 'Candidate Master';
    if (rating < 2400) return 'Master';
    return 'Grandmaster';
}
