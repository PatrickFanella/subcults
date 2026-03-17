# Frontend Development Guide

Patterns, conventions, and practical examples for working on the Subcults React frontend.

## Quick Reference

```bash
cd web
npm ci                   # Install dependencies
npm run dev              # Dev server at localhost:5173
npm run build            # Production build
npm run test             # Run unit tests
npm run test:coverage    # Coverage report
npm run lint             # ESLint check
npx tsc --noEmit         # Type check only
```

Or from the repo root:

```bash
make dev-frontend        # Dev server
make test                # All tests (Go + frontend)
```

## Architecture

The frontend is a React 19 + TypeScript SPA built with Vite, using MapLibre for map-based discovery and LiveKit for live audio streaming.

```
web/src/
├── components/          # Reusable UI and feature components
│   ├── ui/              # Primitives (Button, Input, Modal, LoadingSpinner)
│   └── streaming/       # LiveKit audio components
├── guards/              # Route protection (RequireAuth, RequireAdmin)
├── hooks/               # Custom hooks (data fetching, state derivation)
├── layouts/             # Page shells (AppLayout)
├── lib/                 # Service layer (API client, auth, telemetry)
├── pages/               # Route-level page components
├── routes/              # React Router configuration
├── stores/              # Zustand state management
│   └── slices/          # Store slices (scene, event, user)
├── test/                # Test setup, utilities, mocks
├── types/               # TypeScript type definitions
└── utils/               # Pure utility functions
```

## Key Technologies

| Tool                  | Version | Purpose                               |
| --------------------- | ------- | ------------------------------------- |
| React                 | 19      | Component framework                   |
| TypeScript            | 5.9+    | Type safety (strict mode)             |
| Vite                  | 7       | Build tool + dev server               |
| Tailwind CSS          | 3.4     | Utility-first styling                 |
| Zustand               | 5       | Lightweight state management          |
| React Router          | 6       | Client-side routing                   |
| MapLibre GL           | 5       | Map rendering                         |
| LiveKit Client        | 2       | WebRTC audio streaming                |
| i18next               | 25      | Internationalization (en, es, fr, de) |
| Vitest                | 4       | Unit testing                          |
| React Testing Library | 16      | Component testing                     |
| MSW                   | 2       | API mocking in tests                  |
| Playwright            | 1.58    | E2E testing                           |

## Component Patterns

### Creating a Component

Components live in `components/` and follow this structure:

```tsx
// components/SceneCard.tsx
import { useT } from '../hooks/useT';

interface SceneCardProps {
  name: string;
  description: string;
  isActive?: boolean;
  onSelect: (id: string) => void;
}

export function SceneCard({ name, description, isActive = false, onSelect }: SceneCardProps) {
  const { t } = useT('scenes');

  return (
    <article
      className={`border-b border-border p-4 cursor-pointer
        ${isActive ? 'bg-accent/10' : 'hover:bg-muted'}`}
      onClick={() => onSelect(name)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && onSelect(name)}
      aria-label={t('sceneCard.ariaLabel', { name })}
    >
      <h3 className="font-display text-foreground">{name}</h3>
      <p className="text-sm text-muted-foreground">{description}</p>
    </article>
  );
}
```

**Conventions:**

- Named exports (not default)
- Props interface defined above component
- Functional components only
- Tailwind utilities for all styling
- `dark:` prefix for dark mode variants
- ARIA attributes for accessibility
- i18n via `useT()` hook for user-facing text

### UI Primitives

Primitive components in `components/ui/` are reusable building blocks:

```tsx
// components/ui/Button.tsx
interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
}
```

These accept `className` for additional Tailwind customization.

### Streaming Components

LiveKit integration lives in `components/streaming/`:

- `AudioControls.tsx` — Mute/unmute, volume
- `ParticipantList.tsx` — Connected users
- `StreamLatencyOverlay.tsx` — Latency monitoring

## State Management (Zustand)

All global state lives in `stores/`. Stores use a listener-based pattern:

```typescript
// stores/themeStore.ts — simplified Zustand store
import { create } from 'zustand';

interface ThemeState {
  isDark: boolean;
  toggle: () => void;
}

export const useThemeStore = create<ThemeState>((set) => ({
  isDark: true,
  toggle: () => set((state) => ({ isDark: !state.isDark })),
}));
```

**State categories:**

| Store               | Purpose                                   |
| ------------------- | ----------------------------------------- |
| `authStore`         | Authentication state, tokens              |
| `entityStore`       | Cached entities (scenes, events) with TTL |
| `streamingStore`    | Active stream, participants               |
| `settingsStore`     | User preferences                          |
| `themeStore`        | Dark/light mode                           |
| `toastStore`        | Toast notifications                       |
| `notificationStore` | Push notifications                        |

**Caching** — `entityStore` provides base caching:

| TTL     | Duration | Use Case                 |
| ------- | -------- | ------------------------ |
| SHORT   | 30s      | Frequently changing data |
| DEFAULT | 60s      | Standard cache           |
| LONG    | 5 min    | Stable reference data    |

## Custom Hooks

Hooks in `hooks/` separate data fetching from rendering:

```typescript
// hooks/useScenes.ts
export function useScenes(options?: UseScenesOptions): UseScenesResult {
  const cachedScenes = useEntityStore((state) => state.scene.scenes);

  const scenes = useMemo(() => {
    return filterAndSort(cachedScenes, options);
  }, [cachedScenes, options]);

  return { scenes, loading, error };
}
```

