# TrustPass — Frontend Applications

Three web applications sharing a common design system.

## Structure

```
apps/
├── shared/                 ← Design system — import this first, always
│   ├── tokens.css          — All CSS custom properties (colors, spacing, radii, etc.)
│   ├── base.css            — CSS reset, typography, app shell, navbar, utilities
│   └── components.css      — Reusable .tp-* component classes
│
├── verifier-pwa/           ← Framework-free PWA (plain HTML/CSS/JS)
│   ├── index.html
│   ├── styles.css
│   ├── app.js
│   └── manifest.json
│
├── holder-wallet/          ← SvelteKit app (port 5173)
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +layout.svelte         — Navbar + content shell
│   │   │   ├── +page.svelte           — Credential list
│   │   │   ├── scan/+page.svelte      — Camera QR scanner
│   │   │   ├── claim-review/+page.svelte  — Trust review screen ⭐
│   │   │   └── proof-qr/+page.svelte  — Generated proof QR display
│   │   └── lib/
│   │       ├── components/
│   │       │   ├── Navbar.svelte
│   │       │   ├── Button.svelte
│   │       │   ├── Badge.svelte
│   │       │   ├── Card.svelte
│   │       │   ├── MonoDisplay.svelte
│   │       │   ├── QrDisplay.svelte
│   │       │   ├── CredentialCard.svelte
│   │       │   └── SessionTimer.svelte
│   │       └── stores/
│   │           └── wallet.js
│   ├── package.json
│   ├── svelte.config.js
│   └── vite.config.js
│
└── issuer-console/         ← SvelteKit app (port 5174)
    ├── src/
    │   └── routes/
    │       ├── +layout.svelte         — Navbar with 4 nav items
    │       ├── +page.svelte           — Dashboard (stat cards)
    │       ├── schemas/+page.svelte   — Schema browser
    │       ├── issue/+page.svelte     — Issue credential form
    │       └── credentials/+page.svelte — Credential table + revoke
    ├── package.json
    ├── svelte.config.js
    └── vite.config.js
```

## CSS Import Order (mandatory, never change)

```html
<link rel="stylesheet" href="/shared/tokens.css" />
<link rel="stylesheet" href="/shared/base.css" />
<link rel="stylesheet" href="/shared/components.css" />
<!-- App-specific styles last -->
```

## Running locally

```bash
# holder-wallet
cd apps/holder-wallet && npm install && npm run dev   # http://localhost:5173

# issuer-console
cd apps/issuer-console && npm install && npm run dev  # http://localhost:5174

# verifier-pwa (any static server)
cd apps/verifier-pwa && npx serve .                   # http://localhost:3000
```

## For the next agent

Every route has clear `// TODO (next agent):` comments marking exactly what API calls need wiring. The design, layouts, components, and state management are complete. The next step is:

1. Wire `holder-wallet` `/scan` and `/claim-review` to real `verifier-api` endpoints.
2. Wire `holder-wallet` `/` to load real credentials from `issuer-api`.
3. Wire `issuer-console` `/issue` and `/credentials` to `issuer-api`.
4. Wire `issuer-console` `/schemas` to `schema-registry`.
5. Wire `verifier-pwa` `app.js` `verifyProof()` to `verifier-api POST /verification/verify`.

**Do not redesign.** Use the existing components and tokens. If a component is missing, add it to `apps/shared/components.css` using the existing token system.
