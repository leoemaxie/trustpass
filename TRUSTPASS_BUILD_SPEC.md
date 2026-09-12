# TrustPass — Build Specification

**Audience:** AI coding agent(s) building this system. This document is the single source of truth for what to build, in what order, and to what standard. Where this document conflicts with your own judgment or training-data defaults, this document wins. If something genuinely required to proceed is not specified here, stop and flag it rather than inventing a default silently — leave a `// SPEC-GAP:` comment describing the assumption you made and why.

**Project:** TrustPass — a privacy-preserving digital identity verification platform built for the NITDA ICSC 2026 Universities Hackathon, Track B ("Proving a Fact Without Revealing the Whole Record").

**Core principle every design and code decision must serve:** *Prove, don't reveal.* A verifier must never receive more than the minimum fact needed to answer the question it asked.

---

## 0. How to use this document

Build sequentially, in the order given in Section 9 (Build Plan & Checkpoints). Do not skip ahead to a later checkpoint before the current one passes its acceptance criteria. Each checkpoint is a working, demonstrable increment — not a partial scaffold.

Repo layout, contracts, and schemas below are binding, not illustrative. Field names, service names, and directory names given here should be used verbatim unless a stated constraint (e.g. a language's naming convention) requires adaptation — in which case adapt mechanically (e.g. `snake_case` in Rust/Postgres, `camelCase` in TS/JS at API boundaries where noted) and keep the semantic meaning identical.

---

## 1. System Overview

Three actors:

- **Issuer** — a trusted authority that creates and signs credentials (e.g. a mock national ID authority, a mock university registrar).
- **Holder** — the individual who owns a credential and generates proofs from it.
- **Verifier** — the party requesting confirmation of a claim (e.g. a shop owner checking age).

Core interaction: a Holder possesses a signed credential. A Verifier specifies a claim (e.g. "age >= 18"). The Holder's device generates a cryptographic proof that the claim is true, without revealing the underlying attribute value. The Verifier checks the proof and gets a boolean result plus a non-personal receipt confirming the check occurred.

**Two demo scenarios, one architecture:**
1. **Primary:** point-of-sale age verification. Verifier = a minimal web app usable by a non-technical shop owner on a basic phone. Holder presents a QR code. Result: verified / not verified, big and unambiguous.
2. **Secondary:** academic eligibility (citizenship == Nigerian, enrollment == active, GPA >= 3.50). Proves the same engine generalizes without new proof logic — only a new schema and predicate combination.

---

## 2. Non-Negotiable Requirements (from the challenge brief)

These four must all be demonstrably true in the final build. Treat each as an acceptance criterion for the whole project, independent of individual module checkpoints:

1. A working demo answers a common yes/no identity question **without releasing the underlying record**.
2. The verifier-facing screen is **usable by a non-technical person** (a shop owner) — no login complexity, no jargon, large clear result state.
3. A **record proves the check happened**, retained by the verifier, **containing zero personal data**.
4. There is a **clear, demonstrable answer to "what stops a false yes"** — covering both cryptographic forgery and replay/spoofing of a valid-looking result.

---

## 3. Repository Layout (Monorepo)

```
trustpass/
├── core/                 # Rust: credentials, BBS+, Noir circuit, gRPC service
│   ├── src/
│   │   ├── credential/         # VC-DM issuance, signing, schema validation
│   │   ├── bbs/                # BBS+ signature + selective-disclosure proof logic
│   │   ├── zk/                 # Noir circuit integration (flagship age-check claim)
│   │   ├── predicate/          # Generic predicate evaluation engine
│   │   ├── grpc/               # gRPC service definitions + handlers
│   │   └── main.rs
│   ├── circuits/                # Noir circuit source (age_check/)
│   ├── proto/                   # .proto files shared with Go services
│   └── Cargo.toml
│
├── services/                    # Go: application services
│   ├── issuer-api/              # credential issuance service
│   ├── verifier-api/            # verification request + session token service
│   ├── receipt-service/         # signed non-personal receipt generation/storage
│   ├── schema-registry/         # credential schema CRUD (versioned)
│   └── shared/                  # shared Go types, gRPC client wrappers, config
│
├── apps/
│   ├── holder-wallet/           # Svelte — holder-facing credential + QR display
│   ├── issuer-console/          # Svelte — admin UI for issuing test credentials
│   └── verifier-pwa/            # framework-free minimal PWA — the shop-owner UI
│
├── db/
│   ├── migrations/              # SQL migrations (Postgres)
│   └── seed/                    # synthetic credential seed data
│
├── docker/
│   ├── docker-compose.yml
│   └── Dockerfile.* (per service)
│
├── docs/
│   ├── ARCHITECTURE.md
│   ├── THREAT_MODEL.md
│   └── API.md (generated/maintained alongside proto changes)
│
└── README.md
```

---

## 4. Style & Engineering Conventions

These apply across all languages and are not optional polish — several map directly to what hackathon judges will inspect.

- **No placeholder cryptography.** Every signature, proof, and verification path must use a real implementation from a named, cited library (see Section 6). A `TODO: implement real crypto later` is a spec violation, not an acceptable shortcut, given the explicit decision to build both BBS+ and Noir for real.
- **Standards over invention.** Credentials are W3C VC-DM JSON-LD documents. Do not design a custom credential envelope "for simplicity" — this is one of the two things (alongside the proof scheme) judges are most likely to scrutinize for soundness.
- **Predicate-first, not claim-first.** Never write a function called `checkAge()` or `verifyGPA()`. Write `evaluatePredicate(attribute, operator, threshold)` and configure age/GPA/citizenship as data (schema + predicate config), not as separate code paths. This is the single most important structural rule in this spec — violating it undermines the extensibility story the whole architecture is built to prove.
- **Explicit error types, no silent failures.** Especially in `core`: a failed verification must return a typed reason (`SignatureInvalid`, `PredicateNotSatisfied`, `CredentialExpired`, `CredentialRevoked`, `SessionTokenExpired`, `SessionTokenReused`) — never a bare boolean with no diagnostic path, since the security writeup depends on being able to demonstrate each rejection case individually.
- **Naming:** `snake_case` for Rust and SQL, `camelCase` for Go exported fields intended for JSON/gRPC boundaries follow protobuf-generated conventions (do not hand-edit generated code), kebab-case for directories and Docker service names.
- **Commits/checkpoints:** each checkpoint in Section 9 should correspond to a working state — runnable, testable, demonstrable — not a syntactic milestone. Do not proceed to the next checkpoint with a broken build.
- **No blockchain, no DID registry, no external network dependency for core proof logic.** Use `did:key` (self-certifying, derived directly from the issuer's public key) for issuer/holder identifiers. This is a deliberate simplicity choice, not an oversight — do not "improve" it by adding a ledger.
- **Secrets:** issuer signing keys are generated and stored locally (file or local KMS-style abstraction) for the hackathon build. Do not hardcode key material in source; use environment variables or a local `.env` (gitignored) with a documented `.env.example`.
- **Every service has a health check endpoint** (`/healthz` or gRPC equivalent) — required for the Docker Compose setup to be demonstrably reliable during judging.

---

## 5. Data Model

### 5.1 Credential Schemas (versioned, stored as data — not code)

Table: `credential_schemas`

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | primary key |
| `name` | text | e.g. `NationalIDCredential`, `StudentCredential` |
| `version` | integer | schema version, monotonic per name |
| `attributes` | JSONB | array of `{name, type}` — e.g. `[{"name":"dateOfBirth","type":"date"},{"name":"nationality","type":"string"}]` |
| `issuer_did` | text | which issuer this schema belongs to |
| `created_at` | timestamptz | |

Two schemas to seed at minimum:

**`NationalIDCredential` v1**
```json
{
  "attributes": [
    {"name": "fullName", "type": "string"},
    {"name": "dateOfBirth", "type": "date"},
    {"name": "nationality", "type": "string"},
    {"name": "idNumber", "type": "string"}
  ]
}
```

**`StudentCredential` v1**
```json
{
  "attributes": [
    {"name": "studentId", "type": "string"},
    {"name": "university", "type": "string"},
    {"name": "enrollmentStatus", "type": "string"},
    {"name": "gpa", "type": "decimal"}
  ]
}
```

### 5.2 Credentials (issued instances)

Table: `credentials` — stores only the **issuer's copy** for demo/reissuance purposes; the holder's operative copy lives client-side (wallet). Columns: `id`, `schema_id`, `holder_did`, `attributes` (JSONB, synthetic data only), `signature`, `issued_at`, `expires_at`, `revoked` (boolean), `revocation_reason` (nullable).

### 5.3 Predicate Definitions

Predicates are not stored per-claim; they are a small fixed enum evaluated generically against any attribute:

```
enum PredicateOp {
  GTE,          // >=  (e.g. age, gpa)
  EQ,           // ==  (e.g. nationality, enrollmentStatus)
  IN_SET,       // membership
  BEFORE_DATE,  // date comparison (used to derive age from dateOfBirth)
}
```

A **claim request** from a verifier is `{schemaName, attributeName, operator, value}` — e.g. `{"schemaName": "NationalIDCredential", "attributeName": "dateOfBirth", "operator": "BEFORE_DATE", "value": "2008-09-12"}` to check age >= 18 as of today.

### 5.4 Verification Sessions (replay protection)

Table (or Redis, TTL-backed): `verification_sessions` — `session_token` (random, single-use), `claim_request` (JSONB), `created_at`, `expires_at` (short TTL, e.g. 120s), `consumed` (boolean). A proof presented against an expired or already-consumed token is rejected with `SessionTokenExpired` / `SessionTokenReused`.

### 5.5 Verification Receipts (non-personal, retained by verifier)

Table: `verification_receipts` — `id`, `verifier_id` (which business), `claim_request` (JSONB — the question asked, not the answer's basis), `result` (boolean), `timestamp`, `session_token_hash` (not the raw token). **Must not contain**: holder identity, holder DID, any credential attribute value, or anything traceable to a specific person. This table is what satisfies non-negotiable requirement #3.

---

## 6. Cryptographic Core (`core`, Rust)

### 6.1 Credential format

- W3C Verifiable Credentials Data Model 2.0 JSON-LD structure.
- Issuer identity: `did:key` derived from an Ed25519 or BLS12-381 keypair (BLS12-381 required for BBS+ compatibility — use this as the primary key type).
- Library guidance: use `docknetwork/crypto` (Rust) or an equivalent actively-maintained BBS+ implementation. Cite the exact crate and version used in `docs/ARCHITECTURE.md`.

### 6.2 BBS+ signatures (primary proof mechanism, used for all claim types)

- Issuer signs the full attribute set of a credential with a single BBS+ signature.
- Holder, at proof time, derives a **selective-disclosure proof** revealing only the predicate result, not the attribute value, for the specific attribute(s) named in the claim request.
- Verification confirms: (a) the signature is valid under the issuer's known public key, (b) the disclosed predicate holds, (c) undisclosed attributes remain fully hidden and non-recoverable from the proof.

### 6.3 Noir circuit (flagship claim: age check)

- Implement one real Noir circuit for the age predicate (`dateOfBirth` implies `age >= threshold`), compiled via `nargo`, proved/verified via Barretenberg.
- This circuit is used specifically for the **primary demo path** (point-of-sale age check) as the showcased ZK capability. BBS+ remains the general mechanism for all other claims (citizenship, enrollment, GPA).
- Document in `docs/ARCHITECTURE.md` explicitly: "the proof engine is pluggable — production could migrate every claim to a dedicated ZK circuit; this build demonstrates both approaches side by side."

### 6.4 gRPC interface (`core/proto/proof.proto`)

Define at minimum:
```protobuf
service ProofCore {
  rpc IssueCredential(IssueCredentialRequest) returns (IssueCredentialResponse);
  rpc GenerateProof(GenerateProofRequest) returns (GenerateProofResponse);
  rpc VerifyProof(VerifyProofRequest) returns (VerifyProofResponse);
  rpc CheckRevocation(RevocationCheckRequest) returns (RevocationCheckResponse);
}
```
`VerifyProofResponse` must include a typed `rejection_reason` field (see Section 4) whenever `valid = false`.

### 6.5 Revocation & expiry

- Every credential has `expires_at`. Expired credentials fail proof generation client-side and proof verification server-side.
- Revocation: issuer maintains a revocation list (simple table or bitmask-style status list) keyed by credential ID. `CheckRevocation` is called as part of `VerifyProof`.

---

## 7. Application Services (Go)

All services communicate with `core` via gRPC. Services communicate with each other via internal REST or gRPC (either is fine — pick REST for simplicity unless an agent finds gRPC-to-gRPC meaningfully faster to implement given shared proto definitions).

- **`issuer-api`** — endpoints to create synthetic credentials against a schema, calling `core.IssueCredential`. Also serves schema lookups from `schema-registry`.
- **`verifier-api`** — accepts a claim request from a verifier app, creates a `verification_sessions` row (with token + short TTL), returns the token + claim request encoded for QR generation. Also accepts a submitted proof, calls `core.VerifyProof`, validates the session token (unexpired, unconsumed, marks consumed), and returns the boolean result.
- **`receipt-service`** — on every completed verification (pass or fail), writes a `verification_receipts` row and returns a receipt reference the verifier app can display/store.
- **`schema-registry`** — CRUD for `credential_schemas`, versioned, used by both issuer and verifier flows to validate claim requests against real attribute names/types.

Each service: standard Go project layout (`cmd/`, `internal/`), structured logging, `/healthz`, config via environment variables, no hardcoded ports (use env with sensible documented defaults).

---

## 8. Frontend Applications

### 8.1 `verifier-pwa` (the shop-owner interface — highest scrutiny surface)

- **No framework** — plain HTML/CSS/JS, minimal JS footprint, must load fast on a weak connection and a basic phone browser.
- One primary screen: a "Scan" button (uses device camera via a lightweight QR-reading library, e.g. `jsQR`, loaded from a local vendored copy, not a CDN dependency that could fail offline during a demo).
- Result state: full-screen, high-contrast, unambiguous — green checkmark + "VERIFIED" or red X + "NOT VERIFIED". No secondary information competing for attention on this screen.
- A secondary, clearly separated "Receipts" view lists past verification receipts (non-personal) for the business's own records — this is a distinct screen, not blended into the primary scan flow.
- No login for the MVP demo, or at most a single shared PIN per verifier — do not build a full auth system here; it is out of scope and would work against the "usable by a non-technical shop owner" requirement.

### 8.2 `holder-wallet` (Svelte)

- Displays the holder's credentials (from local storage / mocked secure storage — do not build real secure enclave integration for the hackathon).
- On receiving a claim request (via a secondary channel — e.g. the verifier app displays a claim-request QR that the holder scans, or a shared session code entered manually), generates the proof via a call to `verifier-api` (which itself talks to `core`), and displays a resulting QR code encoding the session token for the verifier to scan.
- Must clearly show the holder, before generating any proof, exactly what is being asked and what will and won't be revealed — this is a trust-building UI moment worth getting right, not an afterthought.

### 8.3 `issuer-console` (Svelte)

- Internal/admin tool. Create synthetic holders, issue credentials against a schema, view/revoke issued credentials. Not part of the judged "usability" demo — can be more conventional in design.

---

## 9. Build Plan & Checkpoints

Build in this order. Each checkpoint must be independently demonstrable before moving to the next.

**Checkpoint 1 — Credential core, no proofs yet.**
`core` can issue a signed VC-DM credential (BBS+ key setup, `did:key` generation, schema validation) and store/retrieve it. `schema-registry` and the two seed schemas exist. Acceptance: issue a synthetic `NationalIDCredential`, inspect it, confirm it validates against its schema and signature.

**Checkpoint 2 — BBS+ selective disclosure, end to end for one claim.**
Generate and verify a BBS+ selective-disclosure proof for one predicate (age >= 18 via `BEFORE_DATE` on `dateOfBirth`) through the full path: `issuer-api` issues → holder holds → `verifier-api` requests claim + session token → proof generated → `core.VerifyProof` confirms → typed result returned. Acceptance: a script/test can run this whole path and print a correct pass and a correct fail (using a synthetic under-18 record).

**Checkpoint 3 — Session tokens and replay protection.**
Verification sessions expire and are single-use; a replayed proof against a consumed or expired token is rejected with the correct typed reason. Acceptance: demonstrate all three rejection cases explicitly (expired, reused, tampered signature).

**Checkpoint 4 — Receipts.**
Every verification (pass/fail) produces a `verification_receipts` row containing zero personal data. Acceptance: inspect a receipt and confirm no holder-identifying field is present anywhere in it.

**Checkpoint 5 — Revocation and expiry.**
A revoked or expired credential fails proof generation/verification with the correct typed reason. Acceptance: revoke a credential, attempt a proof, confirm rejection.

**Checkpoint 6 — Generalize to second and third predicates.**
Wire up `EQ` (nationality == "NG") and `GTE` (gpa >= 3.50) against `StudentCredential` using the *same* predicate engine and services, with no new core logic beyond configuration. Acceptance: this should require adding schema/config data only — if it requires new code paths in `core` beyond generic predicate evaluation, that's a violation of Section 4's predicate-first rule and should be fixed before continuing.

**Checkpoint 7 — Noir circuit for the flagship age claim.**
Real Noir circuit compiled and integrated as an alternate proof path for the age predicate specifically, callable and verifiable alongside the BBS+ path. Acceptance: demonstrate both mechanisms independently producing a correct verified result for the same underlying credential.

**Checkpoint 8 — `verifier-pwa` end to end.**
Full scan → result flow on the minimal PWA, tested on a throttled connection profile (simulate weak connectivity) and at minimum on a mobile browser viewport. Acceptance: a non-technical description of the flow (a written 5-step "how a shop owner uses this" walkthrough) matches what actually happens with no undocumented extra steps.

**Checkpoint 9 — `holder-wallet` and `issuer-console` end to end.**
Full holder-side flow (view credential → receive claim request → generate proof → display result QR) and admin issuance flow. Acceptance: the two demo scenarios (age check, academic eligibility) both run start to finish through the UIs, not just via scripts/tests.

**Checkpoint 10 — Docker Compose, docs, and threat model writeup.**
All services runnable via a single `docker-compose up`. `docs/ARCHITECTURE.md` and `docs/THREAT_MODEL.md` completed, covering the four non-negotiable requirements in Section 2 explicitly, one by one, with a sentence pointing at the specific mechanism that satisfies each.

---

## 10. Explicit Non-Goals (do not build these)

- No blockchain or distributed ledger component of any kind.
- No DID registry or resolver beyond `did:key`.
- No mobile native app — web-based only for all three frontends.
- No production-grade key management (HSM, cloud KMS) — local key storage is acceptable and expected for this build.
- No multi-issuer federation/trust registry beyond a hardcoded list of known issuer DIDs.
- No user account system beyond what's minimally needed to demo (holder identity can be a locally-generated keypair with no registration flow; verifier auth can be a single shared PIN or none).

If an agent finds itself building toward any of the above, stop and re-read Section 2 — it is very likely scope creep away from what is actually being judged.

---

## 11. Glossary (for agent context, avoid re-deriving these from scratch)

- **VC-DM**: W3C Verifiable Credentials Data Model — the standard JSON-LD structure for digitally signed claims.
- **BBS+**: a signature scheme supporting selective disclosure and unlinkable proofs over a multi-attribute signed message — lets a holder prove a subset/predicate of signed attributes without revealing the rest or allowing proofs to be correlated.
- **Noir**: a domain-specific language for writing zero-knowledge circuits, compiled via `nargo`, proved with the Barretenberg backend.
- **`did:key`**: a self-certifying decentralized identifier derived directly from a public key, requiring no registry or ledger.
- **Predicate**: a generic comparison (`>=`, `==`, set membership, date comparison) applied to a named credential attribute — the mechanism that makes the system claim-agnostic rather than hardcoded per use case.
