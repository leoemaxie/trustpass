# TrustPass — Design System & Style Guide

**Status:** Locked. This document is the single source of truth for all visual and interaction design decisions across the three TrustPass applications. Agents must not make design choices that contradict this document. If a gap is discovered, leave a `// DESIGN-GAP:` comment and note it in `docs/PROGRESS.md` rather than inventing silently.

**Motto:** *"Prove, don't show!"*

---

## 0. How to use this document

Read this in full before touching any file in `apps/`. The three applications share a common brand layer but have deliberately different interaction models — understand both the shared rules and the per-app rules before writing a single line of CSS. Violations of per-app constraints (e.g. adding a framework to `verifier-pwa`) are as bad as violations of the shared brand.

---

## 1. Brand Identity

### 1.1 Core Visual Language

TrustPass communicates **cryptographic trust without complexity**. The visual language is built on four ideas:

| Concept | Expression |
|---|---|
| **Proof** | Geometric precision — sharp edges, structured grids |
| **Privacy** | Dark surfaces — information is *hidden by default*, revealed only on demand |
| **Verification** | Luminous signal colors — the answer is unmissable when it arrives |
| **Technology** | Cyan/teal glow — ZK cryptography made tangible and approachable |

The overall mood is **professional, minimal, and slightly cinematic** — think zero-knowledge, not zero-design. The aesthetic draws from security dashboards and terminal UIs, but warmed by the neon glow of the shield emblem.

### 1.2 Logo & Icon

| Asset | Path | Usage |
|---|---|---|
| Full horizontal logo | [`assets/branding/logo.png`](../assets/branding/logo.png) | Header/navbar on `holder-wallet` and `issuer-console`. Never stretched, never recolored. |
| App icon / emblem | [`assets/branding/app_icon.png`](../assets/branding/app_icon.png) | PWA favicon, mobile homescreen shortcut, loading splash. |

**Never** place either asset on a white or light background. They are designed for dark surfaces only (see Section 2 color tokens). If a light-mode context is ever required in the future, a new export must be created — do not attempt to re-use the existing PNGs.

The emblem is a **neon shield with a checkmark and keyhole** — it must not be recreated in code (SVG inline, emoji substitutes, icon fonts). Use the PNG assets only.

### 1.3 Tagline & Copywriting Tone

- Primary tagline: **"Prove, don't show!"** — always in italic when used as a caption or subtitle.
- UX copy must be **plain, active, and direct**. No hedging, no jargon.
- Error messages name the specific rejection reason (matching the typed error enum — `SignatureInvalid`, `CredentialExpired`, etc.) and follow it with a single plain-English sentence explaining what that means to the user.
- Verification results are binary and emphatic — never softened with language like "appears to be valid" or "may not be verified."

---

## 2. Color System

All color values are defined as CSS custom properties. Every application imports them from the shared token sheet. **No hardcoded hex values** anywhere in the app CSS — always use the `--tp-*` variables.

### 2.1 Global Tokens (`apps/shared/tokens.css`)

