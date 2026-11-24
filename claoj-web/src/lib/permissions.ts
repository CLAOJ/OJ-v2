import type { User } from '@/types';

/** Mirrors the API's auth.HasPerm: superusers hold every codename. */
export function hasPerm(user: User | null | undefined, codename: string): boolean {
    return !!user && (!!user.is_superuser || (user.permissions ?? []).includes(codename));
}