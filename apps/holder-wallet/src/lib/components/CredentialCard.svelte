<script>
  /**
   * CredentialCard — displays one credential in the holder wallet list.
   *
   * Props:
   *   credential {Object} — the credential object from local wallet store
   *     .id           {string}
   *     .type         {string}   — schema name e.g. "NationalIDCredential"
   *     .issuerDid    {string}
   *     .issuedAt     {string}   — ISO date
   *     .expiresAt    {string|null}
   *     .revoked      {boolean}
   *
   * Emits:
   *   on:select — when the card is clicked/activated
   */
  export let credential;

  import Badge from './Badge.svelte';
  import MonoDisplay from './MonoDisplay.svelte';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  $: status = credential.revoked
    ? 'fail'
    : isExpired(credential.expiresAt)
      ? 'fail'
      : 'pass';

  $: statusLabel = credential.revoked
    ? 'Revoked'
    : isExpired(credential.expiresAt)
      ? 'Expired'
      : 'Active';

  function isExpired(dateStr) {
    return dateStr ? new Date(dateStr) < new Date() : false;
  }

  function formatDate(dateStr) {
    if (!dateStr) return '—';
    return new Date(dateStr).toLocaleDateString(undefined, {
      year: 'numeric', month: 'short', day: 'numeric',
    });
  }

  function formatType(type) {
    // "NationalIDCredential" → "National ID"
    return (type || '')
      .replace('Credential', '')
      .replace(/([A-Z])/g, ' $1')
      .trim();
  }
</script>

<div
  class="tp-credential-card"
  class:credential-card--revoked={credential.revoked}
  role="button"
  tabindex="0"
  aria-label="View {credential.type}"
  on:click
  on:keydown={e => e.key === 'Enter' && dispatch('select')}
>
  <div class="tp-credential-card__header">
    <span class="tp-credential-card__type">{formatType(credential.type)}</span>
    <Badge variant={status}>{statusLabel}</Badge>
  </div>

  <div class="tp-credential-card__body">
    <div>
      <p class="tp-label">Issuer DID</p>
      <MonoDisplay value={credential.issuerDid} label="Issuer DID" truncate />
    </div>
    <div>
      <p class="tp-label">Issued</p>
      <p style="font-size: var(--tp-text-sm); color: var(--tp-text-primary);">
        {formatDate(credential.issuedAt)}
      </p>
    </div>
  </div>

  <div class="tp-credential-card__footer">
    <span class="tp-credential-card__expiry">
      {credential.expiresAt ? `Expires ${formatDate(credential.expiresAt)}` : 'No expiry'}
    </span>
    <!-- Arrow indicator -->
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
         stroke-width="2" style="color: var(--tp-text-muted);" aria-hidden="true">
      <path d="m9 18 6-6-6-6"/>
    </svg>
  </div>
</div>

<style>
  .credential-card--revoked {
    opacity: 0.6;
    border-style: dashed;
  }
</style>
