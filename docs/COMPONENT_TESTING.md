# Frontend Component Testing Guide

This guide documents the testing approach and patterns used for React component testing in the Subcults project.

## Testing Philosophy

We follow a user-centric testing approach using React Testing Library. Our tests focus on:

1. **User behavior** - Test what users see and do, not implementation details
2. **Accessibility** - Verify ARIA attributes, keyboard navigation, and screen reader compatibility
3. **Real-world interactions** - Use user-event for realistic user interactions
4. **Component behavior** - Test how components respond to user actions

## Testing Stack

- **Vitest** - Fast unit test framework with native ESM support
- **React Testing Library** - User-centric component testing
- **@testing-library/user-event** - Realistic user interaction simulation
- **@testing-library/jest-dom** - Custom matchers for DOM assertions
- **vitest-axe** - Automated accessibility testing

## Test Coverage Requirements

- **Frontend**: >70% coverage (currently at 82.24%)
- **Backend**: >80% coverage

Run coverage reports:
```bash
cd web && npm run test:coverage
```

## Component Testing Patterns

### 1. Query Strategy - Prefer Accessible Queries

Use queries in this priority order:

```typescript
// ✅ Best - Accessible to everyone
screen.getByRole('button', { name: /login/i })
screen.getByLabelText('Email address')
screen.getByPlaceholderText('Search...')

// ✅ Good - Semantic queries
screen.getByText('Welcome')
screen.getByDisplayValue('John')

// ⚠️ Use sparingly - Implementation details
screen.getByTestId('custom-element')
```

**Why?** Queries that resemble how users interact with your app are more resilient to changes.

### 2. User Event Simulation

Always use `@testing-library/user-event` instead of `fireEvent`:

```typescript
import userEvent from '@testing-library/user-event';

// ✅ Realistic user interaction
const user = userEvent.setup();
await user.click(button);
await user.type(input, 'Hello');
await user.keyboard('{Enter}');

// ❌ Don't use fireEvent directly
fireEvent.click(button); // Less realistic
```

### 3. Async Operations with waitFor

Use `waitFor` for async state updates:

```typescript
import { waitFor } from '@testing-library/react';

// Wait for element to appear
await waitFor(() => {
  expect(screen.getByText('Loading complete')).toBeInTheDocument();
});

// Wait for element to disappear
await waitFor(() => {
  expect(screen.queryByText('Loading...')).not.toBeInTheDocument();
});
```

### 4. Accessibility Testing

Every interactive component should verify:

#### ARIA Attributes
```typescript
it('should have proper ARIA attributes', () => {
  render(<SearchBar />);
  const input = screen.getByRole('combobox');
  
  expect(input).toHaveAttribute('aria-expanded', 'false');
  expect(input).toHaveAttribute('aria-controls', 'search-results');
  expect(input).toHaveAttribute('aria-autocomplete', 'list');
});
```

#### Keyboard Navigation
```typescript
it('should support keyboard navigation', async () => {
  const user = userEvent.setup();
  render(<Component />);
  
  await user.tab(); // Navigate to first element
  expect(screen.getByRole('button')).toHaveFocus();
  
  await user.keyboard('{Enter}'); // Activate
  await user.keyboard('{Escape}'); // Close
});
```

#### Screen Reader Support
```typescript
it('should announce loading state', () => {
  render(<Component loading={true} />);
  
  const status = screen.getByRole('status');
  expect(status).toHaveAttribute('aria-live', 'polite');
});
```

### 5. Mock External Dependencies

#### Configuring Stores (Zustand)
```typescript
import { beforeEach } from 'vitest';
import { useThemeStore } from '../stores/themeStore';

// Reset theme store to a known state before each test
beforeEach(() => {
  // Clear any persisted theme state
  localStorage.clear();
  
  // Set initial state using setState
  useThemeStore.setState({ theme: 'light' });
  
  // Clean up DOM if the store modifies it
  document.documentElement.classList.remove('dark');
});
```