```css
:root {
  /* ── Surfaces & Backgrounds ─────────────────────────────── */
  --tp-bg-dark:      #090D16;  /* Deep Obsidian Navy — page background          */
  --tp-bg-surface:   #0F172A;  /* Slate Surface — navbars, sidebars, cards      */
  --tp-bg-card:      #1E293B;  /* Interactive Card — form fields, list items    */
  --tp-bg-overlay:   #0F172Acc; /* Surface at 80% opacity — modal backdrops      */
  --tp-border:       #334155;  /* Subtle border — dividers, input outlines      */
  --tp-border-focus: #38BDF8;  /* Focus ring — keyboard navigation              */

  /* ── Typography ─────────────────────────────────────────── */
  --tp-text-primary: #F8FAFC;  /* High-contrast white — headings, body copy    */
  --tp-text-muted:   #94A3B8;  /* Slate muted — labels, captions, timestamps   */
  --tp-text-accent:  #38BDF8;  /* Sky cyan — motto, links, highlighted values  */

  /* ── Verification Signals (must never be ambiguous) ─────── */
  --tp-signal-pass:        #10B981;  /* Emerald — VERIFIED, success states      */
  --tp-signal-pass-glow:   #10B98133; /* Emerald at 20% — glow / halo effect    */
  --tp-signal-fail:        #EF4444;  /* Crimson — NOT VERIFIED, error states    */
  --tp-signal-fail-glow:   #EF444433; /* Crimson at 20% — glow / halo effect    */
  --tp-signal-pending:     #F59E0B;  /* Amber — loading, processing, QR scan    */
  --tp-signal-pending-glow:#F59E0B33;

  /* ── Cryptographic Accent ───────────────────────────────── */
  --tp-accent-cyan:        #06B6D4;  /* ZK proof badge, session token chips     */
  --tp-accent-cyan-dim:    #06B6D41A; /* Cyan at 10% — subtle tinted backgrounds */
  --tp-accent-emerald:     #34D399;  /* Lighter emerald for icon strokes        */

  /* ── Elevation (box-shadow layers) ─────────────────────── */
  --tp-shadow-card:   0 1px 3px #00000066, 0 1px 2px #0000004d;
  --tp-shadow-raised: 0 4px 6px #00000055, 0 2px 4px #00000044;
  --tp-shadow-modal:  0 20px 60px #000000aa;

  /* ── Radii ──────────────────────────────────────────────── */
  --tp-radius-sm:   4px;
  --tp-radius-md:   8px;
  --tp-radius-lg:   12px;
  --tp-radius-xl:   16px;
  --tp-radius-full: 9999px; /* pill / badge */

  /* ── Spacing Scale (8-pt grid) ──────────────────────────── */
  --tp-space-1:  4px;
  --tp-space-2:  8px;
  --tp-space-3: 12px;
  --tp-space-4: 16px;
  --tp-space-5: 20px;
  --tp-space-6: 24px;
  --tp-space-8: 32px;
  --tp-space-10: 40px;
  --tp-space-12: 48px;
  --tp-space-16: 64px;

  /* ── Transitions ────────────────────────────────────────── */
  --tp-transition-fast:   150ms ease;
  --tp-transition-normal: 250ms ease;
  --tp-transition-slow:   400ms ease;
}
```

### 2.2 Signal Color Rules

These rules are **non-negotiable** — the verification result must be perceivable with no ambiguity, including in low-light conditions and on low-quality mobile screens:

1. `--tp-signal-pass` (emerald `#10B981`) is used **only** to signal a successful, verified state. Never use it decoratively.
2. `--tp-signal-fail` (crimson `#EF4444`) is used **only** to signal a failed or rejected state. Never use it for unrelated warnings or destructive UI actions — use a separate destructive token if needed.
3. Result screens on `verifier-pwa` must use **full-bleed background** in the signal color (with `--tp-bg-dark` as text) or a very large high-contrast glow, not a small badge. The result must be identifiable from 2 metres away on a phone screen.
4. `--tp-signal-pending` (amber) is the only colour used for any in-progress / scanning state. Do not use a spinner on a neutral background.

### 2.3 Dark Mode Only

All three applications are **dark-only**. There is no light mode and no system-preference media query override. The brand identity depends on dark surfaces — the neon glow of the emblem and the signal colours have insufficient contrast on white backgrounds.

---

## 3. Typography

### 3.1 Font Stack

| Role | Family | Source | Weights Used |
|---|---|---|---|
| **Display / Headings** | Inter | Google Fonts (self-hosted) | 600, 700 |
| **Body / UI Text** | Inter | same | 400, 500 |
| **Monospace / Credentials / Keys** | JetBrains Mono | Google Fonts (self-hosted) | 400, 500 |
| **System Fallback** | `ui-sans-serif, system-ui, -apple-system, sans-serif` | — | — |

**Why Inter?** It is optimised for screen legibility at small sizes, has excellent numeral rendering (important for dates and credential values), and has a neutral but confident character suited to a security product.

**Why JetBrains Mono?** Credential IDs, DID strings, session tokens, and proof hashes are monospace data. Using a monospace font for these values makes them visually distinct from prose and correctly signals "this is a cryptographic artifact, not marketing copy."

### 3.2 Self-Hosting Fonts

Fonts **must be self-hosted** — no Google Fonts CDN calls in production. For the hackathon demo this matters because:
1. `verifier-pwa` must load on a weak connection (spec Section 8.1).
2. CDN calls break in offline/demo mode.

Download and place font files in `apps/shared/fonts/`. Use `font-display: swap` to avoid invisible text during load.

