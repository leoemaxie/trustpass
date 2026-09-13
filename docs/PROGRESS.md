# TrustPass — Build Progress Log

This file is the running handoff record between agent sessions, possibly across different models. See `AGENTS.md` Section 7 for the entry format and rules. Do not overwrite prior entries — always append a new entry at the bottom. Do not delete or edit past `Open SPEC-GAP flags` until they've been explicitly resolved and marked as such.

Checkpoints referenced below correspond to `TRUSTPASS_BUILD_SPEC.md` Section 9.

---

## Checkpoint Status Overview

Keep this table current — update it whenever a checkpoint's status changes. This is the fastest thing for a new session to check first.

| # | Checkpoint | Status | Last touched |
|---|---|---|---|
| 1 | Credential core, no proofs yet | acceptance criteria met | 2026-09-13 |
| 2 | BBS+ selective disclosure, end to end | acceptance criteria met | 2026-09-13 |
| 3 | Session tokens and replay protection | acceptance criteria met | 2026-09-13 |
| 4 | Receipts | not started | — |
| 5 | Revocation and expiry | not started | — |
| 6 | Generalize to second/third predicates | not started | — |
| 7 | Noir circuit for flagship age claim | not started | — |
| 8 | `verifier-pwa` end to end | not started | — |
| 9 | `holder-wallet` and `issuer-console` end to end | not started | — |
| 10 | Docker Compose, docs, threat model | not started | — |

Status values: `not started` / `in progress` / `acceptance criteria met`.

---

## Open SPEC-GAP Flags (live list)

Copy each flag here when introduced, and move it to "Resolved" once a human or reviewing agent confirms the assumption. Format: `file:line — assumption made — why`.

*(none yet)*

### Resolved

*(none yet)*

---

## Session Log

<!--
Copy this template for each new session:

## Session [YYYY-MM-DD] — [session identifier, e.g. "gemini-1" or "sonnet-review-1"]
**Model:** [which model ran this session]
**Checkpoint worked on:** [N — name]
**Status:** [not started / in progress / acceptance criteria met]
**What changed:** [short factual list, not a narrative]
**Open SPEC-GAP flags introduced this session:** [list, or "none"]
**Next step:** [specific next action]
-->

## Session [not yet started]
**Model:** —
**Checkpoint worked on:** —
**Status:** Repository scaffolded with `TRUSTPASS_BUILD_SPEC.md`, `AGENTS.md`, and this progress log. No implementation work has begun.
**What changed:** Initial documentation set created.
**Open SPEC-GAP flags introduced this session:** none
**Next step:** Begin Checkpoint 1 — set up `proof-core` Rust project skeleton, implement `did:key` generation, VC-DM credential struct, and schema validation against the two seed schemas (`NationalIDCredential` v1, `StudentCredential` v1) defined in the build spec Section 5.1.

## Session 2026-09-13 — kick-off-cp1
**Model:** Gemini 3.8 Flash
**Checkpoint worked on:** 1 — Credential core, no proofs yet
**Status:** acceptance criteria met
**What changed:**
- Scaffolding of `core` Rust crate (`trustpass-core`): BBS+ key generation, BLS12-381 G2 `did:key` generation and parsing without external ledgers, generic predicate engine (`evaluatePredicate(attribute, operator, threshold)`), W3C VC-DM 2.0 credential issuance, serialization/deserialization, and signature verification.
- Implemented `credential_schemas` validation and seed schemas (`NationalIDCredential` v1, `StudentCredential` v1).
- Implemented PostgreSQL migration 001 (`001_initial_schema.sql`) and SQL/JSON seed data in `db/`.
- Created Go `schema-registry` microservice with thread-safe repository, health check (`/healthz`), versioned schema lookup/creation HTTP endpoints, and unit tests.
- Created `.env.example` with documented environment configurations.
- Created `docs/ARCHITECTURE.md` citing exact cryptographic crate versions (`bbs_plus 0.25.0`, `ark-bls12-381 0.4.0`, etc.).
- Created and executed `core/tests/checkpoint1_acceptance.rs`, verifying end-to-end synthetic `NationalIDCredential` issuance, schema validation, JSON roundtrip, signature verification, and negative tampering tests.
**Open SPEC-GAP flags introduced this session:** none
**Next step:** Begin Checkpoint 2 — BBS+ selective disclosure, end to end for one claim (age >= 18 via `BEFORE_DATE` on `dateOfBirth`). Implement PoK of BBS+ signature revealing only the predicate outcome, wire through `issuer-api` and `verifier-api`.

