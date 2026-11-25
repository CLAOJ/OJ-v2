// Verdict codes come from the judge; labels are translated via the
// Submissions.status.* namespace. Use: t(`status.${statusKey(code)}`)
//
// 'RE' is v1's legacy alias for Runtime Error. The database stores 'RTE' and
// the filters offer VERDICT_CODES, but old links and v1 exports still say 'RE',
// so the alias maps to the same label.
const KNOWN = new Set([
    'AC', 'WA', 'TLE', 'MLE', 'OLE', 'IR', 'RTE', 'RE', 'CE', 'IE', 'AB',
    'QU', 'P', 'G', 'D', 'CP', 'SC',
]);

// The judge's verdict codes in display order (same list the admin filter and
// edit form offer). DB result values, not v1's legacy aliases: runtime errors
// are stored as RTE, and IR/SC are real verdicts.
export const VERDICT_CODES = [
    'AC', 'WA', 'TLE', 'MLE', 'OLE', 'IR', 'RTE', 'CE', 'IE', 'SC', 'AB',
] as const;

export const statusKey = (code: string): string => (KNOWN.has(code) ? code : 'unknown');
