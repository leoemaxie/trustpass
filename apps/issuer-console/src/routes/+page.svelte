<script>
  /**
   * Route: / — Issuer Console Dashboard
   *
   * Shows summary stats: total issued, active, revoked, expired.
   */
  import { onMount } from 'svelte';

  let stats = {
    total:   0,
    active:  0,
    revoked: 0,
    expired: 0,
  };

  onMount(async () => {
    try {
      const res = await fetch('/api/issuer/credentials/stats');
      if (res.ok) {
        stats = await res.json();
      }
    } catch (e) {
      console.warn('Could not fetch stats from issuer-api:', e);
    }
  });
</script>

<svelte:head>
  <title>Dashboard — TrustPass Issuer</title>
</svelte:head>

<div class="tp-page-header">
  <h1 class="tp-page-title">Dashboard</h1>
  <p class="tp-page-subtitle">Overview of issued credentials.</p>
</div>

<div class="tp-stat-grid">
  <div class="tp-stat-card">
    <p class="tp-stat-card__label">Total Issued</p>
    <p class="tp-stat-card__value tp-stat-card__value--cyan">{stats.total}</p>
  </div>
  <div class="tp-stat-card">
    <p class="tp-stat-card__label">Active</p>
    <p class="tp-stat-card__value tp-stat-card__value--pass">{stats.active}</p>
  </div>
  <div class="tp-stat-card">
    <p class="tp-stat-card__label">Revoked</p>
    <p class="tp-stat-card__value tp-stat-card__value--fail">{stats.revoked}</p>
  </div>
  <div class="tp-stat-card">
    <p class="tp-stat-card__label">Expired</p>
    <p class="tp-stat-card__value tp-stat-card__value--pending">{stats.expired}</p>
  </div>
</div>

<hr class="tp-divider" />

<div class="dashboard-links">
  <a href="/issue" class="tp-card tp-card-interactive">
    <div class="tp-card-header">
      <h2 class="tp-card-title">Issue Credential</h2>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--tp-text-muted);" aria-hidden="true">
        <path d="m9 18 6-6-6-6"/>
      </svg>
    </div>
    <p class="tp-card-subtitle">Create a new signed verifiable credential for a holder.</p>
  </a>

  <a href="/credentials" class="tp-card tp-card-interactive">
    <div class="tp-card-header">
      <h2 class="tp-card-title">View Credentials</h2>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--tp-text-muted);" aria-hidden="true">
        <path d="m9 18 6-6-6-6"/>
      </svg>
    </div>
    <p class="tp-card-subtitle">Browse, search, and revoke issued credentials.</p>
  </a>

  <a href="/schemas" class="tp-card tp-card-interactive">
    <div class="tp-card-header">
      <h2 class="tp-card-title">Schemas</h2>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--tp-text-muted);" aria-hidden="true">
        <path d="m9 18 6-6-6-6"/>
      </svg>
    </div>
    <p class="tp-card-subtitle">View registered credential schemas and their attributes.</p>
  </a>
</div>

<style>
  .dashboard-links {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--tp-space-4);
    margin-top: var(--tp-space-6);
  }

  @media (min-width: 640px) {
    .dashboard-links {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  a.tp-card { text-decoration: none; display: block; }
</style>
