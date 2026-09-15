<script>
  import { page } from '$app/stores';

  const navLinks = [
    { href: '/',             label: 'Dashboard'    },
    { href: '/schemas',      label: 'Schemas'      },
    { href: '/issue',        label: 'Issue'        },
    { href: '/credentials',  label: 'Credentials'  },
  ];
</script>

<svelte:head>
  <title>TrustPass Issuer Console</title>
</svelte:head>

<nav class="tp-navbar" aria-label="Main navigation">
  <a href="/" class="tp-navbar__brand" aria-label="TrustPass Issuer Console">
    <img src="/shared/assets/app_icon.png" alt="TrustPass" class="tp-navbar__logo" width="32" height="32" />
    <span class="tp-navbar__app-name">Issuer Console</span>
  </a>
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
</nav>

<main class="tp-main">
  <div class="tp-content-wrap">
    <slot />
  </div>
</main>
