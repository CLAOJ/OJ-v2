import { statusKey, VERDICT_CODES } from '@/lib/submissionStatus';

describe('VERDICT_CODES', () => {
    it('lists the codes the judge stores, not the legacy RE alias', () => {
        expect(VERDICT_CODES).toEqual([
            'AC', 'WA', 'TLE', 'MLE', 'OLE', 'IR', 'RTE', 'CE', 'IE', 'SC', 'AB',
        ]);
        expect(VERDICT_CODES).not.toContain('RE');
    });

    it('every filter code renders a known label', () => {
        for (const code of VERDICT_CODES) {
            expect(statusKey(code)).toBe(code);
        }
        // Old links may still carry v1's alias.
        expect(statusKey('RE')).toBe('RE');
    });
});