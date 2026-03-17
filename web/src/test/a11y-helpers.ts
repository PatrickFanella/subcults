/**
 * Accessibility Testing Helpers
 * Utilities for consistent accessibility testing across the application
 */

import { expect } from 'vitest';
import { axe as axeCore } from 'vitest-axe';
import type { RenderResult } from '@testing-library/react';

/**
 * Standard axe configuration for all accessibility tests
 * Following WCAG 2.1 Level AA compliance
 */
export const axeConfig = {
  rules: {
    // Ensure color contrast meets WCAG AA standards
    'color-contrast': { enabled: true },
    // Ensure landmarks are properly used
    'landmark-one-main': { enabled: true },
    'landmark-unique': { enabled: true },
    // Ensure interactive elements are keyboard accessible
    'focus-order-semantics': { enabled: true },
    // Ensure proper ARIA usage
    'aria-roles': { enabled: true },
    'aria-valid-attr': { enabled: true },
    'aria-valid-attr-value': { enabled: true },
    // Ensure proper heading hierarchy
    'heading-order': { enabled: true },
    // Ensure buttons have accessible names
    'button-name': { enabled: true },
    // Ensure links have accessible names
    'link-name': { enabled: true },
  },
};

/**
 * Run axe accessibility tests on a rendered component
 * @param container - The rendered component container from @testing-library/react
 * @param config - Optional custom axe configuration (merged with defaults)
 * @returns axe results
 */
export async function runAxeTest(
  container: RenderResult['container'],
  config = axeConfig
) {
  const results = await axeCore(container, config);
  return results;
}

/**
 * Common test pattern for checking accessibility violations
 * Use in tests like: await expectNoA11yViolations(container);
 */
export async function expectNoA11yViolations(
  container: RenderResult['container'],
  config = axeConfig
) {
  const results = await runAxeTest(container, config);
  expect(results).toHaveNoViolations();
}
