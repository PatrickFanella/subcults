# Style Guide

Code conventions for the Subcults Go backend and React/TypeScript frontend.

## Go Conventions

### Package Naming

- Lowercase, singular: `scene`, `auth`, `validate` (not `scenes`, `utils`)
- `internal/` for private application code
- `pkg/` for reusable packages (use sparingly)
- `cmd/` for entry points (minimal logic — flag parsing + initialization only)

### File Naming

- Lowercase with underscores: `scene_handlers.go`, `jwt_test.go`
- Tests in the same package: `scene_handlers_test.go` alongside `scene_handlers.go`
- One primary type per file when practical

### Imports

Group imports in three blocks separated by blank lines:

```go
import (
    // Standard library
    "context"
    "fmt"
    "net/http"

    // Third-party
    "github.com/golang-jwt/jwt/v5"

    // Internal
    "github.com/onnwee/subcults/internal/api"
    "github.com/onnwee/subcults/internal/middleware"
)
```

### Error Handling

Wrap errors with context using `%w`:

```go
return fmt.Errorf("failed to create scene: %w", err)
```

Define sentinel errors for expected conditions:

```go
var ErrSceneNotFound = errors.New("scene not found")
```

Check sentinel errors with `errors.Is`:

```go
if errors.Is(err, scene.ErrSceneNotFound) {
    // handle not found
}
```

Never ignore errors. If a return value is intentionally unused, assign to `_` with a comment:

```go
_ = writer.Close() // best-effort cleanup
```

### Context

- Always pass `context.Context` as the first parameter
- Use private types for context keys to avoid collisions:

```go
type userDIDKey struct{}

func SetUserDID(ctx context.Context, did string) context.Context {
    return context.WithValue(ctx, userDIDKey{}, did)
}
```

### Logging

Use `slog` exclusively. Never `fmt.Println` or `log.Println`:

```go
slog.Info("scene created", "scene_id", s.ID, "user_did", userDID)
slog.Error("payment failed", "error", err, "amount", amount)
```

Key-value pairs must alternate between string keys and values. Use structured fields, not string formatting.

### Interfaces

Define interfaces where they are consumed, not where they are implemented:

```go
// In the handler package, not the repository package
type SceneReader interface {
    GetByID(ctx context.Context, id string) (*scene.Scene, error)
}
```

Keep interfaces small. Prefer single-method interfaces when possible.

### Testing

- Table-driven tests with named subtests
- Descriptive test names: `TestCreateScene_DuplicateName_ReturnsConflict`
- In-memory repositories for unit tests (no real database)
- `httptest.NewRequest` + `httptest.NewRecorder` for handler tests
- Race detector always enabled (`go test -race`)

### Forbidden Patterns

| Pattern | Why | Alternative |
|---------|-----|-------------|
| Global mutable singletons | Race conditions, testing difficulty | Dependency injection |
| `time.Sleep` in tests | Flaky, slow | Channels, sync primitives |
| SQL string concatenation | SQL injection | Parameterized queries (`$1`, `$2`) |
| `fmt.Println` / `log.Println` | Unstructured, no levels | `slog.Info`, `slog.Error` |
| `any` / `interface{}` in APIs | Type safety loss | Concrete types or generics |
| Bare `panic` | Crashes the server | Return errors |
| `init()` functions | Hidden side effects, test ordering | Explicit initialization |

## Frontend Conventions

### TypeScript

- Strict mode enabled (`"strict": true` in tsconfig)
- No `any` — use proper types or `unknown` with type guards
- Discriminated unions for state machines:

```tsx
type StreamState =
  | { status: 'idle' }
  | { status: 'connecting' }
  | { status: 'live'; roomId: string }
  | { status: 'error'; message: string };
```

- Use `interface` for object shapes, `type` for unions and intersections

### Component Structure

One component per file. Colocate tests and styles:

```
SearchBar/
  SearchBar.tsx
  SearchBar.test.tsx
  SearchBar.a11y.test.tsx
```

Or flat file naming for simple components:

```
SearchBar.tsx
SearchBar.test.tsx
```

### Component Patterns

Functional components with hooks only. No class components:

```tsx
export function SceneCard({ scene, onClick }: SceneCardProps) {
  const { t } = useTranslation();

  return (
    <article
      role="button"
      tabIndex={0}
      onClick={() => onClick(scene.id)}
      onKeyDown={(e) => e.key === 'Enter' && onClick(scene.id)}
      aria-label={t('scene.viewDetails', { name: scene.name })}
    >
      <h3>{scene.name}</h3>
      <p>{scene.description}</p>
    </article>
  );
}
```

### Hooks

- Prefix custom hooks with `use`: `useScene`, `useSearch`, `useLiveAudio`
- Keep hooks focused — one responsibility per hook
- Extract complex logic into custom hooks to keep components readable

### State Management

| State Type | Tool | Example |
|-----------|------|---------|
| Component-local | `useState`, `useReducer` | Form inputs, toggles |
| Global app state | Zustand stores | Auth state, theme, settings |
| Server state | React Query or `fetch` + hooks | Scene data, search results |
| URL state | React Router | Current route, search params |

Minimize prop drilling. Use Zustand stores or React context for cross-cutting concerns.

### Styling

