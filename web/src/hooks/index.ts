/**
 * Hook Exports
 * Central export for all custom hooks
 */

// Entity hooks
export { useScene, type UseSceneResult } from './useScene';
export { useEvent, type UseEventResult } from './useEvent';
export {
  useScenes,
  usePublicScenes,
  useUserScenes,
  type UseScenesOptions,
  type UseScenesResult,
} from './useScenes';
export {
  useEvents,
  useSceneEvents,
  useUpcomingEvents,
  type UseEventsOptions,
  type UseEventsResult,
} from './useEvents';

// Map hooks
export { useMapBBox } from './useMapBBox';
export { useClusteredData } from './useClusteredData';

// Streaming hooks
export { useLiveAudio, type UseLiveAudioOptions, type UseLiveAudioResult } from './useLiveAudio';

// Search hooks
export { useSearch, type UseSearchResult, type UseSearchOptions } from './useSearch';
export { useSearchHistory, type UseSearchHistoryResult, type SearchHistoryItem } from './useSearchHistory';

// Keyboard hooks
export { useKeyboardShortcut, type KeyboardShortcutOptions } from './useKeyboardShortcut';

// Telemetry hooks
export { useTelemetry, type EmitFunction } from './useTelemetry';
