'use client';

import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';

export interface NameStyle {
    color: string;
    bold: boolean;
    strike: boolean;
    prefix: string;
}

export interface NameStylesData {
    styles: Record<string, NameStyle>;
    users: Record<string, string>;
}

const COLOR = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i;
const cssString = (s: string) => `"${s.replace(/[\\"]/g, '\\$&').replace(/[\r\n<>]/g, '')}"`;

// nameStylesCss turns the admin-configured rank styles (v1 ranks.scss) into
// rules on every link to a styled user's profile, so no page needs to know
// about display ranks. Avatar links (containing an <img>) are left alone.
export function nameStylesCss({ styles, users }: NameStylesData): string {
    const byRank: Record<string, string[]> = {};
    for (const [username, rank] of Object.entries(users)) {
        if (!styles[rank]) continue;
        (byRank[rank] ??= []).push(`a[href$=${cssString('/user/' + username)}]:not(:has(img))`);
    }
    const out: string[] = [];
    for (const [rank, sels] of Object.entries(byRank)) {
        const s = styles[rank];
        const decl = [
            s.color && COLOR.test(s.color) ? `color:${s.color}!important` : '',
            s.bold ? 'font-weight:700!important' : '',
            s.strike ? 'text-decoration:line-through!important' : '',
        ].filter(Boolean).join(';');
        if (decl) out.push(`${sels.flatMap(x => [x, `${x} *`]).join(',')}{${decl}}`);
        if (s.prefix) out.push(`${sels.map(x => `${x}::before`).join(',')}{content:${cssString(s.prefix)};margin-right:.2em}`);
    }
    return out.join('\n');
}

export function useNameStyles() {
    return useQuery({
        queryKey: ['name-styles'],
        queryFn: async () => (await api.get<NameStylesData>('/name-styles')).data,
        staleTime: 5 * 60 * 1000,
    });
}

export function NameStyles() {
    const { data } = useNameStyles();
    if (!data) return null;
    return <style>{nameStylesCss(data)}</style>;
}
