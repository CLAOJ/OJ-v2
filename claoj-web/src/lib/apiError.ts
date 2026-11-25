/** Reads the API's { error } message from a failed axios request. */
export function apiErrorMessage(error: unknown, fallback: string): string {
    const message = (error as { response?: { data?: { error?: unknown } } })?.response?.data?.error;
    return typeof message === 'string' && message !== '' ? message : fallback;
}