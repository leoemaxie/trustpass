<script>
  /**
   * Route: / — Credential List
   *
   * Displays all credentials in the holder wallet.
   * Supports syncing from issuer-api and quick-issuing demo records.
   */
  import { onMount } from 'svelte';
  import { wallet, pendingClaimRequest } from '$lib/stores/wallet.js';
  import CredentialCard from '$lib/components/CredentialCard.svelte';
  import Button from '$lib/components/Button.svelte';
  import { goto } from '$app/navigation';

  let loading = false;
  let statusMsg = '';

  const DEFAULT_HOLDER_DID = 'did:key:zUC724vsrMwHvKyqDdHtrh7z2GNe5xbsfgivth466P4vm2iaJLW9kK48DbgKa32yL944yK9k';

  onMount(async () => {
    if ($wallet.length === 0) {
      await syncFromIssuer();
    }
  });

  async function syncFromIssuer() {
    loading = true;
    statusMsg = 'Syncing from issuer...';
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
          statusMsg = '';
          loading = false;
          return;
        }
      }
    } catch (e) {
      console.warn('Sync failed:', e);
    }
    statusMsg = '';
    loading = false;
  }

  async function loadDemoCredentials() {
    loading = true;
    statusMsg = 'Issuing signed demo credentials from issuer-api...';
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

      statusMsg = 'Demo credentials ready!';
      setTimeout(() => statusMsg = '', 3000);
    } catch (e) {
      alert('Issuance failed: ' + e.message);
    } finally {
      loading = false;
    }
  }

  async function loadMinorCredential() {
    loading = true;
    statusMsg = 'Issuing minor credential (under 18)...';
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
      }
      statusMsg = 'Minor credential issued!';
      setTimeout(() => statusMsg = '', 3000);
    } catch (e) {
      alert('Issuance failed: ' + e.message);
    } finally {
      loading = false;
    }
  }

  // Quick Verification session initiator (Demo scenarios)
  async function startVerificationDemo(type) {
    loading = true;
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

      goto('/claim-review');
    } catch (e) {
      alert('Verification initiation failed: ' + e.message);
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>My Credentials — TrustPass Wallet</title>
</svelte:head>

<div class="tp-page-header">
  <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap;">
    <div>
      <h1 class="tp-page-title">My Credentials</h1>
      <p class="tp-page-subtitle">Your private verifiable credentials. Tap one to view details.</p>
    </div>
    <div style="display: flex; gap: 0.5rem; flex-wrap: wrap;">
      <a href="/scan" class="tp-btn tp-btn-primary tp-btn-sm">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path d="M3 7V5a2 2 0 0 1 2-2h2"/><path d="M17 3h2a2 2 0 0 1 2 2v2"/><path d="M21 17v2a2 2 0 0 1-2 2h-2"/><path d="M7 21H5a2 2 0 0 1-2-2v-2"/>
          <rect width="7" height="7" x="7" y="7" rx="1"/>
        </svg>
        Scan Verifier QR
      </a>
      <button class="tp-btn tp-btn-ghost tp-btn-sm" on:click={syncFromIssuer} disabled={loading}>
        Sync Issuer
      </button>
    </div>
  </div>

  {#if statusMsg}
    <p class="tp-text-muted" style="font-size: var(--tp-text-sm); margin-top: var(--tp-space-2); color: var(--tp-accent-cyan);">
      {statusMsg}
    </p>
  {/if}
</div>

<!-- Demo Scenarios Fast Launcher -->
<div class="tp-card" style="margin-bottom: var(--tp-space-5); border-color: rgba(34, 211, 238, 0.3);">
  <div class="tp-card-header">
    <div>
      <h2 class="tp-card-title" style="font-size: 1rem;">Demo Scenarios (NITDA Track B)</h2>
      <p class="tp-card-subtitle">Generate zero-knowledge proofs directly from your signed credentials:</p>
    </div>
  </div>
  <div class="tp-card-body" style="display: flex; gap: var(--tp-space-3); flex-wrap: wrap;">
    <button
      class="tp-btn tp-btn-secondary tp-btn-sm"
      on:click={() => startVerificationDemo('age')}
      disabled={loading || !$wallet.some(c => c.type === 'NationalIDCredential')}
    >
      Age Verification (Age ≥ 18)
    </button>
    <button
      class="tp-btn tp-btn-secondary tp-btn-sm"
      on:click={() => startVerificationDemo('student')}
      disabled={loading || !$wallet.some(c => c.type === 'StudentCredential')}
    >
      Academic Eligibility (GPA ≥ 3.50)
    </button>
    <button
      class="tp-btn tp-btn-ghost tp-btn-sm"
      on:click={loadDemoCredentials}
      disabled={loading}
    >
      + Load Demo Credentials
    </button>
    <button
      class="tp-btn tp-btn-ghost tp-btn-sm"
      on:click={loadMinorCredential}
      disabled={loading}
      style="color: var(--tp-signal-fail);"
    >
      + Load Minor (Under 18)
    </button>
  </div>
</div>

{#if $wallet.length === 0}
  <div class="tp-empty">
    <div class="tp-empty__icon" aria-hidden="true">
      <svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect width="18" height="11" x="3" y="11" rx="2" ry="2"/>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
    </div>
    <p class="tp-empty__title">No credentials yet</p>
    <p class="tp-empty__description">
      Load demo credentials or issue one from the Issuer Console to get started.
    </p>
    <div style="margin-top: var(--tp-space-4);">
      <Button variant="primary" on:click={loadDemoCredentials} loading={loading}>
        Load Demo Credentials
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

<style>
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
</style>
