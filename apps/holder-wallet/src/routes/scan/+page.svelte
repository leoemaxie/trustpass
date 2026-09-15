<script>
  /**
   * Route: /scan — Camera scanner to read verifier's claim-request QR.
   *
   * On successful decode, parses the claim request and sets
   * pendingClaimRequest store, then navigates to /claim-review.
   *
   * TODO (next agent): the verifier-api encodes
   *   { sessionToken, claimRequest, expiresAt }
   * as JSON into the QR. Validate shape before navigating.
   */
  import { pendingClaimRequest } from '$lib/stores/wallet.js';
  import Button from '$lib/components/Button.svelte';
  import { goto } from '$app/navigation';
  import { onDestroy } from 'svelte';

  let scanning = false;
  let stream = null;
  let scanLoop = null;
  let video;
  let canvas;
  let error = null;

  async function startScan() {
    error = null;
    scanning = true;
    try {
      stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } });
      video.srcObject = stream;
      await video.play();

      const ctx = canvas.getContext('2d', { willReadFrequently: true });

      function tick() {
        if (!scanning) return;
        if (video.readyState === video.HAVE_ENOUGH_DATA) {
          canvas.width  = video.videoWidth;
          canvas.height = video.videoHeight;
          ctx.drawImage(video, 0, 0);
          const imgData = ctx.getImageData(0, 0, canvas.width, canvas.height);
          /* global jsQR */
          const code = jsQR(imgData.data, imgData.width, imgData.height, {
            inversionAttempts: 'dontInvert',
          });
          if (code?.data) {
            stopScan();
            handleQR(code.data);
            return;
          }
        }
        scanLoop = setTimeout(tick, 200);
      }
      tick();
    } catch (e) {
      error = 'Camera unavailable: ' + e.message;
      scanning = false;
    }
  }

  function stopScan() {
    scanning = false;
    clearTimeout(scanLoop);
    stream?.getTracks().forEach(t => t.stop());
    stream = null;
  }

  function handleQR(raw) {
    let parsed;
    try {
      parsed = JSON.parse(raw);
    } catch {
      error = 'Invalid QR code format.';
      return;
    }
    if (!parsed.sessionToken || !parsed.claimRequest) {
      error = 'QR code is missing required fields.';
      return;
    }
    pendingClaimRequest.set(parsed);
    goto('/claim-review');
  }

  onDestroy(stopScan);
</script>

<!-- jsQR must be available globally — vendored at /vendor/jsQR.js -->
<svelte:head>
  <title>Scan Request — TrustPass Wallet</title>
  <script src="/vendor/jsQR.js"></script>
</svelte:head>

<div class="scan-page">
  <div class="tp-page-header">
    <h1 class="tp-page-title">Scan Verifier QR</h1>
    <p class="tp-page-subtitle">Point your camera at the QR code shown by the verifier.</p>
  </div>

  <div class="tp-scanner-container">
    <!-- svelte-ignore a11y-media-has-caption -->
    <video bind:this={video} class="tp-scanner-video" playsinline muted aria-hidden="true"></video>
    <canvas bind:this={canvas} style="display:none;" aria-hidden="true"></canvas>
    {#if scanning}
      <div class="tp-scanner-overlay" aria-hidden="true">
        <div class="tp-scanner-frame"></div>
      </div>
    {/if}
  </div>

  {#if error}
    <p role="alert" style="color: var(--tp-signal-fail); font-size: var(--tp-text-sm);">{error}</p>
  {/if}

  {#if !scanning}
    <Button variant="primary" size="lg" full on:click={startScan}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <path d="M3 7V5a2 2 0 0 1 2-2h2"/><path d="M17 3h2a2 2 0 0 1 2 2v2"/>
        <path d="M21 17v2a2 2 0 0 1-2 2h-2"/><path d="M7 21H5a2 2 0 0 1-2-2v-2"/>
        <rect width="7" height="7" x="7" y="7" rx="1"/>
      </svg>
      Start Camera
    </Button>
  {:else}
    <Button variant="secondary" full on:click={stopScan}>Cancel</Button>
  {/if}
</div>

<style>
  .scan-page {
    max-width: 480px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: var(--tp-space-5);
  }
</style>
