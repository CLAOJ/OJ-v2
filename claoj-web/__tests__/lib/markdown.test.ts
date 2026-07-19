import { normalizeDmojMarkdown } from '@/lib/markdown';

describe('normalizeDmojMarkdown', () => {
    it('converts DMOJ inline tilde math to $...$', () => {
        expect(normalizeDmojMarkdown('the value ~N~ here')).toBe('the value $N$ here');
    });

    it('converts inline math containing LaTeX commands', () => {
        expect(normalizeDmojMarkdown('~1 \\le N \\le 100\\,000~')).toBe('$1 \\le N \\le 100\\,000$');
    });

    it('handles multiple inline math spans on one line', () => {
        expect(normalizeDmojMarkdown('~a~ and ~b~')).toBe('$a$ and $b$');
    });

    it('leaves $$...$$ display math untouched', () => {
        expect(normalizeDmojMarkdown('$$x^2 + y^2$$')).toBe('$$x^2 + y^2$$');
    });

    it('does not convert ~~strikethrough~~ (double tilde)', () => {
        expect(normalizeDmojMarkdown('~~gone~~')).toBe('~~gone~~');
    });

    it('does not touch tildes inside inline code', () => {
        expect(normalizeDmojMarkdown('run `sum ~N~ items`')).toBe('run `sum ~N~ items`');
    });

    it('does not touch tildes inside fenced code blocks', () => {
        const src = '```\n~N~ is not math here\n```';
        expect(normalizeDmojMarkdown(src)).toBe(src);
    });

    it('converts math outside code while preserving code', () => {
        expect(normalizeDmojMarkdown('~N~ then `~x~` then ~M~')).toBe('$N$ then `~x~` then $M$');
    });

    it('returns empty input unchanged', () => {
        expect(normalizeDmojMarkdown('')).toBe('');
    });
});
