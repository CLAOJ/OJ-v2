import api, { type ApiError } from './api';

type Signed = { upload_url: string; key: string; content_type: string };

/**
 * Sends a file straight from the browser to R2: the API signs a short-lived PUT
 * URL, the file goes to R2 (never through the server), then `commit` asks the
 * API to verify and keep it. Resolves null when the server has no R2 (501),
 * so the caller can fall back to its multipart upload.
 */
export async function uploadDirect<T>(signPath: string, file: File, commit: (key: string) => Promise<T>): Promise<T | null> {
    let signed: Signed;
    try {
        signed = (await api.post<Signed>(signPath, { content_type: file.type, size: file.size })).data;
    } catch (e) {
        if ((e as ApiError).response?.status === 501) return null;
        throw e;
    }
    const put = await fetch(signed.upload_url, { method: 'PUT', body: file, headers: { 'Content-Type': signed.content_type } });
    if (!put.ok) throw new Error(`upload failed (${put.status})`);
    return commit(signed.key);
}
