<script>
  /**
   * Card component.
   *
   * Props:
   *   interactive {boolean} — adds hover/focus effects and cursor: pointer
   *   sm          {boolean} — smaller padding variant
   *   as          {'div'|'article'|'section'} — HTML element
   */
  export let interactive = false;
  export let sm          = false;
  export let as          = 'div';

  $: classes = [
    'tp-card',
    sm          ? 'tp-card-sm'          : '',
    interactive ? 'tp-card-interactive' : '',
  ].filter(Boolean).join(' ');
</script>

<svelte:element this={as} class={classes} on:click on:keydown>
  <slot />
</svelte:element>
