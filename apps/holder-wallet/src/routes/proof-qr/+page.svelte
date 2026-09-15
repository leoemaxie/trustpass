<script>
  /**
   * Route: /proof-qr — Display proof QR for verifier to scan.
   *
   * Reads proofQrPayload store. Shows QR + session timer.
   * On expiry, returns home with a message.
   */
  import { proofQrPayload } from '$lib/stores/wallet.js';
  import QrDisplay from '$lib/components/QrDisplay.svelte';
  import SessionTimer from '$lib/components/SessionTimer.svelte';
  import Button from '$lib/components/Button.svelte';
  import { goto } from '$app/navigation';

  const payload = $proofQrPayload;
  if (!payload) goto('/');

  function handleExpired() {
    proofQrPayload.set(null);
    goto('/');
  }
</script>

<svelte:head>
  <title>Show Proof — TrustPass Wallet</title>
</svelte:head>

<div class="proof-qr-page">
  <div class="tp-page-header">
    <h1 class="tp-page-title">Show this to the verifier</h1>
    <p class="tp-page-subtitle">Let them scan the QR code below.</p>
  </div>

  {#if payload}
    <div class="proof-qr-page__timer">
      <SessionTimer expiresAt={payload.expiresAt} on:expired={handleExpired} />
      <span style="font-size: var(--tp-text-xs); color: var(--tp-text-muted);">Session expires</span>
    </div>

    <div class="proof-qr-page__qr">
      <QrDisplay value={payload.encodedProof} label="Proof QR code" size={280}>
        <svelte:fragment slot="label">
          Scan this with the verifier's device
        </svelte:fragment>
      </QrDisplay>
    </div>

    <div class="proof-qr-page__info">
      <p class="tp-text-muted" style="font-size: var(--tp-text-sm); text-align: center;">
        This QR code can only be scanned once. If it expires, go back and generate a new one.
      </p>
    </div>

    <Button variant="ghost" full href="/">Back to Credentials</Button>
  {/if}
</div>

<style>
  .proof-qr-page {
    max-width: 400px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: var(--tp-space-6);
    align-items: center;
  }

  .proof-qr-page__timer {
    display: flex;
    align-items: center;
    gap: var(--tp-space-3);
  }

  .proof-qr-page__qr {
    width: 100%;
    display: flex;
    justify-content: center;
  }

  .proof-qr-page__info {
    max-width: 320px;
  }
</style>
