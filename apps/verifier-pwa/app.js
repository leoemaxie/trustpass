/**
 * verifier-pwa — App Shell & Logic Skeleton
 * ==========================================
 * This file owns:
 *   - Screen routing (show/hide between Scan, Processing, Pass, Fail, Receipts)
 *   - Camera lifecycle (getUserMedia → jsQR frame loop)
 *   - API calls to verifier-api (verify proof)
 *   - Receipt rendering
 *
 * TODO (next agent): wire verifyProof() to the real verifier-api endpoint.
 * The screen-switching and camera plumbing is done; only the API call body
 * and QR payload parsing need implementation.
 *
 * Config
 */

const CONFIG = {
  VERIFIER_API_BASE: window.VERIFIER_API_BASE || 'http://localhost:8082',
  SCAN_INTERVAL_MS: 200,  // how often to decode a camera frame
};

/* ── State ────────────────────────────────────────────────────────────── */
const state = {
  scanning: false,
  stream: null,
  scanLoop: null,
  receipts: JSON.parse(localStorage.getItem('tp_receipts') || '[]'),
};

/* ── DOM refs ─────────────────────────────────────────────────────────── */
const $ = id => document.getElementById(id);
const screens = {
  scan:       $('screen-scan'),
  processing: $('screen-processing'),
  pass:       $('screen-pass'),
  fail:       $('screen-fail'),
  receipts:   $('screen-receipts'),
};

/* ── Screen router ───────────────────────────────────────────────────── */
function showScreen(name) {
  Object.entries(screens).forEach(([key, el]) => {
    el.classList.toggle('pwa-screen--hidden', key !== name);
  });
}

/* ── Camera & jsQR ───────────────────────────────────────────────────── */
async function startCamera() {
  const video  = $('scanner-video');
  const canvas = $('scanner-canvas');
  const ctx    = canvas.getContext('2d', { willReadFrequently: true });

  try {
    state.stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment' },
    });
    video.srcObject = state.stream;
    await video.play();
    state.scanning = true;

    function tick() {
      if (!state.scanning) return;
      if (video.readyState === video.HAVE_ENOUGH_DATA) {
        canvas.width  = video.videoWidth;
        canvas.height = video.videoHeight;
        ctx.drawImage(video, 0, 0);
        const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
        /* global jsQR */
        const code = jsQR(imageData.data, imageData.width, imageData.height, {
          inversionAttempts: 'dontInvert',
        });
        if (code?.data) {
          stopCamera();
          handleQRCode(code.data);
          return;
        }
      }
      state.scanLoop = setTimeout(tick, CONFIG.SCAN_INTERVAL_MS);
    }
    tick();
  } catch (err) {
    showToast('Camera unavailable: ' + err.message, 'fail');
  }
}

function stopCamera() {
  state.scanning = false;
  clearTimeout(state.scanLoop);
  if (state.stream) {
    state.stream.getTracks().forEach(t => t.stop());
    state.stream = null;
  }
}

/* ── QR payload handling ─────────────────────────────────────────────── */
/**
 * TODO (next agent): Parse the real QR payload format from verifier-api.
 * The verifier-api encodes { sessionToken, claimRequest } as JSON in the QR.
 * Expected shape:
 *   { sessionToken: string, claimRequest: { schemaName, attributeName, operator, value } }
 */
async function handleQRCode(raw) {
  showScreen('processing');
  let payload;
  try {
    payload = JSON.parse(raw);
  } catch {
    showResult('fail', null, 'Invalid QR code — could not parse payload.');
    return;
  }

  if (!payload.sessionToken || !payload.claimRequest) {
    showResult('fail', null, 'QR code is missing required fields.');
    return;
  }

  await verifyProof(payload);
}

/* ── API: verify proof ───────────────────────────────────────────────── */
/**
 * TODO (next agent): This function submits the holder-generated proof QR
 * to verifier-api POST /verification/verify.
 *
 * For now it demonstrates the screen flow with a mock response.
 * Replace the mock block with a real fetch() call.
 */
