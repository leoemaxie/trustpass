/**
 * TrustPass Verifier PWA — Service Worker
 * Cache-first for core static assets so the app works reliably on weak or offline networks.
 */

const CACHE_NAME = 'trustpass-verifier-v1';

const STATIC_ASSETS = [
  '/',
  '/index.html',
  '/styles.css',
  '/app.js',
  '/manifest.json',
  '/vendor/jsQR.js',
  '/shared/tokens.css',
  '/shared/base.css',
  '/shared/components.css',
  '/shared/assets/branding/app_icon.png',
  '/shared/assets/branding/logo.png',
];

self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME).then(cache => {
      return cache.addAll(STATIC_ASSETS).catch(err => {
        console.warn('PWA cache pre-load partial warning:', err);
      });
    })
  );
  self.skipWaiting();
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(keys => {
      return Promise.all(
        keys.filter(k => k !== CACHE_NAME).map(k => caches.delete(k))
      );
    })
  );
  self.clients.claim();
});

self.addEventListener('fetch', event => {
  const url = new URL(event.request.url);

  // Network-only for API calls
  if (url.pathname.startsWith('/verification') || url.pathname.startsWith('/api') || url.port === '8083') {
    return;
  }

  // Cache-first, fallback to network for static files
  event.respondWith(
    caches.match(event.request).then(cached => {
      if (cached) return cached;
      return fetch(event.request).then(networkResponse => {
        if (networkResponse && networkResponse.status === 200 && event.request.method === 'GET') {
          const resClone = networkResponse.clone();
          caches.open(CACHE_NAME).then(cache => cache.put(event.request, resClone));
        }
        return networkResponse;
      });
    }).catch(() => {
      // Offline fallback
      return caches.match('/index.html');
    })
  );
});
