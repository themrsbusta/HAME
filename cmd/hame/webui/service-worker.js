// HAME 1.0: the executable is the authority for local UI assets.
// Do not keep an upstream offline cache across HAME engine/UI upgrades.
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', event => event.waitUntil(self.registration.unregister()));
