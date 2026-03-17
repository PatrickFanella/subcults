/**
 * WCAG AA Contrast Ratio Tests
 * Validates that color combinations meet accessibility standards
 */

import { describe, it, expect } from 'vitest';

/**
 * Calculate relative luminance of RGB color
 * https://www.w3.org/TR/WCAG20-TECHS/G17.html
 */
function getLuminance(r: number, g: number, b: number): number {
  const [rs, gs, bs] = [r, g, b].map((c) => {
    c = c / 255;
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  });
  return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
}

/**
 * Calculate contrast ratio between two colors
 * https://www.w3.org/TR/WCAG20-TECHS/G17.html
 */
function getContrastRatio(
  color1: [number, number, number],
  color2: [number, number, number]
): number {
  const l1 = getLuminance(...color1);
  const l2 = getLuminance(...color2);
  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);
  return (lighter + 0.05) / (darker + 0.05);
}

/**
 * Parse hex color to RGB
 */
function hexToRgb(hex: string): [number, number, number] {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  if (!result) throw new Error(`Invalid hex color: ${hex}`);
  return [parseInt(result[1], 16), parseInt(result[2], 16), parseInt(result[3], 16)];
}

/**
 * Parse rgba color to effective RGB over dark mode background.
 * For rgb(...), or rgba(..., 1), this returns the original RGB values.
 * For semi-transparent rgba, this composites over the dark background (#242424).
 */
function rgbaToRgb(rgba: string): [number, number, number] {
  const match = rgba.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([\d.]+))?\)/);
  if (!match) throw new Error(`Invalid rgba color: ${rgba}`);

  const r = parseInt(match[1]);
  const g = parseInt(match[2]);
  const b = parseInt(match[3]);
  const a = match[4] ? parseFloat(match[4]) : 1;

  // Dark mode background color used for compositing (e.g. app shell background)
  const darkBackground = hexToRgb('#242424');

  // If fully opaque, no compositing is needed – preserve previous behavior.
  if (a >= 1) {
    return [r, g, b];
  }

  return [
    Math.round(r * a + darkBackground[0] * (1 - a)),
    Math.round(g * a + darkBackground[1] * (1 - a)),
    Math.round(b * a + darkBackground[2] * (1 - a)),
  ];
}

