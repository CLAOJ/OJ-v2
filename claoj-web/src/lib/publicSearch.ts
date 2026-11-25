// Picker sources for the organization create pages. The admin search helpers
// hit staff-only endpoints, so a permitted non-staff org admin gets these
// instead: the org's own problem list (member-gated) and the public user list.
import api from '@/lib/api';
import type { EntityOption } from '@/components/admin/EntityPicker';
import type { AdminLicense } from '@/types';

const LIMIT = 20;

/** Problems of one organization, for the org contest form's problem picker. */
export function searchOrganizationProblems(orgId: number) {
    return async (term: string): Promise<EntityOption[]> => {
        const res = await api.get(`/organization/${orgId}/problems`, {
            params: { search: term, page_size: LIMIT },
        });
        const rows = res.data.data as { id: number; code: string; name: string }[];
        return rows.map(p => ({ id: p.id, label: p.code, detail: p.name }));
    };
}

/** Public user list search (excludes unlisted profiles), for staff pickers. */
export async function searchPublicUsers(term: string): Promise<EntityOption[]> {
    const res = await api.get('/users', {
        params: { search: term, page_size: LIMIT },
    });
    const rows = res.data.data as { id: number; username: string }[];
    return rows.map(u => ({ id: u.id, label: u.username }));
}

/** Public license metadata for the shared problem form's license select. */
export async function fetchPublicLicenses(): Promise<{ data: AdminLicense[] }> {
    const res = await api.get('/licenses');
    return res.data;
}