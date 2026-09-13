# TrustPass Architecture & Technical Reference

## Cryptographic Primitives & Library Citations

Per project policy in `AGENTS.md` and `TRUSTPASS_BUILD_SPEC.md`, all cryptographic operations in `core` rely on real, cited libraries:

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

---

## Key Representation (`did:key`)

TrustPass uses self-certifying `did:key` identifiers without any distributed ledger, blockchain, or external resolver network.

For BBS+ compatibility over BLS12-381 G2:
- Multicodec header: `0xeb, 0x01` (`bls12_381-g2-pub` 2-byte varint).
- Key payload: 96-byte compressed point serialization of `PublicKeyG2<Bls12_381>`.
- Multibase encoding: Base58BTC with prefix character `z`.
- Resulting string format: `did:key:z<base58btc-encoded-multicodec-prefixed-key>`.

---

## Canonical Attribute Ordering & Deterministic BBS+ Mapping

In `core/src/credential/schema.rs`, credential attributes are sorted alphabetically by attribute name (`canonical_attributes()`) before mapping into BBS+ message vectors:
1. Each key-value pair is formatted as `"{attributeName}:{attributeValue}"`.
2. Formatted strings are hashed with SHA-256 and mapped into scalar field elements `Fr` via `Fr::from_be_bytes_mod_order`.
3. The resulting ordered vector `[m_0, m_1, ..., m_{n-1}]` is signed by the issuer.
4. During verification, verifiers reconstruct canonical message vectors according to the schema definition, ensuring 100% cross-service determinism.

---

## Verifiable Credential Structure (W3C VC-DM 2.0)

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

## BBS+ Selective Disclosure & Proof of Knowledge (PoK)

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
   - The resulting challenge `c = H(chal_bytes)` cryptographically ties the proof to that exact verification session.
   - Any attempt to replay a proof against a different session token or after expiration causes verification failure (`SessionTokenExpired` or `SessionTokenReused`).

