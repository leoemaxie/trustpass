# TrustPass Architecture & Technical Reference

This document serves as the architectural reference for TrustPass — a privacy-preserving digital identity verification platform built for the NITDA ICSC 2026 Universities Hackathon (Track B: "Proving a Fact Without Revealing the Whole Record").

---

## 1. System Topology & Service Boundaries

TrustPass employs a microservice architecture separating cryptographic operations, orchestration APIs, data persistence, and end-user web applications:

```
                          ┌───────────────────────┐
                          │  apps/issuer-console  │ (Port 5174: SvelteKit Admin)
                          └───────────┬───────────┘
                                      │
                                      ▼
┌────────────────────────┐  ┌───────────────────┐  ┌───────────────────────┐
│ apps/verifier-pwa      │  │ services/         │  │ apps/holder-wallet    │
│ (Port 3000: Vanilla JS)│  │ issuer-api        │  │ (Port 5173: SvelteKit)│
└───────────┬────────────┘  │ (Port 8082: Go)   │  └───────────┬───────────┘
            │               └─────────┬─────────┘              │
            ▼                         ▼                        ▼
┌────────────────────────┐  ┌───────────────────┐  ┌───────────────────────┐
│ services/verifier-api  │  │ services/         │  │ core (Rust)           │
│ (Port 8083: Go)        │  │ schema-registry   │  │ (Port 50051: gRPC/    │
└─────┬────────────┬─────┘  │ (Port 8081: Go)   │  │  HTTP Server)         │
      │            │        └───────────────────┘  └───────────────────────┘
      │            ▼
      │    ┌───────────────────┐
      │    │ services/         │
      │    │ receipt-service   │
      └───►│ (Port 8084: Go)   │
           └─────────┬─────────┘
                     ▼
           ┌───────────────────┐
           │ postgres:15       │ (Port 5432: Schemas, Credentials, Receipts)
           │ redis:7           │ (Port 6379: Cache & Fast Sessions)
           └───────────────────┘
```

### Component Roles & Ports
| Component | Directory | Port | Technology | Primary Function |
|---|---|---|---|---|
| **Cryptographic Core** | `core/` | 50051 | Rust (BBS+, BLS12-381, Noir) | Key generation, VC-DM signing, PoK proof generation & verification |
| **Schema Registry** | `services/schema-registry/` | 8081 | Go 1.22 | Versioned credential schemas (`NationalIDCredential`, `StudentCredential`) |
| **Issuer API** | `services/issuer-api/` | 8082 | Go 1.22 | Credential issuance orchestration, audit records, revocation |
| **Verifier API** | `services/verifier-api/` | 8083 | Go 1.22 | Verification session tokens, proof relay, receipt logging trigger |
| **Receipt Service** | `services/receipt-service/` | 8084 | Go 1.22 | Audit receipts storage strictly without personal data |
| **Verifier PWA** | `apps/verifier-pwa/` | 3000 | Vanilla HTML5/JS / Nginx | Non-technical shop-owner QR scanner & binary result screen |
| **Holder Wallet** | `apps/holder-wallet/` | 5173 | SvelteKit / Vite / Nginx | Holder credential storage, triple-disclosure review, proof generation |
| **Issuer Console** | `apps/issuer-console/` | 5174 | SvelteKit / Vite / Nginx | Admin UI for issuing synthetic test credentials and revoking |
| **PostgreSQL** | `db/` | 5432 | PostgreSQL 15 | Relational storage for schemas, credentials, and receipts |
| **Redis** | - | 6379 | Redis 7 | High-performance cache and session store |

---

## 2. Compliance with the Four Non-Negotiable Requirements

Per Section 2 and Section 9 of `TRUSTPASS_BUILD_SPEC.md`, every core requirement is satisfied by a specific, verifiable mechanism:

### Requirement 1: Answering Common Yes/No Identity Questions Without Releasing the Underlying Record
> **Mechanism:** TrustPass implements zero-knowledge Proof of Knowledge over multi-message BBS+ signatures (`bbs_plus::proof::PoKOfSignatureG1Protocol`) using Arkworks BLS12-381 pairings (`core/src/bbs/pok.rs`), blinding unrevealed attributes with fresh Schnorr randomness ($r \leftarrow \mathbb{F}_q$) and evaluating generic comparisons (`OP_GTE`, `OP_EQ`, `OP_LTE`) in `core/src/predicate/engine.rs`. For the flagship age check claim, a compiled Aztec Noir arithmetic circuit (`core/circuits/generic_predicate`) proves $current\_year - birth\_year \ge 18$ via ACIR/Barretenberg ZK-SNARKs without exposing the date of birth or identity record.

