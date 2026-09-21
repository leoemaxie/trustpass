# TrustPass — Brand Visual Identity & Design System

**Motto:** *"Prove, don't show!"*

---

## Brand Logo & App Icon (Transparent PNGs)

### Full Horizontal Logo
![TrustPass Horizontal Logo](assets/branding/logo.png)

### App Icon
![TrustPass App Icon](assets/branding/app_icon.png)

---

## Transparent Assets (`assets/branding/`)

Both assets are 32-bit ARGB PNGs with 100% alpha transparency around the emblem and text:

* **Transparent Full Logo**: [`assets/branding/logo.png`](assets/branding/logo.png)  
  *Features the luminous padlock emblem on the left, bold typography **TRUSTPASS**, and the official motto **"Prove, don't show!"** with transparent background.*
* **Transparent App Icon**: [`assets/branding/app_icon.png`](assets/branding/app_icon.png)  
  *The standalone luminous padlock emblem with transparent alpha background for PWA favicons, navbars, and mobile shortcuts.*

---

## Unified Color Tokens (Cross-App CSS Variables)

```css
:root {
  /* Surfaces & Backgrounds */
  --tp-bg-dark: #090D16;          /* Deep Obsidian Navy */
  --tp-bg-surface: #0F172A;       /* Slate Surface / Navbar */
  --tp-bg-card: #1E293B;          /* Interactive Card Background */
  --tp-border: #334155;           /* Clean Border */

  /* Typography */
  --tp-text-primary: #F8FAFC;      /* High-contrast White */
  --tp-text-muted: #94A3B8;        /* Slate Muted */
  --tp-text-accent: #38BDF8;       /* Motto Cyan */

  /* Verification Signals (High Contrast) */
  --tp-signal-emerald: #10B981;    /* Unambiguous PASS ("VERIFIED") */
  --tp-signal-crimson: #EF4444;    /* Unambiguous FAIL ("NOT VERIFIED") */

  /* Cryptographic Accent */
  --tp-accent-cyan: #06B6D4;       /* ZK proof & session badge */
}
```
