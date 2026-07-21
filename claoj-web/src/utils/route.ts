/**
 * Extract the leading numeric id from a route segment of the form `<id>-<slug>`.
 * v1 URLs look like `/post/93-lấytiền` and `/organization/1-itcla`; the backend
 * API is keyed by the numeric id only, so pages parse it out with this helper.
 */
export function parseLeadingId(segment: string): string {
  const dash = segment.indexOf('-');
  return dash === -1 ? segment : segment.slice(0, dash);
}
