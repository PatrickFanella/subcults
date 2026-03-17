/**
 * SearchResultsPage Tests
 * Validates search results display, filtering, sorting, and pagination
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { SearchResultsPage } from './SearchResultsPage';
import { apiClient } from '../lib/api-client';

// Mock the API client
vi.mock('../lib/api-client', () => ({
  apiClient: {
    searchScenes: vi.fn(),
    searchEvents: vi.fn(),
    searchPosts: vi.fn(),
  },
}));

const mockScenes = [
  { id: 's1', name: 'Techno Underground', description: 'Berlin-style techno', allow_precise: false, coarse_geohash: 'u33d' },
  { id: 's2', name: 'Jazz Cellar', description: 'Intimate jazz sessions', allow_precise: false, coarse_geohash: 'u33d' },
];
const mockEvents = [
  { id: 'e1', scene_id: 's1', name: 'Friday Night Rave', description: 'All night dancing', allow_precise: false },
];
const mockPosts = [
  { id: 'p1', content: 'Looking for techno events near Kreuzberg', created_at: '2024-01-15T20:00:00Z' },
];

const renderSearchResultsPage = (search = '?q=techno') => {
  const router = createMemoryRouter(
    [
      {
        path: '/search',
        element: <SearchResultsPage />,
      },
    ],
    {
      initialEntries: [`/search${search}`],
      future: {
        v7_startTransition: true,
        v7_relativeSplatPath: true,
      },
    }
  );

  return render(<RouterProvider router={router} />);
};

describe('SearchResultsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(apiClient.searchScenes).mockResolvedValue(mockScenes);
    vi.mocked(apiClient.searchEvents).mockResolvedValue(mockEvents);
    vi.mocked(apiClient.searchPosts).mockResolvedValue(mockPosts);
  });

  describe('Empty State', () => {
    it('shows prompt when no query is provided', () => {
      renderSearchResultsPage('');
      // i18n mocked to return keys; check the h1 heading
      expect(screen.getByRole('heading', { level: 1, name: 'search.results.emptyQueryHeading' })).toBeInTheDocument();
    });

    it('shows no-results message when query returns nothing', async () => {
      vi.mocked(apiClient.searchScenes).mockResolvedValue([]);
      vi.mocked(apiClient.searchEvents).mockResolvedValue([]);
      vi.mocked(apiClient.searchPosts).mockResolvedValue([]);

      renderSearchResultsPage('?q=xyznotfound');

      await waitFor(() => {
        // The h1 heading shows the noResultsHeading key
        expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('search.results.noResultsHeading');
      }, { timeout: 500 });
    });
  });

  describe('Results Display', () => {
    it('shows scene results grouped under Scenes heading', async () => {
      renderSearchResultsPage('?q=techno');

      await waitFor(() => {
        expect(screen.getByText('Techno Underground')).toBeInTheDocument();
        expect(screen.getByText('Jazz Cellar')).toBeInTheDocument();
      }, { timeout: 500 });
    });

    it('shows event results', async () => {
      renderSearchResultsPage('?q=techno');

      await waitFor(() => {
        expect(screen.getByText('Friday Night Rave')).toBeInTheDocument();
      }, { timeout: 500 });
    });

    it('shows post results', async () => {
      renderSearchResultsPage('?q=techno');

      await waitFor(() => {
        expect(screen.getByText(/Looking for techno events near Kreuzberg/i)).toBeInTheDocument();
      }, { timeout: 500 });
    });

    it('renders result links with correct hrefs', async () => {
      renderSearchResultsPage('?q=techno');

      await waitFor(() => {
        expect(screen.getByRole('link', { name: /Techno Underground/i })).toHaveAttribute('href', '/scenes/s1');
      }, { timeout: 500 });
    });
  });

  describe('Type Filter', () => {
    it('shows all type filter buttons', () => {
      renderSearchResultsPage('?q=techno');
      // i18n mock returns keys directly
      const allButtons = screen.getAllByRole('button', { name: 'search.results.types.all' });
      expect(allButtons.length).toBeGreaterThan(0);
    });

    it('clicking Scenes filter updates aria-pressed', async () => {
      renderSearchResultsPage('?q=techno');

      const scenesButtons = screen.getAllByRole('button', { name: 'search.results.types.scenes' });
      const sidebarScenesBtn = scenesButtons[0];

      await userEvent.click(sidebarScenesBtn);

      await waitFor(() => {
        expect(sidebarScenesBtn).toHaveAttribute('aria-pressed', 'true');
      });
    });
  });

  describe('Sort Selector', () => {
    it('renders sort options in desktop sidebar using i18n keys', () => {
      renderSearchResultsPage('?q=techno');
      // i18n mock returns keys directly
      expect(screen.getAllByRole('button', { name: 'search.results.sort.relevance' }).length).toBeGreaterThan(0);
      expect(screen.getAllByRole('button', { name: 'search.results.sort.recent' }).length).toBeGreaterThan(0);
      expect(screen.getAllByRole('button', { name: 'search.results.sort.trending' }).length).toBeGreaterThan(0);
    });
  });

  describe('Loading State', () => {
    it('shows loading spinner while searching', async () => {
      // Keep the promise unresolved
      vi.mocked(apiClient.searchScenes).mockReturnValue(new Promise(() => {}));
      vi.mocked(apiClient.searchEvents).mockReturnValue(new Promise(() => {}));
      vi.mocked(apiClient.searchPosts).mockReturnValue(new Promise(() => {}));

      renderSearchResultsPage('?q=techno');

      // Loading shows after debounce fires - wait for it
      await waitFor(() => {
        expect(screen.getByRole('status')).toBeInTheDocument();
      }, { timeout: 500 });
    });
  });

  describe('Accessibility', () => {
    it('has a main landmark', () => {
      renderSearchResultsPage('?q=techno');
      expect(screen.getByRole('main')).toBeInTheDocument();
    });

    it('has filter aside with accessible label', () => {
      renderSearchResultsPage('?q=techno');
      expect(screen.getByRole('complementary')).toBeInTheDocument();
    });
  });
});
