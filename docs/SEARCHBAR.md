# SearchBar Component

## Overview

The `SearchBar` component provides a unified global search interface for discovering scenes, events, and posts across the platform. It features debounced typeahead suggestions, keyboard navigation, and full ARIA accessibility compliance.

## Features

- **Unified Search**: Searches across scenes, events, and posts in parallel
- **Typeahead Suggestions**: Displays grouped results as you type
- **Debounced Queries**: 300ms debounce to reduce API load
- **Request Cancellation**: Automatically cancels in-flight requests when new queries are typed
- **Keyboard Navigation**: Full keyboard support (↑/↓/Enter/Esc)
- **ARIA Compliance**: Follows ARIA combobox pattern for screen readers
- **Loading States**: Visual feedback during search
- **Empty States**: Helpful messaging when no results found
- **Privacy-First**: Does not expose private scenes/events in suggestions

## Usage

### Basic Usage

```tsx
import { SearchBar } from '../components/SearchBar';

function MyComponent() {
  return (
    <div>
      <SearchBar />
    </div>
  );
}
```

### With Custom Placeholder

```tsx
<SearchBar placeholder="Find underground music..." />
```

### With Selection Callback

```tsx
<SearchBar 
  onSelect={(item) => {
    console.log('Selected:', item.type, item.data);
  }}
/>
```

### Auto-focus on Mount

```tsx
<SearchBar autoFocus />
```

### With Custom Styling

```tsx
<SearchBar className="max-w-lg mx-auto" />
```

## API

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `placeholder` | `string` | `"Search scenes, events, posts..."` | Placeholder text for input |
| `className` | `string` | `""` | Additional CSS classes |
| `onSelect` | `(item: SearchResultItem) => void` | `undefined` | Callback when result is selected |
| `autoFocus` | `boolean` | `false` | Auto-focus input on mount |

### SearchResultItem Type

```ts
type SearchResultItem = 
  | { type: 'scene'; data: SceneSearchResult }
  | { type: 'event'; data: EventSearchResult }
  | { type: 'post'; data: PostSearchResult };
```

## Keyboard Navigation

| Key | Action |
|-----|--------|
| `↓` (Arrow Down) | Move selection down |
| `↑` (Arrow Up) | Move selection up |
| `Enter` | Navigate to selected result |
| `Escape` | Close dropdown |

## Accessibility

The component follows the [ARIA combobox pattern](https://www.w3.org/WAI/ARIA/apg/patterns/combobox/) and includes:

- `role="combobox"` on input
- `role="listbox"` on results container
- `role="option"` on result items
- `aria-expanded` state
- `aria-controls` linking input to results
- `aria-autocomplete="list"`
- `aria-activedescendant` for keyboard selection
- `aria-selected` on selected items

Screen reader announcements:
- Dropdown state changes
- Selected item updates
- Result counts
- Error states

## Performance

### Debouncing

The component debounces search queries with a 300ms delay to reduce API load. This can be customized via the `useSearch` hook:

```tsx
const { results, loading, search } = useSearch({ debounceMs: 500 });
```

### Request Cancellation

When a new query is typed before the previous search completes, the in-flight request is automatically cancelled using `AbortController`. This prevents:
- Wasted network bandwidth
- Race conditions where stale results overwrite fresh ones
- Unnecessary API load

### Result Limits

By default, each category (scenes, events, posts) returns up to 5 results for a total of 15 possible results. This can be customized:

```tsx
const { results } = useSearch({ limit: 10 });
```

## States

### Loading State

Displayed while search is in progress:
- Animated spinner
- "Searching..." message

### Empty State

Displayed when no results match the query:
- "No results found for '{query}'" message

### Error State

Displayed when API request fails:
- Error message with details

## Extension Hooks

### Custom Search Logic

The `useSearch` hook can be used independently for custom search implementations:

```tsx
import { useSearch } from '../hooks/useSearch';

function CustomSearch() {
  const { results, loading, search, clear } = useSearch({
    debounceMs: 500,
    limit: 10,
  });

  // Custom implementation
}
```

### Custom Result Rendering

Override the default result rendering by forking the component and customizing:

- `getIcon()` - Change result type icons
- `getDisplayName()` - Change primary text
- `getSecondaryInfo()` - Change secondary text

## Integration

### App Shell

Add to the app shell for global access:

```tsx
// In App.tsx or Layout.tsx
import { SearchBar } from './components/SearchBar';

function AppShell() {
  return (
    <header>
      <nav>
        <SearchBar className="flex-1 max-w-xl" />
      </nav>
    </header>
  );
}
```

### Mobile Considerations

For mobile layouts, consider:

```tsx
<div className="hidden md:block">
  <SearchBar />
</div>

{/* Mobile: Full-screen modal */}
<button 
  className="md:hidden"
  onClick={() => setMobileSearchOpen(true)}
>
  🔍
</button>
```

## Testing

The component includes comprehensive tests covering:

- ✅ Rendering and initial state
- ✅ ARIA attribute compliance
- ✅ Debounce behavior
- ✅ Parallel search execution
- ✅ Keyboard navigation (↑/↓/Enter/Esc)
- ✅ Request cancellation
- ✅ Loading/empty/error states
- ✅ Clear functionality
- ✅ Outside click handling
- ✅ Result selection

Run tests:

```bash
npm test SearchBar
```

## Dependencies

The search functionality requires the following API endpoints:

- `GET /search/scenes?q={query}&limit={limit}`
- `GET /search/events?q={query}&limit={limit}`
- `GET /search/posts?q={query}&limit={limit}`

**Note**: As of the initial implementation, these endpoints may not all be available yet. The component gracefully handles missing endpoints by showing empty results for unavailable categories.

## Security & Privacy

- **Private Content**: The search respects visibility settings and does not expose private scenes/events
- **Input Sanitization**: All query strings are URL-encoded before sending to API
- **XSS Prevention**: All user-generated content is rendered through React's built-in escaping

## Performance Targets

- ✅ Typing shows suggestions within <500ms post-debounce
- ✅ Keyboard navigation responds instantly
- ✅ No memory leaks from cancelled requests
- ✅ Dropdown render time <50ms

## Known Limitations

1. **Endpoint Availability**: Depends on `/search/scenes`, `/search/events`, and `/search/posts` endpoints (tracked in issues #98-#100)
2. **Result Ranking**: Current implementation returns results in API order; trust-based ranking not yet implemented
3. **Autocomplete**: No query autocomplete/suggestions beyond result names
4. **History**: No search history persistence

## Future Enhancements

- [ ] Search history with localStorage
- [ ] Recent searches section
- [ ] Popular searches
- [ ] Filters (date range, location, tags)
- [ ] Advanced query syntax (quotes, boolean operators)
- [ ] Trust-based result ranking
- [ ] Keyboard shortcuts (Cmd+K to focus)
- [ ] Mobile-optimized full-screen modal

## Related Components

- `useSearch` hook - Core search logic
- `LoadingSkeleton` - Loading state patterns
- `ErrorBoundary` - Error handling

## Related Issues

- #21 - Frontend App Shell (parent epic)
- #98 - Scene search endpoint
- #99 - Event search endpoint  
- #100 - Post search endpoint

## License

See repository LICENSE file.
