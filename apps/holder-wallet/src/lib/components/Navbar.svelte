<script>
  /**
   * Navbar component — shared across holder-wallet routes.
   *
   * Props:
   *   appName  {string}  — app label shown beside the logo
   *   navLinks {Array<{href, label, icon?}>} — navigation items
   */
  export let appName  = 'Holder Wallet';
  export let navLinks = [];

  import { page } from '$app/stores';
</script>

<nav class="tp-navbar" aria-label="Main navigation">
  <a href="/" class="tp-navbar__brand" aria-label="TrustPass home">
    <img
      src="/shared/assets/branding/app_icon.png"
      alt="TrustPass logo"
      class="tp-navbar__logo"
      width="32"
      height="32"
    />
    <span class="tp-navbar__app-name">{appName}</span>
  </a>

  {#if navLinks.length}
    <ul class="tp-navbar__nav" role="list">
      {#each navLinks as link}
        <li>
          <a
            href={link.href}
            class="tp-nav-link"
            class:active={$page.url.pathname === link.href}
            aria-current={$page.url.pathname === link.href ? 'page' : undefined}
          >
            {link.label}
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</nav>