**Available hooks:**

| Hook                     | Purpose                   |
| ------------------------ | ------------------------- |
| `useScenes` / `useScene` | Scene data                |
| `useEvents` / `useEvent` | Event data                |
| `useSearch`              | Search with debounce      |
| `useSearchHistory`       | Recent search terms       |
| `useLiveAudio`           | LiveKit audio state       |
| `useMapBBox`             | Map bounding box tracking |
| `useClusteredData`       | MapLibre clustering       |
| `useKeyboardShortcut`    | Keyboard bindings         |
| `useT`                   | i18n translations         |
| `useTelemetry`           | Analytics events          |

## API Integration

The API client in `lib/api-client.ts` handles authentication and retries:

```typescript
// Automatic token refresh on 401
// Exponential backoff retry (3 attempts)
// Request timeout: 10s default

const response = await apiClient.fetch('/api/scenes', {
  method: 'GET',
  timeout: 5000,
});
```

The Vite dev server proxies `/api/*` to `http://localhost:8080` (Go API).

## Styling

All styling uses Tailwind CSS utilities. No custom CSS files for components.

**Dark mode:** Class-based (`dark:` prefix). Toggle via `ThemeProvider`.

```tsx
<div className="bg-background text-foreground dark:bg-gray-950">
  <h1 className="text-foreground dark:text-white">Title</h1>
</div>
```

**Design system colors** use CSS variables:

- `background`, `foreground` — Base colors
- `border`, `muted`, `muted-foreground` — Subtle elements
- `accent` — Vermillion brand accent
- `primary`, `secondary` — Action colors

**Responsive:** Mobile-first with breakpoints `xs: 375px`, `sm: 640px`, `md: 768px`, `lg: 1024px`.

**Touch targets:** Minimum 44px for interactive elements.

## Internationalization (i18n)

All user-facing text must be translated. Translations load lazily via HTTP backend.

**Supported languages:** English (`en`), Spanish (`es`), French (`fr`), German (`de`)

**Namespaces:** `common`, `scenes`, `events`, `streaming`, `auth`

```tsx
import { useTranslation } from 'react-i18next';

function SceneHeader() {
  const { t } = useTranslation('scenes');
  return <h1>{t('header.title')}</h1>;
}
```

Add new translation keys to `web/public/locales/{lang}/{namespace}.json`.

## Testing

### Unit Tests

Tests use Vitest + React Testing Library. Place test files next to the code they test:

```
components/SearchBar.tsx
components/SearchBar.test.tsx
```

```tsx
// components/SearchBar.test.tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SearchBar } from './SearchBar';

describe('SearchBar', () => {
  it('calls onSearch with input value', async () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} />);

    await userEvent.type(screen.getByRole('searchbox'), 'jazz');
    await userEvent.keyboard('{Enter}');

    expect(onSearch).toHaveBeenCalledWith('jazz');
  });
});
```

**Conventions:**

- Use `screen` queries over destructured `getBy*`
- Prefer `userEvent` over `fireEvent` for realistic interactions
- Test behavior, not implementation
- Use MSW handlers (`test/mocks/`) for API calls
- Include accessibility checks with `vitest-axe`

### Running Tests

```bash
npm run test               # Watch mode
npm run test -- --run      # Single run
npm run test:coverage      # With coverage
```

### Coverage Targets

| Scope            | Target |
| ---------------- | ------ |
| Overall frontend | >70%   |
| Components       | >70%   |
| Hooks            | >80%   |
| Stores           | >80%   |

### E2E Tests

Playwright tests live in `e2e/`. Run with:

```bash
make test-e2e
```

## Accessibility

Every component must meet WCAG 2.1 AA. Check:

- [ ] Keyboard navigation (Tab, Enter, Escape)
- [ ] ARIA labels on interactive elements
- [ ] Color contrast (4.5:1 text, 3:1 large text)
- [ ] Focus indicators visible
- [ ] Screen reader text for icon-only buttons

See [A11Y_CHECKLIST.md](A11Y_CHECKLIST.md) for the full checklist.

## Performance

**Build-time optimizations:**

- Code splitting: `vendor-react`, `vendor-router`, `vendor-i18n` chunks
- Tree shaking via Vite/Rollup
- Bundle analysis: `npm run build` generates `stats.html`

**Runtime budgets:**

- FCP: <1.0s
- Map render: <1.2s
- Bundle size monitored via `rollup-plugin-visualizer`

**Lazy loading:** Use `React.lazy()` for route-level code splitting:

```tsx
const SceneDetailPage = React.lazy(() => import('./pages/SceneDetailPage'));
```

## Adding a New Feature

1. **Types** — Define types in `types/` if needed
2. **Store** — Add state slice if global state is needed
3. **Hook** — Create a custom hook in `hooks/` for data logic
4. **Component** — Build the UI component in `components/`
5. **Page** — Create a page in `pages/` if it's a new route
6. **Route** — Register in `routes/index.tsx`
7. **i18n** — Add translation keys to all locale files
8. **Tests** — Write unit tests alongside each file
9. **Accessibility** — Verify keyboard nav + screen reader

## Related Docs

- [Design System](../design-system.md) — Color specs, typography, component styles
- [Accessibility Checklist](A11Y_CHECKLIST.md) — WCAG 2.1 AA requirements
- [Testing Guide](TESTING_GUIDE.md) — Full testing strategy
- [STYLE_GUIDE.md](../STYLE_GUIDE.md) — Code conventions