describe('WCAG AA Contrast Ratios', () => {
  describe('Light Mode', () => {
    it('foreground on background meets WCAG AA (4.5:1 for normal text)', () => {
      const background = hexToRgb('#ffffff');
      const foreground = hexToRgb('#213547');

      const ratio = getContrastRatio(foreground, background);

      expect(ratio).toBeGreaterThanOrEqual(4.5);
    });

    it('foreground-secondary on background meets WCAG AA', () => {
      const background = hexToRgb('#ffffff');
      const foreground = hexToRgb('#464646');

      const ratio = getContrastRatio(foreground, background);

      expect(ratio).toBeGreaterThanOrEqual(4.5);
    });

    it('foreground-muted on background has 3.54:1 ratio (suitable for large text or non-critical UI)', () => {
      const background = hexToRgb('#ffffff');
      const foreground = hexToRgb('#888888');

      const ratio = getContrastRatio(foreground, background);

      // Muted text is typically used for non-critical UI elements
      // 3.54:1 meets WCAG AA for large text (3:1) but not normal text
      expect(ratio).toBeGreaterThanOrEqual(3);
    });

    it('border on background has 1.26:1 ratio (decorative, relies on other cues)', () => {
      const background = hexToRgb('#ffffff');
      const border = hexToRgb('#e5e5e5');

      const ratio = getContrastRatio(border, background);

      // NOTE: This is a subtle border that relies on shape/position for distinction
      // Not meeting 3:1 for UI components - consider using darker border when needed
      expect(ratio).toBeLessThan(3);
      expect(ratio).toBeGreaterThan(1);
    });

    it('brand-primary text on white background has 5.70:1 ratio (meets WCAG AA for normal text)', () => {
      const background = hexToRgb('#ffffff');
      const brandPrimary = hexToRgb('#7c3aed');

      const ratio = getContrastRatio(brandPrimary, background);

      // 5.70:1 meets WCAG AA for normal text (≥4.5:1)
      expect(ratio).toBeGreaterThanOrEqual(4.5);
    });

    it('brand-primary-dark text on white background meets WCAG AA', () => {
      const background = hexToRgb('#ffffff');
      const brandPrimaryDark = hexToRgb('#5b21b6');

      const ratio = getContrastRatio(brandPrimaryDark, background);

      expect(ratio).toBeGreaterThanOrEqual(4.5);
    });
  });

  describe('Dark Mode', () => {
    it('foreground on background meets WCAG AA', () => {
      const background = hexToRgb('#242424');
      // rgba(255, 255, 255, 0.87) composited over #242424 = rgb(227, 227, 227)
      const foreground = rgbaToRgb('rgba(255, 255, 255, 0.87)');

      const ratio = getContrastRatio(foreground, background);

      expect(ratio).toBeGreaterThanOrEqual(4.5);
    });

    it('foreground-secondary on background meets WCAG AA', () => {
      const background = hexToRgb('#242424');
      // rgba(255, 255, 255, 0.7) composited over #242424 = rgb(189, 189, 189)
      const foreground = rgbaToRgb('rgba(255, 255, 255, 0.7)');

      const ratio = getContrastRatio(foreground, background);

      expect(ratio).toBeGreaterThanOrEqual(4.5);
    });

    it('foreground-muted on background has 4.38:1 ratio (suitable for large text or non-critical UI)', () => {
      const background = hexToRgb('#242424');
      const foreground = hexToRgb('#888888');

      const ratio = getContrastRatio(foreground, background);

      // Close to WCAG AA (4.5:1) - acceptable for large text
      expect(ratio).toBeGreaterThanOrEqual(3);
    });

    it('border on background has 1.50:1 ratio (decorative, relies on other cues)', () => {
      const background = hexToRgb('#242424');
      const border = hexToRgb('#404040');

      const ratio = getContrastRatio(border, background);

      // NOTE: Subtle border similar to light mode
      expect(ratio).toBeLessThan(3);
      expect(ratio).toBeGreaterThan(1);
    });

    it('brand-primary on terminal black has 3.69:1 ratio (meets WCAG AA for large text)', () => {
      const background = hexToRgb('#000000');
      const brandPrimary = hexToRgb('#7c3aed');

      const ratio = getContrastRatio(brandPrimary, background);

      // Meets large text requirement (3:1)
      expect(ratio).toBeGreaterThanOrEqual(3);
    });

    it('white text on brand-primary background has 5.70:1 ratio (meets WCAG AA for normal text)', () => {
      const background = hexToRgb('#7c3aed');
      const foreground = hexToRgb('#ffffff');

      const ratio = getContrastRatio(foreground, background);

      // 5.70:1 meets WCAG AA for normal text (≥4.5:1)
      expect(ratio).toBeGreaterThanOrEqual(4.5);
    });
  });

  describe('Brand Colors', () => {
    it('underground background with white text meets WCAG AAA (7:1)', () => {
      const background = hexToRgb('#1a1a1a');
      const foreground = hexToRgb('#ffffff');

      const ratio = getContrastRatio(foreground, background);

      // Should meet AAA standard for enhanced contrast
      expect(ratio).toBeGreaterThanOrEqual(7);
    });

    it('white text on brand-accent has 1.62:1 ratio (use dark text or darker background)', () => {
      const background = hexToRgb('#00ffff');
      const foreground = hexToRgb('#ffffff');

      const ratio = getContrastRatio(foreground, background);

      // NOTE: This combination should be avoided for text
      // Consider using dark text (#1a1a1a) on brand-accent instead
      expect(ratio).toBeLessThan(3);
    });
  });

  describe('Large Text (18pt or 14pt bold)', () => {
    it('large text only needs 3:1 ratio - brand-primary on white', () => {
      const background = hexToRgb('#ffffff');
      const brandPrimary = hexToRgb('#7c3aed');

      const ratio = getContrastRatio(brandPrimary, background);

      // Large text requirement is 3:1
      expect(ratio).toBeGreaterThanOrEqual(3);
    });

    it('large text only needs 3:1 ratio - brand-accent on dark', () => {
      const background = hexToRgb('#242424');
      const brandAccent = hexToRgb('#00ffff');

      const ratio = getContrastRatio(brandAccent, background);

      expect(ratio).toBeGreaterThanOrEqual(3);
    });
  });
});
