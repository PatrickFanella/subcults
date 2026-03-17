# PWA Features

This document describes the Progressive Web App (PWA) features implemented in Subcults.

## Overview

Subcults is now a fully installable Progressive Web App with offline support and advanced caching strategies.

## Features

### 1. Web App Manifest

The app includes a comprehensive manifest (`/manifest.json`) with:

- **App Identity**: Name, short name, and description
- **Display Mode**: Standalone (opens without browser UI)
- **Theme**: Black theme color for consistent branding
- **Icons**: Multiple sizes (192x192, 512x512) with maskable variants for adaptive icons
- **Shortcuts**: Quick actions for common tasks
- **Share Target**: Ability to receive shared content from other apps

### 2. Service Worker

The service worker (`/sw.js`) implements three caching strategies:

#### Cache-First Strategy
- **Used for**: Static assets (JS, CSS, fonts, images)
- **Behavior**: Serves from cache if available, fetches from network if not
- **Benefits**: Fastest loading for static content

#### Network-First Strategy
- **Used for**: Search endpoints, navigation requests
- **Behavior**: Tries network first, falls back to cache if offline
- **Benefits**: Always fresh data when online, offline fallback

#### Stale-While-Revalidate Strategy
- **Used for**: GET /api/scenes endpoints
- **Behavior**: Serves cached version immediately, updates cache in background
- **Benefits**: Fast response + fresh data

### 3. Offline Support

- **Offline Page**: Custom offline fallback page at `/offline.html`
- **Cached Assets**: Critical assets cached on install
- **Cache Management**: Automatic cache cleanup and size limits
  - API cache: 50 entries max
  - Image cache: 100 entries max

### 4. Cache Versioning

The service worker uses versioned caches:
- `subcults-cache-static-v1.1.0`
- `subcults-cache-api-v1.1.0`
- `subcults-cache-images-v1.1.0`

When the version changes, old caches are automatically deleted.

### 5. Update Detection

The app automatically:
- Checks for service worker updates hourly
- Notifies users when a new version is available
- Provides option to reload and activate the update

## Installation

### Desktop (Chrome/Edge)
1. Visit the site
2. Look for the install icon in the address bar
3. Click "Install"

### Mobile (Android)
1. Visit the site in Chrome
2. Tap the menu (three dots)
3. Select "Add to Home screen"

### Mobile (iOS)
1. Visit the site in Safari
2. Tap the Share button
3. Select "Add to Home Screen"

## Testing

### Manual Testing

1. **Install the app**:
   ```bash
   npm run build
   npm run preview
   ```
   Open Chrome DevTools > Application > Manifest to verify

2. **Test offline mode**:
   - Open Chrome DevTools > Network tab
   - Check "Offline"
   - Navigate the app - should show cached content or offline page

3. **Test caching**:
   - Open Chrome DevTools > Application > Cache Storage
   - Verify caches are created and populated

### Automated Testing

Run the service worker tests:
```bash
npm test -- src/lib/service-worker.test.ts
```

All 18 tests should pass.

### PWA Validation

Use Lighthouse to validate PWA features:
```bash
npm run lighthouse
```

Or use the Chrome DevTools Lighthouse panel:
1. Open Chrome DevTools (F12)
2. Go to the "Lighthouse" tab
3. Select "Progressive Web App" category
4. Click "Generate report"

You can also use online tools like [web.dev/measure](https://web.dev/measure) to audit your deployed site.

## API Reference

### Service Worker Registration

```typescript
import { registerServiceWorker } from './lib/service-worker';

registerServiceWorker({
  onUpdateInstalled: () => {
    // Handle update notification
    console.log('New version available!');
  }
});
```

### Checking Registration

```typescript
import { isServiceWorkerRegistered } from './lib/service-worker';

const registered = await isServiceWorkerRegistered();
console.log('Service worker registered:', registered);
```

### Unregistering

```typescript
import { unregisterServiceWorker } from './lib/service-worker';

const success = await unregisterServiceWorker();
console.log('Unregistered:', success);
```

## Cache Strategies Detail

### Request Type → Strategy Mapping

| Request Type | Strategy | Cache Name | Max Entries |
|-------------|----------|------------|-------------|
| Static Assets (JS, CSS, fonts) | Cache-First | static | N/A |
| Images | Cache-First | images | 100 |
| Search API | Network-First | api | 50 |
| Scenes API | Stale-While-Revalidate | api | 50 |
| Other API | Network-First | api | 50 |
| Navigation | Network-First | static | N/A |

## Performance Benefits

- **First Load**: Normal network speed
- **Repeat Visits**: Instant loading from cache
- **Offline**: Cached content available
- **Background Updates**: Fresh data without waiting

## Browser Support

- ✅ Chrome/Edge (Desktop & Mobile)
- ✅ Firefox (Desktop & Mobile)
- ✅ Safari (Desktop & Mobile)
- ✅ Samsung Internet
- ⚠️ Opera Mini (limited support)

## Troubleshooting

### Service worker not updating

1. Force refresh (Ctrl+Shift+R)
2. Clear site data in DevTools
3. Unregister service worker manually

### Cache not working

1. Check DevTools > Application > Service Workers
2. Verify service worker is active
3. Check Network tab for cache hits

### Manifest not recognized

1. Verify manifest.json is accessible at `/manifest.json`
2. Check manifest format with DevTools > Application > Manifest
3. Ensure all required fields are present

## Security Considerations

- Service worker only works over HTTPS (or localhost)
- Cache size limits prevent excessive storage use
- Only same-origin requests are cached
- Automatic cache cleanup prevents stale data

## Future Enhancements

- Background sync for offline actions
- Push notifications integration
- Periodic background sync
- File upload queue when offline
- Optimistic UI updates

## References

- [Web App Manifest Spec](https://www.w3.org/TR/appmanifest/)
- [Service Worker API](https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API)
- [Cache API](https://developer.mozilla.org/en-US/docs/Web/API/Cache)
- [PWA Checklist](https://web.dev/pwa-checklist/)
