import { creationOptions, credentialJSON } from '@/lib/webauthn';

describe('webauthn helpers', () => {
    it('decodes base64url options from {publicKey} and encodes the credential back', () => {
        // "-__8" is bytes fb ff fc: url-safe chars a plain atob rejects.
        const opts = creationOptions({ publicKey: { challenge: '-__8', user: { id: 'AQ', name: 'u', displayName: 'u' }, rp: { name: 'x' }, pubKeyCredParams: [] } });
        expect(Array.from(new Uint8Array(opts.challenge as ArrayBuffer))).toEqual([0xfb, 0xff, 0xfc]);
        expect(Array.from(new Uint8Array(opts.user.id as ArrayBuffer))).toEqual([1]);

        const buf = (a: number[]) => new Uint8Array(a).buffer;
        const json = credentialJSON({
            id: 'id', type: 'public-key', rawId: buf([0xfb, 0xff, 0xfc]),
            getClientExtensionResults: () => ({}),
            response: { clientDataJSON: buf([1]), attestationObject: buf([2, 3]) },
        } as unknown as PublicKeyCredential);
        expect(json.rawId).toBe('-__8');
        expect(json.response).toEqual({ clientDataJSON: 'AQ', attestationObject: 'AgM' });
    });
});
