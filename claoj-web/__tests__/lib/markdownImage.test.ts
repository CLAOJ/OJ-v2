import { insertImageMarkdown } from '@/lib/markdownImage';

describe('insertImageMarkdown', () => {
    it('inserts at the cursor when there is no selection', () => {
        expect(insertImageMarkdown('hello world', 5, 5, 'shot', '/api/media/a.png')).toEqual({
            value: 'hello![shot](/api/media/a.png) world',
            cursor: 30,
        });
    });

    it('replaces the selected text', () => {
        expect(insertImageMarkdown('hello world', 6, 11, 'shot', '/api/media/a.png')).toEqual({
            value: 'hello ![shot](/api/media/a.png)',
            cursor: 31,
        });
    });

    it('drops brackets from the alt text', () => {
        expect(insertImageMarkdown('', 0, 0, 'a[b]c', 'u').value).toBe('![abc](u)');
    });
});