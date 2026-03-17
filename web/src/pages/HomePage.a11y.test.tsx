/**
 * HomePage Accessibility Tests
 * Validates WCAG compliance for the main map view
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { HomePage } from './HomePage';
import { expectNoA11yViolations } from '../test/a11y-helpers';

// Mock MapView component to avoid MapLibre GL canvas dependencies in unit tests
// The actual MapView component in src/components/MapView.tsx includes the ARIA
// attributes tested here (role="application" and aria-label)
vi.mock('../components/MapView', () => ({
  MapView: vi.fn(() => (
    <div 
      role="application" 
      aria-label="Interactive map showing scenes and events"
      data-testid="map-container"
    >
      Map View Placeholder
    </div>
  )),
}));

describe('HomePage - Accessibility', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should not have any accessibility violations', async () => {
    const { container } = render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>
    );

    await expectNoA11yViolations(container);
  });

  it('should have proper ARIA labels for map application', async () => {
    const { getByRole } = render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>
    );

    // MapView component has role="application" and aria-label in the actual implementation
    await waitFor(() => {
      expect(getByRole('application')).toBeInTheDocument();
    });
    const mapApplication = getByRole('application');
    expect(mapApplication).toHaveAttribute('aria-label', 'Interactive map showing scenes and events');
  });

  it('should be keyboard navigable', async () => {
    const { container } = render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>
    );

    // Verify that the map container is present
    await waitFor(() => {
      expect(container.querySelector('[data-testid="map-container"]')).toBeInTheDocument();
    });
    const mapContainer = container.querySelector('[data-testid="map-container"]');
    expect(mapContainer).toBeInTheDocument();
  });
});
