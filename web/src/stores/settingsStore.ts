/**
 * Settings Store
 * User preferences and privacy settings with localStorage persistence
 */

import { create } from 'zustand';

interface SettingsState {
  /** Opt-out of telemetry collection */
  telemetryOptOut: boolean;
  /** Opt-in to session replay recording (default: false) */
  sessionReplayOptIn: boolean;
}

interface SettingsActions {
  setTelemetryOptOut: (optOut: boolean) => void;
  setSessionReplayOptIn: (optIn: boolean) => void;
  initializeSettings: () => void;
}

export type SettingsStore = SettingsState & SettingsActions;

/**
 * Local storage key for settings
 */
const SETTINGS_STORAGE_KEY = 'subcults-settings';

/**
 * Default settings
 * PRIVACY-FIRST: Telemetry is OPT-IN by default (telemetryOptOut: true)
 */
const DEFAULT_SETTINGS: SettingsState = {
  telemetryOptOut: true, // Default: telemetry OFF (user must opt-in)
  sessionReplayOptIn: false, // Default OFF for privacy
};

/**
 * Load settings from localStorage
 */
function loadSettings(): SettingsState {
  try {
    const stored = localStorage.getItem(SETTINGS_STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored);
      return {
        telemetryOptOut: parsed.telemetryOptOut ?? DEFAULT_SETTINGS.telemetryOptOut,
        sessionReplayOptIn: parsed.sessionReplayOptIn ?? DEFAULT_SETTINGS.sessionReplayOptIn,
      };
    }
  } catch (error) {
    console.warn('[settingsStore] Failed to load settings from localStorage:', error);
  }
  return DEFAULT_SETTINGS;
}

/**
 * Save settings to localStorage
 */
function saveSettings(settings: SettingsState): void {
  try {
    localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
  } catch (error) {
    console.warn('[settingsStore] Failed to save settings to localStorage:', error);
  }
}

/**
 * Settings store with localStorage persistence
 * 
 * IMPORTANT: Call initializeSettings() on app startup to load persisted preferences.
 * Without initialization, user opt-out preferences will not be respected until
 * they manually change settings in the current session.
 * 
 * Example:
 * ```typescript
 * // In App.tsx or root component:
 * useEffect(() => {
 *   useSettingsStore.getState().initializeSettings();
 * }, []);
 * ```
 */
export const useSettingsStore = create<SettingsStore>((set, get) => ({
  ...DEFAULT_SETTINGS,

  setTelemetryOptOut: (optOut: boolean) => {
    set({ telemetryOptOut: optOut });
    saveSettings(get());
  },

  setSessionReplayOptIn: (optIn: boolean) => {
    set({ sessionReplayOptIn: optIn });
    saveSettings(get());
  },

  initializeSettings: () => {
    const loaded = loadSettings();
    set(loaded);
  },
}));

/**
 * Hook for telemetry opt-out flag only (optimized for re-renders)
 */
export function useTelemetryOptOut(): boolean {
  return useSettingsStore((state) => state.telemetryOptOut);
}

/**
 * Hook for session replay opt-in flag only (optimized for re-renders)
 */
export function useSessionReplayOptIn(): boolean {
  return useSettingsStore((state) => state.sessionReplayOptIn);
}

/**
 * Hook for settings actions only (stable reference)
 */
export function useSettingsActions() {
  const setTelemetryOptOut = useSettingsStore((state) => state.setTelemetryOptOut);
  const setSessionReplayOptIn = useSettingsStore((state) => state.setSessionReplayOptIn);
  const initializeSettings = useSettingsStore((state) => state.initializeSettings);

  return {
    setTelemetryOptOut,
    setSessionReplayOptIn,
    initializeSettings,
  };
}
