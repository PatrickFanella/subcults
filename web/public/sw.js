/**
 * Service Worker
 * Handles Web Push notifications and offline capabilities with advanced caching strategies
 */

// Service worker version (increment on changes to force update)
const VERSION = '1.1.0';
const CACHE_PREFIX = 'subcults-cache';
const STATIC_CACHE = `${CACHE_PREFIX}-static-v${VERSION}`;
const API_CACHE = `${CACHE_PREFIX}-api-v${VERSION}`;
const IMAGE_CACHE = `${CACHE_PREFIX}-images-v${VERSION}`;

// Files to cache immediately on install
const STATIC_ASSETS = [
  '/',
  '/offline.html',
  '/manifest.json',
  '/icon-192.svg',
  '/icon-512.svg',
  '/icon-maskable-192.svg',
  '/icon-maskable-512.svg',
];

// Cache size limits
const MAX_API_CACHE_SIZE = 50;
const MAX_IMAGE_CACHE_SIZE = 100;

/**
 * Install Event - Cache static assets
 */
self.addEventListener('install', (event) => {
  console.log(`[ServiceWorker v${VERSION}] Installing...`);
  
  event.waitUntil(
    caches.open(STATIC_CACHE).then((cache) => {
      console.log('[ServiceWorker] Caching static assets');
      return cache.addAll(STATIC_ASSETS).catch((error) => {
        console.error('[ServiceWorker] Failed to cache some static assets:', error);
        // Log which assets failed
        return Promise.allSettled(
          STATIC_ASSETS.map(async (asset) => {
            try {
              const response = await fetch(asset);
              if (response.ok) {
                await cache.put(asset, response);
              } else {
                console.error(`[ServiceWorker] Failed to cache ${asset}: ${response.status}`);
              }
            } catch (err) {
              console.error(`[ServiceWorker] Failed to cache ${asset}:`, err);
            }
          })
        );
      });
    }).then(() => {
      // Skip waiting to activate immediately
      return self.skipWaiting();
    })
  );
});

/**
 * Activate Event - Clean up old caches
 */
self.addEventListener('activate', (event) => {
  console.log(`[ServiceWorker v${VERSION}] Activating...`);
  
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          // Delete old caches that don't match current version
          if (cacheName.startsWith(CACHE_PREFIX) && 
              cacheName !== STATIC_CACHE && 
              cacheName !== API_CACHE && 
              cacheName !== IMAGE_CACHE) {
            console.log('[ServiceWorker] Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => {
      // Claim all clients immediately
      return self.clients.claim();
    })
  );
});

/**
 * Fetch Event - Apply caching strategies based on request type
 */
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Only handle same-origin requests
  if (url.origin !== self.location.origin) {
    return;
  }

  // Apply strategy based on request type
  if (request.method !== 'GET') {
    // Only cache GET requests
    return;
  }

  // API requests
  if (url.pathname.startsWith('/api/')) {
    if (url.pathname.includes('/search')) {
      // Network-first for search
      event.respondWith(networkFirst(request, API_CACHE));
    } else if (url.pathname.startsWith('/api/scenes')) {
      // Stale-while-revalidate for GET /scenes
      event.respondWith(staleWhileRevalidate(request, API_CACHE));
    } else {
      // Network-first for other API requests
      event.respondWith(networkFirst(request, API_CACHE));
    }
  }
  // Image requests
  else if (request.destination === 'image') {
    event.respondWith(cacheFirst(request, IMAGE_CACHE));
  }
  // Manifest - network-first to get updates
  else if (url.pathname === '/manifest.json') {
    event.respondWith(networkFirst(request, STATIC_CACHE));
  }
  // Static assets (JS, CSS, fonts, etc.)
  else if (
    request.destination === 'script' ||
    request.destination === 'style' ||
    request.destination === 'font' ||
    url.pathname.endsWith('.svg')
  ) {
    event.respondWith(cacheFirst(request, STATIC_CACHE));
  }
  // Navigation requests
  else if (request.mode === 'navigate') {
    event.respondWith(networkFirst(request, STATIC_CACHE));
  }
});

/**
 * Cache-First Strategy
 * Serve from cache if available, fetch and cache if not
 */
async function cacheFirst(request, cacheName) {
  const cachedResponse = await caches.match(request);
  
  if (cachedResponse) {
    return cachedResponse;
  }

  try {
    const response = await fetch(request);
    
    // Cache successful responses with valid content
    if (response.ok && isValidResponse(response)) {
      const cache = await caches.open(cacheName);
      cache.put(request, response.clone());
      
      // Limit cache size
      await limitCacheSize(cacheName, getMaxCacheSize(cacheName));
    }
    
    return response;
  } catch (error) {
    console.error('[ServiceWorker] Fetch failed:', error);
    
    // Return offline page for navigation requests
    if (request.mode === 'navigate') {
      const offlineResponse = await caches.match('/offline.html');
      if (offlineResponse) {
        return offlineResponse;
      }
    }
    
    throw error;
  }
}

/**
 * Network-First Strategy
 * Try network first, fall back to cache if offline
 */
