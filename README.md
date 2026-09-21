# TrustPass

**Privacy-preserving digital identity verification — prove a fact, don't reveal the record.**  
*Built for NITDA ICSC 2026 Universities Hackathon — Track B.*
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Rust](https://img.shields.io/badge/Rust-1.85%2B-orange.svg?logo=rust&logoColor=white)](https://www.rust-lang.org/)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg?logo=go&logoColor=white)](https://go.dev/)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-5-FF3E00.svg?logo=svelte&logoColor=white)](https://kit.svelte.dev/)
[![Noir ZK](https://img.shields.io/badge/Noir-ZK%20Circuits-black.svg?logo=noir&logoColor=white)](https://noir-lang.org/)
[![Docker Compose](https://img.shields.io/badge/Docker-Compose%20v2-2496ED.svg?logo=docker&logoColor=white)](https://www.docker.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1.svg?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D.svg?logo=redis&logoColor=white)](https://redis.io/)

---

## What this is

Most identity checks today ask for far more than they need. A shop confirming you're old enough to buy something doesn't need your date of birth, your name, or your ID number — it needs a yes or no. TrustPass replaces document disclosure with cryptographic proof: a person holding a signed digital credential can prove a specific claim about themselves (age, citizenship, enrollment status, academic eligibility) without exposing the record that claim is drawn from.

The verifier gets a confirmed result and a signed, non-personal receipt proving the check happened. Nothing else crosses the boundary.

**Core principle:** *Prove, don't show!*

---

## 5-Minute Quickstart

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose (v2.0+)
- *Or for local bare-metal dev:* Rust 1.85+, Go 1.22+, Node.js 20+, and `pnpm`.

### 1. Launch All Services with Docker
```bash
git clone https://github.com/leoemaxie/trustpass.git
cd trustpass
cp .env.example .env

# Single command to build and launch all 10 services
docker compose up --build
```

### 2. Service Access Endpoints

| Application / Service | URL / Port | Purpose |
|---|---|---|
| **Verifier PWA** | [`http://localhost:3000`](http://localhost:3000) | Zero-login camera QR scanner & binary result screen |
| **Holder Wallet** | [`http://localhost:5173`](http://localhost:5173) | Holder credential storage, claim review, & proof display |
| **Issuer Console** | [`http://localhost:5174`](http://localhost:5174) | Admin dashboard for issuing credentials & testing revocation |
| **Schema Registry** | [`http://localhost:8081`](http://localhost:8081) | Versioned W3C credential schema definitions |
| **Issuer API** | [`http://localhost:8082`](http://localhost:8082) | Credential issuance orchestration & revocation registry |
| **Verifier API** | [`http://localhost:8083`](http://localhost:8083) | Verification session tokens & proof verification |
| **Receipt Service** | [`http://localhost:8084`](http://localhost:8084) | Audit trail of checks with **zero personal data** |
| **Cryptographic Core** | [`http://localhost:50051`](http://localhost:50051) | Pure Rust BBS+ BLS12-381 & Noir ZK circuit server |
| **PostgreSQL** | `localhost:5432` | Relational storage (schemas, issued records, receipts) |
| **Redis** | `localhost:6379` | High-speed cache and session TTL storage |

---

## End-to-End Demo Scenarios

### Scenario A: Point-of-Sale Age Verification (Age ≥ 18)
1. Open the **Verifier PWA** at [`http://localhost:3000`](http://localhost:3000). Click **"Start Verification Scan"**. A single-use verification QR code and session token are generated.
2. Open the **Holder Wallet** at [`http://localhost:5173`](http://localhost:5173).
   - In Demo Quick Launchers, click **"Test Over 18 (DOB: 1998-05-14)"**.
   - Review the **Triple-Disclosure Trust Screen** — notice that your Name, ID Number, and exact Date of Birth remain private; only the boolean condition (`Age ≥ 18`) will be proven.
   - Click **"Generate Cryptographic Proof"**. A QR code with the BBS+ Proof of Knowledge is rendered.
3. In the Verifier PWA, scan or paste the proof payload.
   - The verifier immediately displays a full-bleed **VERIFIED (Green)** state!
   - A non-personal audit receipt is recorded in the receipt service containing only a SHA-256 session hash.
4. Try scanning with **"Test Under 18 (DOB: 2012-08-20)"** — the verifier unambiguously displays **NOT VERIFIED (Red)** with typed diagnostic reason `PredicateNotSatisfied`.

### Scenario B: Academic Eligibility (Nigerian Citizen & GPA ≥ 3.50)
1. Open the **Holder Wallet** at [`http://localhost:5173`](http://localhost:5173).
   - Click **"Academic Check (Pass: GPA 3.85)"**.
   - Confirm proof generation: evaluates `nationality == "NG"` and `gpa >= 3.50` using the *identical* generic predicate engine (`core::predicate::evaluate_predicate`) without any claim-specific code!
   - Verifier confirms `VERIFIED`.
2. Try the failing GPA credential (`gpa = 3.10`) — proof verification rejects cleanly.

### Scenario C: Testing Revocation & Replay Protection
1. In the **Issuer Console** at [`http://localhost:5174/credentials`](http://localhost:5174/credentials), click **"Revoke"** on any issued credential.
2. In the **Holder Wallet**, attempt to generate a proof using that credential.
3. Verification is rejected with typed diagnostic: `CredentialRevoked`.
4. Try submitting an already-used session token — rejected with `SessionTokenReused`.

---

## Running the Automated Test Suites

Every checkpoint in TrustPass is validated by automated unit, integration, and negative security tests:

```bash
# 1. Run all Go application service tests (Checkpoints 4, 8, 9)
cd services
go test -v ./...
cd ..

# 2. Run all Rust Cryptographic Core tests (Checkpoints 1, 2, 3, 5, 6, 7)
cd core
cargo test
cd ..

# 3. Run SvelteKit type checks on frontend apps
pnpm --filter trustpass-holder-wallet check
pnpm --filter trustpass-issuer-console check
```

---

## Architectural Soundness & Defense in Depth

- **Standards-based credentials** — W3C Verifiable Credentials Data Model 2.0 JSON-LD documents, not proprietary formats.
- **Genuine Cryptography** — Dock Network BBS+ selective-disclosure signatures over pairing-friendly curve BLS12-381 (`ark-bls12-381`), plus Aztec Noir arithmetic ZK circuit integration. Zero placeholder cryptography.
- **Generic Predicate Engine** — Zero claim-specific functions (`checkAge()`, `isNigerian()`). All claims evaluate generically via `attribute`, `operator` (`GTE`, `LTE`, `EQ`, `IN_RANGE`), and `threshold`.
- **Replay & Session Protection** — Ephemeral 120s session tokens bound into the Fiat-Shamir challenge bytes $c = H(pk \parallel pok \parallel session\_token \parallel \dots)$ and consumed atomically upon first use.
- **Zero Personal Data in Receipts** — Auditable verification receipts retain only non-personal metadata and a SHA-256 session token hash.
- **Independent from Ledgers** — Self-certifying `did:key` identifiers require no blockchain, external registry, or cloud dependency.

See [`docs/THREAT_MODEL.md`](docs/THREAT_MODEL.md) for the complete security analysis answering *"what stops a false yes"*, and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for cryptographic primitive specifications.

---

## Project Documentation

- [`TRUSTPASS_BUILD_SPEC.md`](./TRUSTPASS_BUILD_SPEC.md) — Authoritative technical specification, data models, and API contracts.
- [`AGENTS.md`](./AGENTS.md) — Operating rules, prime directives, and code quality invariants.
- [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) — Architectural reference, service topologies, and library citations.
- [`docs/THREAT_MODEL.md`](./docs/THREAT_MODEL.md) — Threat actor taxonomy, attack mitigation vectors, and proof soundness.
- [`docs/DEPLOYMENT.md`](./docs/DEPLOYMENT.md) — Production deployment guide with Oracle Cloud Infrastructure (OCI) reference and SSL configuration.
- [`docs/VERIFIER_WALKTHROUGH.md`](./docs/VERIFIER_WALKTHROUGH.md) — 5-step non-technical retail walkthrough for shop owners.

---

## License

Licensed under the Apache License, Version 2.0 (the "License"). See [LICENSE](LICENSE) for details.