### Requirement 2: Verifier-Facing Screen Usable by a Non-Technical Person
> **Mechanism:** `apps/verifier-pwa` is an offline-capable, framework-free Progressive Web App requiring zero authentication, zero configuration, and no cryptographic jargon; it launches directly into an active camera viewfinder (`jsQR.js`) and transitions immediately upon proof submission to a full-bleed, high-contrast binary state card (`VERIFIED` in `#059669` green or `NOT VERIFIED` in `#dc2626` red with explicit plain-language error explanations), followed by an automatic 8-second auto-reset for seamless retail point-of-sale operation.

### Requirement 3: Verifier Record Proving the Check Happened with Zero Personal Data
> **Mechanism:** `services/receipt-service` and PostgreSQL table `verification_receipts` persist auditable verification records containing exclusively non-identifying metadata (`id`, `verifier_id`, `claim_request` predicate, boolean `result`, UTC `timestamp`, and a 64-character hex SHA-256 hash of the session token `session_token_hash`), strictly omitting any holder DID, subject name, national identity number, or attribute values.

### Requirement 4: Clear, Demonstrable Answer to "What Stops a False Yes"
> **Mechanism:** A multi-layered defense-in-depth architecture eliminates false positive verifications across all attack vectors:
> 1. *Cryptographic Forgery:* BBS+ signatures over BLS12-381 G2 provide 128-bit computational security under the $q$-SDH assumption.
> 2. *Replay Attacks:* Verification proofs incorporate a single-use 120-second session token cryptographically bound into the Fiat-Shamir challenge bytes $c = H(pk \parallel pok \parallel session\_token \parallel schema \parallel attribute)$.
> 3. *Token Reuse:* Atomic single-use consumption in `services/verifier-api/internal/session/` immediately invalidates session tokens upon first check attempt (`SessionTokenReused`).
> 4. *Credential Expiration:* Proof verification evaluates the credential's ISO-8601 expiration timestamp against UTC now, returning `CredentialExpired`.
> 5. *Revocation:* Issuer revocation state is checked synchronously at verification time in `core` (`verify_proof_internal`), returning `CredentialRevoked`.
> 6. *Attribute Tampering:* Canonical alphabetical sorting of attributes (`canonical_attributes`) ensures that modifying attribute names or indices invalidates the BBS+ PoK signature relation.

---

## 3. Cryptographic Primitives & Library Citations

All cryptographic operations rely on named, cited libraries per project policy in `AGENTS.md`:

