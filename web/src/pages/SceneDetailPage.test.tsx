/**
 * SceneDetailPage Component Tests
 * Tests scene detail display and user interactions
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { SceneDetailPage } from './SceneDetailPage';

describe('SceneDetailPage', () => {
  const renderSceneDetailPage = (sceneId = '123') => {
    const router = createMemoryRouter(
      [
        {
          path: '/scenes/:id',
          element: <SceneDetailPage />,
        },
      ],
      {
        initialEntries: [`/scenes/${sceneId}`],
        future: {
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        },
      }
    );

    return render(<RouterProvider router={router} />);
  };

  describe('Rendering', () => {
    it('should render scene detail heading', () => {
      renderSceneDetailPage();
      expect(screen.getByRole('heading', { name: /Scene Detail/i })).toBeInTheDocument();
    });

    it('should display scene ID from route params', () => {
      renderSceneDetailPage('abc123');
      expect(screen.getByText(/Scene ID: abc123/i)).toBeInTheDocument();
    });

    it('should display placeholder text', () => {
      renderSceneDetailPage();
      expect(screen.getByText(/Scene details will be displayed here/i)).toBeInTheDocument();
    });
  });

  describe('Route Parameters', () => {
    it('should display different IDs based on route', () => {
      renderSceneDetailPage('scene-456');
      expect(screen.getByText(/Scene ID: scene-456/i)).toBeInTheDocument();
    });

    it('should handle numeric IDs', () => {
      renderSceneDetailPage('999');
      expect(screen.getByText(/Scene ID: 999/i)).toBeInTheDocument();
    });

    it('should handle UUID-like IDs', () => {
      const uuid = '550e8400-e29b-41d4-a716-446655440000';
      renderSceneDetailPage(uuid);
      expect(screen.getByText(new RegExp(`Scene ID: ${uuid}`, 'i'))).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper heading hierarchy', () => {
      renderSceneDetailPage();
      
      const heading = screen.getByRole('heading', { name: /Scene Detail/i });
      expect(heading.tagName).toBe('H1');
    });

    it('should have readable text content', () => {
      renderSceneDetailPage('123');
      
      const container = screen.getByText(/Scene ID: 123/i).parentElement;
      expect(container).toBeInTheDocument();
    });
  });

  describe('Layout', () => {
    it('should apply padding to container', () => {
      const { container } = renderSceneDetailPage();
      
      const mainDiv = container.querySelector('div');
      expect(mainDiv).toHaveStyle({ padding: '2rem' });
    });
  });
});
