import { nameStylesCss } from '@/components/providers/NameStyles';

describe('nameStylesCss', () => {
    it('styles links to users with a configured rank, escaping user data', () => {
        const css = nameStylesCss({
            styles: {
                staff: { color: '#ff888a', bold: true, strike: false, prefix: '🔥' },
                bad: { color: 'red;}body{display:none', bold: false, strike: false, prefix: '"}x' },
            },
            users: { alice: 'staff', 'ev"il': 'bad', bob: 'unknown' },
        });
        expect(css).toContain('color:#ff888a!important;font-weight:700!important');
        expect(css).toContain('content:"🔥"');
        expect(css).not.toContain('display:none');
        expect(css).toContain('a[href$="/user/ev\\"il"]');
        expect(css).toContain('content:"\\"}x"');
        expect(css).not.toContain('bob');
        // A link that marks its name styles only that element.
        expect(css).toContain('a[href$="/user/alice"] [data-name]::before');
        expect(css).toContain('a[href$="/user/alice"]:not(:has(img,[data-name])):not([role=menuitem],[data-plain])');
    });

    it('sets the configured decoration, and none when unset', () => {
        const css = (s: object) => nameStylesCss({ styles: { r: { color: '', bold: false, strike: false, prefix: '', ...s } }, users: { a: 'r' } });
        expect(css({})).toContain('text-decoration:none!important');
        expect(css({ underline: true })).toContain('text-decoration:underline!important');
        expect(css({ underline: true, strike: true })).toContain('text-decoration:underline line-through!important');
    });
});
