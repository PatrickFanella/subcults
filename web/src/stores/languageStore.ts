/**
 * Language Store
 * User language preference with localStorage persistence
 */

import { create } from 'zustand';
import i18n, { SUPPORTED_LANGUAGES } from '../i18n';

export type Language = (typeof SUPPORTED_LANGUAGES)[number];

interface LanguageState {
  /** Current language */
  language: Language;
}

interface LanguageActions {
  /** Change the current language */
  setLanguage: (language: Language) => void;
  /** Initialize language from i18n (called on app startup) */
  initializeLanguage: () => void;
}

export type LanguageStore = LanguageState & LanguageActions;

/**
 * Language store with localStorage persistence
 * 
 * IMPORTANT: Call initializeLanguage() on app startup to sync with i18n's
 * detected language.
 * 
 * Example:
 * ```typescript
 * // In App.tsx or root component:
 * useEffect(() => {
 *   useLanguageStore.getState().initializeLanguage();
 * }, []);
 * ```
 */
export const useLanguageStore = create<LanguageStore>((set) => ({
  language: 'en',

  setLanguage: (language: Language) => {
    set({ language });
    // Update i18n (will also persist to localStorage via i18next-browser-languagedetector)
    i18n.changeLanguage(language);
  },

  initializeLanguage: () => {
    // Sync with i18n's detected language (i18next LanguageDetector handles detection)
    const currentLang = i18n.language as Language;
    set({ language: currentLang });
  },
}));

/**
 * Hook for current language only (optimized for re-renders)
 */
export function useLanguage(): Language {
  return useLanguageStore((state) => state.language);
}

/**
 * Hook for language actions only (stable reference)
 */
export function useLanguageActions() {
  const setLanguage = useLanguageStore((state) => state.setLanguage);
  const initializeLanguage = useLanguageStore((state) => state.initializeLanguage);

  return {
    setLanguage,
    initializeLanguage,
  };
}