**Note:** For custom stores like `authStore` that don't use Zustand, call the store's methods directly:
```typescript
import { authStore } from '../stores/authStore';

beforeEach(() => {
  // Reset to logged out state
  authStore.logout();
});
```

#### Mocking API Calls
```typescript
vi.mock('../lib/api-client', () => ({
  apiClient: {
    searchScenes: vi.fn().mockResolvedValue([]),
    searchEvents: vi.fn().mockResolvedValue([]),
  },
}));
```

#### Mocking React Router
```typescript
import { vi } from 'vitest';

// Mock useNavigate to test navigation
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// In your test
it('should navigate on button click', async () => {
  const user = userEvent.setup();
  render(<Component />);
  
  await user.click(screen.getByRole('button'));
  expect(mockNavigate).toHaveBeenCalledWith('/expected-route');
});
```

**Alternative:** Use `createMemoryRouter` for integration testing with real routing:
```typescript
import { createMemoryRouter, RouterProvider } from 'react-router-dom';

const renderWithRouter = (component, initialRoute = '/') => {
  const router = createMemoryRouter(
    [{ path: '/', element: component }],
    { 
      initialEntries: [initialRoute],
      future: {
        v7_startTransition: true,
        v7_relativeSplatPath: true,
      },
    }
  );
  return render(<RouterProvider router={router} />);
};
```

### 6. Testing Form Interactions

```typescript
describe('LoginPage', () => {
  it('should submit form with user input', async () => {
    const user = userEvent.setup();
    const mockLogin = vi.fn();
    
    render(<LoginPage onLogin={mockLogin} />);
    
    // Fill in form
    await user.type(screen.getByLabelText(/email/i), 'user@example.com');
    await user.type(screen.getByLabelText(/password/i), 'password123');
    
    // Submit
    await user.click(screen.getByRole('button', { name: /login/i }));
    
    // Verify
    expect(mockLogin).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'password123',
    });
  });
});
```

### 7. Testing Conditional Rendering

```typescript
it('should show error state', () => {
  render(<Component error="Something went wrong" />);
  
  expect(screen.getByRole('alert')).toHaveTextContent('Something went wrong');
});

it('should show loading state', () => {
  render(<Component loading={true} />);
  
  expect(screen.getByText(/loading/i)).toBeInTheDocument();
});

it('should show empty state', () => {
  render(<Component data={[]} />);
  
  expect(screen.getByText(/no results/i)).toBeInTheDocument();
});
```

### 8. Testing Component Behavior

```typescript
describe('MiniPlayer', () => {
  it('should not render when disconnected', () => {
    setupMocks({ isConnected: false });
    const { container } = render(<MiniPlayer />);
    
    expect(container.firstChild).toBeNull();
  });
  
  it('should toggle mute on button click', async () => {
    const user = userEvent.setup();
    const mockToggleMute = vi.fn();
    setupMocks({ toggleMute: mockToggleMute });
    
    render(<MiniPlayer />);
    await user.click(screen.getByLabelText(/mute/i));
    
    expect(mockToggleMute).toHaveBeenCalled();
  });
});
```

## Test Organization

### File Structure
```
src/
├── components/
│   ├── SearchBar.tsx
│   ├── SearchBar.test.tsx          # Component tests
│   └── MiniPlayer.tsx
├── pages/
│   ├── LoginPage.tsx
│   ├── LoginPage.test.tsx          # Page tests
│   └── HomePage.a11y.test.tsx      # Accessibility-specific tests
└── test/
    ├── setup.ts                    # Global test setup
    └── mocks/                      # Shared mocks
```

### Test Structure
```typescript
describe('ComponentName', () => {
  // Setup and teardown
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render with default props', () => {});
    it('should render with custom props', () => {});
  });

  describe('User Interactions', () => {
    it('should handle click events', async () => {});
    it('should handle keyboard events', async () => {});
  });

  describe('Accessibility', () => {
    it('should have proper ARIA attributes', () => {});
    it('should support keyboard navigation', async () => {});
  });

  describe('Edge Cases', () => {
    it('should handle empty state', () => {});
    it('should handle error state', () => {});
  });
});
```

