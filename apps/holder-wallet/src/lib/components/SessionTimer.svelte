<script>
  /**
   * SessionTimer — countdown chip for the active verification session.
   *
   * Props:
   *   expiresAt {string} — ISO timestamp when the session token expires
   *
   * Displays amber timer chip. Goes red and pulses when < 30s remaining.
   *
   * Emits:
   *   on:expired — when countdown reaches zero
   */
  export let expiresAt;

  import { onDestroy, createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let remaining = 0;
  let interval;

  function tick() {
    remaining = Math.max(0, Math.floor((new Date(expiresAt).getTime() - Date.now()) / 1000));
    if (remaining === 0) {
      clearInterval(interval);
      dispatch('expired');
    }
  }

  $: if (expiresAt) {
    clearInterval(interval);
    tick();
    interval = setInterval(tick, 1000);
  }

  onDestroy(() => clearInterval(interval));

  function formatRemaining(secs) {
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return `${m}:${String(s).padStart(2, '0')}`;
  }

  $: expiring = remaining < 30 && remaining > 0;
</script>

<div
  class="tp-timer-chip"
  class:tp-timer-chip--expiring={expiring}
  role="timer"
  aria-label="Session expires in {remaining} seconds"
  aria-live="polite"
>
  <!-- Clock icon -->
  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
    <circle cx="12" cy="12" r="10"/>
    <polyline points="12 6 12 12 16 14"/>
  </svg>
  {#if remaining > 0}
    {formatRemaining(remaining)}
  {:else}
    Expired
  {/if}
</div>
