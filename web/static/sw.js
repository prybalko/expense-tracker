const CACHE_NAME = 'expense-tracker-v1';

// Assets to cache for offline/instant startup
const STATIC_ASSETS = [
    '/',
    '/static/style.css',
    '/static/datepicker.js',
    '/static/pull-to-refresh.js',
    '/static/manifest.json',
    '/static/favicon.svg',
    '/static/apple-touch-icon.png'
];

// Install: pre-cache static assets
self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            return cache.addAll(STATIC_ASSETS);
        })
    );
    // Activate immediately
    self.skipWaiting();
});

// Activate: clean up old caches
self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((cacheNames) => {
            return Promise.all(
                cacheNames
                    .filter((name) => name !== CACHE_NAME)
                    .map((name) => caches.delete(name))
            );
        })
    );
    // Take control of all pages immediately
    self.clients.claim();
});

// Fetch: cache-first for static assets, stale-while-revalidate for pages
self.addEventListener('fetch', (event) => {
    const url = new URL(event.request.url);
    
    // Skip non-GET requests
    if (event.request.method !== 'GET') {
        return;
    }
    
    // Skip external requests (like htmx CDN)
    if (url.origin !== location.origin) {
        return;
    }
    
    // Static assets: stale-while-revalidate
    // Serve cached version immediately, fetch fresh version in background
    if (url.pathname.startsWith('/static/')) {
        event.respondWith(
            caches.match(event.request).then((cached) => {
                const fetchPromise = fetch(event.request).then((response) => {
                    if (response.ok) {
                        const clone = response.clone();
                        caches.open(CACHE_NAME).then((cache) => {
                            cache.put(event.request, clone);
                        });
                    }
                    return response;
                });
                return cached || fetchPromise;
            })
        );
        return;
    }
    
    // HTML pages: cache-first for instant startup
    // Pull-to-refresh will update content when user wants fresh data
    if (event.request.headers.get('accept')?.includes('text/html') || 
        url.pathname === '/' ||
        url.pathname === '/stats') {
        
        // Check if this is a pull-to-refresh or HTMX request (wants fresh data)
        const isRefreshRequest = event.request.headers.get('HX-Request') === 'true';
        
        if (isRefreshRequest) {
            // HTMX/pull-to-refresh: network-first, update cache
            event.respondWith(
                fetch(event.request).then((response) => {
                    if (response.ok) {
                        const clone = response.clone();
                        caches.open(CACHE_NAME).then((cache) => {
                            cache.put(event.request, clone);
                        });
                    }
                    return response;
                }).catch(() => {
                    return caches.match(event.request);
                })
            );
        } else {
            // Initial page load: cache-first for instant startup
            event.respondWith(
                caches.match(event.request).then((cached) => {
                    if (cached) {
                        return cached;
                    }
                    return fetch(event.request).then((response) => {
                        if (response.ok) {
                            const clone = response.clone();
                            caches.open(CACHE_NAME).then((cache) => {
                                cache.put(event.request, clone);
                            });
                        }
                        return response;
                    });
                })
            );
        }
        return;
    }
    
    // API requests (like POST/DELETE expenses): network only
    // These are handled by the method check above
});
