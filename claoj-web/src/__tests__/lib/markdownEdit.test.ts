import { insertText, prefixLines, wrapSelection } from '@/lib/markdownEdit';

describe('markdownEdit', () => {
    it('wraps the selection and keeps it selected', () => {
        expect(wrapSelection('a bc d', 2, 4, '**', '**', 'bold')).toEqual({ value: 'a **bc** d', start: 4, end: 6 });
    });

    it('wraps a placeholder when nothing is selected', () => {
        const e = wrapSelection('ab', 1, 1, '`', '`', 'code');
        expect(e.value).toBe('a`code`b');
        expect(e.value.slice(e.start, e.end)).toBe('code');
    });

    it('prefixes every touched line', () => {
        expect(prefixLines('x\none\ntwo\ny', 4, 8, '> ').value).toBe('x\n> one\n> two\ny');
        expect(prefixLines('a\nb', 0, 3, i => `${i + 1}. `).value).toBe('1. a\n2. b');
        expect(prefixLines('', 0, 0, '- ').value).toBe('- ');
    });

    it('inserts text at the cursor', () => {
        expect(insertText('ab', 1, 1, 'X')).toEqual({ value: 'aXb', start: 2, end: 2 });
    });
});
