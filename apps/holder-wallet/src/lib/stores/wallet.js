/**
 * Wallet Store — holder-wallet state management
 * ===============================================
 * Persists credentials to localStorage (synthetic data only for hackathon).
 *
 * TODO (next agent): wire loadCredentials() to issuer-api GET /credentials
 * filtered by the holder's local DID. For now it initialises from localStorage
 * with an optional fallback to mock data.
 */

import { writable, derived } from 'svelte/store';

// ── Credential store ─────────────────────────────────────────────────────────

function createWalletStore() {
  const stored = localStorage.getItem('tp_wallet_credentials');
  const initial = stored ? JSON.parse(stored) : [];

  const { subscribe, set, update } = writable(initial);

  // Persist on every change
  subscribe(creds => {
    localStorage.setItem('tp_wallet_credentials', JSON.stringify(creds));
  });

  return {
    subscribe,

    /** Replace all credentials (called after fetching from issuer-api) */
    setCredentials(creds) {
      set(creds);
    },

    /** Add or update a single credential by id */
    upsert(credential) {
      update(creds => {
        const idx = creds.findIndex(c => c.id === credential.id);
        if (idx >= 0) {
          creds[idx] = credential;
          return [...creds];
        }
        return [credential, ...creds];
      });
    },

    /** Remove a credential by id */
    remove(id) {
      update(creds => creds.filter(c => c.id !== id));
    },
  };
}

export const wallet = createWalletStore();

// ── Derived: active (non-revoked, non-expired) credentials ───────────────────
export const activeCredentials = derived(wallet, $creds =>
  $creds.filter(c => !c.revoked && (!c.expiresAt || new Date(c.expiresAt) > new Date()))
);

// ── Active claim request (set when holder receives a verifier QR) ────────────
export const pendingClaimRequest = writable(null);
/**
 * Shape of pendingClaimRequest:
 * {
 *   sessionToken: string,
 *   claimRequest: { schemaName, attributeName, operator, value },
 *   expiresAt: string (ISO)
 * }
 */

// ── Proof QR result (set after proof generation) ────────────────────────────
export const proofQrPayload = writable(null);
/**
 * Shape of proofQrPayload:
 * {
 *   encodedProof: string,   — JSON string to encode into QR
 *   sessionToken: string,
 *   expiresAt: string
 * }
 */
