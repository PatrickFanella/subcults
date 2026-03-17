# State Management Guide

## Overview

The Subcults frontend uses **Zustand** for global state management. The state is organized into domain-specific slices (scenes, events, users) with built-in caching, TTL-based invalidation, and optimistic updates.

## Architecture

### Entity Store

The core state management is provided by the `EntityStore` which combines three slices:

```typescript
// web/src/stores/entityStore.ts
{
  scene: {
    scenes: Record<string, CachedEntity<Scene>>,
    optimisticUpdates: Record<string, Scene>
  },
  event: {
    events: Record<string, CachedEntity<Event>>
  },
  user: {
    users: Record<string, CachedEntity<User>>
  }
}
```

Each cached entity includes:
- `data`: The actual entity data
- `metadata`: Cache metadata (timestamp, loading, error, stale)

### Slices

State is organized into three domain slices:

1. **Scene Slice** (`web/src/stores/slices/sceneSlice.ts`)
   - Scene CRUD operations
   - Optimistic membership updates with rollback
   - Privacy-aware caching

2. **Event Slice** (`web/src/stores/slices/eventSlice.ts`)
   - Event CRUD operations
   - Scene association

3. **User Slice** (`web/src/stores/slices/userSlice.ts`)
   - User profile caching
   - **Privacy-safe**: Only basic profile (DID, role)

## Caching Strategy

### Stale-While-Revalidate

The store implements stale-while-revalidate pattern:

1. **Fresh Data**: Returned immediately from cache
2. **Stale Data**: Returned from cache + background refetch triggered
3. **Missing Data**: Loading state shown + fetch initiated

```typescript
// TTL Configuration
TTL_CONFIG = {
  DEFAULT: 60000,  // 60 seconds
  SHORT: 30000,    // 30 seconds
  LONG: 300000,    // 5 minutes
}
```

### Request Deduplication

Concurrent requests for the same entity are automatically deduplicated:

```typescript
// Multiple components requesting same scene
useScene('scene-1') // Initiates fetch
useScene('scene-1') // Reuses in-flight request
useScene('scene-1') // Reuses in-flight request
// ↪ Only 1 API call made
```

### Manual Invalidation

You can manually mark entries as stale:

```typescript
const { markSceneStale } = useEntityStore()
markSceneStale('scene-1') // Next access triggers refetch
```

## Usage Patterns

### Fetching Individual Entities

Use the `useScene` and `useEvent` hooks for individual entities:

```tsx
function SceneDetail({ sceneId }: { sceneId: string }) {
  const { scene, loading, error, refetch } = useScene(sceneId)

  if (loading) return <Spinner />
  if (error) return <Error message={error} onRetry={refetch} />
  if (!scene) return <NotFound />

  return <SceneDisplay scene={scene} />
}
```

### Fetching Lists

Use `useScenes` and `useEvents` for filtered lists:

```tsx
function MyScenes() {
  const { user } = useAuth()
  const { scenes, activeCount, loading } = useUserScenes(user?.did)

  return (
    <div>
      <h2>My Scenes ({activeCount} active)</h2>
      {scenes.map(scene => (
        <SceneCard key={scene.id} scene={scene} />
      ))}
    </div>
  )
}
```

### Filtering & Derived Data

Hooks provide built-in filtering and derived selectors:

```tsx
// Filter by owner
const { scenes } = useScenes({ filterByOwner: 'user-123' })

// Filter by visibility
const { scenes } = useScenes({ filterByVisibility: 'public' })

// Get public scenes (convenience hook)
const { scenes } = usePublicScenes()

// Filter events by scene
const { events } = useSceneEvents('scene-1')

// Get upcoming events
const { events, upcomingCount } = useUpcomingEvents()
```

## Optimistic Updates

The store supports optimistic updates with automatic rollback on failure.

### Example: Scene Membership Join

```tsx
function JoinSceneButton({ sceneId }: { sceneId: string }) {
  const { user } = useAuth()
  const { 
    optimisticJoinScene, 
    rollbackSceneUpdate,
    commitSceneUpdate 
  } = useEntityStore()

  const handleJoin = async () => {
    if (!user) return

    // Apply optimistic update
    optimisticJoinScene(sceneId, user.did)

    try {
      // Attempt actual API call
      await apiClient.post(`/scenes/${sceneId}/join`)
      
      // Commit on success
      commitSceneUpdate(sceneId)
    } catch (error) {
      // Rollback on failure
      rollbackSceneUpdate(sceneId)
      toast.error('Failed to join scene')
    }
  }

  return <button onClick={handleJoin}>Join Scene</button>
}
```

### How It Works

1. **Backup**: Original state saved to `optimisticUpdates`
2. **Apply**: Optimistic change applied to cache
3. **API Call**: Actual request sent to backend
4. **Commit/Rollback**: 
   - Success → Remove backup
   - Failure → Restore from backup

## Performance Optimization

### Minimal Re-renders

Zustand uses shallow comparison by default. Use selectors to extract only needed data:

```tsx
// ✅ Good - subscribes only to specific scene
const scene = useEntityStore(state => state.scene.scenes['scene-1'])

// ❌ Bad - subscribes to entire store
const { scene } = useEntityStore()
```

### Memoized Selectors

