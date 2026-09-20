# TrustPass Verifier — Shop Owner Walkthrough

This document satisfies the Checkpoint 8 acceptance criterion: a non-technical, 5-step description of how a shop owner or merchant uses the `verifier-pwa` web application at the point of sale.

---

## 5-Step Point-of-Sale Verification Walkthrough

### Step 1: Open the Verifier App
The shop owner opens the TrustPass Verifier PWA on their mobile phone browser (or home-screen shortcut). The screen opens immediately with no login screens, no account creation, and zero technical jargon.

### Step 2: Tap "Start Scan"
The shop owner taps the large **Start Scan** button at the bottom of the screen. The device camera activates within the viewfinder frame.

### Step 3: Point Camera at the Customer's QR Code
The customer (credential holder) shows their proof QR code on their phone screen. The shop owner points the camera at the code. The app automatically recognizes the code within 200ms and displays a "Verifying…" indicator.

### Step 4: Read the Result (VERIFIED or NOT VERIFIED)
The entire phone screen instantly changes to an unambiguous, full-bleed color state:
- **Green checkmark with "VERIFIED"** and the satisfied condition (e.g. `Age ≥ 18`). The shop owner proceeds with the sale.
- **Red X with "NOT VERIFIED"** and an everyday explanation (e.g. *"The condition is not met"* or *"This QR code has already been used"*).
The shop owner taps **Scan Another** to return to the scanner for the next customer.

### Step 5: Audit Verification Proof in "Receipts"
At any time, the shop owner can tap **Receipts** in the top header. This displays a time-stamped log of every completed verification (`PASS` or `FAIL`) and its unique receipt identifier.
Crucially: **no customer name, date of birth, ID number, photo, or personal identifier is ever shown, captured, or stored.**
