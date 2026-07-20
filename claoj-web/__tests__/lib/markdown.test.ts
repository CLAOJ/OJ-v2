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

    it('converts [user:name] to a markdown link to the user page', () => {
        expect(normalizeDmojMarkdown('by [user:dinhwe2612] here')).toBe('by [dinhwe2612](/user/dinhwe2612) here');
    });

    it('converts [ruser:name] (rated user) to a user-page link', () => {
        expect(normalizeDmojMarkdown('[ruser:alice]')).toBe('[alice](/user/alice)');
    });

    it('handles usernames with digits and underscores', () => {
        expect(normalizeDmojMarkdown('[user:a_b12]')).toBe('[a_b12](/user/a_b12)');
    });

    it('does not convert unknown reference types like [problem:x] or [Link:x]', () => {
        expect(normalizeDmojMarkdown('[problem:aplusb] [Link:x]')).toBe('[problem:aplusb] [Link:x]');
    });

    it('does not convert user references inside code', () => {
        expect(normalizeDmojMarkdown('`[user:x]`')).toBe('`[user:x]`');
    });

    it('handles user references and math together', () => {
        expect(normalizeDmojMarkdown('[user:bob] solved ~N~ problems')).toBe('[bob](/user/bob) solved $N$ problems');
    });

    // DMOJ content frequently omits the space CommonMark requires after the
    // leading '#'s of an ATX heading (e.g. "##Input"). v1's markdown processor
    // rendered these as headings; remark does not, so we insert the space.
    it('inserts the missing space after a level-2 ATX heading', () => {
        expect(normalizeDmojMarkdown('##Đầu vào:')).toBe('## Đầu vào:');
    });

    it('inserts the missing space after a level-1 ATX heading', () => {
        expect(normalizeDmojMarkdown('#Title')).toBe('# Title');
    });

    it('leaves already-spaced headings untouched', () => {
        expect(normalizeDmojMarkdown('### Already spaced')).toBe('### Already spaced');
    });

    it('fixes ATX headings across multiple lines', () => {
        expect(normalizeDmojMarkdown('##Đầu vào:\ntext\n###Ràng buộc:')).toBe('## Đầu vào:\ntext\n### Ràng buộc:');
    });

    it('does not treat a mid-line hash as a heading', () => {
        expect(normalizeDmojMarkdown('the value #5 here')).toBe('the value #5 here');
    });

    it('does not fix headings inside fenced code blocks', () => {
        const src = '```\n##notheading\n```';
        expect(normalizeDmojMarkdown(src)).toBe(src);
    });

    // Legacy DMOJ statements contain literal <br>, <br/>, and the malformed
    // </br>. Without rehype-raw these render as literal text, so convert them
    // to a markdown hard line break.
    it('converts a malformed </br> tag to a hard line break', () => {
        expect(normalizeDmojMarkdown('line1</br>line2')).toBe('line1  \nline2');
    });

    it('converts <br> and <br/> variants to a hard line break', () => {
        expect(normalizeDmojMarkdown('a<br>b<br/>c<br />d')).toBe('a  \nb  \nc  \nd');
    });

    it('does not convert <br> inside inline code', () => {
        expect(normalizeDmojMarkdown('`a<br>b`')).toBe('`a<br>b`');
    });
});
