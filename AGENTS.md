# AGENTS.md — TrustPass

This file governs how any AI agent operates in this repository. Read this in full before making any change. If this file and a prompt/instruction conflict, this file wins unless a human explicitly overrides it in-session and says so.

The authoritative technical specification is `TRUSTPASS_BUILD_SPEC.md` at the repo root. This file does not repeat that content — it governs *behavior*, not architecture. Read both before starting work.

---

## 1. Prime directive

**Prove, don't reveal.** Every module you touch must serve the principle that a verifier receives the minimum fact needed and nothing else. If you are ever unsure whether a change leaks more than it should, treat that as a blocking question, not a judgment call to resolve silently in favor of shipping.

---

## 2. Before you write any code

1. Read `TRUSTPASS_BUILD_SPEC.md` in full, not just the section that seems relevant to your current task.
2. Identify which **checkpoint** (Section 9 of the build spec) you are working toward. Do not start work on a checkpoint whose predecessor hasn't passed its acceptance criteria.
3. Check `docs/PROGRESS.md` (create it if it doesn't exist — see Section 7 below) for what previous agent sessions already did and any open flags they left.
4. If the task you've been given doesn't map cleanly onto a checkpoint in the build spec, stop and ask rather than inventing new scope.

---

## 3. Model-routing policy

Different models are in use on this project. This section tells you which kind of task you're being trusted with, so calibrate your caution accordingly — it is not a comment on the humans' opinion of you.

- **High-volume / low-stakes work** (boilerplate scaffolding, repetitive CRUD, lint/type fixes, iterative UI polish on `holder-wallet`, `issuer-console`, `verifier-pwa`): move fast, iterate freely, cost of a wrong attempt is low.
- **Correctness-critical work** (anything in `core`, the predicate engine, session-token/replay logic, revocation, receipt generation): move slower, be conservative, prefer an explicit `// SPEC-GAP:` flag over a silent assumption, and treat Section 4 of this file as binding, not advisory.
- If you are a fast/cheap-tier model and you land on a correctness-critical file, that is a signal to slow down, not a reason to skip the file's higher bar.

---

## 4. Hard rules (violating these is worse than an incomplete feature)

- **Never write placeholder or mocked cryptography** in `core` and call it done. If BBS+ or Noir integration isn't ready, leave it visibly unfinished with a `// SPEC-GAP:` comment and a failing test — do not stub a fake "always returns true" verifier.
- **Never write a claim-specific function.** No `checkAge()`, `verifyGPA()`, `isNigerian()`. Every claim must route through the generic predicate engine (`evaluatePredicate(attribute, operator, threshold)`). If you catch yourself writing a special case, stop and restructure as configuration instead.
- **Never let `verification_receipts` gain a personal-data column.** Before adding any field to that table, ask: could this identify a specific holder? If yes, it does not belong there.
- **Never add a blockchain, DID registry, or external ledger dependency.** Use `did:key` only. This is a deliberate simplicity constraint, not a gap to "helpfully" fix.
- **Never silently swallow a verification failure.** Every rejection path returns one of the typed reasons defined in the build spec (`SignatureInvalid`, `PredicateNotSatisfied`, `CredentialExpired`, `CredentialRevoked`, `SessionTokenExpired`, `SessionTokenReused`) — never a bare `false` with no diagnostic.
- **Never commit key material, secrets, or `.env` files.** Use `.env.example` with placeholder values only.
- **Never break a passing checkpoint to start the next one.** If your change to checkpoint N+1 causes checkpoint N's acceptance test to fail, that's a regression — fix it before continuing, don't leave it for "later cleanup."

---

## 5. Coding conventions

See `TRUSTPASS_BUILD_SPEC.md` Section 4 for the full list (naming conventions, error typing, health checks, etc.). Highlights worth repeating here because they're easy to drift on across a long agent session:

- `snake_case` (Rust, SQL) / `camelCase` at Go JSON/gRPC boundaries / kebab-case (dirs, Docker services).
- Every service exposes `/healthz` (or gRPC equivalent).
- No hand-edits to generated protobuf code — regenerate from `.proto` instead.
- Cite the exact crate/library version you used for cryptographic primitives in `docs/ARCHITECTURE.md` when you touch `core`.

---

## 6. When you're unsure

Do not guess silently on anything that affects:
- what data crosses the verifier boundary,
- what counts as "personal data" for the receipts table,
- whether a shortcut would violate the predicate-first rule,
- session-token/replay semantics.

Instead: leave a `// SPEC-GAP: <what you assumed and why>` comment at the point of ambiguity, make the most conservative assumption (favor revealing less, not more), and note it in `docs/PROGRESS.md` for a human or reviewing agent to confirm.

For everything else (variable naming, internal file structure within a service, minor UI layout choices) — use your judgment and move on. Not every decision needs a flag; save flags for things that actually touch the four non-negotiable requirements in the build spec's Section 2.

---

## 7. Session handoff — `docs/PROGRESS.md`

Every agent session should leave this file in a state the next session (possibly a different model) can pick up from cold. Before ending a session, update it with:

```markdown
## Session [date/identifier]
**Model:** [which model ran this session]
**Checkpoint worked on:** [N — name from build spec Section 9]
**Status:** [not started / in progress / acceptance criteria met]
**What changed:** [short factual list, not a narrative]
**Open SPEC-GAP flags introduced this session:** [list file:line + assumption, or "none"]
**Next step:** [the specific next action, not "continue building"]
```

Do not overwrite previous entries — append. If `docs/PROGRESS.md` doesn't exist yet, create it with this structure.

---

## 8. Testing expectations

- Every checkpoint's acceptance criteria (build spec Section 9) must be runnable as an actual test or script, not just something a human eyeballs. If a criterion is currently only manually verifiable, write the automation for it as part of finishing that checkpoint.
- `core` correctness paths (Checkpoints 2, 3, 5, 6, 7) need explicit **negative** tests, not just happy-path ones: a tampered signature, an expired credential, a revoked credential, a replayed session token, an unsatisfied predicate. A checkpoint that only demonstrates the positive case is not done.
- Don't delete or weaken an existing test to make a build pass. If a test seems wrong, flag it in `docs/PROGRESS.md` rather than quietly loosening it.

---

## 9. Non-goals reminder

Re-read Section 10 of `TRUSTPASS_BUILD_SPEC.md` ("Explicit Non-Goals") before adding any new dependency, service, or infrastructure component that isn't already named in the repo layout. If what you're about to build isn't in the spec's layout or explicitly requested, that's a signal to stop and ask, not a gap to fill proactively.

---

## 10. Communication style for this project

When reporting back (in PR descriptions, `docs/PROGRESS.md`, or session summaries): be factual and specific, not narrative. State what was built, what test proves it works, and what's still open. Avoid marketing language ("robust," "seamless," "production-ready") — say what's actually true, including limitations.
