export type ContestPhase = 'upcoming' | 'ongoing' | 'past';

/**
 * Phase of a contest at `now`. A missing or unparseable time falls through to
 * 'ongoing' so a malformed row still renders instead of disappearing.
 */
export function contestPhase(
    start: string | null | undefined,
    end: string | null | undefined,
    now: number = Date.now()
): ContestPhase {
    const startsAt = start ? Date.parse(start) : NaN;
    const endsAt = end ? Date.parse(end) : NaN;
    if (!Number.isNaN(endsAt) && now > endsAt) return 'past';
    if (!Number.isNaN(startsAt) && now < startsAt) return 'upcoming';
    return 'ongoing';
}