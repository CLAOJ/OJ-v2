// The server (go-webauthn) sends options as {publicKey: {...}} with binary
// fields base64url-encoded, and expects the credential back in the standard
// JSON form (binary fields base64url). The browser API wants ArrayBuffers.

/* eslint-disable @typescript-eslint/no-explicit-any */

const fromB64url = (s: string): ArrayBuffer => {
    const b64 = s.replace(/-/g, '+').replace(/_/g, '/') + '='.repeat((4 - (s.length % 4)) % 4);
    return Uint8Array.from(atob(b64), c => c.charCodeAt(0)).buffer;
};

const toB64url = (buf: ArrayBuffer | null): string | null => {
    if (!buf) return null;
    let bin = '';
    new Uint8Array(buf).forEach(b => { bin += String.fromCharCode(b); });
    return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
};

const descriptors = (list?: any[]) => list?.map(c => ({ ...c, id: fromB64url(c.id) }));

export function creationOptions(options: any): PublicKeyCredentialCreationOptions {
    const pk = options.publicKey ?? options;
    return {
        ...pk,
        challenge: fromB64url(pk.challenge),
        user: { ...pk.user, id: fromB64url(pk.user.id) },
        excludeCredentials: descriptors(pk.excludeCredentials),
    };
}

export function requestOptions(options: any): PublicKeyCredentialRequestOptions {
    const pk = options.publicKey ?? options;
    return { ...pk, challenge: fromB64url(pk.challenge), allowCredentials: descriptors(pk.allowCredentials) };
}

export function credentialJSON(cred: PublicKeyCredential) {
    const r = cred.response as any;
    const response: Record<string, string | string[] | null> = { clientDataJSON: toB64url(r.clientDataJSON) };
    if ('attestationObject' in r) {
        response.attestationObject = toB64url(r.attestationObject);
        if (typeof r.getTransports === 'function') response.transports = r.getTransports();
    } else {
        response.authenticatorData = toB64url(r.authenticatorData);
        response.signature = toB64url(r.signature);
        response.userHandle = toB64url(r.userHandle);
    }
    return {
        id: cred.id,
        rawId: toB64url(cred.rawId),
        type: cred.type,
        clientExtensionResults: cred.getClientExtensionResults(),
        response,
    };
}
