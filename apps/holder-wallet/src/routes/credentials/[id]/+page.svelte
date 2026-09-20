<script>
  import { page } from '$app/stores';
  import { wallet } from '$lib/stores/wallet.js';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import MonoDisplay from '$lib/components/MonoDisplay.svelte';

  const credId = $page.params.id;
  $: credential = $wallet.find(c => c.id === credId || c.credentialUrn === credId);

  function formatDate(iso) {
    if (!iso) return '—';
    return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
  }

  $: attributes = credential?.attributes || (credential?.data?.credentialSubject?.claims) || {};
</script>

<svelte:head>
  <title>Credential Details — TrustPass Wallet</title>
</svelte:head>

<div class="cred-detail">
  <div class="tp-page-header">
    <a href="/" class="tp-btn tp-btn-ghost tp-btn-sm" style="margin-bottom: var(--tp-space-3); display: inline-flex;">
      ← Back to Credentials
    </a>
    <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap;">
      <div>
        <h1 class="tp-page-title">{credential ? credential.type : 'Credential Details'}</h1>
        <p class="tp-page-subtitle">Stored securely on your device.</p>
      </div>
      {#if credential}
        <Badge variant={credential.revoked ? 'fail' : 'pass'}>
          {credential.revoked ? 'Revoked' : 'Active'}
        </Badge>
      {/if}
    </div>
  </div>

  {#if !credential}
    <div class="tp-empty">
      <p class="tp-empty__title">Credential not found</p>
      <p class="tp-empty__description">This credential may have been removed or does not exist.</p>
      <Button href="/" variant="primary">Return Home</Button>
    </div>
  {:else}
    <div class="tp-card">
      <div class="tp-card-header">
        <h2 class="tp-card-title">Credential Metadata</h2>
      </div>
      <div class="tp-card-body">
        <div class="tp-field">
          <p class="tp-label">Credential ID</p>
          <code class="tp-mono" style="word-break: break-all;">{credential.id}</code>
        </div>
        <div class="tp-field">
          <p class="tp-label">Issuer DID</p>
          <MonoDisplay value={credential.issuerDid || credential.data?.issuer} label="Issuer DID" />
        </div>
        <div class="metadata-row">
          <div>
            <p class="tp-label">Issued Date</p>
            <p style="font-size: var(--tp-text-sm);">{formatDate(credential.issuedAt)}</p>
          </div>
          <div>
            <p class="tp-label">Expiry Date</p>
            <p style="font-size: var(--tp-text-sm);">{formatDate(credential.expiresAt)}</p>
          </div>
        </div>
      </div>
    </div>

    <div class="tp-card" style="margin-top: var(--tp-space-4);">
      <div class="tp-card-header">
        <div>
          <h2 class="tp-card-title">Attributes (Private Record)</h2>
          <p class="tp-card-subtitle">These are stored only on your device and are never sent whole to a verifier.</p>
        </div>
      </div>
      <div class="tp-card-body">
        <table class="tp-table">
          <thead>
            <tr>
              <th>Attribute</th>
              <th>Stored Value</th>
            </tr>
          </thead>
          <tbody>
            {#each Object.entries(attributes) as [key, val]}
              <tr>
                <td><code class="tp-mono">{key}</code></td>
                <td style="font-weight: 500;">{String(val)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}
</div>

<style>
  .cred-detail {
    max-width: 640px;
    margin: 0 auto;
  }
  .metadata-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--tp-space-4);
    margin-top: var(--tp-space-3);
  }
</style>