| Primitive / Purpose | Crate / Library | Version | Upstream Repository / Notes |
|---|---|---|---|
| BBS+ Signatures & PoK | `bbs_plus` | `0.25.0` | [`docknetwork/crypto`](https://github.com/docknetwork/crypto) |
| Pairing-Friendly Curve | `ark-bls12-381` | `0.4.0` | Arkworks BLS12-381 implementation |
| Elliptic Curves Core | `ark-ec` | `0.4.2` | Arkworks EC abstraction |
| Finite Fields | `ark-ff` | `0.4.2` | Arkworks Finite Field arithmetic |
| Canonical Serialization | `ark-serialize` | `0.4.2` | Deterministic point/scalar serialization |
| Crypto Utilities | `dock_crypto_utils` | `0.23.0` | [`docknetwork/crypto`](https://github.com/docknetwork/crypto) |
| Hashing to Scalar Field | `sha2` | `0.10.9` | SHA-256 for deterministic attribute scalar mapping |
| Base58 Multibase Encoding | `bs58` | `0.5.1` | Multibase 'z' base58btc for `did:key` representation |
| ZK DSL & Backend | `noir_rs` / Aztec ACIR | `0.30+` | Aztec Network Noir circuit compiler & Barretenberg |

---

## 4. Key Representation (`did:key`)

TrustPass uses self-certifying `did:key` identifiers without any distributed ledger, blockchain, or external resolver network.

For BBS+ compatibility over BLS12-381 G2:
- Multicodec header: `0xeb, 0x01` (`bls12_381-g2-pub` 2-byte varint).
- Key payload: 96-byte compressed point serialization of `PublicKeyG2<Bls12_381>`.
- Multibase encoding: Base58BTC with prefix character `z`.
- Resulting string format: `did:key:z<base58btc-encoded-multicodec-prefixed-key>`.

---

## 5. Canonical Attribute Ordering & Deterministic BBS+ Mapping

In `core/src/credential/schema.rs`, credential attributes are sorted alphabetically by attribute name (`canonical_attributes()`) before mapping into BBS+ message vectors:
1. Each key-value pair is formatted as `"{attributeName}:{attributeValue}"`.
2. Formatted strings are hashed with SHA-256 and mapped into scalar field elements `Fr` via `Fr::from_be_bytes_mod_order`.
3. The resulting ordered vector $[m_0, m_1, \dots, m_{n-1}]$ is signed by the issuer.
4. During verification, verifiers reconstruct canonical message vectors according to the schema definition, ensuring 100% cross-service determinism.

---

## 6. Verifiable Credential Structure (W3C VC-DM 2.0)

Credentials conform to W3C Verifiable Credentials Data Model 2.0 JSON-LD specification:
- `@context`: `["https://www.w3.org/ns/credentials/v2", "https://w3id.org/security/suites/bbs-2023/v1"]`
- `id`: `urn:uuid:<uuid-v4>`
- `type`: `["VerifiableCredential", "<SchemaName>"]`
- `issuer`: `did:key:z...`
- `issuanceDate`: ISO 8601 UTC timestamp
- `expirationDate`: Optional ISO 8601 UTC timestamp
- `credentialSchema`: Reference to schema ID (`schema:national-id:v1`, etc.)
- `credentialSubject`: Holds `id` (Holder DID) and the schema attribute claims
- `proof`: BBS+ BLS signature envelope (`BbsBlsSignature2020`) containing hex-encoded compressed signature bytes, timestamp, and verificationMethod.

---

## 7. BBS+ Selective Disclosure & Proof of Knowledge (PoK)

TrustPass implements genuine Proof of Knowledge of BBS+ signatures using `bbs_plus::proof::PoKOfSignatureG1Protocol`:
1. **Blinding Unrevealed Attributes**:
   - For all undisclosed attributes, fresh Schnorr randomness is generated (`MessageOrBlinding::BlindMessageRandomly(msg)`).
   - The verifier receives **zero attribute values**; only the mathematical relation is proven.
2. **Session Token Binding (Replay Protection)**:
   - When generating the Fiat-Shamir challenge, `chal_bytes` incorporates:
     - Issuer's public key bytes.
     - The protocol's challenge contribution (`pok.challenge_contribution`).
     - The verifier's single-use, short-lived `session_token`.
     - The target `schema_name` and `attribute_name`.
   - The resulting challenge $c = H(chal\_bytes)$ cryptographically ties the proof to that exact verification session.
   - Any attempt to replay a proof against a different session token or after expiration causes verification failure (`SessionTokenExpired` or `SessionTokenReused`).

---

## 8. Dual-Proof Architecture: BBS+ and Noir ZK Circuits

TrustPass provides a dual-proof architecture demonstrating both multi-message signature proofs and zero-knowledge arithmetic circuits side by side.

### 1. General Mechanism: BBS+ Selective Disclosure
- Implemented natively in pure Rust using `docknetwork/crypto`.
- Issuer signs multiple claims with a single signature over BLS12-381.
- Holder presents a Proof of Knowledge (PoK) of the signature with undisclosed attributes blinded by fresh Schnorr randomness.
- Generalizes across all credential types (National ID, Student Credentials, etc.) with sub-second performance on commodity hardware.

### 2. General Zero-Knowledge Predicate Circuit: Noir (`generic_predicate`)
- Located in `core/circuits/generic_predicate/`.
- Written in Aztec Network's Noir domain-specific language.
- Implements arithmetic constraints for generic predicate evaluation:
  - `OP_GTE`: Numeric greater-than-or-equal constraint (e.g. GPA $\ge 3.50$, Age $\ge 18$).
  - `OP_LTE`: Numeric less-than-or-equal constraint (e.g. Date of Birth on or before cutoff date).
  - `OP_EQ`: Structural and cryptographic hash equality.
  - `OP_IN_RANGE`: Two-sided interval range checks.
- Compiles to ACIR intermediate representation via `nargo` and proves/verifies via Barretenberg backend.
- Purely generic: zero claim-specific functions (`checkAge()`, `isNigerian()`, etc.) exist anywhere in the codebase.