```css
@font-face {
  font-family: 'Inter';
  font-style: normal;
  font-weight: 400 700;
  font-display: swap;
  src: url('/fonts/inter-variable.woff2') format('woff2');
}

@font-face {
  font-family: 'JetBrains Mono';
  font-style: normal;
  font-weight: 400 500;
  font-display: swap;
  src: url('/fonts/jetbrains-mono-variable.woff2') format('woff2');
}
```

### 3.3 Type Scale

All sizes use `rem` units. Base: `1rem = 16px`.

| Token | rem | px | Usage |
|---|---|---|---|
| `--tp-text-xs`   | 0.75rem | 12px | Timestamps, helper text, fine print |
| `--tp-text-sm`   | 0.875rem | 14px | Labels, secondary body, table cells |
| `--tp-text-base` | 1rem | 16px | Primary body, form inputs |
| `--tp-text-lg`   | 1.125rem | 18px | Sub-headings, card titles |
| `--tp-text-xl`   | 1.25rem | 20px | Section headings |
| `--tp-text-2xl`  | 1.5rem | 24px | Page headings |
| `--tp-text-3xl`  | 1.875rem | 30px | Hero / dashboard titles |
| `--tp-text-4xl`  | 2.25rem | 36px | Verification result labels (`verifier-pwa`) |
| `--tp-text-display` | 4rem | 64px | Full-screen result text (`VERIFIED` / `NOT VERIFIED`) |

Add these to the `:root` block in `tokens.css`.

### 3.4 Line Height & Letter Spacing

```css
:root {
  --tp-leading-tight:  1.25;
  --tp-leading-snug:   1.375;
  --tp-leading-normal: 1.5;
  --tp-leading-relaxed: 1.625;

  --tp-tracking-tight:  -0.025em;
  --tp-tracking-normal: 0em;
  --tp-tracking-wide:   0.05em;
  --tp-tracking-wider:  0.1em;  /* Use for ALL_CAPS labels and badge text */
}
```

All-caps labels (e.g., "VERIFIED", "NOT VERIFIED", "SESSION TOKEN") always use `--tp-tracking-wider` and `font-weight: 600+`.

---

## 4. Component Library (Shared Primitives)

These components exist in `apps/shared/` as CSS classes (plain CSS, no framework dependency). Svelte apps import them; `verifier-pwa` uses them directly.

### 4.1 Buttons

Three variants:

```
.tp-btn-primary   — filled, --tp-accent-cyan background, dark text
.tp-btn-secondary — outlined, --tp-border border, --tp-text-primary text
.tp-btn-danger    — filled, --tp-signal-fail background, white text
```

Rules:
- Minimum touch target: `44px` height, `44px` width on mobile.
- Always include a visible `:focus-visible` ring using `--tp-border-focus`.
- Never disable buttons without also providing an adjacent explanation of why.
- Loading state: replace button text with an animated ellipsis or spinner; never just freeze the UI.

### 4.2 Badges / Status Chips

Used for session token status, credential validity, predicate result:

```
.tp-badge-pass    — emerald background at 15%, emerald text, emerald border
.tp-badge-fail    — crimson background at 15%, crimson text, crimson border
.tp-badge-pending — amber background at 15%, amber text, amber border
.tp-badge-crypto  — cyan background at 10%, cyan text — for ZK proof type labels
```

Letter-spacing: `--tp-tracking-wider`. Text always uppercase.

### 4.3 Cards

```css
.tp-card {
  background: var(--tp-bg-card);
  border: 1px solid var(--tp-border);
  border-radius: var(--tp-radius-lg);
  box-shadow: var(--tp-shadow-card);
  padding: var(--tp-space-6);
}
```

Interactive cards (clickable) add `:hover` with `border-color: var(--tp-border-focus)` and `box-shadow: var(--tp-shadow-raised)`.

### 4.4 Form Inputs

```css
.tp-input {
  background: var(--tp-bg-dark);
  border: 1px solid var(--tp-border);
  border-radius: var(--tp-radius-md);
  color: var(--tp-text-primary);
  font-family: inherit;
  font-size: var(--tp-text-base);
  padding: var(--tp-space-3) var(--tp-space-4);
}
.tp-input:focus {
  outline: 2px solid var(--tp-border-focus);
  outline-offset: 2px;
}
.tp-input::placeholder {
  color: var(--tp-text-muted);
}
```

