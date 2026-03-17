/**
 * i18n Configuration
 * Internationalization setup using i18next with lazy loading and namespaces
 */

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import HttpBackend from 'i18next-http-backend';
import LanguageDetector from 'i18next-browser-languagedetector';

// Supported languages
export const SUPPORTED_LANGUAGES = ['en', 'es', 'fr', 'de'] as const;
export type SupportedLanguage = typeof SUPPORTED_LANGUAGES[number];

// Namespaces for organizing translations
export const NAMESPACES = ['common', 'scenes', 'events', 'streaming', 'auth'] as const;
export type Namespace = typeof NAMESPACES[number];

i18n
  // Load translations using HTTP backend (lazy loading)
  .use(HttpBackend)
  // Detect user language
  .use(LanguageDetector)
  // Pass the i18n instance to react-i18next
  .use(initReactI18next)
  // Initialize i18next
  .init({
    // Fallback language
    fallbackLng: 'en',
    
    // Supported languages
    supportedLngs: SUPPORTED_LANGUAGES,
    
    // Default namespace
    defaultNS: 'common',
    
    // Load all namespaces
    ns: NAMESPACES,
    
    // Language detection options
    detection: {
      // Detection order: localStorage > navigator > htmlTag
      order: ['localStorage', 'navigator', 'htmlTag'],
      // Cache user language preference
      caches: ['localStorage'],
      // localStorage key
      lookupLocalStorage: 'subcults-language',
    },
    
    // Backend options for loading translations
    backend: {
      // Path to translation files
      loadPath: '/locales/{{lng}}/{{ns}}.json',
    },
    
    // React options
    react: {
      // Disable React Suspense because the app is not wrapped in a Suspense boundary.
      // This prevents runtime crashes when translations are loaded asynchronously.
      useSuspense: false,
    },
    
    // Interpolation options
    interpolation: {
      // React already escapes values
      escapeValue: false,
    },
    
    // Development mode settings
    debug: import.meta.env.DEV,
    
    // Show missing keys in development
    saveMissing: import.meta.env.DEV,
    
    // Missing key handler (warn in dev, silent in prod)
    missingKeyHandler: (lng, ns, key) => {
      if (import.meta.env.DEV) {
        console.warn(`[i18n] Missing translation key: ${ns}:${key} (${lng})`);
      }
    },
  });

export default i18n;
