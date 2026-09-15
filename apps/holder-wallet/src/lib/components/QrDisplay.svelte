<script>
  /**
   * QrDisplay — renders a QR code for the proof payload.
   *
   * Props:
   *   value   {string}  — data to encode in QR
   *   label   {string}  — accessible description
   *   size    {number}  — pixel dimension (default: 240, minimum spec: 240)
   *
   * Uses the `qrcode` npm package (canvas-based, no CDN).
   *
   * TODO (next agent): call this component from the proof-qr route,
   * passing the encoded proof payload returned by verifier-api.
   */
  export let value = '';
  export let label = 'Proof QR code';
  export let size  = 260;

  import { onMount } from 'svelte';

  let canvas;
  let error = null;

  onMount(async () => {
    if (!value) return;
    try {
      const QRCode = (await import('qrcode')).default;
      await QRCode.toCanvas(canvas, value, {
        width: size,
        margin: 2,
        color: { dark: '#000000', light: '#FFFFFF' },
        errorCorrectionLevel: 'M',
      });
    } catch (e) {
      error = e.message;
    }
  });

  $: if (canvas && value) {
    // Re-render if value changes after mount
    import('qrcode').then(m => {
      m.default.toCanvas(canvas, value, {
        width: size,
        margin: 2,
        color: { dark: '#000000', light: '#FFFFFF' },
        errorCorrectionLevel: 'M',
      }).catch(e => { error = e.message; });
    });
  }
</script>

<div class="tp-qr-wrapper">
  <div class="tp-qr-container">
    {#if error}
      <p style="color: var(--tp-signal-fail); font-size: var(--tp-text-sm); padding: 1rem;">
        QR generation failed: {error}
      </p>
    {:else if !value}
      <div style="width:{size}px;height:{size}px;background:#fff;display:flex;align-items:center;justify-content:center;">
        <div class="tp-spinner"></div>
      </div>
    {:else}
      <canvas
        bind:this={canvas}
        width={size}
        height={size}
        role="img"
        aria-label={label}
      ></canvas>
    {/if}
  </div>
  {#if $$slots.label}
    <p class="tp-qr-label"><slot name="label" /></p>
  {/if}
</div>
