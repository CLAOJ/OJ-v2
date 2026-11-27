import axios from 'axios';
import { cookies } from 'next/headers';
import api from '@/lib/api';
import SubmissionPageContent, { type SubmissionDetail } from './SubmissionPageContent';

// Submissions are permission-gated, so this render is per visitor: fetched with
// the visitor's cookies, and never put in the shared nginx page cache (v2.conf
// only shares the public list/detail pages). Plain axios on purpose: the shared
// client's 401 refresh queue is module-global, i.e. shared by every visitor this
// worker serves. Any failure (logged out, expired token, 403) falls back to the
// client-side fetch, which can refresh the session.
async function fetchSubmission(id: string): Promise<SubmissionDetail | undefined> {
    const store = await cookies();
    if (!store.has('access_token')) return undefined;
    return axios
        .get<SubmissionDetail>(`${api.defaults.baseURL}/submission/${id}`, {
            headers: { Cookie: store.toString() },
            timeout: 3000,
        })
        .then(r => r.data)
        .catch(() => undefined);
}

export default async function SubmissionPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = await params;
    return <SubmissionPageContent params={params} initial={await fetchSubmission(id)} />;
}
