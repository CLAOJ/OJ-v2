// Server-side search functions for EntityPicker.
import { adminUserApi, adminProblemApi, adminContestApi } from '@/lib/adminApi';
import type { EntityOption } from '@/components/admin/EntityPicker';

const LIMIT = 20;

export async function searchUsers(term: string): Promise<EntityOption[]> {
    const res = await adminUserApi.list(1, LIMIT, term);
    return res.data.data.map(u => ({ id: u.id, label: u.username }));
}

export async function searchProblems(term: string): Promise<EntityOption[]> {
    const res = await adminProblemApi.list(1, LIMIT, { search: term });
    return res.data.data.map(p => ({ id: p.id, label: p.code, detail: p.name }));
}

export async function searchContests(term: string): Promise<EntityOption[]> {
    const res = await adminContestApi.list(1, LIMIT, { search: term });
    return res.data.data.map(c => ({ id: c.id, label: c.key, detail: c.name }));
}