Labels always sit **above** their input, never inside (placeholder-only labels violate accessibility and disappear on focus). Validation errors appear below the input in `--tp-signal-fail` colour with an accompanying icon.

### 4.5 Monospace Data Display

Credential IDs, DID strings, session tokens, proof hex — any cryptographic artifact displayed to the user:

```css
.tp-mono {
  font-family: 'JetBrains Mono', monospace;
  font-size: var(--tp-text-sm);
  background: var(--tp-accent-cyan-dim);
  border: 1px solid var(--tp-border);
  border-radius: var(--tp-radius-sm);
  padding: var(--tp-space-1) var(--tp-space-2);
  color: var(--tp-accent-cyan);
  word-break: break-all;
}
```

Long strings (DIDs, proof bytes) are truncated with a "copy" button — never displayed in full by default.

### 4.6 QR Code Display

QR codes are always rendered on a **white background tile** (QR readers need high-contrast light modules), embedded on the dark card surface:

```css
.tp-qr-container {
  background: #FFFFFF;
  border-radius: var(--tp-radius-md);
  padding: var(--tp-space-4);
  display: inline-block;
}
```

Minimum QR tile size: `240×240px` on mobile. Never scale below this — camera-based scanners fail on small/blurry codes.

---

## 5. Per-Application Design Specifications

### 5.1 `verifier-pwa` — The Shop-Owner Interface

> **Design constraint:** This is the highest-scrutiny surface. One job: show a clear binary result. Every design decision must serve that one job, not compete with it.

**Stack:** Plain HTML + CSS + minimal vanilla JS. No CSS frameworks (no Tailwind, no Bootstrap), no build step for CSS. A single `styles.css` importing `tokens.css`.

#### Screens

| Screen | Description |
|---|---|
| **Scan** | Primary screen. One large camera viewfinder, one action label. Nothing else. |
| **Processing** | Full-screen amber overlay while proof is being verified. A simple spinner + "Verifying…" text. |
| **Result: VERIFIED** | Full-screen emerald background. Oversized ✓ icon. "VERIFIED" in `--tp-text-display`. No other information on screen. |
| **Result: NOT VERIFIED** | Full-screen crimson background. Oversized ✗ icon. "NOT VERIFIED" in `--tp-text-display`. Reason in `--tp-text-lg` below. |
| **Receipts** | Separate tab/route. A simple list of past receipts (timestamp, predicate type, result) on dark card backgrounds. No holder data. |

#### Interaction Rules

- No modals, no sheets, no overlays beyond the result state itself.
- No secondary information on the result screen. The shop owner should be able to glance and know. Return-to-scan is a single large tap target filling the bottom third of the result screen.
- Keyboard/touch target minimum: `64px` for primary actions (larger than default because basic-phone users often have larger fingers and less precise touch).
- Fonts loaded inline (single `@font-face` import at top of `styles.css`). The PWA must be visually correct even on first visit before the font has cached.

#### Performance Budget

| Metric | Target |
|---|---|
| Total page weight (HTML + CSS + JS) | < 80 KB (excluding QR library) |
| QR library | jsQR, vendored locally, < 50 KB |
| Fonts | Variable woff2 subsets, < 30 KB |
| Time to interactive on 3G throttle | < 3 seconds |

### 5.2 `holder-wallet` — The Credential Holder Interface

> **Design constraint:** Earn the holder's trust before asking them to generate a proof. The UI moment where the holder sees "what will be shared" must be designed with care.

**Stack:** Svelte. Uses shared `tokens.css` + Svelte scoped styles. May use simple Svelte transitions for screen-to-screen flows.

#### Screens

| Screen | Description |
|---|---|
| **Credential List** | Grid of credential cards. Each card shows schema type, issuer DID (truncated + copy), and expiry status badge. Revoked credentials shown with crimson badge but not hidden. |
| **Credential Detail** | Expanded view of one credential. Shows all attributes with labels. Private attributes that will be hidden in proof generation are dimmed (40% opacity) with a lock icon. |
| **Claim Request Review** | **Most important screen.** Broken into three sections: (1) "What is being asked" — the predicate in plain English; (2) "What will be revealed" — explicitly nothing beyond the yes/no result (design must make this visually emphatic); (3) "What will stay hidden" — list of all attributes that remain private. A large primary button to "Generate Proof" sits at the bottom. |
| **Proof QR** | Displays the generated proof as a QR code (white background tile on dark surface). Instructions for the holder: "Show this to the verifier to scan." A timer chip showing session expiry countdown in amber. |
| **Scan Claim Request** | Camera view to scan the verifier's claim-request QR. |

