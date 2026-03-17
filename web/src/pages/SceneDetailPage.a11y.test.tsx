/**
 * SceneDetailPage Accessibility Tests
 * Validates WCAG compliance for scene detail views
 */

import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { SceneDetailPage } from './SceneDetailPage';
import { expectNoA11yViolations } from '../test/a11y-helpers';

describe('SceneDetailPage - Accessibility', () => {
  it('should not have any accessibility violations', async () => {
    const { container } = render(
      <MemoryRouter initialEntries={['/scene/test-scene-123']}>
        <Routes>
          <Route path="/scene/:id" element={<SceneDetailPage />} />
        </Routes>
      </MemoryRouter>
    );

    await expectNoA11yViolations(container);
  });

  it('should have proper heading hierarchy', () => {
    const { container } = render(
      <MemoryRouter initialEntries={['/scene/test-scene-123']}>
        <Routes>
          <Route path="/scene/:id" element={<SceneDetailPage />} />
        </Routes>
      </MemoryRouter>
    );

    const h1 = container.querySelector('h1');
    expect(h1).toBeInTheDocument();
    expect(h1?.textContent).toBe('Scene Detail');
  });

  it('should display content in a readable format', () => {
    const { getByText } = render(
      <MemoryRouter initialEntries={['/scene/test-scene-123']}>
        <Routes>
          <Route path="/scene/:id" element={<SceneDetailPage />} />
        </Routes>
      </MemoryRouter>
    );

    expect(getByText(/Scene ID:/)).toBeInTheDocument();
    expect(getByText(/test-scene-123/)).toBeInTheDocument();
  });

  it('should have sufficient padding for readability', () => {
    const { container } = render(
      <MemoryRouter initialEntries={['/scene/test-scene-123']}>
        <Routes>
          <Route path="/scene/:id" element={<SceneDetailPage />} />
        </Routes>
      </MemoryRouter>
    );

    const wrapper = container.querySelector('div');
    expect(wrapper).toHaveStyle({ padding: '2rem' });
  });
});
