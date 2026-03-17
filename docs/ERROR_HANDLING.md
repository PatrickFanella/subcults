# Error Handling & Toast Notifications

This document describes how to use the error boundary and toast notification system in the Subcults frontend application.

## Table of Contents

- [Error Boundary](#error-boundary)
- [Toast Notifications](#toast-notifications)
- [API Error Handling](#api-error-handling)
- [Accessibility](#accessibility)
- [Best Practices](#best-practices)

## Error Boundary

The `ErrorBoundary` component catches React rendering errors and displays a user-friendly fallback UI.

### Basic Usage

The error boundary is already integrated at the root level in `App.tsx`, so all components are protected:

```tsx
import { ErrorBoundary } from './components/ErrorBoundary';

function App() {
  return (
    <ErrorBoundary>
      <YourAppContent />
    </ErrorBoundary>
  );
}
```

### Custom Fallback UI

You can provide a custom fallback UI for specific sections:

```tsx
<ErrorBoundary fallback={<CustomErrorUI />}>
  <CriticalComponent />
</ErrorBoundary>
```

### Features

- **Automatic Error Catching**: Catches errors during rendering, in lifecycle methods, and in constructors
- **User-Friendly Fallback**: Shows a clean error screen with reload and report options
- **Development Details**: In development mode, shows filtered stack traces for debugging
- **Production Safety**: Stack traces are hidden in production to avoid exposing sensitive information
- **Accessibility**: 
  - Uses `role="alert"` and `aria-live="assertive"` for screen reader announcements
  - Automatically focuses the reload button when error occurs
  - Semantic HTML structure
- **Error Reporting**: Provides a "Report Issue" button that opens email with pre-filled error details

### Logging

In development mode, errors are logged to the console with filtered stack traces:

```javascript
console.warn('[ErrorBoundary] Rendering error caught:', {
  message: error.message,
  stack: filterStackTrace(error.stack),
  componentStack: errorInfo.componentStack,
});
```

In production, integrate with an error tracking service (e.g., Sentry):

```typescript
// In componentDidCatch method
if (import.meta.env.PROD) {
  sentryClient.captureException(error, { extra: errorInfo });
}
```

## Toast Notifications

The toast system provides ephemeral notifications for user feedback.

### Basic Usage

Use the `useToasts()` hook to show notifications:

```tsx
import { useToasts } from '../stores/toastStore';

function MyComponent() {
  const toasts = useToasts();

  const handleSuccess = () => {
    toasts.success('Scene created successfully!');
  };

  const handleError = () => {
    toasts.error('Failed to create scene. Please try again.');
  };

  const handleInfo = () => {
    toasts.info('Loading scenes...');
  };

  return (
    <button onClick={handleSuccess}>Create Scene</button>
  );
}
```

### Custom Duration

Control how long toasts are displayed (default: 5000ms):

```tsx
// Show for 3 seconds
toasts.success('Quick notification', 3000);

// Show indefinitely (must be manually dismissed)
toasts.error('Critical error', 0);
```

### Advanced Usage

Use the `custom` method for full control:

```tsx
toasts.custom({
  type: 'error',
  message: 'Cannot delete active scene',
  duration: 0, // Don't auto-dismiss
  dismissible: false, // User cannot dismiss
});
```

### Manual Dismissal

Get the toast ID to dismiss programmatically:

```tsx
const id = toasts.info('Processing...');

// Later, dismiss it
toasts.dismiss(id);

// Or clear all toasts
toasts.clearAll();
```

### Integration with ToastContainer

The `ToastContainer` is already integrated in `App.tsx` and will automatically display all toasts:

```tsx
import { ToastContainer } from './components/ToastContainer';

function App() {
  return (
    <>
      <YourAppContent />
      <ToastContainer />
    </>
  );
}
```

### Toast Types

Three toast types are available:

- **Success** (✓): Green, for successful operations
- **Error** (✕): Red, for errors and failures
- **Info** (ℹ): Blue, for informational messages

### Accessibility

Toasts are fully accessible:

- Container uses `role="region"` with `aria-label="Notifications"`
- Each toast uses `role="status"` with `aria-live="polite"` and `aria-atomic="true"`
- Dismiss buttons have proper `aria-label="Dismiss notification"`
- Screen readers will announce new toasts automatically

## API Error Handling

### Error Mapping

Use the error mapping utilities to convert API errors to user-friendly messages:

```tsx
import { getErrorMessage } from '../utils/errorMapping';
import { ApiClientError } from '../lib/api-client';

try {
  await apiClient.request('/api/scenes', { method: 'POST', body: data });
} catch (error) {
  if (error instanceof ApiClientError) {
    const message = getErrorMessage(error);
    toasts.error(message);
  }
}
```

### Auto-Toast Pattern

Create a wrapper for API calls with automatic error toasting:

```tsx
import { getErrorMessage, shouldShowToast } from '../utils/errorMapping';
import { apiClient, ApiClientError } from '../lib/api-client';
import { useToasts } from '../stores/toastStore';

async function apiCallWithToast<T>(
  request: () => Promise<T>,
  successMessage?: string
): Promise<T | null> {
  const toasts = useToasts();
  
  try {
    const result = await request();
    
    if (successMessage) {
      toasts.success(successMessage);
    }
    
    return result;
  } catch (error) {
    if (error instanceof ApiClientError && shouldShowToast(error)) {
      toasts.error(getErrorMessage(error));
    }
    return null;
  }
}

// Usage
const scene = await apiCallWithToast(
  () => apiClient.request('/api/scenes', { method: 'POST', body: data }),
  'Scene created successfully!'
);
```

### Known Error Codes

The following error codes are mapped to user-friendly messages:

**Auth Errors:**
- `unauthorized`: "Your session has expired. Please log in again."
- `invalid_token`: "Invalid authentication token. Please log in again."
- `auth_failed`: "Authentication failed. Please check your credentials."

**Network Errors:**
- `network_error`: "Network error. Please check your connection and try again."
- `timeout`: "Request timed out. Please try again."

**Validation Errors:**
- `validation`: "Invalid input. Please check your data and try again."
- `invalid_scene_name`: "Scene name is invalid. Use only letters, numbers, spaces, dashes, underscores, and periods."
- `duplicate_scene_name`: "A scene with this name already exists."
- `invalid_time_range`: "Invalid time range. End time must be after start time."

**Resource Errors:**
- `not_found`: "The requested resource was not found."
- `conflict`: "A conflict occurred. The resource may have been modified."

**Permission Errors:**
- `forbidden`: "You do not have permission to perform this action."

**Server Errors:**
- `internal_error`: "An internal server error occurred. Please try again later."
- `service_unavailable`: "Service temporarily unavailable. Please try again later."

## Accessibility

Both the error boundary and toast system follow accessibility best practices:

### Error Boundary
- Uses semantic HTML (`role="alert"`, `aria-live="assertive"`)
- Automatically manages focus to reload button
- Buttons have descriptive `aria-describedby` attributes
- Works with keyboard navigation

### Toast Notifications
- Live region announces toasts to screen readers (`aria-live="polite"`)
- Each toast has `role="status"` for semantic meaning
- `aria-atomic="true"` ensures full toast content is announced
- Dismiss buttons have clear labels
- Positioned to not block content (top-right corner)
- High contrast colors for visibility

## Best Practices

### Do's

✅ Use error boundary at component boundaries for resilience
✅ Show success toasts for user-initiated actions
✅ Use appropriate toast types (success/error/info)
✅ Keep toast messages concise and actionable
✅ Map API error codes to user-friendly messages
✅ Test error states and toast behavior
✅ Ensure toasts don't overlap critical UI elements

### Don'ts

❌ Don't show toasts for every API call (avoid notification fatigue)
❌ Don't use long toast messages (keep under 2 lines)
❌ Don't show multiple toasts for the same error
❌ Don't expose sensitive data in error messages
❌ Don't show stack traces in production
❌ Don't rely solely on toasts for critical information
❌ Don't auto-dismiss critical error messages

### Performance

- Toasts are rendered in a fixed position overlay (no reflow)
- Auto-dismiss uses setTimeout (cleanup handled automatically)
- Error boundary only re-renders on error state change
- Zustand store is optimized for minimal re-renders

### Testing

Always test error handling:

```tsx
import { render, screen } from '@testing-library/react';
import { ErrorBoundary } from './ErrorBoundary';

it('catches and displays errors', () => {
  const ThrowError = () => { throw new Error('Test error'); };
  
  render(
    <ErrorBoundary>
      <ThrowError />
    </ErrorBoundary>
  );
  
  expect(screen.getByRole('alert')).toBeInTheDocument();
  expect(screen.getByText('Something went wrong')).toBeInTheDocument();
});
```

Test toast notifications:

```tsx
import { renderHook, act } from '@testing-library/react';
import { useToasts } from '../stores/toastStore';

it('shows success toast', () => {
  const { result } = renderHook(() => useToasts());
  
  act(() => {
    result.current.success('Test message');
  });
  
  const toasts = useToastStore.getState().toasts;
  expect(toasts[0]).toMatchObject({
    type: 'success',
    message: 'Test message',
  });
});
```

## Examples

### Scene Creation with Error Handling

```tsx
import { useToasts } from '../stores/toastStore';
import { apiClient } from '../lib/api-client';
import { getErrorMessage } from '../utils/errorMapping';

function CreateSceneForm() {
  const toasts = useToasts();
  
  const handleSubmit = async (data: SceneData) => {
    try {
      await apiClient.request('/api/scenes', {
        method: 'POST',
        body: JSON.stringify(data),
      });
      
      toasts.success('Scene created successfully!');
      navigate('/scenes');
    } catch (error) {
      toasts.error(getErrorMessage(error));
    }
  };
  
  return <form onSubmit={handleSubmit}>...</form>;
}
```

### Loading State with Toast

```tsx
function DataLoader() {
  const toasts = useToasts();
  
  useEffect(() => {
    const loadData = async () => {
      const loadingToast = toasts.info('Loading data...', 0);
      
      try {
        await fetchData();
        toasts.dismiss(loadingToast);
        toasts.success('Data loaded successfully!');
      } catch (error) {
        toasts.dismiss(loadingToast);
        toasts.error(getErrorMessage(error));
      }
    };
    
    loadData();
  }, []);
  
  return <div>...</div>;
}
```

### Custom Error Boundary for Section

```tsx
function CriticalSection() {
  return (
    <ErrorBoundary
      fallback={
        <div>
          <h2>This section is temporarily unavailable</h2>
          <button onClick={() => window.location.reload()}>
            Reload
          </button>
        </div>
      }
    >
      <CriticalFeature />
    </ErrorBoundary>
  );
}
```