#### Trust Design Rules

- The "What will stay hidden" section on the Claim Request Review screen must be visually prominent — not a footnote. Use a `tp-badge-crypto` labelled "HIDDEN BY ZK PROOF" next to each private attribute.
- The proof generation button must not appear until the holder has scrolled past the disclosure summary.
- Never show raw cryptographic bytes to the holder unless they explicitly tap a "Technical Details" disclosure control.

### 5.3 `issuer-console` — Admin Interface

> **Design constraint:** Internal tool. Can be more conventional. Prioritise clarity and function over visual drama. The verification signal colors (`signal-pass`, `signal-fail`) must retain identical meaning here — do not use them decoratively.

**Stack:** Svelte. Uses shared `tokens.css`. Follows same dark surface approach.

#### Screens

| Screen | Description |
|---|---|
| **Dashboard** | Summary stats: total credentials issued, active, revoked, expired. Simple numeric cards. |
| **Schemas** | List of `credential_schemas` with version, attribute count, and issuer DID. |
| **Issue Credential** | Form: select schema → fill synthetic attributes → submit. Result shows issued credential JSON in `.tp-mono` block with copy button. |
| **Credentials** | Searchable/filterable table of issued credentials. Status column uses signal badges. Revoke action uses `tp-btn-danger`. |
| **Revoke Confirmation** | Inline confirmation (not a modal) — "You are about to revoke credential [ID]. This cannot be undone." Two buttons: confirm (danger), cancel (secondary). |

---

## 6. Layout & Grid

### 6.1 Application Shell

All three apps use a consistent shell pattern:

```
┌─────────────────────────────────────────────┐
│  Navbar (--tp-bg-surface, 56px height)       │
│  [Logo emblem] [App name]          [Nav items]│
├─────────────────────────────────────────────┤
│                                              │
│  Content area (--tp-bg-dark, flex-1)         │
│  max-width: 1200px, centered, px-4           │
│                                              │
└─────────────────────────────────────────────┘
```

`verifier-pwa` exception: no navbar on the Scan and Result screens. The scan view is **full-bleed, edge-to-edge**. The navbar only appears on the Receipts view.

### 6.2 Responsive Breakpoints

| Name | Min-width | Usage |
|---|---|---|
| `mobile` | 0 | Default (mobile-first) |
| `tablet` | 640px | Two-column layouts, wider cards |
| `desktop` | 1024px | Sidebar navigation (issuer-console), wider grids |

`verifier-pwa` is **mobile-only design**. It must work correctly on a 320px-wide phone. Desktop is acceptable but not optimised.

`holder-wallet` is mobile-first but should be usable on desktop (a holder might generate proofs from their laptop).

`issuer-console` targets desktop but must not break on tablet.

### 6.3 Spacing Discipline

- All spacing uses the `--tp-space-*` token scale (8pt grid).
- No arbitrary `margin: 13px` or `padding: 7px` values.
- Sections within a page are separated by `--tp-space-12` (48px).
- Components within a section are separated by `--tp-space-6` (24px).
- Items within a component are separated by `--tp-space-4` (16px) or smaller.

---

## 7. Iconography

