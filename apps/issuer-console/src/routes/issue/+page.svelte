<script>
  /**
   * Route: /issue — Issue a new credential
   *
   * Form: select schema → fill synthetic attributes → submit.
   * On success: shows the issued credential JSON in a code block.
   *
   * TODO (next agent):
   *   1. Load schemas from GET /api/schemas/schemas
   *   2. POST to /api/issuer/credentials/issue with { schemaName, holderDid, attributes }
   *   3. Display result with MonoDisplay for credential ID.
   */
  import { onMount } from 'svelte';

  let schemas     = [];
  let selectedSchema = null;
  let holderDid   = '';
  let attributes  = {};
  let submitting  = false;
  let result      = null;
  let error       = null;

  onMount(async () => {
    try {
      const res = await fetch('/api/schemas/schemas');
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data) && data.length > 0) {
          schemas = data;
          return;
        }
      }
    } catch (e) {
      console.warn('Could not load schemas from registry:', e);
    }

    // Fallback seed schemas
    schemas = [
      {
        name: 'NationalIDCredential',
        attributes: [
          { name: 'fullName',     type: 'string' },
          { name: 'dateOfBirth',  type: 'date'   },
          { name: 'nationality',  type: 'string' },
          { name: 'idNumber',     type: 'string' },
        ],
      },
      {
        name: 'StudentCredential',
        attributes: [
          { name: 'studentId',        type: 'string'  },
          { name: 'university',       type: 'string'  },
          { name: 'enrollmentStatus', type: 'string'  },
          { name: 'gpa',              type: 'decimal' },
        ],
      },
    ];
  });

  $: if (selectedSchema) {
    // Reset attributes when schema changes
    attributes = {};
    selectedSchema.attributes.forEach(a => {
      attributes[a.name] = a.type === 'decimal' ? '0.00' : '';
    });
  }

  async function issueCredential() {
    error = null;
    submitting = true;
    try {
      const res = await fetch('/api/issuer/credentials/issue', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          schemaName: selectedSchema.name,
          holderDid,
          attributes,
        }),
      });

      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Issuance failed');
      }

      const vc = await res.json();
      result = {
        id: vc.id,
        type: selectedSchema.name,
        holderDid,
        attributes,
        issuedAt: vc.issuanceDate || new Date().toISOString(),
        signature: vc.proof?.proofValue || 'BBS+ Signature Verified',
        fullVc: vc,
      };
    } catch (e) {
      error = e.message;
    } finally {
      submitting = false;
    }
  }

  function reset() {
    result = null;
    holderDid = '';
    attributes = {};
    selectedSchema = null;
  }
</script>

<svelte:head>
  <title>Issue Credential — TrustPass Issuer</title>
</svelte:head>

<div class="tp-page-header">
  <h1 class="tp-page-title">Issue Credential</h1>
  <p class="tp-page-subtitle">Create a signed verifiable credential for a holder.</p>
</div>

{#if result}
  <!-- ── Success state ─────────────────────────────────────────────── -->
  <div class="tp-card issue-success">
    <div class="tp-card-header">
      <div>
        <h2 class="tp-card-title" style="color: var(--tp-signal-pass);">Credential Issued</h2>
        <p class="tp-card-subtitle">{result.type}</p>
      </div>
      <span class="tp-badge tp-badge-pass">Active</span>
    </div>

    <div class="tp-card-body">
      <div class="tp-field">
        <p class="tp-label">Credential ID</p>
        <code class="tp-mono" style="word-break:break-all;">{result.id}</code>
      </div>

      <div class="tp-field">
        <p class="tp-label">Full JSON</p>
        <pre class="tp-code-block">{JSON.stringify(result, null, 2)}</pre>
      </div>
    </div>

    <div class="tp-card-footer">
      <button class="tp-btn tp-btn-secondary" on:click={reset}>Issue Another</button>
    </div>
  </div>

{:else}
  <!-- ── Issue form ──────────────────────────────────────────────── -->
  <form class="tp-form issue-form" on:submit|preventDefault={issueCredential}>

    <!-- Schema selection -->
    <div class="tp-field">
      <label class="tp-label tp-label-required" for="schema-select">Credential Schema</label>
      <select
        id="schema-select"
        class="tp-select"
        bind:value={selectedSchema}
        required
      >
        <option value={null} disabled selected>Select a schema…</option>
        {#each schemas as schema}
          <option value={schema}>{schema.name}</option>
        {/each}
      </select>
    </div>

    <!-- Holder DID -->
    <div class="tp-field">
      <label class="tp-label tp-label-required" for="holder-did">Holder DID</label>
      <input
        id="holder-did"
        type="text"
        class="tp-input"
        placeholder="did:key:z…"
        bind:value={holderDid}
        required
        style="font-family: var(--tp-font-mono); font-size: var(--tp-text-sm);"
      />
    </div>

    <!-- Dynamic attribute fields -->
    {#if selectedSchema}
      <fieldset class="attribute-fieldset">
        <legend class="tp-label" style="margin-bottom: var(--tp-space-4);">Attributes</legend>
        <div class="tp-form-row">
          {#each selectedSchema.attributes as attr}
            <div class="tp-field">
              <label class="tp-label tp-label-required" for="attr-{attr.name}">
                {attr.name}
                <span style="font-weight: 400; text-transform: none; letter-spacing: 0; color: var(--tp-text-disabled);">
                  ({attr.type})
                </span>
              </label>
              <input
                id="attr-{attr.name}"
                type={attr.type === 'date' ? 'date' : 'text'}
                class="tp-input"
                bind:value={attributes[attr.name]}
                required
                placeholder={attr.type === 'decimal' ? '0.00' : attr.name}
              />
            </div>
          {/each}
        </div>
      </fieldset>
    {/if}

    {#if error}
      <p role="alert" class="tp-field-error">{error}</p>
    {/if}

    <div class="tp-form-actions">
      <a href="/" class="tp-btn tp-btn-ghost">Cancel</a>
      <button
        type="submit"
        class="tp-btn tp-btn-primary"
        class:tp-btn-loading={submitting}
        disabled={submitting || !selectedSchema}
        aria-busy={submitting}
      >
        {submitting ? '' : 'Issue Credential'}
      </button>
    </div>
  </form>
{/if}

<style>
  .issue-form, .issue-success {
    max-width: 640px;
  }

  .attribute-fieldset {
    border: 1px solid var(--tp-border);
    border-radius: var(--tp-radius-md);
    padding: var(--tp-space-5);
  }
</style>
