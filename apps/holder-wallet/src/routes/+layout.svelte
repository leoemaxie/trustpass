<script>
  import Navbar from '$lib/components/Navbar.svelte';

  const navLinks = [
    { href: '/',       label: 'Credentials' },
    { href: '/scan',   label: 'Scan Request' },
  ];
</script>

<svelte:head>
  <title>TrustPass Wallet</title>
</svelte:head>

<Navbar appName="Wallet" {navLinks} />

<main class="tp-main">
  <div class="tp-content-wrap">
    <slot />
  </div>
</main>
