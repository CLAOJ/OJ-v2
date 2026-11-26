/** Text edits for the markdown toolbar. Each returns the new value and the
 * selection to restore, so the caller can keep the cursor where it belongs. */
export interface Edit {
    value: string;
    start: number;
    end: number;
}

/** Wraps the selection (or a placeholder when nothing is selected) in
 * `before`/`after`, leaving the wrapped text selected. */
export function wrapSelection(value: string, start: number, end: number, before: string, after: string, placeholder: string): Edit {
    const text = value.slice(start, end) || placeholder;
    return {
        value: value.slice(0, start) + before + text + after + value.slice(end),
        start: start + before.length,
        end: start + before.length + text.length,
    };
}

/** Prefixes every line touched by the selection (`> `, `- `, `1. `, `### `). */
export function prefixLines(value: string, start: number, end: number, prefix: string | ((i: number) => string)): Edit {
    const lineStart = value.lastIndexOf('\n', start - 1) + 1;
    const nl = value.indexOf('\n', end);
    const lineEnd = nl === -1 ? value.length : nl;
    const lines = value.slice(lineStart, lineEnd).split('\n');
    const block = lines.map((l, i) => (typeof prefix === 'string' ? prefix : prefix(i)) + l).join('\n');
    return {
        value: value.slice(0, lineStart) + block + value.slice(lineEnd),
        start: lineStart,
        end: lineStart + block.length,
    };
}

/** Replaces the selection with `text`, cursor after it. */
export function insertText(value: string, start: number, end: number, text: string): Edit {
    const cursor = start + text.length;
    return { value: value.slice(0, start) + text + value.slice(end), start: cursor, end: cursor };
}
