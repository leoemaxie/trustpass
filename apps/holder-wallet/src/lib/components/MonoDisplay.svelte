<script>
  /**
   * MonoDisplay — cryptographic artifact display with copy-to-clipboard.
   *
   * Props:
   *   value   {string}  — the value to display and copy
   *   label   {string}  — accessible label (e.g. "Credential ID")
   *   block   {boolean} — use code-block (pre) style instead of inline pill
   *   truncate {boolean} — truncate long values with ellipsis (default: true)
   */
  export let value    = '';
  export let label    = 'Value';
  export let block    = false;
  export let truncate = true;

  let copied = false;

  async function copyToClipboard() {
    try {
      await navigator.clipboard.writeText(value);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    } catch {
      // Clipboard API unavailable — silently fail
    }
  }
</script>

{#if block}
  <div>
    <div class="tp-copy-field">
      <span class="tp-copy-field__value" class:tp-truncate={truncate} aria-label={label}>{value}</span>
      <button
        class="tp-copy-field__btn tp-btn tp-btn-ghost tp-btn-sm"
        on:click={copyToClipboard}
        aria-label="Copy {label} to clipboard"
        title={copied ? 'Copied!' : 'Copy'}
      >
        {#if copied}
          <!-- Check icon -->
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        {:else}
          <!-- Copy icon -->
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <rect width="14" height="14" x="8" y="8" rx="2" ry="2"/>
            <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>
          </svg>
        {/if}
      </button>
    </div>
  </div>
{:else}
  <code class="tp-mono" aria-label={label}>{value}</code>
{/if}
