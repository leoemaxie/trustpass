<script>
  /**
   * Route: /credentials — Credential management table
   *
   * Lists all issued credentials. Supports search and revocation.
   *
   * TODO (next agent):
   *   1. GET /api/issuer/credentials → populate credentials[]
   *   2. POST /api/issuer/credentials/{id}/revoke → call revokeCredential()
   */
  import { onMount } from 'svelte';

  let credentials = [];
  let loading     = true;
  let search      = '';
  let revoking    = null;   // id currently being revoked
  let confirmId   = null;   // id awaiting inline revoke confirmation

  onMount(async () => {
    try {
      const res = await fetch('/api/issuer/credentials');
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data) && data.length > 0) {
          credentials = data.map(c => ({
            id: c.credentialUrn || c.id,
            rawId: c.id,
            type: c.type || c.schemaName || 'Credential',
            holderDid: c.holderDid || 'did:key:...',
            issuedAt: c.issuedAt,
            expiresAt: c.expiresAt,
            revoked: !!c.revoked,
          }));
          loading = false;
          return;
        }
      }
    } catch (e) {
      console.warn('Could not fetch credentials from issuer-api:', e);
    }

    // Fallback sample data
    credentials = [
      {
        id: 'urn:uuid:mock-001',
        rawId: 'mock-001',
        type: 'NationalIDCredential',
        holderDid: 'did:key:zUC724vsrMwHvKyqDdHtrh7z2GNe5xbsfgivth466P4vm2iaJLW9kK48DbgKa32yL944yK9k',
        issuedAt: '2026-09-13T10:00:00Z',
        expiresAt: '2027-09-13T10:00:00Z',
        revoked: false,
      },
      {
        id: 'urn:uuid:mock-002',
        rawId: 'mock-002',
        type: 'StudentCredential',
        holderDid: 'did:key:zUC724vsrMwHvKyqDdHtrh7z2GNe5xbsfgivth466P4vm2iaJLW9kK48DbgKa32yL944yK9k',
        issuedAt: '2026-09-13T12:30:00Z',
        expiresAt: null,
        revoked: false,
      },
    ];
    loading = false;
  });

  $: filtered = credentials.filter(c =>
    c.id.toLowerCase().includes(search.toLowerCase()) ||
    c.type.toLowerCase().includes(search.toLowerCase()) ||
    c.holderDid.toLowerCase().includes(search.toLowerCase())
  );

  async function confirmRevoke(id) {
    revoking = id;
    confirmId = null;
    try {
      const res = await fetch(`/api/issuer/credentials/${encodeURIComponent(id)}/revoke`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: 'Revoked via Issuer Console' }),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      credentials = credentials.map(c => c.id === id || c.rawId === id ? { ...c, revoked: true } : c);
    } catch (e) {
      alert('Revocation failed: ' + e.message);
    } finally {
      revoking = null;
    }
  }

  function formatDate(iso) {
    if (!iso) return '—';
    return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
  }

  function statusVariant(c) {
    if (c.revoked) return 'fail';
    if (c.expiresAt && new Date(c.expiresAt) < new Date()) return 'fail';
    return 'pass';
  }
  function statusLabel(c) {
    if (c.revoked) return 'Revoked';
    if (c.expiresAt && new Date(c.expiresAt) < new Date()) return 'Expired';
    return 'Active';
  }
</script>

<svelte:head>
  <title>Credentials — TrustPass Issuer</title>
</svelte:head>

<div class="tp-page-header">
  <h1 class="tp-page-title">Credentials</h1>
  <p class="tp-page-subtitle">All issued credentials. Revoke here if compromised.</p>
  <div class="tp-page-header__actions">
    <input
      type="search"
      class="tp-input"
      placeholder="Search by ID, type, or holder DID…"
      bind:value={search}
      style="max-width: 320px;"
      aria-label="Search credentials"
    />
    <a href="/issue" class="tp-btn tp-btn-primary">+ Issue New</a>
  </div>
</div>

{#if loading}
  <div class="tp-loading-screen">
    <div class="tp-spinner tp-spinner-lg" aria-hidden="true"></div>
    <p>Loading credentials…</p>
  </div>
{:else if filtered.length === 0}
  <div class="tp-empty">
    <p class="tp-empty__title">No credentials found</p>
    <p class="tp-empty__description">{search ? 'Try a different search term.' : 'Issue a credential to get started.'}</p>
  </div>
{:else}
  <div class="tp-table-wrap">
    <table class="tp-table" aria-label="Issued credentials">
      <thead>
        <tr>
          <th>Type</th>
          <th>Credential ID</th>
          <th>Holder DID</th>
          <th>Issued</th>
          <th>Expires</th>
          <th>Status</th>
          <th><span class="tp-sr-only">Actions</span></th>
        </tr>
      </thead>
      <tbody>
        {#each filtered as cred (cred.id)}
          <tr>
            <td style="font-weight: 500;">{cred.type.replace('Credential', '')}</td>
            <td>
              <code class="tp-mono" title={cred.id}>
                {cred.id.slice(0, 28)}…
              </code>
            </td>
            <td>
              <code class="tp-mono" title={cred.holderDid}>
                {cred.holderDid.slice(0, 20)}…
              </code>
            </td>
            <td class="tp-text-muted">{formatDate(cred.issuedAt)}</td>
            <td class="tp-text-muted">{formatDate(cred.expiresAt)}</td>
            <td>
              <span class="tp-badge tp-badge-{statusVariant(cred)}">{statusLabel(cred)}</span>
            </td>
            <td>
              <div class="tp-table-actions">
                {#if !cred.revoked}
                  {#if confirmId === cred.id}
                    <!-- Inline confirmation — no modal -->
                    <span style="font-size: var(--tp-text-xs); color: var(--tp-signal-fail);">Revoke this credential?</span>
                    <button
                      class="tp-btn tp-btn-danger tp-btn-sm"
                      class:tp-btn-loading={revoking === cred.id}
                      on:click={() => confirmRevoke(cred.id)}
                      aria-busy={revoking === cred.id}
                    >
                      {revoking === cred.id ? '' : 'Confirm'}
                    </button>
                    <button class="tp-btn tp-btn-ghost tp-btn-sm" on:click={() => confirmId = null}>
                      Cancel
                    </button>
                  {:else}
                    <button
                      class="tp-btn tp-btn-ghost tp-btn-sm"
                      style="color: var(--tp-signal-fail);"
                      on:click={() => confirmId = cred.id}
                    >
                      Revoke
                    </button>
                  {/if}
                {/if}
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
