import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';

// The OAuth providers the server has configured; buttons for the rest would
// only lead to an error, so pages render just these.
export function useOAuthProviders(): string[] {
    const { data } = useQuery({
        queryKey: ['oauth-providers'],
        queryFn: async () => (await api.get<{ providers: string[] }>('/auth/oauth-providers')).data.providers,
        staleTime: Infinity,
    });
    return data ?? [];
}
