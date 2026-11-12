// Calciotto's service worker. Deliberately empty of any caching: it exists so
// the app is installable and, later, so it can receive push notifications —
// not to work offline. A worker that caches the app shell is the classic
// cause of "the app never updates", and nothing here needs offline use.
//
// No `fetch` handler either: without one, every request goes straight to the
// network exactly as it would with no worker at all.

// Take over as soon as a new version is installed, instead of waiting for
// every tab of the old one to close: there is no cached state to migrate.
self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});
