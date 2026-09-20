<script>
  /**
   * Route: /claim-review — Claim Request Review
   *
   * THE most important trust screen in the wallet.
   * Shows the holder exactly what is being asked and what will/won't be revealed.
   *
   * Reads from the pendingClaimRequest store (set by /scan route after QR decode).
   * Proof generate button only appears after the holder scrolls past the disclosure.
   *
   * TODO (next agent):
   *   1. Wire generateProof() to call verifier-api POST /verification/sessions/prove
   *      passing { sessionToken, credentialId, claimRequest }.
   *   2. On success, set proofQrPayload store and navigate to /proof-qr.
   */
  import { pendingClaimRequest, proofQrPayload, wallet } from '$lib/stores/wallet.js';
  import Button from '$lib/components/Button.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import SessionTimer from '$lib/components/SessionTimer.svelte';
  import { goto } from '$app/navigation';
  import { get } from 'svelte/store';

  let generating = false;
  let buttonVisible = false;
  let disclosureEnd;

  const claim = $pendingClaimRequest;

  // Guard: redirect if no pending claim
  if (!claim) {
    goto('/scan');
  }

  // Intersection observer to reveal the generate button after scrolling
  import { onMount } from 'svelte';
  onMount(() => {
    if (!disclosureEnd) return;
    const observer = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting) buttonVisible = true; },
      { threshold: 0.5 }
    );
    observer.observe(disclosureEnd);
    return () => observer.disconnect();
  });

  // Find the matching credential for this claim
  $: matchingCredential = $wallet.find(
    c => c.type === claim?.claimRequest?.schemaName
  );

  // All attribute names for the matching schema (to list hidden ones)
  $: allAttributes = matchingCredential
    ? Object.keys(matchingCredential.attributes || {})
    : [];

  $: revealedAttribute = claim?.claimRequest?.attributeName;
  $: hiddenAttributes  = allAttributes.filter(a => a !== revealedAttribute);

  function formatOperator(op) {
    const map = { GTE: '≥', EQ: '=', IN_SET: 'in', BEFORE_DATE: 'before' };
    return map[op] || op;
  }

  async function generateProof() {
    generating = true;
    try {
      if (!matchingCredential) {
        throw new Error(`No credential found for schema "${claim?.claimRequest?.schemaName}" in your wallet.`);
      }

      let credentialPayload = matchingCredential.data || matchingCredential;
      if (typeof credentialPayload === 'string') {
        try {
          credentialPayload = JSON.parse(credentialPayload);
        } catch {}
      }

      const res = await fetch('/api/verifier/verification/sessions/prove', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sessionToken: claim.sessionToken,
          credential: credentialPayload,
          claimRequest: claim.claimRequest,
        }),
      });

      const data = await res.json();
      if (!res.ok || data.valid === false) {
        throw new Error(data.errorMessage || data.rejectionReason || 'Proof generation failed');
      }

      proofQrPayload.set({
        encodedProof: JSON.stringify({
          sessionToken: claim.sessionToken,
          proof: data.proof,
          claimRequest: claim.claimRequest,
          claimSummary: `${claim.claimRequest.attributeName} ${formatOperator(claim.claimRequest.operator)} ${claim.claimRequest.value}`,
          expiresAt: claim.expiresAt,
        }),
        sessionToken: claim.sessionToken,
        expiresAt: claim.expiresAt,
      });

      goto('/proof-qr');
    } catch (e) {
      alert('Proof generation failed: ' + e.message);
    } finally {
      generating = false;
    }
  }
</script>

<svelte:head>
  <title>Verify Request — TrustPass Wallet</title>
</svelte:head>

