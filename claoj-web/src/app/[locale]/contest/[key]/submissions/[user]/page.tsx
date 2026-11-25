import { redirect } from 'next/navigation';

// v1 /contest/<key>/submissions/<user>/: the contest-scoped submission list
// for one participant. v2 serves it from the public list with ?contest and
// ?user, which already works with both filters together.
export default async function ContestUserSubmissionsPage({
    params,
}: {
    params: Promise<{ key: string; user: string }>;
}) {
    const { key, user } = await params;
    redirect(`/submissions?contest=${encodeURIComponent(key)}&user=${encodeURIComponent(user)}`);
}