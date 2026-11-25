'use client';

import UsersListPage from '../users/page';

/** v1 `/contributors`: the user list ranked by contribution points. */
export default function ContributorsPage() {
    return <UsersListPage initialRank="contribution" />;
}