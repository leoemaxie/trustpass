<script>
  /**
   * Button component — wraps shared .tp-btn classes.
   *
   * Props:
   *   variant   {'primary'|'secondary'|'ghost'|'danger'}
   *   size      {'sm'|'md'|'lg'}
   *   full      {boolean}  — full-width
   *   loading   {boolean}  — show spinner, disable interaction
   *   disabled  {boolean}
   *   type      {'button'|'submit'|'reset'}
   *   href      {string}   — if set, renders an <a> instead of <button>
   */
  export let variant  = 'primary';
  export let size     = 'md';
  export let full     = false;
  export let loading  = false;
  export let disabled = false;
  /** @type {'button'|'submit'|'reset'} */
  export let type     = 'button';
  export let href     = null;

  $: classes = [
    'tp-btn',
    `tp-btn-${variant}`,
    size === 'sm' ? 'tp-btn-sm' : size === 'lg' ? 'tp-btn-lg' : '',
    full    ? 'tp-btn-full'    : '',
    loading ? 'tp-btn-loading' : '',
  ].filter(Boolean).join(' ');
</script>

{#if href}
  <a {href} class={classes} aria-disabled={disabled || loading} role="button">
    <slot />
  </a>
{:else}
  <button
    {type}
    class={classes}
    disabled={disabled || loading}
    aria-busy={loading}
    on:click
    on:submit
  >
    <slot />
  </button>
{/if}