async function verifyProof(payload) {
  try {
    // TODO: Replace mock with real API call:
    // const res = await fetch(`${CONFIG.VERIFIER_API_BASE}/verification/verify`, {
    //   method: 'POST',
    //   headers: { 'Content-Type': 'application/json' },
    //   body: JSON.stringify(payload),
    // });
    // const data = await res.json();

    // --- MOCK RESPONSE (remove when API is wired) ---
    await new Promise(r => setTimeout(r, 1200)); // simulate network
    const data = { valid: true, claimSummary: 'Age ≥ 18', receiptId: 'mock-receipt-001' };
    // --- END MOCK ---

    const claimSummary = data.claimSummary || formatClaim(payload.claimRequest);
    showResult(data.valid ? 'pass' : 'fail', claimSummary, data.rejectionReason);
    storeReceipt({ result: data.valid, claimSummary, receiptId: data.receiptId });
  } catch (err) {
    showResult('fail', null, 'Network error — could not reach verification service.');
  }
}

/* ── Result rendering ────────────────────────────────────────────────── */
function showResult(type, claimSummary, reason) {
  if (type === 'pass') {
    $('pass-claim-text').textContent = claimSummary || '';
    showScreen('pass');
  } else {
    $('fail-reason-text').textContent = formatReason(reason);
    showScreen('fail');
  }
}

/**
 * Maps typed rejection reasons to plain-English messages for shop owners.
 * Matches the typed error enum from the build spec.
 */
function formatReason(reason) {
  const map = {
    SignatureInvalid:      'The credential signature is invalid.',
    PredicateNotSatisfied: 'The condition is not met.',
    CredentialExpired:     'The credential has expired.',
    CredentialRevoked:     'The credential has been revoked.',
    SessionTokenExpired:   'The verification session has timed out. Ask the holder to try again.',
    SessionTokenReused:    'This QR code has already been used.',
  };
  return map[reason] || reason || 'Verification could not be completed.';
}

function formatClaim(req) {
  if (!req) return '';
  return `${req.attributeName} ${req.operator} ${req.value}`;
}

/* ── Receipts ────────────────────────────────────────────────────────── */
function storeReceipt(receipt) {
  receipt.timestamp = new Date().toISOString();
  state.receipts.unshift(receipt);
  localStorage.setItem('tp_receipts', JSON.stringify(state.receipts.slice(0, 50)));
}

function renderReceipts() {
  const list = $('receipts-list');
  if (!state.receipts.length) return; // empty state already in HTML

  list.innerHTML = state.receipts.map(r => `
    <div class="pwa-receipt-item">
      <div class="pwa-receipt-item__indicator pwa-receipt-item__indicator--${r.result ? 'pass' : 'fail'}" aria-hidden="true"></div>
      <div class="pwa-receipt-item__body">
        <p class="pwa-receipt-item__predicate">${escHtml(r.claimSummary || 'Unknown claim')}</p>
        <p class="pwa-receipt-item__time">${formatTime(r.timestamp)}</p>
      </div>
      <span class="pwa-receipt-item__result pwa-receipt-item__result--${r.result ? 'pass' : 'fail'}">
        ${r.result ? 'PASS' : 'FAIL'}
      </span>
    </div>
  `).join('');
}

function formatTime(iso) {
  return new Date(iso).toLocaleString(undefined, {
    month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
  });
}

function escHtml(str) {
  return str.replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

/* ── Toast ───────────────────────────────────────────────────────────── */
function showToast(message, type = 'neutral') {
  const region = $('toast-region');
  const toast  = document.createElement('div');
  toast.className = `tp-toast tp-toast-${type}`;
  toast.innerHTML = `<p class="tp-toast__message">${escHtml(message)}</p>`;
  region.appendChild(toast);
  setTimeout(() => toast.remove(), 4000);
}

/* ── Event wiring ────────────────────────────────────────────────────── */
function bindEvents() {
  $('btn-start-scan').addEventListener('click', () => {
    startCamera();
  });

  $('btn-receipts').addEventListener('click', () => {
    renderReceipts();
    showScreen('receipts');
  });

  $('btn-back-to-scan').addEventListener('click', () => {
    showScreen('scan');
  });

  [$('btn-scan-again-pass'), $('btn-scan-again-fail')].forEach(btn => {
    btn.addEventListener('click', () => {
      showScreen('scan');
    });
  });
}

/* ── Init ────────────────────────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  bindEvents();
  showScreen('scan');
});

/* ── PWA Service Worker registration ─────────────────────────────────── */
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {
      // SW optional — app works without it
    });
  });
}
