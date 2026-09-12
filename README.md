# TrustPass

**Privacy-preserving digital identity verification — prove a fact, don't reveal the record.**

---

## What this is

Most identity checks today ask for far more than they need. A shop confirming you're old enough to buy something doesn't need your date of birth, your name, or your ID number — it needs a yes or no. TrustPass replaces document disclosure with cryptographic proof: a person holding a signed digital credential can prove a specific claim about themselves (age, citizenship, enrollment status, academic eligibility) without exposing the record that claim is drawn from.

The verifier gets a confirmed result and a signed, non-personal receipt proving the check happened. Nothing else crosses the boundary.

**Core principle:** *Prove, don't reveal.*

---

## How it works

Three roles:

- **Issuer** — a trusted authority (e.g. a mock national ID office or university registrar) that creates and signs a credential.
- **Holder** — the person who owns the credential and controls when a proof is generated from it.
- **Verifier** — the party asking a specific question, who receives a proof instead of a record.

```
Credential (signed, private) → Proof (claim-specific, generated on demand) → Verifier → Result
```

The record never travels. Only the proof does.

---

## Demo scenarios

1. **Primary — point-of-sale age verification.** A shop owner, using nothing more than a phone camera and a minimal web page, scans a code and gets a clear verified / not verified result. No login, no jargon, no document exchange.
2. **Secondary — academic eligibility.** The same underlying engine verifies citizenship, enrollment status, and a minimum GPA threshold, using synthetic student records, to demonstrate that the architecture generalizes rather than being built around a single claim.

---

## Architecture at a glance

| Layer | Technology | Purpose |
|---|---|---|
| Credential & proof core | Rust — W3C Verifiable Credentials, BBS+ signatures, Noir ZK circuit | Issues signed credentials and generates/verifies selective-disclosure proofs |
| Application services | Go, over gRPC | Issuance, verification requests, schema registry, receipt logging |
| Data layer | PostgreSQL, Redis | Credential schemas, revocation state, receipts, short-lived session tokens |
| Verifier interface | Framework-free PWA | The shop-owner-facing scan screen |
| Holder & issuer interfaces | Svelte | Credential wallet and admin issuance console |
| Deployment | Docker Compose | Single-command local environment |

The system is built around a **generic predicate engine** (`attribute` + `operator` + `threshold`) rather than one function per claim type. Age verification, citizenship checks, and GPA thresholds are all the same code path, configured differently — which is what lets the system extend to new claim types by adding a schema, not by writing new proof logic.

Full technical detail lives in [`TRUSTPASS_BUILD_SPEC.md`](./TRUSTPASS_BUILD_SPEC.md).

---

## Repository layout

```
trustpass/
├── core/               # Rust: credentials, BBS+, Noir circuit, gRPC service
├── services/           # Go: issuer-api, verifier-api, receipt-service, schema-registry
├── apps/               # holder-wallet, issuer-console (Svelte); verifier-pwa (vanilla)
├── db/                 # migrations + synthetic seed data
├── docker/             # docker-compose.yml and per-service Dockerfiles
└── docs/               # architecture, threat model, progress log
```

---

## Getting started

```bash
git clone https://github.com/leoemaxie/trustpass.git
cd trustpass
cp .env.example .env        # fill in local config; never commit real secrets
docker compose -f docker/docker-compose.yml up --build
```

This brings up all services (`core`, `issuer-api`, `verifier-api`, `receipt-service`, `schema-registry`, Postgres, Redis) plus the three frontends. See each service's own README (once scaffolded) for standalone dev instructions.

**Seeding demo data:**

```bash
# from repo root, once services are up
./db/seed/seed.sh   # creates synthetic issuers, schemas, and test holders
```

---

## Running the demo

1. Open `issuer-console`, issue a synthetic credential to a test holder (age check: under- and over-18 examples are seeded; academic check: a passing and failing GPA example are seeded).
2. Open `holder-wallet` as that holder, review the credential.
3. Open `verifier-pwa` on a second device or browser tab, choose a claim (e.g. "age ≥ 18"), and generate the scannable code.
4. Scan it from `holder-wallet`; the wallet shows exactly what's being asked before generating the proof.
5. `verifier-pwa` displays the result — verified or not verified — and logs a non-personal receipt.

---

## What makes this sound, not just functional

- **Standards-based credentials** — W3C Verifiable Credentials Data Model, not a custom format.
- **Real cryptography** — BBS+ selective-disclosure signatures for general claims, plus a genuine Noir zero-knowledge circuit for the flagship age-check path.
- **Replay protection** — every verification is bound to a short-lived, single-use session token.
- **Lifecycle-aware** — credentials expire and can be revoked; both are enforced at proof-verification time.
- **Receipts without records** — every check produces an auditable, timestamped receipt containing zero personal data.

See [`docs/THREAT_MODEL.md`](./docs/THREAT_MODEL.md) for the full answer to "what stops a false yes."

---

## Project documentation

- [`TRUSTPASS_BUILD_SPEC.md`](./TRUSTPASS_BUILD_SPEC.md) — full technical specification: data model, API contracts, build order, and checkpoints.
- [`AGENTS.md`](./AGENTS.md) — operating rules for AI agents contributing to this codebase.
- [`docs/PROGRESS.md`](./docs/PROGRESS.md) — running build/session log.
- [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) — architecture notes and cited library versions.
- [`docs/THREAT_MODEL.md`](./docs/THREAT_MODEL.md) — security and privacy analysis.

---

## Status

This is an active hackathon build. See `docs/PROGRESS.md` for current checkpoint status.

## License
