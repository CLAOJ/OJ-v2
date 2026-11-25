import { ratingLevel, ratingProgress, getRankTitle, getRankColor } from '@/lib/utils';

// Boundaries must match v1 OJ/judge/ratings.py (bisect_right over RATING_VALUES).
describe('rating tiers match v1', () => {
    it.each([
        [0, 'Newbie'], [1199, 'Newbie'], [1200, 'Pupil'], [1599, 'Specialist'],
        [1600, 'Expert'], [1899, 'Expert'], [1900, 'Candidate Master'], [2150, 'Candidate Master'],
        [2200, 'Master'], [2300, 'International Master'], [2441, 'Grandmaster'],
        [2600, 'International Grandmaster'], [2900, 'Legendary Grandmaster'],
    ])('%i is %s', (rating, title) => {
        expect(getRankTitle(rating)).toBe(title);
    });

    it('unrated users are not coloured as a tier', () => {
        expect(getRankTitle(null)).toBe('Unrated');
        expect(getRankColor(null)).toContain('zinc');
    });

    it('progress runs through the current tier like v1', () => {
        expect(ratingLevel(1300)).toBe(1);
        expect(ratingProgress(1300)).toBeCloseTo(0.5);
        expect(ratingProgress(3500)).toBe(1);
    });
});
