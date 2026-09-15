<script>
  /**
   * Badge / status chip component.
   *
   * Props:
   *   variant  {'pass'|'fail'|'pending'|'crypto'|'neutral'}
   */
  export let variant = 'neutral';
</script>

<span class="tp-badge tp-badge-{variant}">
  <slot />
</span>