Tailwind CSS exclusively. No inline styles, CSS modules, or styled-components:

```tsx
// Correct
<div className="flex items-center gap-4 bg-background text-foreground">

// Wrong
<div style={{ display: 'flex' }}>
```

Follow the design system (`design-system.md`):

- Dark mode via `dark:` prefix (dark-first approach)
- Sharp corners everywhere (no `rounded-*` except avatars/pills which use `rounded-full`)
- No shadows — use text scale, accent bars, and color alternation for depth
- Accent color: vermillion (`#FF3D00`)

### Typography

Per the design system:

| Use | Font | Class |
|-----|------|-------|
| Headlines | Inter Tight | `font-display tracking-tighter` |
| Body text | Inter | `font-body leading-relaxed` |
| Data/metrics | JetBrains Mono | `font-mono tracking-wider` |

### Internationalization

All user-facing text must use i18next:

```tsx
import { useTranslation } from 'react-i18next';

function Component() {
  const { t } = useTranslation();
  return <h1>{t('page.title')}</h1>;
}
```

Never hardcode user-visible strings. Verify with: `npm run check:i18n`

### Accessibility

Every interactive element must be keyboard-navigable and screen-reader-friendly:

```tsx
// Button-like elements need role + keyboard handler
<div
  role="button"
  tabIndex={0}
  onClick={handleClick}
  onKeyDown={(e) => e.key === 'Enter' && handleClick()}
  aria-label="descriptive label"
>
```

Requirements:
- ARIA labels on all interactive elements
- Focus management for modals and dialogs
- Color contrast meeting WCAG AA
- Test with `vitest-axe` in every component test
- Page-level accessibility tests in `.a11y.test.tsx` files

### Import Ordering

```tsx
// React and framework
import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

// Third-party libraries
import maplibregl from 'maplibre-gl';

// Internal - stores, hooks, services
import { useAuthStore } from '@/stores/authStore';
import { useScene } from '@/hooks/useScene';

// Internal - components
import { SearchBar } from '@/components/SearchBar';

// Types
import type { Scene } from '@/types';
```

### Forbidden Frontend Patterns

| Pattern | Why | Alternative |
|---------|-----|-------------|
| `any` type | Type safety loss | Proper types or `unknown` |
| Inline styles | Inconsistent, hard to maintain | Tailwind classes |
| Class components | Legacy pattern | Functional components + hooks |
| Snapshot tests | Brittle, low signal | Behavior tests with Testing Library |
| `console.log` in production | Noise, potential data leak | Telemetry service |
| Direct DOM manipulation | Breaks React reconciliation | Refs when absolutely necessary |
| `// @ts-ignore` | Hides real type errors | Fix the types |
| Hardcoded strings | Can't be translated | i18next `t()` function |

## Linting and Formatting

### Go

```bash
make lint    # go vet
make fmt     # gofmt
```

### Frontend

```bash
cd web
npm run lint   # ESLint
```

Configuration files:
- `.eslintrc.js` — ESLint rules (extends `eslint:recommended`)
- `.prettierrc` — Formatting: semicolons, single quotes, trailing commas, 100 char width, 2-space indent

### Prettier Settings

```json
{
  "semi": true,
  "singleQuote": true,
  "trailingComma": "es5",
  "tabWidth": 2,
  "printWidth": 100
}
```

## Naming Conventions

### Go

| Element | Convention | Example |
|---------|-----------|---------|
| Package | lowercase singular | `scene`, `auth` |
| Exported function | PascalCase | `HandleCreateScene` |
| Unexported function | camelCase | `validateInput` |
| Interface | PascalCase, noun or -er suffix | `SceneRepository`, `Reader` |
| Constants | PascalCase | `DefaultPort`, `MaxRetries` |
| Error variables | `Err` prefix | `ErrSceneNotFound` |
| Context keys | Private struct type | `type userDIDKey struct{}` |

### TypeScript / React

| Element | Convention | Example |
|---------|-----------|---------|
| Component | PascalCase | `SceneCard`, `SearchBar` |
| Hook | camelCase with `use` prefix | `useScene`, `useLiveAudio` |
| Store | camelCase with `Store` suffix | `authStore`, `streamingStore` |
| Type/Interface | PascalCase | `Scene`, `SearchQuery` |
| Constant | SCREAMING_SNAKE_CASE | `MAX_RETRY_COUNT` |
| CSS class | Tailwind utilities | `bg-background text-foreground` |
| Translation key | dot-separated lowercase | `scene.create.title` |
| File (component) | PascalCase | `SceneCard.tsx` |
| File (utility) | camelCase | `formatDate.ts` |
| Test file | Same name + `.test` | `SceneCard.test.tsx` |

## Performance Patterns

### Go

- Use `sync.Pool` for frequently allocated objects
- Prefer slices over maps for small collections
- Profile with `pprof` before optimizing (never guess)
- Set `GOMAXPROCS` appropriately for containers

### Frontend

- Lazy-load routes with `React.lazy()` and `Suspense`
- Memoize expensive computations with `useMemo`
- Avoid re-renders: use `React.memo` for pure components receiving stable props
- Virtualize long lists (don't render 1000+ DOM nodes)
- Keep main bundle under 150KB gzip
