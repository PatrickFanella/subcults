# Integration Tests

This directory contains integration tests for critical user flows in the Subcults application.

## Overview

Integration tests validate complete user journeys by testing how multiple components work together with mocked API responses. They use:

- **React Testing Library**: For rendering components and simulating user interactions
- **MSW (Mock Service Worker)**: For intercepting and mocking API requests
- **Vitest**: As the test runner

## Test Coverage

### 1. Login to Dashboard Flow (`login-flow.integration.test.tsx`)
Tests the complete authentication flow:
- User enters credentials and logs in
- Error handling for invalid credentials  
- Remember me functionality
- Loading states during login

### 2. Scene Detail to Events Navigation (`scene-events-navigation.integration.test.tsx`)
Tests scene detail page and event listing:
- Scene information display
- Events list for a scene
- Navigation from scene to event details
- Error handling for missing scenes

### 3. Admin Create Scene (`admin-create-scene.integration.test.tsx`)
Tests admin functionality:
- Admin authentication and access control
- Scene creation form
- Form validation
- Success and error handling
- Role-based access restrictions

### 4. Stream Start to Live Listeners (`stream-listeners.integration.test.tsx`)
Tests streaming functionality:
- Stream page access
- Joining a stream
- Participant list display
- Audio controls
- Real-time participant updates
- Disconnection handling
- Settings persistence

### 5. Settings Modifications (`settings-modifications.integration.test.tsx`)
Tests settings page and persistence:
- Theme toggling (light/dark mode)
- Theme persistence across page reloads
- Notification settings
- Privacy settings
- Settings synchronization

### 6. Search to Navigation (`search-navigation.integration.test.tsx`)
Tests search functionality:
- Search interface
- Search results display
- Navigation to scene/event details from results
- Loading states
- Empty results handling
- Error handling
- Result filtering

## Running the Tests

### Run all integration tests
```bash
cd web
npm test -- integration-tests/
```

### Run a specific test file
```bash
npm test -- integration-tests/login-flow.integration.test.tsx
```

### Run tests in watch mode
```bash
npm test -- integration-tests/ --watch
```

### Run tests with coverage
```bash
npm test -- integration-tests/ --coverage
```

### Run tests in UI mode (interactive)
```bash
npm test -- integration-tests/ --ui
```

## Writing Integration Tests

### Basic Structure

```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { setupMockServer } from '../test/mocks/server';

// Setup MSW mock server
setupMockServer();

describe('Integration: Your Flow Name', () => {
  beforeEach(() => {
    // Reset state before each test
    localStorage.clear();
  });

  it('should complete user flow', async () => {
    const user = userEvent.setup();

    // Create router with necessary routes
    const router = createMemoryRouter(
      [
        { path: '/', element: <YourPage /> },
      ],
      {
        initialEntries: ['/'],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    render(<RouterProvider router={router} />);

    // Your test assertions here
    expect(screen.getByText('Expected Text')).toBeInTheDocument();
  });
});
```

### Best Practices

1. **Use MSW handlers** for API mocking (defined in `web/src/test/mocks/handlers.ts`)
2. **Call `setupMockServer()`** at the top of your test file to enable MSW
3. **Reset state** in `beforeEach()` to prevent test pollution
4. **Use `waitFor`** for async assertions
5. **Test user journeys**, not implementation details
6. **Use `userEvent`** for realistic user interactions
7. **Keep tests focused** on one flow per test
8. **Use descriptive test names** that explain the user's goal

### Adding New API Mocks

Edit `web/src/test/mocks/handlers.ts`:

```typescript
const yourHandlers = [
  http.get(`${API_BASE}/your-endpoint`, () => {
    return HttpResponse.json({
      // Your mock response
    });
  }),
];

// Add to exports
export const handlers = [
  ...authHandlers,
  ...yourHandlers, // Add your handlers
];
```

## Test Results

Current status (as of last run):
- **Total tests**: 41
- **Passing**: 29 (71%)
- **Failing**: 12 (reveal API contract issues)
- **Runtime**: ~11 seconds

### Known Issues

Some tests fail due to API contract mismatches between frontend expectations and MSW mock responses:

1. **Login flow**: Error message format differences ("Unauthorized" vs "Invalid credentials")
2. **Login flow**: Response format differences (camelCase vs snake_case)
3. **Stream tests**: Async timing issues to be addressed

These failures are **valuable** - they reveal real integration issues that need to be fixed in either the frontend or backend code.

## Continuous Integration

To run integration tests in CI, add to your GitHub Actions workflow:

```yaml
- name: Run integration tests
  run: |
    cd web
    npm test -- integration-tests/ --run
```

## Troubleshooting

### Tests timeout
Increase the timeout in your test:
```typescript
it('should complete flow', async () => {
  // Test code
}, { timeout: 10000 }); // 10 seconds
```

### MSW warnings about unhandled requests
Add a handler in `handlers.ts` for the endpoint being called.

### Tests fail in CI but pass locally
Ensure you're clearing state in `beforeEach()` and not relying on timing-dependent behavior.

### State pollution between tests
Make sure to reset all stores and clear localStorage in `beforeEach()`:
```typescript
beforeEach(() => {
  authStore.logout();
  localStorage.clear();
  useThemeStore.setState({ theme: 'light' });
});
```

## Resources

- [MSW Documentation](https://mswjs.io/)
- [React Testing Library](https://testing-library.com/react)
- [Vitest Documentation](https://vitest.dev/)
- [User Event Documentation](https://testing-library.com/docs/user-event/intro)