## Session 2026-09-13 — cp2-selective-disclosure
**Model:** Gemini 3.8 Flash
**Checkpoint worked on:** 2 — BBS+ selective disclosure, end to end
**Status:** acceptance criteria met
**What changed:**
- Implemented genuine BBS+ Proof of Knowledge (PoK) of signatures with selective disclosure in `core/src/bbs/proof.rs`, blinding all unrevealed attributes with fresh Schnorr randomness (`MessageOrBlinding::BlindMessageRandomly`).
- Cryptographically bound the verifier's single-use `session_token`, `schema_name`, and `attribute_name` into the Fiat-Shamir challenge calculation for non-malleability and replay defense.
- Implemented `core/proto/proof.proto` defining gRPC service contracts (`IssueCredential`, `GenerateProof`, `VerifyProof`, `CheckRevocation`).
- Added lightweight HTTP API server in `core/src/server.rs` (`/healthz`, `/api/v1/issue`, `/api/v1/generate-proof`, `/api/v1/verify-proof`).
- Implemented Go `services/shared/coreclient.go` HTTP client wrapper for `core`.
- Implemented Go `services/issuer-api` with `/healthz`, `/credentials/issue`, `/credentials` endpoints.
- Implemented Go `services/verifier-api` with `/healthz`, `/verification/sessions`, `/verification/verify`, and single-use session store with TTL expiry.
- Added comprehensive unit tests in Go for `verifier-api` (session lifecycle, expiration, replay rejection) and `issuer-api`.
- Created and passed `core/tests/checkpoint2_acceptance.rs`, proving adult pass (DOB 1999-07-20 >= 18 with 0 personal attributes revealed), minor rejection (`PredicateNotSatisfied`), tampered proof rejection (`SignatureInvalid`), and mismatched token rejection (`SessionTokenExpired`).
**Open SPEC-GAP flags introduced this session:** none
**Next step:** Begin Checkpoint 3 — Session tokens and replay protection. Expand verification session tests to demonstrate all three rejection cases explicitly (expired, reused, tampered signature) through live HTTP services and persistence.

## Session 2026-09-13 — cp3-session-tokens-replay
**Model:** Gemini 3.8 Flash
**Checkpoint worked on:** 3 — Session tokens and replay protection
**Status:** acceptance criteria met
**What changed:**
- Implemented atomic `Consume(token string)` in `verifier-api/internal/session` ensuring atomic state transition to prevent race condition replays.
- Updated `verifier-api/internal/handler/handler.go` to strictly enforce atomic single-use session token consumption.
- Created `core/tests/checkpoint3_acceptance.rs` demonstrating the complete lifecycle and all three required rejection cases:
  1. `SessionTokenExpired`: expired TTL or token mismatch.
  2. `SessionTokenReused`: attempted reuse of an already-consumed single-use token.
  3. `SignatureInvalid`: modified Schnorr commitments / bit-flipped proof bytes or unauthorized issuer key.
- Created `services/verifier-api/internal/handler/checkpoint3_test.go` verifying all three typed rejection responses across the live Go HTTP API.
**Open SPEC-GAP flags introduced this session:** none
**Next step:** Begin Checkpoint 4 — Receipts. Implement `receipt-service` and non-personal receipt generation/storage (`verification_receipts`) on every completed verification, proving the check occurred with zero personal data.



