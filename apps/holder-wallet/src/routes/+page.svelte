<script>
  /**
   * Route: / — Credential List
   *
   * Displays all credentials in the holder wallet.
   * TODO (next agent): call loadCredentials() on mount to sync from issuer-api.
   */
  import { wallet } from '$lib/stores/wallet.js';
  import CredentialCard from '$lib/components/CredentialCard.svelte';
  import Button from '$lib/components/Button.svelte';
  import { goto } from '$app/navigation';

  // TODO: replace mock with real API fetch
  // import { loadCredentials } from '$lib/api/issuer.js';
  // onMount(loadCredentials);
</script>

<svelte:head>
  <title>My Credentials — TrustPass Wallet</title>
</svelte:head>

<div class="tp-page-header">
  <h1 class="tp-page-title">My Credentials</h1>
  <p class="tp-page-subtitle">Your verifiable credentials. Tap one to view details.</p>
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
      Credentials issued to you will appear here. Contact your issuer to get started.
    </p>
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
