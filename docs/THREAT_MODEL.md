# TrustPass Threat Model & Security Architecture

**Authoritative Security Reference for Track B: "Proving a Fact Without Revealing the Whole Record"**  
**Core Question Answered:** *What stops a false yes?*

---

## 1. Executive Summary

In traditional identity systems, verification relies on presenting physical or digital documents (passports, driver's licenses, student ID cards). This introduces two fatal flaws:
1. **Over-disclosure:** Verifiers receive sensitive personal records (full names, home addresses, dates of birth, national identity numbers) when they only required an answer to a single boolean question (e.g., "is age $\ge 18$?").
2. **Fragile Security:** A verifier that inspects a document can easily be fooled by high-resolution visual counterfeits, forged inspection screens, or borrowed credentials.

TrustPass replaces visual inspection and record transmission with **zero-knowledge cryptographic proofs** over W3C Verifiable Credentials using BBS+ selective disclosure and Noir ZK circuits.

A verification is only successful if and only if:
$$\text{Valid}(proof) \iff \text{ValidSig}(issuer, cred) \land \text{SatisfiesPredicate}(attrib, op, threshold) \land \text{ValidSession}(token) \land \neg\text{Expired}(cred) \land \neg\text{Revoked}(cred)$$

---

## 2. Threat Actor Taxonomy

We evaluate the system against five adversarial profiles:

| Threat Actor | Capabilities | Objective |
|---|---|---|
| **Dishonest Holder** | Possesses valid/expired/invalid credentials; controls holder device; can modify client code. | Falsely satisfy an identity check (e.g. claim age $\ge 18$ when under 18; claim GPA $\ge 3.50$ when lower). |
| **Replay Attacker** | Network eavesdropper or bystander who captures a valid proof QR code from another user. | Replay a legitimate user's proof to gain unauthorized access or benefits. |
| **Malicious Verifier** | Operates a verifier terminal; has access to receipts database and local logs. | Harvest personal data, correlate holder visits across multiple sessions, or deanonymize holders. |
| **Revoked / Suspended User** | Possesses a credential that was historically valid but has been formally revoked or has expired. | Continue using the invalidated credential at unsuspecting verifier checkpoints. |
| **Rogue / Impersonating Issuer** | Untrusted third-party generating arbitrary signatures with their own keys. | Issue fraudulent credentials claiming to be a trusted authority (e.g. National Identity Office). |

---

## 3. What Stops a False Yes? (Defense-in-Depth Analysis)

The fundamental integrity requirement of TrustPass is that **no adversary can cause a verifier to display a `VERIFIED` state unless a legitimate issuer signed authentic attributes that strictly satisfy the verifier's predicate in an active, unexpired, single-use session.**

Below is the detailed breakdown of the cryptographic and architectural barriers preventing a "false yes":

```
[Incoming Verification Request]
               │
               ▼
   [Gate 1: Session Token Valid & Active?] ──NO──► Reject: SessionTokenExpired / Reused
               │ YES
               ▼
   [Gate 2: Issuer Public Key Trusted?] ───NO──► Reject: SignatureInvalid
               │ YES
               ▼
   [Gate 3: Credential Unexpired?] ────────NO──► Reject: CredentialExpired
               │ YES
               ▼
   [Gate 4: Credential Not Revoked?] ──────NO──► Reject: CredentialRevoked
               │ YES
               ▼
   [Gate 5: BBS+ / Noir Proof Sound?] ─────NO──► Reject: SignatureInvalid / PredicateNotSatisfied
               │ YES
               ▼
   [Gate 6: Session Token Bound in Fiat-Shamir?] ─NO─► Reject: SessionTokenReused
               │ YES
               ▼
        [RESULT: VERIFIED] ──► Generate Non-Personal Audit Receipt
```

### 3.1 Vector 1: Cryptographic Forgery of Credentials
- **Attack:** A holder alters their birth date or GPA in the credential JSON and attempts to generate a proof.
- **Defense:**
  - Credentials are signed using BBS+ signatures over the pairing-friendly elliptic curve **BLS12-381** (`ark-bls12-381`).
  - BBS+ provides 128-bit computational security under the $q$-Strong Diffie-Hellman ($q$-SDH) assumption.
  - Computing a valid signature for an altered attribute vector without knowledge of the issuer's private key $x \in \mathbb{F}_q$ is computationally infeasible.
  - Verification rejects forged credentials with typed error: `SignatureInvalid`.

### 3.2 Vector 2: Replay of a Valid Past Proof
- **Attack:** An adversary photographs a valid QR code displayed on a holder's phone or intercepts the network payload, then presents it to the verifier seconds or minutes later.
- **Defense:**
  - Proofs are bound to an ephemeral, cryptographically random **session token** (UUIDv4) issued by `services/verifier-api`.
  - When the holder generates a Proof of Knowledge (`bbs_plus::proof::PoKOfSignatureG1Protocol`), the Fiat-Shamir challenge $c$ is computed over:
    $$c = H(pk \parallel \text{pok.challenge\_contribution} \parallel session\_token \parallel schema\_name \parallel attribute\_name)$$
  - During verification, `core` independently reconstructs $c$ using the expected session token provided by the verifier terminal.
  - If an adversary presents a proof generated for session $A$ to session $B$, the challenges will not match:
    $$c_{\text{proof}} \ne c_{\text{expected}}$$
  - Proof verification fails algebraically: `SignatureInvalid` / `SessionTokenExpired`.

### 3.3 Vector 3: Concurrent Session Reuse & Race Attacks
- **Attack:** An adversary tries to submit the same valid proof simultaneously to multiple terminals before the session expires.
- **Defense:**
  - `services/verifier-api` implements atomic single-use session consumption (`session.Store.ConsumeSession(token)`).
  - The first verification attempt marks `consumed = true`. Any subsequent attempt using the same session token is immediately blocked before cryptographic verification begins.
  - Rejection returns typed error: `SessionTokenReused`.

### 3.4 Vector 4: Stale / Pre-generated Proofs
- **Attack:** A user pre-generates a proof at home and presents it hours later at a store.
- **Defense:**
  - Verification sessions have an enforced time-to-live (`VERIFICATION_SESSION_TTL_SECONDS = 120s`).
  - Once the TTL elapses, the session token is purged or marked invalid.
  - Rejection returns typed error: `SessionTokenExpired`.

### 3.5 Vector 5: Claim Request / Predicate Tampering
- **Attack:** A holder modifies the predicate threshold (e.g. changing `threshold = 18` to `threshold = 10` or changing `operator = GTE` to `operator = LTE`).
- **Defense:**
  - The claim request parameters (`schema_name`, `attribute_name`, `operator`, `threshold`) are embedded into the Fiat-Shamir challenge bytes and signed into the proof context.
  - The verifier validates the proof against *its own* stored claim request, not whatever the holder claims was requested.
  - If the predicate evaluation fails against the true threshold, `core/src/predicate/engine.rs` returns typed error: `PredicateNotSatisfied`.

### 3.6 Vector 6: Expired Credential Presentation
- **Attack:** A holder presents a credential whose `expirationDate` has passed.
- **Defense:**
  - `core/src/credential/vc.rs` extracts the `expirationDate` attribute and compares it deterministically against UTC now.
  - If `now > expirationDate`, verification halts immediately with typed error: `CredentialExpired`.

### 3.7 Vector 7: Revoked Credential Presentation
- **Attack:** A university or national agency revokes a credential (e.g. student expelled or passport cancelled), but the holder still possesses the signed JSON in their wallet.
- **Defense:**
  - `core` verifies the credential's revocation status against the issuer's revocation registry (`verify_proof_internal`).
  - If the credential ID is marked as revoked, verification is denied with typed error: `CredentialRevoked`.

### 3.8 Vector 8: Attribute Permutation & Canonical Ordering
- **Attack:** An attacker reorders attributes in a multi-claim credential to match a higher value to a different field (e.g. swapping GPA `3.90` into the Age field).
- **Defense:**
  - Schema attributes are deterministically sorted alphabetically (`canonical_attributes`) and formatted as `"{name}:{value}"` before SHA-256 scalar field mapping.
  - Index permutation disrupts the BBS+ generator-message binding $(h_i^{m_i})$, causing signature verification to fail completely.

### 3.9 Vector 9: Zero-Knowledge Circuit Soundness (Noir)
- **Attack:** An attacker attempts to produce a fake SNARK proof for the Noir `generic_predicate` circuit.
- **Defense:**
  - Noir compiles constraints into ACIR (Abstract Circuit Intermediate Representation) executed by the Barretenberg prover.
  - Barretenberg enforces polynomial arithmetic constraints with cryptographic soundness ($< 2^{-100}$ probability of false proof generation).
  - Private witness inputs ($birth\_year$) never leave the prover's local memory.

---

## 4. Compliance with the Four Non-Negotiable Requirements

Per Section 2 of `TRUSTPASS_BUILD_SPEC.md`, the architecture explicitly satisfies all challenge directives:

| Requirement | Threat / Risk Addressed | Mitigating Mechanism | Automated Test Citation |
|---|---|---|---|
| **1. Answer common yes/no without releasing record** | Identity theft, surveillance, over-retention by verifiers. | BBS+ Proof of Knowledge (`PoKOfSignatureG1Protocol`) with blinded unrevealed attributes + Noir arithmetic ZK circuits. Attribute values never cross the wire. | `core/tests/checkpoint2_acceptance.rs`<br>`core/tests/checkpoint7_acceptance.rs` |
| **2. Usable by non-technical person (shop owner)** | Operator confusion, delayed queues, misinterpreting cryptographic diagnostics. | `apps/verifier-pwa`: Zero-login, mobile camera scanner, high-contrast binary state cards (`VERIFIED` green / `NOT VERIFIED` red), plain-language failure diagnostics, and 8-second auto-reset. | `apps/verifier-pwa/app.js`<br>`docs/VERIFIER_WALKTHROUGH.md` |
| **3. Proves check happened with zero personal data** | Verifier database leaks exposing customer identity records. | `services/receipt-service`: Logs only `verifier_id`, `claim_request`, boolean `result`, timestamp, and `session_token_hash`. Zero holder DIDs, names, or attribute values stored. | `services/receipt-service/internal/handler_test.go`<br>`db/migrations/001_initial_schema.sql` |
| **4. Demonstrable answer to "what stops a false yes"** | Forged credentials, replayed proofs, expired/revoked documents, altered thresholds. | Multi-gate defense: BLS12-381 signature integrity, Fiat-Shamir session token binding, atomic single-use session consumption, timestamp expiration, and real-time revocation. | `core/tests/checkpoint5_acceptance.rs`<br>`services/verifier-api/internal/session/session_test.go` |

---

## 5. Privacy Guarantees & Non-Personal Receipts

### 5.1 Unlinkability Across Verifications
Because BBS+ proofs randomize unrevealed attributes using fresh Schnorr blinding factors $r \leftarrow \mathbb{F}_q$ on every proof generation:
$$\text{Proof}_1 \not\sim \text{Proof}_2$$
Two verifications performed by the same holder at different merchants cannot be correlated, even if both merchants collude and compare their proof payloads.

### 5.2 Zero Personal Data in Verification Receipts
The `verification_receipts` table schema is strictly non-personal:

```sql
CREATE TABLE verification_receipts (
    id UUID PRIMARY KEY,
    verifier_id VARCHAR(255) NOT NULL,
    claim_request JSONB NOT NULL,       -- e.g. {"attributeName": "dateOfBirth", "operator": "GTE", "threshold": "18"}
    result BOOLEAN NOT NULL,            -- true / false
    timestamp TIMESTAMPTZ NOT NULL,     -- UTC check time
    session_token_hash VARCHAR(64) NOT NULL -- SHA-256(session_token)
);
```

**What is present:** Mathematical proof that a verification occurred at time $T$ for condition $C$ by verifier $V$ with outcome $O$.  
**What is strictly absent:** Holder DID, Subject Name, National ID, Phone Number, Date of Birth, Address, or any raw attribute.

---

## 6. Trust Boundaries & Operational Scope

1. **Issuer Trust:** Verifiers must possess or trust the issuer's public key (`did:key`). In this build, trusted issuer keys are seeded in `services/schema-registry`.
2. **Local Key Management:** In accordance with Section 10 of `TRUSTPASS_BUILD_SPEC.md` ("Explicit Non-Goals"), private keys are managed locally via deterministic seeds rather than cloud HSMs.
3. **No Ledger Dependency:** `did:key` resolves self-certifying public keys directly without relying on external blockchains or distributed consensus networks.
