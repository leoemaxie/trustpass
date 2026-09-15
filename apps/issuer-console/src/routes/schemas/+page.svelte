<script>
  /**
   * Route: /schemas — Schema registry browser
   *
   * Lists all registered credential schemas from schema-registry.
   *
   * TODO (next agent): GET /api/schemas/schemas to load real schemas.
   */
  import { onMount } from 'svelte';

  let schemas = [];
  let loading = true;

  onMount(async () => {
    // TODO: real fetch
    // const res = await fetch('/api/schemas/schemas');
    // schemas = await res.json();

    // MOCK
    await new Promise(r => setTimeout(r, 400));
    schemas = [
      {
        id: 'schema:national-id:v1',
        name: 'NationalIDCredential',
        version: 1,
        issuerDid: 'did:key:zMOCK_ISSUER_DID',
        createdAt: '2026-09-13T10:00:00Z',
        attributes: [
          { name: 'fullName',    type: 'string' },
          { name: 'dateOfBirth', type: 'date'   },
          { name: 'nationality', type: 'string' },
          { name: 'idNumber',    type: 'string' },
        ],
      },
      {
        id: 'schema:student:v1',
        name: 'StudentCredential',
        version: 1,
        issuerDid: 'did:key:zMOCK_ISSUER_DID',
        createdAt: '2026-09-13T10:00:00Z',
        attributes: [
          { name: 'studentId',        type: 'string'  },
          { name: 'university',       type: 'string'  },
          { name: 'enrollmentStatus', type: 'string'  },
          { name: 'gpa',              type: 'decimal' },
        ],
      },
    ];
    loading = false;
  });

  let expanded = {};
  function toggle(id) {
    expanded[id] = !expanded[id];
  }
</script>

<svelte:head>
  <title>Schemas — TrustPass Issuer</title>
</svelte:head>

<div class="tp-page-header">
  <h1 class="tp-page-title">Credential Schemas</h1>
  <p class="tp-page-subtitle">Registered schemas. Attributes drive the predicate engine — no code changes needed to add new claim types.</p>
</div>

{#if loading}
  <div class="tp-loading-screen">
    <div class="tp-spinner tp-spinner-lg" aria-hidden="true"></div>
  </div>
{:else}
  <div class="schemas-list">
    {#each schemas as schema (schema.id)}
      <div class="tp-card">
        <div class="tp-card-header">
          <div>
            <h2 class="tp-card-title">{schema.name}</h2>
            <p class="tp-card-subtitle">
              Version {schema.version} ·
              {schema.attributes.length} attributes ·
              <code class="tp-mono" style="font-size: 0.75rem;">{schema.id}</code>
            </p>
          </div>
          <button
            class="tp-btn tp-btn-ghost tp-btn-sm"
            on:click={() => toggle(schema.id)}
            aria-expanded={!!expanded[schema.id]}
          >
            {expanded[schema.id] ? 'Hide' : 'Show'} Attributes
          </button>
        </div>

        {#if expanded[schema.id]}
          <div class="schema-attrs">
            <table class="tp-table">
              <thead>
                <tr>
                  <th>Attribute Name</th>
                  <th>Type</th>
                  <th>Predicate Operators</th>
                </tr>
              </thead>
              <tbody>
                {#each schema.attributes as attr}
                  <tr>
                    <td><code class="tp-mono">{attr.name}</code></td>
                    <td><span class="tp-badge tp-badge-neutral">{attr.type}</span></td>
                    <td class="tp-text-muted" style="font-size: var(--tp-text-xs);">
                      {#if attr.type === 'date'}    BEFORE_DATE, EQ
                      {:else if attr.type === 'decimal'} GTE, EQ
                      {:else if attr.type === 'string'}  EQ, IN_SET
                      {:else}                       EQ
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}

        <div class="tp-card-footer" style="justify-content: flex-start;">
          <span style="font-size: var(--tp-text-xs); color: var(--tp-text-muted);">
            Issuer: <code class="tp-mono" style="font-size: 0.75rem;">{schema.issuerDid.slice(0, 30)}…</code>
          </span>
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .schemas-list {
    display: flex;
    flex-direction: column;
    gap: var(--tp-space-4);
  }
  .schema-attrs {
    margin-top: var(--tp-space-4);
    border: 1px solid var(--tp-border);
    border-radius: var(--tp-radius-md);
    overflow: hidden;
  }
</style>
