/**
 * Store Exports
 * Central export for all store-related modules
 */

// Entity Store
export {
  useEntityStore,
  isStale,
  createFreshMetadata,
  setLoadingMetadata,
  setSuccessMetadata,
  setErrorMetadata,
  TTL_CONFIG,
  type CacheMetadata,
  type CachedEntity,
  type SceneState,
  type EventState,
  type UserState,
  type EntityStoreState,
  type SceneActions,
  type EventActions,
  type UserActions,
  type EntityStoreActions,
  type EntityStore,
} from './entityStore';

// Auth Store - User type is the canonical definition
export { authStore, useAuth, type User } from './authStore';

// Toast Store
export {
  useToastStore,
  useToasts,
  type Toast,
  type ToastType,
  type ToastStore,
} from './toastStore';

// Participant Store
export {
  useParticipantStore,
  useParticipants,
  useParticipant,
  useLocalParticipant,
  normalizeIdentity,
  type ParticipantMetadata,
  type CachedParticipant,
  type ParticipantStore,
} from './participantStore';

// Streaming Store
export {
  useStreamingStore,
  useStreamingConnection,
  useStreamingAudio,
  useStreamingActions,
  type StreamingStore,
} from './streamingStore';

// Notification Store
export {
  useNotificationStore,
  useNotificationState,
  useNotificationActions,
  type NotificationPermission,
  type PushSubscriptionData,
  type NotificationStore,
} from './notificationStore';

// Settings Store
export {
  useSettingsStore,
  useTelemetryOptOut,
  useSettingsActions,
  type SettingsStore,
} from './settingsStore';

// Language Store
export {
  useLanguageStore,
  useLanguage,
  useLanguageActions,
  type Language,
  type LanguageStore,
} from './languageStore';

// Theme Store
export {
  useThemeStore,
  useTheme,
  useThemeActions,
  type Theme,
  type ThemeStore,
} from './themeStore';