async function networkFirst(request, cacheName) {
  try {
    const response = await fetch(request);
    
    // Cache successful responses with valid content
    if (response.ok && isValidResponse(response)) {
      const cache = await caches.open(cacheName);
      cache.put(request, response.clone());
      
      // Limit cache size
      await limitCacheSize(cacheName, getMaxCacheSize(cacheName));
    }
    
    return response;
  } catch (error) {
    console.log('[ServiceWorker] Network failed, trying cache:', request.url);
    
    const cachedResponse = await caches.match(request);
    
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // Return offline page for navigation requests
    if (request.mode === 'navigate') {
      const offlineResponse = await caches.match('/offline.html');
      if (offlineResponse) {
        return offlineResponse;
      }
    }
    
    throw error;
  }
}

/**
 * Stale-While-Revalidate Strategy
 * Serve from cache immediately, update cache in background
 */
async function staleWhileRevalidate(request, cacheName) {
  const cachedResponse = await caches.match(request);
  
  const fetchPromise = fetch(request).then(async (response) => {
    if (response.ok && isValidResponse(response)) {
      const cache = await caches.open(cacheName);
      await cache.put(request, response.clone());
      await limitCacheSize(cacheName, getMaxCacheSize(cacheName));
    }
    return response;
  }).catch((error) => {
    console.log('[ServiceWorker] Background fetch failed:', error);
    return null;
  });

  // Return cached response immediately, or wait for network
  return cachedResponse || fetchPromise;
}

/**
 * Limit cache size by removing excess entries
 * Note: Cache key iteration order is not guaranteed across browsers
 */
async function limitCacheSize(cacheName, maxSize) {
  const cache = await caches.open(cacheName);
  const keys = await cache.keys();
  
  if (keys.length > maxSize) {
    // Remove excess entries (order not guaranteed - may not be FIFO)
    const keysToDelete = keys.slice(0, keys.length - maxSize);
    await Promise.all(keysToDelete.map((key) => cache.delete(key)));
    console.log(`[ServiceWorker] Trimmed cache ${cacheName} to ${maxSize} entries`);
  }
}

/**
 * Get max cache size for a given cache name
 */
function getMaxCacheSize(cacheName) {
  if (cacheName === API_CACHE) {
    return MAX_API_CACHE_SIZE;
  }
  if (cacheName === IMAGE_CACHE) {
    return MAX_IMAGE_CACHE_SIZE;
  }
  return 50; // Default
}

/**
 * Validate response before caching to prevent cache poisoning
 */
function isValidResponse(response) {
  // Only cache successful responses
  if (!response || !response.ok) {
    return false;
  }
  
  // Check content-type for expected types
  const contentType = response.headers.get('content-type');
  if (!contentType) {
    return false;
  }
  
  // Allow common web content types
  const validTypes = [
    'text/html',
    'text/css',
    'text/javascript',
    'application/javascript',
    'application/json',
    'image/',
    'font/',
    'application/font',
    'image/svg+xml',
  ];
  
  return validTypes.some(type => contentType.includes(type));
}

/**
 * Push Event Handler
 * Receives push notifications from the server
 */
self.addEventListener('push', (event) => {
  console.log('[ServiceWorker] Push notification received:', event);

  // Default notification data
  let title = 'Subcults';
  let options = {
    body: 'You have a new notification',
    icon: '/vite.svg',
    badge: '/vite.svg',
    vibrate: [200, 100, 200],
    tag: 'subcults-notification',
    requireInteraction: false,
  };

  // Parse notification data if available
  if (event.data) {
    try {
      const data = event.data.json();
      console.log('[ServiceWorker] Push data:', data);
      
      if (data.title) {
        title = data.title;
      }
      
      if (data.body) {
        options.body = data.body;
      }
      
      if (data.icon) {
        options.icon = data.icon;
      }
      
      if (data.tag) {
        options.tag = data.tag;
      }
      
      if (data.url) {
        options.data = { url: data.url };
      }
    } catch (error) {
      console.error('[ServiceWorker] Failed to parse push data:', error);
    }
  }

  // Show notification
  event.waitUntil(
    self.registration.showNotification(title, options)
  );
});

/**
 * Notification Click Handler
 * Handles user interaction with notifications
 */
self.addEventListener('notificationclick', (event) => {
  console.log('[ServiceWorker] Notification clicked:', event);

  // Close the notification
  event.notification.close();

  // Handle notification click action
  event.waitUntil(
    (async () => {
      const url = event.notification.data?.url || '/';
      
      // Normalize URL to absolute for comparison
      const targetUrl = new URL(url, self.location.origin).href;
      
      // Try to focus existing window/tab
      const clients = await self.clients.matchAll({
        type: 'window',
        includeUncontrolled: true,
      });

      for (const client of clients) {
        if (client.url === targetUrl && 'focus' in client) {
          return client.focus();
        }
      }

      // Open new window if no matching client found
      if (self.clients.openWindow) {
        return self.clients.openWindow(targetUrl);
      }
    })()
  );
});

/**
 * Notification Close Handler
 * Handles notification dismissal
 */
self.addEventListener('notificationclose', (event) => {
  console.log('[ServiceWorker] Notification closed:', event);
  // Future: Track notification dismissals for analytics
});

console.log(`[ServiceWorker v${VERSION}] Loaded`);