The provided hooks use memoized selectors to prevent unnecessary re-renders:

```tsx
// Memoized filtering
const scenes = useMemo(() => {
  return Object.values(cachedScenes)
    .filter(/* ... */)
    .map(cached => cached.data)
}, [cachedScenes, filterByOwner, filterByVisibility])
```

## API Integration

### Automatic Token Refresh

The store integrates with the API client which handles:
- Automatic access token injection
- Token refresh on 401
- Retry on network errors

```typescript
// Slices use apiClient directly
const scene = await apiClient.get<Scene>(`/scenes/${id}`)
```

### Error Handling

Errors are captured and stored in metadata:

```tsx
const { scene, error } = useScene('scene-1')

if (error) {
  // Display user-friendly error
  return <ErrorMessage message={error} />
}
```

## Privacy & Security

### User Data

Only minimal user data is cached:
- `did`: Decentralized identifier
- `role`: User role ('user' | 'admin')

**Never cache**:
- Email addresses
- Phone numbers
- Personal addresses
- Payment information

### Location Privacy

Scene/event coordinates respect privacy flags:
- `allow_precise: false` → Only jittered coordinates cached
- Backend enforces consent before returning precise coordinates

## Testing

### Unit Tests

Test stores and hooks in isolation:

```typescript
describe('useScene', () => {
  beforeEach(() => {
    // Reset store
    useEntityStore.setState({
      scene: { scenes: {}, optimisticUpdates: {} },
      event: { events: {} },
      user: { users: {} }
    })
  })

  it('fetches scene if not in cache', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(mockScene)
    const { result } = renderHook(() => useScene('scene-1'))
    await waitFor(() => expect(result.current.scene).toEqual(mockScene))
  })
})
```

### Mock Data

Use `setState` to populate store in tests:

```typescript
useEntityStore.setState({
  scene: {
    scenes: {
      'scene-1': {
        data: mockScene,
        metadata: createFreshMetadata()
      }
    },
    optimisticUpdates: {}
  },
  // ...
})
```

## Common Patterns

### Loading States

```tsx
const { scene, loading } = useScene(sceneId)

if (loading && !scene) {
  return <Spinner /> // Initial load
}

if (loading && scene) {
  return (
    <div>
      <SceneDisplay scene={scene} /> {/* Show stale data */}
      <RefreshIndicator /> {/* Show updating */}
    </div>
  )
}
```

### Pagination

For paginated lists, fetch and merge results:

```tsx
function SceneList() {
  const [page, setPage] = useState(1)
  const { scenes } = useScenes()

  useEffect(() => {
    // Fetch next page and merge into store
    fetchScenePage(page).then(newScenes => {
      newScenes.forEach(scene => {
        useEntityStore.getState().setScene(scene)
      })
    })
  }, [page])

  return (
    <>
      {scenes.map(scene => <SceneCard key={scene.id} scene={scene} />)}
      <button onClick={() => setPage(p => p + 1)}>Load More</button>
    </>
  )
}
```

### Prefetching

Prefetch data on hover or route transition:

```tsx
function SceneLink({ sceneId }: { sceneId: string }) {
  const { fetchScene } = useEntityStore()

  const handleMouseEnter = () => {
    // Prefetch on hover
    fetchScene(sceneId).catch(() => {
      // Ignore errors during prefetch
    })
  }

  return (
    <Link 
      to={`/scenes/${sceneId}`}
      onMouseEnter={handleMouseEnter}
    >
      View Scene
    </Link>
  )
}
```

### Cache Warming

Warm cache after login:

```tsx
function App() {
  const { user, isAuthenticated } = useAuth()
  const { fetchScene } = useEntityStore()

  useEffect(() => {
    if (isAuthenticated && user) {
      // Warm cache with user's scenes
      fetchUserScenes(user.did).then(scenes => {
        scenes.forEach(scene => {
          useEntityStore.getState().setScene(scene)
        })
      })
    }
  }, [isAuthenticated, user])

  return <Router />
}
```

## Troubleshooting

### Stale Data Not Refreshing

Check TTL configuration and manual stale marking:

```typescript
// Force refresh
markSceneStale('scene-1')
await fetchScene('scene-1')
```

### Memory Leaks

Remove entities when no longer needed:

```typescript
const { removeScene } = useEntityStore()
removeScene('scene-1') // Free memory
```

### Race Conditions

Use request deduplication (automatic) or implement cancellation:

```typescript
useEffect(() => {
  const controller = new AbortController()
  
  fetchWithAbort(id, controller.signal)
    .catch(err => {
      if (err.name !== 'AbortError') {
        console.error(err)
      }
    })

  return () => controller.abort()
}, [id])
```

## Future Enhancements

- **Offline Support**: IndexedDB persistence
- **Background Sync**: Sync changes when back online
- **Pagination**: Built-in cursor-based pagination
- **Real-time Updates**: WebSocket integration for live data
- **Cache Strategies**: LRU eviction, size limits

## References

- [Zustand Documentation](https://github.com/pmndrs/zustand)
- [Stale-While-Revalidate Pattern](https://web.dev/stale-while-revalidate/)
- [API Client Reference](./API_REFERENCE.md)
- [Privacy Guidelines](./PRIVACY.md)
