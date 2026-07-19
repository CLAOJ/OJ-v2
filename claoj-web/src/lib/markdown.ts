/**
 * Normalize CLAOJ/DMOJ-flavored markdown into the dialect that
 * react-markdown + remark-math understands.
 *
 * CLAOJ inherits DMOJ's math conventions: inline math is delimited with a
 * single tilde (`~x~`) and display math with `$$...$$`
 * (see judge/utils/mathoid.py in the v1 codebase). remark-math only speaks
 * `$...$` / `$$...$$`, and remark-gfm would otherwise treat `~x~` as
 * strikethrough — so without this step every inline formula renders as
 * struck-through raw LaTeX.
 *
 * This rewrites inline `~...~` to `$...$`, leaves `$$...$$` display math and
 * `~~strikethrough~~` alone, and never touches text inside inline code spans
 * or fenced code blocks.
 */
export function normalizeDmojMarkdown(content: string): string {
    if (!content) return content;

    // Split out code so tildes inside it are preserved verbatim. A capturing
    // group means the code segments land at odd indices of the result array.
    const codePattern = /(```[\s\S]*?```|~~~[\s\S]*?~~~|`[^`\n]+`)/g;

    return content
        .split(codePattern)
        .map((segment, i) => (i % 2 === 1 ? segment : convertInlineMath(segment)))
        .join('');
}

// `~latex~` -> `$latex$`. The lookarounds keep double-tilde runs (`~~strike~~`)
// intact, and `[^~\n]+?` stops a span from swallowing newlines or a stray tilde.
function convertInlineMath(text: string): string {
    return text.replace(/(?<!~)~(?!~)([^~\n]+?)~(?!~)/g, (_match, body) => `$${body}$`);
}
