/**
 * Flat page path normalization: the web catch-all route receives the URL path
 * as segments and Django stores flatpage urls with a leading and trailing
 * slash ("/about/"), so both are enforced before the lookup.
 */
export function normalizeFlatPagePath(raw: string | string[]): string {
    const joined = Array.isArray(raw) ? raw.join('/') : raw;
    const trimmed = joined.trim();
    const withLeading = trimmed.startsWith('/') ? trimmed : '/' + trimmed;
    const collapsed = withLeading.replace(/\/{2,}/g, '/');
    return collapsed.endsWith('/') ? collapsed : collapsed + '/';
}