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
 * It also repairs two other DMOJ-isms that remark renders wrong: ATX headings
 * written without the space CommonMark requires (`##Input` -> `## Input`) and
 * literal `<br>` / `</br>` tags (which render as raw text without rehype-raw).
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
        .map((segment, i) =>
            i % 2 === 1
                ? segment
                : convertInlineMath(convertUserReferences(convertLineBreaks(convertHeadings(segment)))))
        .join('');
}

// `##Heading` -> `## Heading`. DMOJ statements often omit the space CommonMark
// requires after the leading 1-6 `#`s, so remark renders the line as literal
// text. Only fires at the start of a line (multiline `^`) when a non-space,
// non-`#` character follows, so mid-line hashes (`#5`) are left alone.
function convertHeadings(text: string): string {
    return text.replace(/^(#{1,6})(?=[^\s#])/gm, '$1 ');
}

// `<br>`, `<br/>`, `<br />`, and the malformed `</br>` -> a markdown hard line
// break (two trailing spaces + newline). These render as literal text when
// rehype-raw is disabled, so normalizing keeps the intended break in both modes.
function convertLineBreaks(text: string): string {
    return text.replace(/<\/?br\s*\/?>/gi, '  \n');
}

// `~latex~` -> `$latex$`. The lookarounds keep double-tilde runs (`~~strike~~`)
// intact, and `[^~\n]+?` stops a span from swallowing newlines or a stray tilde.
function convertInlineMath(text: string): string {
    return text.replace(/(?<!~)~(?!~)([^~\n]+?)~(?!~)/g, (_match, body) => `$${body}$`);
}

// DMOJ reference syntax `[user:name]` / `[ruser:name]` (see judge/jinja2/reference.py,
// regex \[(r?user):(\w+)\]) -> a plain markdown link to the user's page. The rated
// `ruser` badge is rendered as a normal link here; only `user`/`ruser` are recognized,
// so `[problem:x]`, `[Link:x]`, etc. pass through untouched.
function convertUserReferences(text: string): string {
    return text.replace(/\[r?user:([\p{L}\p{N}_]+)\]/gu, (_match, name) => `[${name}](/user/${name})`);
}