{#if claim}
<div class="claim-review">

  <!-- ── Header ───────────────────────────────────────────────────────── -->
  <div class="claim-review__header">
    <div>
      <h1 class="tp-page-title">Verification Request</h1>
      <p class="tp-page-subtitle">Review what's being asked before you proceed.</p>
    </div>
    <SessionTimer expiresAt={claim.expiresAt} on:expired={() => goto('/')} />
  </div>

  <!-- ── Section 1: What is being asked ──────────────────────────────── -->
  <div class="tp-disclosure-section">
    <div class="tp-disclosure-section__header">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tp-text-muted)" stroke-width="2" aria-hidden="true">
        <circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>
      </svg>
      <h2 class="tp-disclosure-section__title" style="color: var(--tp-text-muted)">What's being asked</h2>
    </div>
    <div class="tp-disclosure-section__body">
      <p style="font-size: var(--tp-text-base); color: var(--tp-text-primary);">
        The verifier is asking to confirm:
      </p>
      <div class="claim-predicate-display">
        <span class="claim-predicate__attr">{claim.claimRequest.attributeName}</span>
        <span class="claim-predicate__op">{formatOperator(claim.claimRequest.operator)}</span>
        <span class="claim-predicate__val">{claim.claimRequest.value}</span>
      </div>
      <p class="tp-text-muted" style="font-size: var(--tp-text-sm);">
        Schema: <code style="font-family: var(--tp-font-mono); font-size: 0.8em;">{claim.claimRequest.schemaName}</code>
      </p>
    </div>
  </div>

  <!-- ── Section 2: What will be revealed ────────────────────────────── -->
  <div class="tp-disclosure-section">
    <div class="tp-disclosure-section__header">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tp-signal-pass)" stroke-width="2" aria-hidden="true">
        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>
      </svg>
      <h2 class="tp-disclosure-section__title" style="color: var(--tp-signal-pass)">What will be revealed</h2>
    </div>
    <div class="tp-disclosure-section__body">
      <div class="tp-attr-row">
        <span class="tp-attr-row__name">{claim.claimRequest.attributeName}</span>
        <div class="tp-attr-row__value">
          <Badge variant="pass">Yes / No result only</Badge>
        </div>
      </div>
      <p style="font-size: var(--tp-text-xs); color: var(--tp-text-muted);">
        Only a cryptographic proof that the condition is met — the actual value is never sent.
      </p>
    </div>
  </div>

  <!-- ── Section 3: What will stay hidden ─────────────────────────────── -->
  <div class="tp-disclosure-section">
    <div class="tp-disclosure-section__header">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tp-accent-cyan)" stroke-width="2" aria-hidden="true">
        <rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
      <h2 class="tp-disclosure-section__title" style="color: var(--tp-accent-cyan)">What stays hidden</h2>
    </div>
    <div class="tp-disclosure-section__body">
      {#if hiddenAttributes.length}
        {#each hiddenAttributes as attr}
          <div class="tp-attr-row tp-attr-row--hidden">
            <span class="tp-attr-row__name">{attr}</span>
            <div class="tp-attr-row__value">
              <span class="tp-zk-hidden-badge">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
                  <rect width="18" height="11" x="3" y="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                </svg>
                Hidden by ZK proof
              </span>
            </div>
          </div>
        {/each}
      {:else}
        <p class="tp-text-muted" style="font-size: var(--tp-text-sm);">No other attributes in this credential.</p>
      {/if}
    </div>
  </div>

  <!-- ── Scroll anchor for button reveal ──────────────────────────────── -->
  <div bind:this={disclosureEnd} aria-hidden="true"></div>

  <!-- ── Generate proof button ─────────────────────────────────────────── -->
  {#if buttonVisible}
    <div class="claim-review__action">
      <Button variant="primary" full size="lg" loading={generating} on:click={generateProof}>
        {generating ? 'Generating Proof…' : 'Generate Proof & Continue'}
      </Button>
      <Button variant="ghost" full href="/" disabled={generating}>Cancel</Button>
    </div>
  {:else}
    <p class="scroll-prompt" aria-hidden="true">↓ Scroll to review, then continue</p>
  {/if}

</div>
{/if}

<style>
  .claim-review {
    display: flex;
    flex-direction: column;
    gap: var(--tp-space-5);
    max-width: 560px;
    margin: 0 auto;
  }

  .claim-review__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--tp-space-4);
    flex-wrap: wrap;
  }

  .claim-predicate-display {
    display: flex;
    align-items: center;
    gap: var(--tp-space-3);
    background: var(--tp-bg-dark);
    border: 1px solid var(--tp-border);
    border-radius: var(--tp-radius-md);
    padding: var(--tp-space-3) var(--tp-space-4);
  }

  .claim-predicate__attr {
    font-family: var(--tp-font-mono);
    font-size: var(--tp-text-sm);
    color: var(--tp-accent-cyan);
  }
  .claim-predicate__op {
    font-size: var(--tp-text-lg);
    font-weight: 700;
    color: var(--tp-text-muted);
  }
  .claim-predicate__val {
    font-family: var(--tp-font-mono);
    font-size: var(--tp-text-sm);
    color: var(--tp-signal-pass);
  }

  .claim-review__action {
    display: flex;
    flex-direction: column;
    gap: var(--tp-space-3);
    padding: var(--tp-space-6) 0;
  }

  .scroll-prompt {
    text-align: center;
    color: var(--tp-text-muted);
    font-size: var(--tp-text-sm);
    padding: var(--tp-space-4) 0;
    animation: tp-scan-pulse 2s ease-in-out infinite;
  }
</style>
