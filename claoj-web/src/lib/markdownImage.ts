/**
 * Insert a markdown image at the current textarea selection and return the
 * new value plus the cursor position just after the inserted snippet.
 */
export function insertImageMarkdown(value: string, start: number, end: number, alt: string, url: string) {
    const snippet = `![${alt.replace(/[[\]]/g, '')}](${url})`;
    return {
        value: value.slice(0, start) + snippet + value.slice(end),
        cursor: start + snippet.length,
    };
}