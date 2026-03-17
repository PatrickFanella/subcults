/**
 * i18n Hooks and Components
 * Convenience exports for internationalization
 */

import { useTranslation, Trans as I18nextTrans } from 'react-i18next';
import type { Namespace } from '../i18n';

/**
 * Custom hook for translations with type safety
 * 
 * @param ns - Optional namespace (defaults to 'common')
 * @returns Translation function and i18n instance
 * 
 * @example
 * ```tsx
 * const { t } = useT('scenes');
 * const createTitle = t('create.title'); // Translation from scenes namespace
 * ```
 */
export function useT(ns?: Namespace) {
  return useTranslation(ns);
}

/**
 * Trans component for complex translations with React components
 * Re-export from react-i18next for convenience
 * 
 * @example
 * ```tsx
 * <Trans i18nKey="app.tagline" ns="common">
 *   Discover <strong>Underground Music</strong> Scenes
 * </Trans>
 * ```
 */
export const Trans = I18nextTrans;

export default useT;
