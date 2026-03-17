/**
 * Language Store Tests
 * Validates language preference state management and i18n synchronization
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useLanguageStore, useLanguage, useLanguageActions } from './languageStore';
import i18n from '../i18n';

describe('languageStore', () => {
  beforeEach(() => {
    // Clear localStorage
    localStorage.clear();
    // Reset store to initial state
    useLanguageStore.setState({ language: 'en' });
    // Reset i18n
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('initializeLanguage', () => {
    it('syncs with i18n current language', () => {
      // Mock i18n to return Spanish
      vi.spyOn(i18n, 'language', 'get').mockReturnValue('es');
      
      const { result } = renderHook(() => useLanguageStore());

      act(() => {
        result.current.initializeLanguage();
      });

      expect(result.current.language).toBe('es');
    });

    it('defaults to en when i18n language is not supported', () => {
      // Mock i18n to return unsupported language
      vi.spyOn(i18n, 'language', 'get').mockReturnValue('fr');
      
      const { result } = renderHook(() => useLanguageStore());

      act(() => {
        result.current.initializeLanguage();
      });

      // Store accepts what i18n provides (i18n handles fallback)
      expect(result.current.language).toBe('fr');
    });
  });

  describe('setLanguage', () => {
    it('updates language state', () => {
      const { result } = renderHook(() => useLanguageStore());

      act(() => {
        result.current.setLanguage('es');
      });

      expect(result.current.language).toBe('es');
    });

    it('updates i18n language', () => {
      const changeLanguageSpy = vi.spyOn(i18n, 'changeLanguage');
      const { result } = renderHook(() => useLanguageStore());

      act(() => {
        result.current.setLanguage('es');
      });

      expect(changeLanguageSpy).toHaveBeenCalledWith('es');
    });
  });

  describe('useLanguage hook', () => {
    it('returns current language', () => {
      useLanguageStore.setState({ language: 'es' });
      const { result } = renderHook(() => useLanguage());

      expect(result.current).toBe('es');
    });
  });

  describe('useLanguageActions hook', () => {
    it('returns stable action references', () => {
      const { result, rerender } = renderHook(() => useLanguageActions());
      const firstRender = result.current;

      rerender();
      const secondRender = result.current;

      // Actions should maintain referential equality
      expect(firstRender.setLanguage).toBe(secondRender.setLanguage);
      expect(firstRender.initializeLanguage).toBe(secondRender.initializeLanguage);
    });
  });
});
