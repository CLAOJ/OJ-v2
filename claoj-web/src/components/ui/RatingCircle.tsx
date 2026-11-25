import { cn, getRankColor, getRankTitle, ratingProgress } from '@/lib/utils';

/**
 * v1's rating circle (templates/user/rating.html): a ring in the tier colour,
 * filled from the bottom by progress through the tier, so the tier reads
 * from shape and fill as well as hue. The title names the tier.
 */
export function RatingCircle({ rating, className }: { rating: number | null | undefined; className?: string }) {
    if (rating == null) return null;
    const fill = 16 * ratingProgress(rating);
    return (
        <svg
            viewBox="0 0 16 16"
            className={cn('inline-block size-[0.9em] shrink-0 align-[-0.1em]', getRankColor(rating), className)}
            role="img"
            aria-label={`${getRankTitle(rating)} (${rating})`}
        >
            <title>{`${getRankTitle(rating)} (${rating})`}</title>
            <clipPath id={`rc-${rating}`}>
                <rect x="0" y={16 - fill} width="16" height={fill} />
            </clipPath>
            <circle cx="8" cy="8" r="6.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
            <circle cx="8" cy="8" r="6.5" fill="currentColor" clipPath={`url(#rc-${rating})`} />
        </svg>
    );
}
