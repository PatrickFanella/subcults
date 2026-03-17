import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { Outlet } from 'react-router-dom';

const SIMULATED_CHUNK_LOAD_MS = 25;
let isHomeResolved = false;
let homeSuspensePromise: Promise<void> | null = null;

vi.mock('../layouts/AppLayout', () => ({
  AppLayout: () => <Outlet />,
}));

vi.mock('../pages/HomePage', () => {
  return {
    HomePage: () => {
      if (!isHomeResolved) {
        if (!homeSuspensePromise) {
          homeSuspensePromise = new Promise<void>((resolve) => {
            setTimeout(() => {
              isHomeResolved = true;
              resolve();
            }, SIMULATED_CHUNK_LOAD_MS);
          });
        }
        throw homeSuspensePromise;
      }
      return <div>Lazy Home Page</div>;
    },
  };
});

describe('AppRouter lazy loading', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/');
    isHomeResolved = false;
    homeSuspensePromise = null;
  });

  it('shows loading skeleton while loading the home route chunk', async () => {
    const { AppRouter } = await import('./index');
    render(<AppRouter />);

    expect(screen.getByRole('status', { name: 'Loading content' })).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('Lazy Home Page')).toBeInTheDocument();
    });
  });
});
