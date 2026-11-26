import { Fragment } from 'react';
import { Link } from '@/navigation';
import { cn, getRankColor } from '@/lib/utils';

/** v1 link_users: comma-separated profile links coloured by rating; the
 * display-rank style (prefix, colour) comes from NameStyles. */
export function UserLinks({ users }: { users: { username: string; rating?: number | null }[] }) {
    return (
        <>
            {users.map((u, i) => (
                <Fragment key={u.username}>
                    {i > 0 && ', '}
                    <Link href={`/user/${u.username}`} className={cn('font-semibold hover:underline', getRankColor(u.rating))}>
                        {u.username}
                    </Link>
                </Fragment>
            ))}
        </>
    );
}