## Common Testing Scenarios

### Testing SearchBar with Debounce
```typescript
it('should debounce search requests', async () => {
  const user = userEvent.setup({ delay: null }); // Fast typing
  const mockSearch = vi.fn();
  
  render(<SearchBar onSearch={mockSearch} />);
  
  await user.type(screen.getByRole('combobox'), 'test');
  
  // Should not call immediately
  expect(mockSearch).not.toHaveBeenCalled();
  
  // Should call after debounce
  await waitFor(() => {
    expect(mockSearch).toHaveBeenCalledWith('test');
  }, { timeout: 500 });
});
```

### Testing Dropdown Navigation
```typescript
it('should navigate dropdown with arrow keys', async () => {
  const user = userEvent.setup();
  render(<Dropdown items={['Option 1', 'Option 2']} />);
  
  await user.click(screen.getByRole('button'));
  await user.keyboard('{ArrowDown}');
  
  const options = screen.getAllByRole('option');
  expect(options[0]).toHaveAttribute('aria-selected', 'true');
});
```

### Testing Modal Dialogs
```typescript
it('should close modal on Escape key', async () => {
  const user = userEvent.setup();
  const mockClose = vi.fn();
  
  render(<Modal open={true} onClose={mockClose} />);
  
  await user.keyboard('{Escape}');
  
  expect(mockClose).toHaveBeenCalled();
});
```

## Performance Testing

While component tests focus on functionality, consider:

- **Memoization**: Verify components don't re-render unnecessarily
- **Lazy Loading**: Test components render correctly when loaded
- **Large Lists**: Ensure virtualization works with test data

## Debugging Tests

### Run specific tests
```bash
npm test -- LoginPage.test.tsx
npm test -- --grep "should handle click"
```

### Interactive UI mode
```bash
npm run test:ui
```

### Debug with console logs
```typescript
import { screen } from '@testing-library/react';

// View current DOM
screen.debug();

// View specific element
screen.debug(screen.getByRole('button'));
```

### View available queries
```typescript
import { logRoles } from '@testing-library/react';

const { container } = render(<Component />);
logRoles(container); // Shows all roles in component
```

## Best Practices

### ✅ DO
- Test user behavior, not implementation
- Use accessible queries (role, label, text)
- Test keyboard navigation and ARIA attributes
- Mock external dependencies consistently
- Write descriptive test names
- Group related tests with `describe` blocks
- Clean up after tests (clearAllMocks, unmount)

### ❌ DON'T
- Test implementation details (state, props)
- Use test IDs unless necessary
- Make tests dependent on each other
- Test third-party libraries
- Skip accessibility tests
- Use `waitFor` with hardcoded delays

## Resources

- [React Testing Library Docs](https://testing-library.com/react)
- [User Event API](https://testing-library.com/docs/user-event/intro)
- [Vitest Documentation](https://vitest.dev/)
- [Testing Accessibility](https://testing-library.com/docs/queries/about/#priority)
- [Common Testing Patterns](https://kentcdodds.com/blog/common-mistakes-with-react-testing-library)

## Component Test Examples

See these files for comprehensive examples:

- **SearchBar**: `src/components/SearchBar.test.tsx` - Keyboard nav, debounce, results, API mocking
- **DarkModeToggle**: `src/components/DarkModeToggle.test.tsx` - Zustand store integration, theme toggling
- **MiniPlayer**: `src/components/MiniPlayer.test.tsx` - Streaming controls, volume
- **MapView**: `src/components/MapView.test.tsx` - Map interaction, geolocation

## Continuous Integration

Tests run automatically in GitHub Actions on:
- Every pull request
- Push to main/develop
- Manual workflow dispatch

Ensure all tests pass before merging:
```bash
npm test -- --run
```

---

**Questions?** Check [TESTING.md](../../TESTING.md) or ask in team discussions.
