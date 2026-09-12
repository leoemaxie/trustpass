# TrustPass — Build Progress Log

This file is the running handoff record between agent sessions, possibly across different models. See `AGENTS.md` Section 7 for the entry format and rules. Do not overwrite prior entries — always append a new entry at the bottom. Do not delete or edit past `Open SPEC-GAP flags` until they've been explicitly resolved and marked as such.

Checkpoints referenced below correspond to `TRUSTPASS_BUILD_SPEC.md` Section 9.

---

## Checkpoint Status Overview

Keep this table current — update it whenever a checkpoint's status changes. This is the fastest thing for a new session to check first.

| # | Checkpoint | Status | Last touched |
|---|---|---|---|
| 1 | Credential core, no proofs yet | not started | — |
| 2 | BBS+ selective disclosure, end to end | not started | — |
| 3 | Session tokens and replay protection | not started | — |
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
