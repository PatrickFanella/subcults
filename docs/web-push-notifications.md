# Web Push Notifications

This directory contains the implementation of Web Push notifications for Subcults, enabling real-time user re-engagement through browser notifications.

## Architecture Overview

The Web Push notification system consists of four main components:

1. **Notification Store** (`stores/notificationStore.ts`) - Zustand store managing subscription state
2. **Notification Service** (`lib/notification-service.ts`) - Web Push API wrapper and backend communication
3. **Service Worker** (`public/sw.js`) - Background script handling push events
4. **UI Component** (`components/NotificationSettings.tsx`) - User interface for opt-in/opt-out

## Privacy & Consent

**Privacy-first design principles:**

- **Explicit Opt-In**: Notifications are NEVER auto-prompted. Users must explicitly enable them from Settings.
- **User Control**: Users can disable notifications at any time, both in-app and via browser settings.
- **Transparent Purpose**: Clear communication about what notifications will be sent (new events, stream starts, membership approvals).
- **Local Storage**: Subscription endpoints and keys are treated as sensitive data and are not persisted to localStorage. The active subscription is derived from the browser's PushManager on each page load.
- **No Tracking**: Subscription endpoints are only used for sending notifications, not for analytics or tracking user behavior across the app or web.

## User Flow

1. User navigates to **Settings** page
2. User clicks "Enable" button in Notifications section
3. Browser displays native permission prompt
4. If granted:
   - Service worker subscribes to push notifications
   - Subscription sent to backend API
   - UI updates to show "Enabled" status
5. User can click "Disable" to unsubscribe at any time

## Technical Implementation

### Permission States

The Notification API has three permission states:

- `default` - Permission not yet requested
- `granted` - User has granted permission
- `denied` - User has blocked notifications

### Subscription Flow

```typescript
// 1. Request permission
const permission = await Notification.requestPermission();

// 2. Subscribe to push notifications (requires service worker)
const subscription = await registration.pushManager.subscribe({
  userVisibleOnly: true,
  applicationServerKey: vapidPublicKey,
});

// 3. Send subscription to backend
await fetch('/api/notifications/subscribe', {
  method: 'POST',
  body: JSON.stringify(subscription),
});
```

### Service Worker

The service worker (`public/sw.js`) handles:

- **Push Events**: Receives notifications from server, displays to user
- **Notification Clicks**: Opens app when user clicks notification
- **Offline Support**: Future enhancement for offline capabilities

### Backend Integration

The frontend expects the following backend endpoints:

- `POST /api/notifications/subscribe` - Register new push subscription
  - Body: `{ endpoint: string, keys: { p256dh: string, auth: string } }`
  - Returns: 200 OK

- `DELETE /api/notifications/subscribe` - Delete push subscription
  - Body: `{ endpoint: string }`
  - Returns: 200 OK or 404 if not found

### VAPID Keys

Web Push requires VAPID (Voluntary Application Server Identification) keys for authentication.

**Configuration:**

Set the `VITE_VAPID_PUBLIC_KEY` environment variable in `configs/dev.env`:

```env
VITE_VAPID_PUBLIC_KEY=BEl62iUYgUivxIkv69yViEuiBIa-Ib37J8xQmr8Db5s...
```

The backend must generate a VAPID key pair and provide the public key to the frontend.

## Browser Compatibility

Web Push notifications are supported in:

- ✅ Chrome/Edge (Desktop & Android)
- ✅ Firefox (Desktop & Android)
- ✅ Safari 16+ (Desktop & iOS 16.4+)
- ❌ Opera Mini
- ❌ IE 11

The UI automatically displays a fallback message for unsupported browsers.

## Testing

### Unit Tests

```bash
cd web
npm test -- src/stores/notificationStore.test.ts
npm test -- src/lib/notification-service.test.ts
```

### Manual Testing

1. Start dev server: `npm run dev`
2. Navigate to Settings page
3. Click "Enable" in Notifications section
4. Grant permission in browser prompt
5. Verify "Enabled" badge appears
6. Check browser DevTools > Application > Service Workers
7. Verify subscription in DevTools > Application > Push Messaging

### Testing Push Notifications

To test receiving notifications, you'll need:

1. A backend service to send push notifications
2. The user's subscription endpoint and keys
3. A tool like [web-push](https://www.npmjs.com/package/web-push) CLI

Example using web-push CLI:

```bash
npx web-push send-notification \
  --endpoint="https://fcm.googleapis.com/fcm/send/..." \
  --key="p256dh-key" \
  --auth="auth-key" \
  --vapid-subject="mailto:admin@subcults.com" \
  --vapid-pubkey="BEl62..." \
  --vapid-pvtkey="..." \
  --payload='{"title":"Test","body":"Hello!"}'
```

## Security Considerations

1. **VAPID Keys**: Keep private key secure on backend. Never expose in frontend code.
2. **Subscription Endpoints**: Treat as sensitive data. Don't log or expose unnecessarily.
3. **HTTPS Required**: Service workers and push notifications require HTTPS (or localhost).
4. **User Verification**: Backend should verify subscription belongs to authenticated user.
5. **Rate Limiting**: Implement rate limits to prevent notification spam.

## Future Enhancements

- [ ] Notification preferences (event types, frequency)
- [ ] In-app notification center for message history
- [ ] Notification action buttons (e.g., "View Event", "Dismiss")
- [ ] Rich notifications with images and custom sounds
- [ ] Analytics for notification engagement
- [ ] A/B testing for notification content
- [ ] Notification scheduling and batching

## Troubleshooting

### Permission Denied

If permission is denied, users must manually reset it:

- **Chrome**: Settings > Privacy > Site Settings > Notifications
- **Firefox**: Address bar (i) icon > Permissions > Notifications
- **Safari**: Safari > Settings > Websites > Notifications

### Service Worker Not Registered

Check browser console for errors. Common issues:

- HTTPS required (except localhost)
- Service worker file must be at root of scope
- Syntax errors in service worker code

### Subscription Fails

Check:

1. Service worker is active: DevTools > Application > Service Workers
2. VAPID public key is valid base64 URL-safe string
3. Network tab shows successful API request
4. Browser console for detailed error messages

## Resources

- [Web Push API (MDN)](https://developer.mozilla.org/en-US/docs/Web/API/Push_API)
- [Service Worker API (MDN)](https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API)
- [VAPID Specification](https://datatracker.ietf.org/doc/html/rfc8292)
- [Web Push Protocol](https://datatracker.ietf.org/doc/html/rfc8030)
- [Push Notifications UX Best Practices](https://web.dev/push-notifications-overview/)
