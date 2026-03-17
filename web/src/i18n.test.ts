/**
 * i18n Configuration Tests
 * Validates internationalization setup and exports
 */

import { describe, it, expect } from 'vitest';
import { SUPPORTED_LANGUAGES, NAMESPACES } from './i18n';

describe('i18n configuration', () => {
  describe('exports', () => {
    it('should export supported languages', () => {
      expect(SUPPORTED_LANGUAGES).toEqual(['en', 'es']);
    });

    it('should export all namespaces', () => {
      expect(NAMESPACES).toEqual(['common', 'scenes', 'events', 'streaming', 'auth']);
    });

    it('should have at least one supported language', () => {
      expect(SUPPORTED_LANGUAGES.length).toBeGreaterThan(0);
    });

    it('should have English as a supported language', () => {
      expect(SUPPORTED_LANGUAGES).toContain('en');
    });

    it('should have Spanish as a supported language', () => {
      expect(SUPPORTED_LANGUAGES).toContain('es');
    });

    it('should have all required namespaces', () => {
      expect(NAMESPACES).toContain('common');
      expect(NAMESPACES).toContain('scenes');
      expect(NAMESPACES).toContain('events');
      expect(NAMESPACES).toContain('streaming');
      expect(NAMESPACES).toContain('auth');
    });
  });
});