- **Icon library:** [Lucide Icons](https://lucide.dev/) — SVG sprite or inline SVGs. Chosen for its clean, consistent 24px grid and alignment with the geometric brand language.
- Icons are always paired with a text label unless the meaning is universal (e.g., ✕ close, ✓ check on the result screen).
- Icon size: `20px` in-line with text, `24px` standalone, `48px+` for result-state icons.
- Icon color: inherits from parent text color by default (`currentColor`). Override only for signal icons (`--tp-signal-pass`, `--tp-signal-fail`).

**Core icons used across all apps:**

| Icon | Usage |
|---|---|
| `shield-check` | Credential valid state |
| `shield-x` | Credential invalid / revoked |
| `lock` | Hidden / private attribute marker |
| `scan-line` | QR scan action |
| `qr-code` | QR display state |
| `clock` | Session expiry timer |
| `copy` | Copy to clipboard |
| `check-circle-2` | VERIFIED result |
| `x-circle` | NOT VERIFIED result |
| `file-text` | Receipt / log items |

---

## 8. Motion & Animation

### 8.1 Principles

- Motion communicates state change, not aesthetics. Every animation must have a purpose.
- Duration: use `--tp-transition-fast` (150ms) for micro-interactions (hover states, focus rings), `--tp-transition-normal` (250ms) for content transitions, `--tp-transition-slow` (400ms) only for significant context changes (result reveal).
- Never animate between verification states with a "fade in" — the result reveal should feel **decisive** (e.g., a fast scale + fade, not a gentle crossfade).

### 8.2 Result Reveal Animation (verifier-pwa)

The result screen transition is the most important animation in the product:

```css
/* Result screen entry */
@keyframes tp-result-reveal {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.tp-result-screen {
  animation: tp-result-reveal var(--tp-transition-slow) ease-out forwards;
}
```

The glow halo behind the check/X icon pulses once on entry using a `box-shadow` keyframe, then settles.

### 8.3 Reduced Motion

Always respect `prefers-reduced-motion`:

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 9. Accessibility

- **Contrast ratio:** All text on dark backgrounds must meet WCAG AA (4.5:1 for body text, 3:1 for large text). `--tp-text-primary` (#F8FAFC) on `--tp-bg-dark` (#090D16) passes AAA. Verify signal color text contrast before shipping.
- **Focus management:** Keyboard navigation must be fully functional in all apps. The `:focus-visible` ring using `--tp-border-focus` (cyan) must be visible on every focusable element.
- **Screen readers:** Result states on `verifier-pwa` must use `role="alert"` and `aria-live="assertive"` so the result is announced immediately to screen reader users.
- **Touch targets:** 44px minimum (64px for primary actions on `verifier-pwa`).
- **No color as the only signal:** Result states use icon + text + colour — never colour alone.

---

## 10. File & Asset Conventions

```
apps/
├── shared/
│   ├── fonts/
│   │   ├── inter-variable.woff2
│   │   └── jetbrains-mono-variable.woff2
│   ├── tokens.css          ← All CSS custom properties (Section 2 & 3)
│   ├── base.css            ← CSS reset + html/body defaults
│   ├── components.css      ← Shared .tp-* component classes (Section 4)
│   └── icons/              ← Lucide SVG sprites or individual SVG files
├── holder-wallet/
│   └── src/
│       └── styles/         ← App-specific Svelte scoped styles only
├── issuer-console/
│   └── src/
│       └── styles/
└── verifier-pwa/
    ├── styles.css           ← Imports tokens.css + components.css, adds pwa-specific rules
    └── index.html
```

**Import order** (always this order, no exceptions):
1. `tokens.css`
2. `base.css`
3. `components.css`
4. App-specific styles

---

## 11. What Agents Must Not Do

- **Do not** introduce a CSS framework (Tailwind, Bootstrap, UnoCSS) into `verifier-pwa`. The PWA must remain framework-free and under the weight budget.
- **Do not** use signal colors (`--tp-signal-pass`, `--tp-signal-fail`) for anything other than their defined purpose.
- **Do not** render logos or the brand emblem as inline SVG or emoji substitutes.
- **Do not** load fonts from an external CDN at runtime.
- **Do not** use hardcoded hex color values in any app CSS file.
- **Do not** add animations longer than 500ms to any core interaction path.
- **Do not** invent new color tokens. If an existing token doesn't serve the need, flag it as a `// DESIGN-GAP:` and append a proposal to this file.

---

## 12. Open Design Questions (pre-locked)

These were resolved before locking this document. Listed for traceability:

| Question | Decision |
|---|---|
| Light mode? | No. Dark-only. The brand aesthetic and signal legibility require it. |
| CSS framework for Svelte apps? | No framework. Shared `tokens.css` + `components.css` is sufficient and keeps parity with `verifier-pwa`. |
| Custom icon set vs library? | Lucide Icons. Consistent grid, MIT license, good coverage for our use cases. |
| Variable fonts vs static? | Variable woff2 for both Inter and JetBrains Mono. Single file per family reduces requests. |
| Tailwind for `verifier-pwa`? | Explicitly no — weight budget and framework-free constraint from build spec. |
| Font rendering on Android low-end? | Inter chosen partly for good rendering on Android system WebView without subpixel AA. |
