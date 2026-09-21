<script>
  /**
   * Route: / — Digital Credentials Vault
   *
   * Main dashboard for TrustPass Holder Wallet.
   * Displays cryptographic verifiable credentials, wallet identity state,
   * verification request scanners, and zero-knowledge policy simulation.
   */
  import { onMount } from 'svelte';
  import { wallet, pendingClaimRequest } from '$lib/stores/wallet.js';
  import CredentialCard from '$lib/components/CredentialCard.svelte';
  import Button from '$lib/components/Button.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import MonoDisplay from '$lib/components/MonoDisplay.svelte';
  import { goto } from '$app/navigation';

  let loading = false;
  let banner = { show: false, message: '', type: 'info' };
  let showVerificationTools = true;

  const DEFAULT_HOLDER_DID = 'did:key:zUC724vsrMwHvKyqDdHtrh7z2GNe5xbsfgivth466P4vm2iaJLW9kK48DbgKa32yL944yK9k';

  $: activeCount = $wallet.filter(c => !c.revoked && (!c.expiresAt || new Date(c.expiresAt) > new Date())).length;
  $: hasNationalId = $wallet.some(c => c.type === 'NationalIDCredential' && !c.revoked);
  $: hasStudentId = $wallet.some(c => c.type === 'StudentCredential' && !c.revoked);

  onMount(async () => {
    if ($wallet.length === 0) {
      await syncFromIssuer();
    }
  });

  function showNotification(message, type = 'info', timeoutMs = 4000) {
    banner = { show: true, message, type };
    if (timeoutMs > 0) {
      setTimeout(() => {
        if (banner.message === message) banner.show = false;
      }, timeoutMs);
    }
  }

  async function syncFromIssuer() {
    loading = true;
    showNotification('Synchronizing credentials from identity authority...', 'info', 0);
    try {
      const res = await fetch('/api/issuer/credentials');
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data) && data.length > 0) {
          data.forEach(c => {
            wallet.upsert({
              id: c.credentialUrn || c.id,
              type: c.type || c.schemaName,
              holderDid: c.holderDid,
              issuerDid: DEFAULT_HOLDER_DID,
              issuedAt: c.issuedAt,
              expiresAt: c.expiresAt,
              revoked: !!c.revoked,
              attributes: c.attributes || {},
              data: c.data,
            });
          });
          showNotification(`Synchronized ${data.length} credentials successfully.`, 'success');
          loading = false;
          return;
        }
      }
      showNotification('Wallet is up to date.', 'info');
    } catch (e) {
      showNotification('Unable to reach issuer service. Operating in local mode.', 'info');
    } finally {
      loading = false;
    }
  }

  async function provisionStandardCredentials() {
    loading = true;
    showNotification('Provisioning cryptographically signed identity credentials...', 'info', 0);
    try {
      // 1. Issue NationalIDCredential (Adult, DOB: 1999-07-20)
      const res1 = await fetch('/api/issuer/credentials/issue', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          schemaName: 'NationalIDCredential',
          holderDid: DEFAULT_HOLDER_DID,
          attributes: {
            fullName: 'Adebayo Olawale',
            dateOfBirth: '1999-07-20',
            nationality: 'NG',
            idNumber: 'NIN-8392104',
          },
        }),
      });

      if (res1.ok) {
        const vc1 = await res1.json();
        wallet.upsert({
          id: vc1.id,
          type: 'NationalIDCredential',
          holderDid: DEFAULT_HOLDER_DID,
          issuerDid: vc1.issuer,
          issuedAt: vc1.issuanceDate,
          expiresAt: vc1.expirationDate,
          revoked: false,
          attributes: vc1.credentialSubject.claims,
          data: vc1,
        });
      }

      // 2. Issue StudentCredential (GPA 3.82)
      const res2 = await fetch('/api/issuer/credentials/issue', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          schemaName: 'StudentCredential',
          holderDid: DEFAULT_HOLDER_DID,
          attributes: {
            studentId: 'UNILAG-2023-8821',
            university: 'University of Lagos',
            enrollmentStatus: 'active',
            gpa: '3.82',
          },
        }),
      });

      if (res2.ok) {
        const vc2 = await res2.json();
        wallet.upsert({
          id: vc2.id,
          type: 'StudentCredential',
          holderDid: DEFAULT_HOLDER_DID,
          issuerDid: vc2.issuer,
          issuedAt: vc2.issuanceDate,
          expiresAt: vc2.expirationDate,
          revoked: false,
          attributes: vc2.credentialSubject.claims,
          data: vc2,
        });
      }

      showNotification('Standard identity credentials successfully issued and stored.', 'success');
    } catch (e) {
      showNotification('Issuance failed: ' + e.message, 'error', 6000);
    } finally {
      loading = false;
    }
  }

  async function provisionMinorCredential() {
    loading = true;
    showNotification('Provisioning National ID (under 18 profile)...', 'info', 0);
    try {
      const res = await fetch('/api/issuer/credentials/issue', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          schemaName: 'NationalIDCredential',
          holderDid: DEFAULT_HOLDER_DID,
          attributes: {
            fullName: 'Chidinma Eze (Minor)',
            dateOfBirth: '2012-05-15',
            nationality: 'NG',
            idNumber: 'NIN-9912041',
          },
        }),
      });

      if (res.ok) {
        const vc = await res.json();
        wallet.upsert({
          id: vc.id,
          type: 'NationalIDCredential',
          holderDid: DEFAULT_HOLDER_DID,
          issuerDid: vc.issuer,
          issuedAt: vc.issuanceDate,
          expiresAt: vc.expirationDate,
          revoked: false,
          attributes: vc.credentialSubject.claims,
          data: vc,
        });
        showNotification('Minor National ID credential provisioned.', 'success');
      } else {
        throw new Error(`Server returned ${res.status}`);
      }
    } catch (e) {
      showNotification('Issuance failed: ' + e.message, 'error', 6000);
    } finally {
      loading = false;
    }
  }

  async function simulateVerification(type) {
    loading = true;
    showNotification('Initiating zero-knowledge verification session...', 'info', 0);
    try {
      let claimRequest;
      if (type === 'age') {
        claimRequest = {
          schemaName: 'NationalIDCredential',
          attributeName: 'dateOfBirth',
          operator: 'BEFORE_DATE',
          value: '2008-09-12',
        };
      } else {
        claimRequest = {
          schemaName: 'StudentCredential',
          attributeName: 'gpa',
          operator: 'GTE',
          value: '3.50',
        };
      }

      const res = await fetch('/api/verifier/verification/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ claimRequest, ttlSeconds: 120 }),
      });

      if (!res.ok) {
        throw new Error('Failed to initiate verification session');
      }

      const session = await res.json();
      pendingClaimRequest.set({
        sessionToken: session.sessionToken,
        claimRequest: session.claimRequest,
        expiresAt: session.expiresAt,
      });

      banner.show = false;
      goto('/claim-review');
    } catch (e) {
      showNotification('Verification session failed: ' + e.message, 'error', 6000);
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>Credentials Vault — TrustPass Identity</title>
</svelte:head>

<div class="wallet-dashboard">
  <!-- Header & Primary Actions -->
  <header class="tp-page-header">
    <div class="header-content">
      <div>
        <div class="header-badges">
          <Badge variant="crypto">Zero-Knowledge</Badge>
          <span class="protocol-chip">did:key · Ed25519</span>
        </div>
        <h1 class="tp-page-title">Digital Credentials</h1>
        <p class="tp-page-subtitle">
          Encrypted, privacy-preserving identity claims. Present zero-knowledge proofs without exposing underlying personal data.
        </p>
      </div>

      <div class="header-actions">
        <a href="/scan" class="tp-btn tp-btn-primary">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path d="M3 7V5a2 2 0 0 1 2-2h2"/><path d="M17 3h2a2 2 0 0 1 2 2v2"/><path d="M21 17v2a2 2 0 0 1-2 2h-2"/><path d="M7 21H5a2 2 0 0 1-2-2v-2"/>
            <rect width="7" height="7" x="7" y="7" rx="1"/>
          </svg>
          Scan Verifier QR
        </a>

        <button class="tp-btn tp-btn-secondary" on:click={syncFromIssuer} disabled={loading}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class:spin={loading} aria-hidden="true">
            <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/>
          </svg>
          Sync Issuer
        </button>
      </div>
    </div>

    <!-- Notification Banner -->
    {#if banner.show}
      <div class="feedback-banner feedback-banner--{banner.type}" role="status">
        <div class="feedback-banner__content">
          {#if banner.type === 'success'}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <path d="M20 6L9 17l-5-5"/>
            </svg>
          {:else if banner.type === 'error'}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
            </svg>
          {:else}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spin" aria-hidden="true">
              <circle cx="12" cy="12" r="10" stroke-dasharray="32" stroke-dashoffset="12"/>
            </svg>
          {/if}
          <span>{banner.message}</span>
        </div>
        <button class="feedback-banner__close" on:click={() => (banner.show = false)} aria-label="Dismiss notification">
          &times;
        </button>
      </div>
    {/if}
  </header>

  <!-- Wallet Metrics Overview -->
  <section class="tp-stat-grid" aria-label="Wallet Overview">
    <div class="tp-stat-card">
      <span class="tp-stat-card__label">Active Credentials</span>
      <span class="tp-stat-card__value tp-stat-card__value--pass">{activeCount}</span>
      <span class="stat-meta">Verified cryptographic records</span>
    </div>

    <div class="tp-stat-card">
      <span class="tp-stat-card__label">Total Vault Items</span>
      <span class="tp-stat-card__value tp-stat-card__value--cyan">{$wallet.length}</span>
      <span class="stat-meta">Stored on-device</span>
    </div>

    <div class="tp-stat-card">
      <span class="tp-stat-card__label">ZK Privacy Engine</span>
      <div class="stat-status-row">
        <span class="status-indicator"></span>
        <span class="status-text">Active</span>
      </div>
      <span class="stat-meta">Zero data disclosure</span>
    </div>
  </section>

  <!-- Interactive Verification & Testing Suite -->
  <section class="verification-suite tp-card">
    <div class="suite-header" on:click={() => (showVerificationTools = !showVerificationTools)} role="button" tabindex="0" on:keydown={(e) => e.key === 'Enter' && (showVerificationTools = !showVerificationTools)}>
      <div class="suite-title-group">
        <div class="suite-icon-wrap">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            <path d="m9 12 2 2 4-4"/>
          </svg>
        </div>
        <div>
          <div style="display: flex; align-items: center; gap: var(--tp-space-2);">
            <h2 class="suite-title">Verification Policy Simulator</h2>
            <Badge variant="neutral">Evaluation Suite</Badge>
          </div>
          <p class="suite-subtitle">Trigger zero-knowledge predicate verifications against active credentials.</p>
        </div>
      </div>
      <button class="suite-toggle-btn" aria-label="Toggle policy simulation tools">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class:rotate-180={showVerificationTools}>
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </button>
    </div>

    {#if showVerificationTools}
      <div class="suite-body">
        <div class="policy-actions">
          <div class="policy-card">
            <div class="policy-info">
              <span class="policy-label">Predicate Policy</span>
              <h3 class="policy-name">Age Verification (18+)</h3>
              <p class="policy-spec">Proves <code>dateOfBirth &le; 18 years</code> without revealing actual birthdate or NIN.</p>
            </div>
            <button
              class="tp-btn tp-btn-secondary tp-btn-sm"
              on:click={() => simulateVerification('age')}
              disabled={loading || !hasNationalId}
            >
              Verify Age Policy
            </button>
          </div>

          <div class="policy-card">
            <div class="policy-info">
              <span class="policy-label">Predicate Policy</span>
              <h3 class="policy-name">Academic Standing (GPA &ge; 3.50)</h3>
              <p class="policy-spec">Proves <code>gpa &ge; 3.50</code> without revealing actual GPA score or transcript.</p>
            </div>
            <button
              class="tp-btn tp-btn-secondary tp-btn-sm"
              on:click={() => simulateVerification('student')}
              disabled={loading || !hasStudentId}
            >
              Verify GPA Policy
            </button>
          </div>
        </div>

        <div class="issuance-utilities">
          <span class="utility-label">Identity Provisioning Utilities:</span>
          <div class="utility-buttons">
            <button
              class="tp-btn tp-btn-ghost tp-btn-sm"
              on:click={provisionStandardCredentials}
              disabled={loading}
              title="Issues standard National ID and Student credentials"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 5v14M5 12h14"/>
              </svg>
              Provision Standard Profiles
            </button>

            <button
              class="tp-btn tp-btn-ghost tp-btn-sm text-subtle-fail"
              on:click={provisionMinorCredential}
              disabled={loading}
              title="Issues National ID for a holder under 18"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>
              </svg>
              Provision Minor Profile (&lt;18)
            </button>
          </div>
        </div>
      </div>
    {/if}
  </section>

  <!-- Credential Vault Section -->
  <section class="vault-section" aria-label="Credential Vault">
    <div class="vault-section__header">
      <div>
        <h2 class="vault-section__title">Stored Credentials</h2>
        <span class="vault-section__count">{$wallet.length} {$wallet.length === 1 ? 'credential' : 'credentials'}</span>
      </div>

      {#if $wallet.length > 0}
        <button class="tp-btn tp-btn-ghost tp-btn-sm" on:click={provisionStandardCredentials} disabled={loading}>
          + Add Credential
        </button>
      {/if}
    </div>

    {#if $wallet.length === 0}
      <div class="tp-empty tp-card empty-vault">
        <div class="empty-icon-shield" aria-hidden="true">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect width="18" height="14" x="3" y="5" rx="2"/>
            <circle cx="12" cy="12" r="3"/>
            <path d="M3 10h18"/>
          </svg>
        </div>
        <h3 class="tp-empty__title">No Verifiable Credentials Stored</h3>
        <p class="tp-empty__description">
          Your zero-knowledge identity vault does not currently hold any active credentials. Sync with an authorized issuer or provision standard identity profiles to begin verifying claims.
        </p>
        <div class="empty-actions">
          <Button variant="primary" on:click={provisionStandardCredentials} {loading}>
            Provision Standard Credentials
          </Button>
          <Button variant="secondary" on:click={syncFromIssuer} disabled={loading}>
            Sync Identity Authority
          </Button>
        </div>
      </div>
    {:else}
      <div class="credentials-grid">
        {#each $wallet as credential (credential.id)}
          <CredentialCard
            {credential}
            on:click={() => goto(`/credentials/${credential.id}`)}
          />
        {/each}
      </div>
    {/if}
  </section>
</div>

<style>
  .wallet-dashboard {
    display: flex;
    flex-direction: column;
    gap: var(--tp-space-6);
    padding-bottom: var(--tp-space-8);
  }

  .header-content {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: var(--tp-space-4);
    flex-wrap: wrap;
  }

  .header-badges {
    display: flex;
    align-items: center;
    gap: var(--tp-space-2);
    margin-bottom: var(--tp-space-2);
  }

  .protocol-chip {
    font-family: var(--tp-font-mono);
    font-size: var(--tp-text-xs);
    color: var(--tp-text-muted);
    background: var(--tp-bg-card);
    border: 1px solid var(--tp-border);
    padding: 2px 8px;
    border-radius: var(--tp-radius-full);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: var(--tp-space-3);
    flex-wrap: wrap;
  }

  /* Feedback Banner */
  .feedback-banner {
    margin-top: var(--tp-space-4);
    padding: var(--tp-space-3) var(--tp-space-4);
    border-radius: var(--tp-radius-md);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--tp-space-3);
    font-size: var(--tp-text-sm);
    border: 1px solid transparent;
    animation: fadeIn 0.2s ease-out;
  }

  .feedback-banner--info {
    background: var(--tp-accent-cyan-dim);
    border-color: var(--tp-accent-cyan-border);
    color: var(--tp-accent-cyan);
  }

  .feedback-banner--success {
    background: var(--tp-signal-pass-bg);
    border-color: rgba(16, 185, 129, 0.3);
    color: var(--tp-signal-pass);
  }

  .feedback-banner--error {
    background: var(--tp-signal-fail-bg);
    border-color: rgba(239, 68, 68, 0.3);
    color: var(--tp-signal-fail);
  }

  .feedback-banner__content {
    display: flex;
    align-items: center;
    gap: var(--tp-space-2);
  }

  .feedback-banner__close {
    background: none;
    border: none;
    color: inherit;
    font-size: 1.25rem;
    line-height: 1;
    cursor: pointer;
    opacity: 0.7;
  }
  .feedback-banner__close:hover {
    opacity: 1;
  }

  /* Stat Card Enhancements */
  .stat-meta {
    font-size: var(--tp-text-xs);
    color: var(--tp-text-muted);
  }

  .stat-status-row {
    display: flex;
    align-items: center;
    gap: var(--tp-space-2);
    line-height: 1;
  }

  .status-indicator {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--tp-signal-pass);
    box-shadow: 0 0 8px var(--tp-signal-pass);
  }

  .status-text {
    font-size: var(--tp-text-xl);
    font-weight: 700;
    color: var(--tp-text-primary);
  }

  /* Verification Suite Card */
  .verification-suite {
    border-color: var(--tp-border);
    background: linear-gradient(180deg, var(--tp-bg-card) 0%, rgba(15, 23, 42, 0.6) 100%);
    overflow: hidden;
  }

  .suite-header {
    padding: var(--tp-space-4) var(--tp-space-5);
    display: flex;
    align-items: center;
    justify-content: space-between;
    cursor: pointer;
    user-select: none;
    border-bottom: 1px solid var(--tp-border-subtle);
    transition: background var(--tp-transition-fast);
  }
  .suite-header:hover {
    background: rgba(255, 255, 255, 0.02);
  }

  .suite-title-group {
    display: flex;
    align-items: center;
    gap: var(--tp-space-3);
  }

  .suite-icon-wrap {
    width: 36px;
    height: 36px;
    border-radius: var(--tp-radius-md);
    background: var(--tp-accent-cyan-dim);
    color: var(--tp-accent-cyan);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .suite-title {
    font-size: var(--tp-text-base);
    font-weight: 600;
    color: var(--tp-text-primary);
    margin: 0;
  }

  .suite-subtitle {
    font-size: var(--tp-text-xs);
    color: var(--tp-text-muted);
    margin: 2px 0 0 0;
  }

  .suite-toggle-btn {
    background: none;
    border: none;
    color: var(--tp-text-muted);
    cursor: pointer;
    display: flex;
    align-items: center;
    padding: var(--tp-space-1);
    transition: transform var(--tp-transition-fast);
  }

  .rotate-180 {
    transform: rotate(180deg);
  }

  .suite-body {
    padding: var(--tp-space-5);
    display: flex;
    flex-direction: column;
    gap: var(--tp-space-4);
  }

  .policy-actions {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--tp-space-3);
  }

  @media (min-width: 768px) {
    .policy-actions {
      grid-template-columns: 1fr 1fr;
    }
  }

  .policy-card {
    background: var(--tp-bg-surface);
    border: 1px solid var(--tp-border);
    border-radius: var(--tp-radius-md);
    padding: var(--tp-space-4);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: var(--tp-space-3);
  }

  .policy-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: var(--tp-tracking-wider);
    color: var(--tp-text-muted);
    font-weight: 600;
  }

  .policy-name {
    font-size: var(--tp-text-sm);
    font-weight: 600;
    color: var(--tp-text-primary);
    margin: var(--tp-space-1) 0;
  }

  .policy-spec {
    font-size: var(--tp-text-xs);
    color: var(--tp-text-muted);
    margin: 0;
    line-height: 1.4;
  }

  .policy-spec code {
    font-family: var(--tp-font-mono);
    color: var(--tp-accent-cyan);
  }

  .issuance-utilities {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: var(--tp-space-2);
    padding-top: var(--tp-space-3);
    border-top: 1px solid var(--tp-border-subtle);
  }

  .utility-label {
    font-size: var(--tp-text-xs);
    color: var(--tp-text-muted);
    font-weight: 500;
  }

  .utility-buttons {
    display: flex;
    align-items: center;
    gap: var(--tp-space-2);
    flex-wrap: wrap;
  }

  .text-subtle-fail:hover:not(:disabled) {
    color: var(--tp-signal-fail) !important;
  }

  /* Credential Vault Grid */
  .vault-section {
    display: flex;
    flex-direction: column;
    gap: var(--tp-space-4);
  }

  .vault-section__header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    border-bottom: 1px solid var(--tp-border);
    padding-bottom: var(--tp-space-2);
  }

  .vault-section__title {
    font-size: var(--tp-text-lg);
    font-weight: 600;
    color: var(--tp-text-primary);
    margin: 0;
    display: inline-block;
  }

  .vault-section__count {
    font-size: var(--tp-text-xs);
    color: var(--tp-text-muted);
    margin-left: var(--tp-space-2);
  }

  .empty-vault {
    padding: var(--tp-space-8) var(--tp-space-6);
    background: var(--tp-bg-card);
    border-radius: var(--tp-radius-lg);
    text-align: center;
  }

  .empty-icon-shield {
    color: var(--tp-text-muted);
    margin-bottom: var(--tp-space-3);
    display: inline-flex;
  }

  .empty-actions {
    display: flex;
    justify-content: center;
    gap: var(--tp-space-3);
    margin-top: var(--tp-space-5);
    flex-wrap: wrap;
  }

  .credentials-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--tp-space-4);
  }

  @media (min-width: 640px) {
    .credentials-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (min-width: 1024px) {
    .credentials-grid {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  .spin {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: translateY(0); }
  }
</style>
