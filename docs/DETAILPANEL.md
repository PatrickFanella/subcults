# DetailPanel Component

## Overview

The `DetailPanel` component is a sliding side panel that displays detailed information about scenes and events when markers are clicked on the map. It provides an accessible, keyboard-friendly interface for deeper engagement with spatial content.

## Features

- **Sliding Animation**: Smooth slide-in from the right with fade-in backdrop
- **Accessibility**: Full keyboard navigation, focus trap, ARIA attributes
- **Privacy-First**: Respects location consent flags, never displays precise coordinates without consent
- **Analytics**: Built-in event tracking for panel open/close actions
- **Performance**: Opens within <300ms with caching support

## Usage

```tsx
import { DetailPanel } from './components/DetailPanel';
import type { Scene, Event } from './types/scene';

function MyComponent() {
  const [selectedEntity, setSelectedEntity] = useState<Scene | Event | null>(null);
  const [loading, setLoading] = useState(false);

  const handleClose = () => {
    setSelectedEntity(null);
  };

  const handleAnalytics = (eventName: string, data?: Record<string, unknown>) => {
    console.log('Analytics:', eventName, data);
  };

  return (
    <DetailPanel
      isOpen={selectedEntity !== null}
      onClose={handleClose}
      entity={selectedEntity}
      loading={loading}
      onAnalyticsEvent={handleAnalytics}
    />
  );
}
```

## Props

### `isOpen: boolean`
Controls whether the panel is visible. When `true`, the panel slides in from the right.

### `onClose: () => void`
Callback function called when the panel should close. Triggered by:
- Clicking the close button (×)
- Clicking the backdrop
- Pressing the ESC key

### `entity: Scene | Event | null`
The scene or event to display. Can be `null` when loading or if no entity is selected.

### `loading?: boolean`
Optional flag to indicate the panel is fetching entity details. Shows loading state while `true`.

### `onAnalyticsEvent?: (eventName: string, data?: Record<string, unknown>) => void`
Optional callback for analytics tracking. Called with:
- `'detail_panel_open'` when panel opens
- `'detail_panel_close'` when panel closes

## Accessibility Features

### Keyboard Navigation
- **ESC**: Closes the panel and returns focus to previously focused element
- **TAB/Shift+TAB**: Cycles through focusable elements within panel (focus trap)
- Close button receives focus automatically when panel opens

### ARIA Attributes
- `role="dialog"`: Identifies panel as modal dialog
- `aria-modal="true"`: Indicates modal behavior
- `aria-labelledby`: Links to entity title for screen readers
- `aria-label`: Descriptive label for close button
- `aria-hidden="true"`: Hides backdrop from screen readers

### Focus Management
- Automatically focuses close button when panel opens
- Returns focus to previously focused element when panel closes
- Traps focus within panel while open (prevents focus escaping)

## Privacy Enforcement

The DetailPanel component enforces privacy by:

1. **Never displaying precise coordinates** without explicit consent
2. **Showing privacy notices** indicating when approximate location is used
3. **Respecting `allow_precise` flag** on all entities

```tsx
// Privacy notice displayed based on entity.allow_precise
{entity.allow_precise
  ? "Precise location shared"
  : "Approximate location (privacy preserved)"
}
```

## Displayed Information

### For Scenes
- Scene name
- Description (if available)
- Tags (if available)
- Trust score (placeholder)
- Upcoming events (placeholder)
- Privacy notice

### For Events
- Event name
- Description (if available)
- Associated scene
- Active stream status (placeholder)
- Join stream button (placeholder)
- Privacy notice

## Performance Considerations

### Fast Opening (<300ms target)
- Basic entity info shown immediately from GeoJSON properties
- Full details fetched asynchronously in background
- Uses entity cache to avoid redundant API calls

### Caching Strategy
Entities are cached by `${type}-${id}` key to avoid refetching:
```tsx
const cacheKey = `${type}-${id}`;
if (entityCache.has(cacheKey)) {
  return entityCache.get(cacheKey);
}
```

### Pre-fetching Adjacent Markers
When panel opens, the implementation can pre-fetch adjacent markers for snappy navigation (future enhancement).

## Analytics Events

The component emits analytics events for tracking user engagement:

### `detail_panel_open`
Emitted when panel opens with entity details
```tsx
{
  entity_type: 'scene' | 'event',
  entity_id: string
}
```

### `detail_panel_close`
Emitted when panel closes
```tsx
{
  entity_type: 'scene' | 'event',
  entity_id: string
}
```

## Styling

The component uses inline styles for simplicity but supports CSS classes for customization:

- `.detail-panel-backdrop`: Semi-transparent backdrop overlay
- `.detail-panel`: Main panel container

### Animations
```css
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideInRight {
  from { transform: translateX(100%); }
  to { transform: translateX(0); }
}
```

## Integration with ClusteredMapView

The `ClusteredMapView` component integrates `DetailPanel` with marker click handlers:

```tsx
// Click handler for markers
map.on('click', 'unclustered-scene-point', (e) => {
  const feature = e.features[0];
  handleMarkerClick(feature);
});

// Shows panel with entity details
const handleMarkerClick = (feature) => {
  const basicEntity = createFromGeoJSON(feature);
  setSelectedEntity(basicEntity); // Shows immediately
  fetchEntityDetails(entity.id, entity.type); // Fetches full details
};
```

## Testing

The component includes comprehensive tests covering:

- **Rendering**: Open/close states, entity types
- **Interactions**: Click, ESC key, backdrop click
- **Accessibility**: ARIA attributes, focus management, keyboard navigation
- **Privacy**: Location consent enforcement
- **Analytics**: Event emission tracking

Run tests:
```bash
npm test -- DetailPanel.test.tsx
```

## Future Enhancements

Planned features (currently showing placeholders):

1. **Trust Score Display**: Show computed trust scores for scenes
2. **Upcoming Events**: List upcoming events for scenes
3. **Join Stream Button**: Enable joining active streams for events
4. **View Posts**: Navigate to scene/event posts
5. **Navigation Between Entities**: Arrow keys to navigate adjacent markers
6. **Image Gallery**: Display scene/event photos
7. **Social Actions**: Follow, share, bookmark

## Security Considerations

- **XSS Prevention**: All user-provided content is rendered safely (React auto-escapes)
- **No Injection Risks**: Inline styles prevent CSS injection
- **Privacy-First**: No precise coordinates exposed without explicit consent
- **Analytics Privacy**: Entity IDs only, no PII in analytics events

## Browser Compatibility

Tested and supported in:
- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Mobile browsers (iOS Safari, Chrome Android)

## Related Components

- **ClusteredMapView**: Map component with marker clustering
- **MapView**: Base map component
- **Scene/Event Types**: Type definitions in `types/scene.ts`

## References

- [WAI-ARIA Authoring Practices: Dialog (Modal)](https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/)
- [React Testing Library: Accessibility Queries](https://testing-library.com/docs/queries/about#priority)
- [MapLibre GL JS: Event Handling](https://maplibre.org/maplibre-gl-js/docs/API/classes/Map/)
