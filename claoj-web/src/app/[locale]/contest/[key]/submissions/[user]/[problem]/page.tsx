import { redirect } from 'next/navigation';

// v1 /contest/<key>/submissions/<user>/<problem>/: one participant's
// submissions for one contest problem.
export default async function ContestUserProblemSubmissionsPage({
    params,
}: {
    params: Promise<{ key: string; user: string; problem: string }>;
}) {
    const { key, user, problem } = await params;
    redirect(
        `/submissions?contest=${encodeURIComponent(key)}&user=${encodeURIComponent(user)}&problem=${encodeURIComponent(problem)}`
    );
}