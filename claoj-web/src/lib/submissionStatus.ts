// Verdict codes come from the judge; labels are translated via the
// Submissions.status.* namespace. Use: t(`status.${statusKey(code)}`)
//
// 'RE' is a legacy alias for Runtime Error still used by some filters
// (e.g. src/app/[locale]/submissions/page.tsx) alongside 'RTE'; it was
// found in the existing STATUS_NAMES/STATUS_INFO maps this task replaces,
// so it's kept as a known code mapping to the same label as 'RTE'.
const KNOWN = new Set([
    'AC', 'WA', 'TLE', 'MLE', 'OLE', 'IR', 'RTE', 'RE', 'CE', 'IE', 'AB',
    'QU', 'P', 'G', 'D', 'CP', 'SC',
]);

export const statusKey = (code: string): string => (KNOWN.has(code) ? code : 'unknown');
