/**
 * SearchResultsPage Component
 * Full-page search results with sidebar filters, sort selector, and pagination
 * Route: /search?q=&type=&sort=&page=
 */

import React, { useEffect, useState, useCallback } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useSearch } from '../hooks/useSearch';
import { SearchBar } from '../components/SearchBar';
import type {
  SearchResultItem,
  SceneSearchResult,
  EventSearchResult,
  PostSearchResult,
} from '../types/search';

// Constants
const RESULTS_PER_PAGE = 10;
const POST_TITLE_TRUNCATE_LENGTH = 60;
const UNTITLED_POST_LABEL = 'Untitled Post';

// Icons
const ICONS = {
  SCENE: '🎭',
  EVENT: '📅',
  POST: '📝',
  FILTER: '⚙️',
  SORT: '↕️',
  EMPTY: '🔍',
} as const;

type ResultType = 'all' | 'scenes' | 'events' | 'posts';
type SortOption = 'relevance' | 'recent' | 'trending';

const SORT_OPTIONS: { value: SortOption; labelKey: string }[] = [
  { value: 'relevance', labelKey: 'search.results.sort.relevance' },
  { value: 'recent', labelKey: 'search.results.sort.recent' },
  { value: 'trending', labelKey: 'search.results.sort.trending' },
];

const TYPE_OPTIONS: { value: ResultType; labelKey: string; icon: string }[] = [
  { value: 'all', labelKey: 'search.results.types.all', icon: ICONS.EMPTY },
  { value: 'scenes', labelKey: 'search.results.types.scenes', icon: ICONS.SCENE },
  { value: 'events', labelKey: 'search.results.types.events', icon: ICONS.EVENT },
  { value: 'posts', labelKey: 'search.results.types.posts', icon: ICONS.POST },
];

/** Returns whether the given type filter value matches the current URL param */
function isTypeActive(typeParam: ResultType, value: ResultType): boolean {
  return typeParam === value;
}

/** Returns whether the given sort value matches the current URL param */
function isSortActive(sortParam: SortOption, value: SortOption): boolean {
  return sortParam === value;
}

/**
 * Get display name for a search result item
 */
function getDisplayName(item: SearchResultItem): string {
  switch (item.type) {
    case 'scene':
      return item.data.name;
    case 'event':
      return item.data.name;
    case 'post':
      return (
        item.data.title ||
        item.data.content?.substring(0, POST_TITLE_TRUNCATE_LENGTH) ||
        UNTITLED_POST_LABEL
      );
  }
}

/**
 * Get secondary info for a search result item
 */
function getSecondaryInfo(item: SearchResultItem): string | null {
  switch (item.type) {
    case 'scene':
      return item.data.description ?? null;
    case 'event':
      return item.data.description ?? null;
    case 'post':
      // Only show content as secondary info when a title is present; otherwise
      // the display name IS the content, so showing it again would duplicate.
      return item.data.title ? (item.data.content ?? null) : null;
  }
}

/**
 * Get the navigation path for a search result item
 */
function getResultPath(item: SearchResultItem): string {
  switch (item.type) {
    case 'scene':
      return `/scenes/${item.data.id}`;
    case 'event':
      return `/events/${item.data.id}`;
    case 'post':
      return `/posts/${item.data.id}`;
  }
}

/**
 * Get the type icon for a result
 */
function getTypeIcon(type: SearchResultItem['type']): string {
  switch (type) {
    case 'scene':
      return ICONS.SCENE;
    case 'event':
      return ICONS.EVENT;
    case 'post':
      return ICONS.POST;
  }
}

interface ResultCardProps {
  item: SearchResultItem;
}

/**
 * Individual result card component
 */
const ResultCard: React.FC<ResultCardProps> = ({ item }) => {
  const displayName = getDisplayName(item);
  const secondaryInfo = getSecondaryInfo(item);

  return (
    <Link
      to={getResultPath(item)}
      className="
        block p-4 rounded-none border border-border
        bg-background-secondary hover:bg-background-hover
        focus:outline-none focus-visible:ring-2 focus-visible:ring-neon-purple
        transition-colors theme-transition
      "
    >
      <div className="flex items-start gap-3">
        <span className="text-2xl flex-shrink-0 mt-0.5" aria-hidden="true">
          {getTypeIcon(item.type)}
        </span>
        <div className="flex-1 min-w-0">
          <div className="text-sm font-semibold text-foreground truncate">{displayName}</div>
          {secondaryInfo && (
            <div className="mt-1 text-xs text-foreground-secondary line-clamp-2">
              {secondaryInfo}
            </div>
          )}
        </div>
      </div>
    </Link>
  );
};

interface ResultsSectionProps {
  titleKey: string;
  icon: string;
  items: SearchResultItem[];
}

/**
 * Section of grouped results
 */
const ResultsSection: React.FC<ResultsSectionProps> = ({ titleKey, icon, items }) => {
  const { t } = useTranslation('common');

  if (items.length === 0) return null;

  return (
    <section aria-labelledby={`results-section-${titleKey}`}>
      <h2
        id={`results-section-${titleKey}`}
        className="flex items-center gap-2 text-sm font-semibold text-foreground-tertiary uppercase tracking-wider mb-3"
      >
        <span aria-hidden="true">{icon}</span>
        {t(titleKey)}
        <span className="ml-1 text-xs font-normal normal-case bg-background-secondary border border-border rounded-full px-2 py-0.5">
          {items.length}
        </span>
      </h2>
      <ul className="space-y-2" role="list">
        {items.map((item, idx) => (
          <li key={`${item.type}-${idx}`}>
            <ResultCard item={item} />
          </li>
        ))}
      </ul>
    </section>
  );
};

/**
 * Build a list of page numbers / ellipsis strings for the pagination bar.
 * Always shows first/last pages and up to 2 pages around the current page.
 * Uses '…' as a sentinel for ellipsis gaps.
 */
function getPaginationPages(current: number, total: number): (number | '…')[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }

  const pages: (number | '…')[] = [];
  const addPage = (p: number) => {
    if (!pages.includes(p)) pages.push(p);
  };

  addPage(1);
  if (current > 3) pages.push('…');
  for (let p = Math.max(2, current - 1); p <= Math.min(total - 1, current + 1); p++) {
    addPage(p);
  }
  if (current < total - 2) pages.push('…');
  addPage(total);

  return pages;
}

/**
 * SearchResultsPage – full-page search UI with filters, sort, and pagination
 */
export const SearchResultsPage: React.FC = () => {
  const { t } = useTranslation('common');
  const [searchParams, setSearchParams] = useSearchParams();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  // Read URL state
  const query = searchParams.get('q') ?? '';
  const typeParam = (searchParams.get('type') as ResultType) ?? 'all';
  const sortParam = (searchParams.get('sort') as SortOption) ?? 'relevance';
  const pageParam = parseInt(searchParams.get('page') ?? '1', 10);
  const currentPage = isNaN(pageParam) || pageParam < 1 ? 1 : pageParam;

  // Search hook with higher limit for the results page
  const { results, loading, error, search, clear } = useSearch({ limit: 20 });

  // Run search whenever query changes
  useEffect(() => {
    if (query.trim()) {
      search(query);
    } else {
      clear();
    }
  }, [query, search, clear]); // search and clear are stable useCallback references from useSearch

  /**
   * Update a single URL param while preserving others
   */
  const updateParam = useCallback(
    (key: string, value: string) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (value) {
            next.set(key, value);
          } else {
            next.delete(key);
          }
          // Reset to page 1 when filter/sort changes
          if (key !== 'page') {
            next.delete('page');
          }
          return next;
        },
        { replace: true }
      );
    },
    [setSearchParams]
  );

  const handleTypeChange = (type: ResultType) => {
    updateParam('type', type === 'all' ? '' : type);
  };

  const handleSortChange = (sort: SortOption) => {
    updateParam('sort', sort === 'relevance' ? '' : sort);
  };

  const handlePageChange = (page: number) => {
    updateParam('page', page === 1 ? '' : String(page));
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  // Flatten and filter results by type
  const allItems: SearchResultItem[] = [
    ...results.scenes.map((s: SceneSearchResult) => ({ type: 'scene' as const, data: s })),
    ...results.events.map((e: EventSearchResult) => ({ type: 'event' as const, data: e })),
    ...results.posts.map((p: PostSearchResult) => ({ type: 'post' as const, data: p })),
  ];

  // Apply type filter
  const filteredItems =
    typeParam === 'all'
      ? allItems
      : allItems.filter((item) => {
          if (typeParam === 'scenes') return item.type === 'scene';
          if (typeParam === 'events') return item.type === 'event';
          if (typeParam === 'posts') return item.type === 'post';
          return true;
        });

  // Apply sort (client-side approximation; the API should own authoritative sorting.
  // 'recent' uses created_at for posts; scenes/events will sort to the end since they
  // don't carry a created_at field in the current type definitions).
  const sortedItems = [...filteredItems].sort((a, b) => {
    if (sortParam === 'recent') {
      const aDate = a.type === 'post' ? (a.data.created_at ?? '') : '';
      const bDate = b.type === 'post' ? (b.data.created_at ?? '') : '';
      return bDate.localeCompare(aDate);
    }
    // relevance / trending: keep original order from API
    return 0;
  });

  // Pagination
  const totalItems = sortedItems.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / RESULTS_PER_PAGE));
  const safePage = Math.min(currentPage, totalPages);
  const pagedItems = sortedItems.slice(
    (safePage - 1) * RESULTS_PER_PAGE,
    safePage * RESULTS_PER_PAGE
  );

  // For grouped view (when type === 'all'), separate by type
  const groupedScenes = pagedItems.filter((i) => i.type === 'scene');
  const groupedEvents = pagedItems.filter((i) => i.type === 'event');
  const groupedPosts = pagedItems.filter((i) => i.type === 'post');

  const hasResults = totalItems > 0;
  const showGrouped = typeParam === 'all';

  const totalResultCount = results.scenes.length + results.events.length + results.posts.length;

  return (
    <div className="min-h-screen bg-background text-foreground theme-transition">
      {/* Search header bar */}
      <div className="bg-background-secondary border-b border-border px-4 py-3">
        <div className="max-w-5xl mx-auto">
          <SearchBar placeholder={t('search.placeholder')} className="max-w-2xl" />
        </div>
      </div>

      <div className="max-w-5xl mx-auto px-4 py-6">
        {/* Page heading */}
        {query ? (
          <h1 className="text-lg font-semibold text-foreground mb-4">
            {loading
              ? t('search.results.searching')
              : hasResults
                ? t('search.results.heading', { query, count: totalResultCount })
                : t('search.results.noResultsHeading', { query })}
          </h1>
        ) : (
          <h1 className="text-lg font-semibold text-foreground mb-4">
            {t('search.results.emptyQueryHeading')}
          </h1>
        )}

        <div className="flex gap-6">
          {/* Sidebar – desktop */}
          <aside
            className="hidden md:block w-56 flex-shrink-0"
            aria-label={t('search.results.filters.label')}
          >
            <div className="bg-background-secondary border border-border rounded-none p-4 space-y-5 sticky top-4">
              {/* Type filter */}
              <div>
                <h2 className="text-xs font-semibold text-foreground-tertiary uppercase tracking-wider mb-2">
                  {t('search.results.filters.type')}
                </h2>
                <ul className="space-y-1" role="list">
                  {TYPE_OPTIONS.map(({ value, labelKey, icon }) => (
                    <li key={value}>
                      <button
                        onClick={() => handleTypeChange(value)}
                        aria-pressed={isTypeActive(typeParam, value)}
                        className={`
                          w-full flex items-center gap-2 px-3 py-1.5 rounded-none text-sm
                          focus:outline-none focus-visible:ring-2 focus-visible:ring-neon-purple
                          transition-colors
                          ${
                            isTypeActive(typeParam, value)
                              ? 'bg-neon-purple text-white font-medium'
                              : 'text-foreground hover:bg-background-hover'
                          }
                        `}
                      >
                        <span aria-hidden="true">{icon}</span>
                        {t(labelKey)}
                      </button>
                    </li>
                  ))}
                </ul>
              </div>

              {/* Sort */}
              <div>
                <h2 className="text-xs font-semibold text-foreground-tertiary uppercase tracking-wider mb-2">
                  {t('search.results.filters.sort')}
                </h2>
                <ul className="space-y-1" role="list">
                  {SORT_OPTIONS.map(({ value, labelKey }) => (
                    <li key={value}>
                      <button
                        onClick={() => handleSortChange(value)}
                        aria-pressed={isSortActive(sortParam, value)}
                        className={`
                          w-full text-left px-3 py-1.5 rounded-none text-sm
                          focus:outline-none focus-visible:ring-2 focus-visible:ring-neon-purple
                          transition-colors
                          ${
                            isSortActive(sortParam, value)
                              ? 'bg-neon-purple text-white font-medium'
                              : 'text-foreground hover:bg-background-hover'
                          }
                        `}
                      >
                        {t(labelKey)}
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </aside>

          {/* Mobile filter bar */}
          <div className="md:hidden mb-4 w-full">
            <div className="flex items-center gap-2">
              <button
                onClick={() => setIsSidebarOpen(!isSidebarOpen)}
                aria-expanded={isSidebarOpen}
                aria-controls="mobile-filters"
                className="
                  flex items-center gap-1.5 px-3 py-1.5 rounded-none text-sm
                  bg-background-secondary border border-border text-foreground
                  hover:bg-underground-lighter
                  focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
                "
              >
                <span aria-hidden="true">{ICONS.FILTER}</span>
                {t('search.results.filters.label')}
              </button>

              {/* Sort select – always visible on mobile */}
              <select
                value={sortParam}
                onChange={(e) => handleSortChange(e.target.value as SortOption)}
                aria-label={t('search.results.filters.sort')}
                className="
                  px-3 py-1.5 rounded-lg text-sm
                  bg-background-secondary border border-border text-foreground
                  focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
                  theme-transition
                "
              >
                {SORT_OPTIONS.map(({ value, labelKey }) => (
                  <option key={value} value={value}>
                    {t(labelKey)}
                  </option>
                ))}
              </select>
            </div>

            {/* Collapsible mobile filters */}
            {isSidebarOpen && (
              <div
                id="mobile-filters"
                className="mt-3 bg-background-secondary border border-border rounded-none p-4"
              >
                <h2 className="text-xs font-semibold text-foreground-tertiary uppercase tracking-wider mb-2">
                  {t('search.results.filters.type')}
                </h2>
                <div className="flex flex-wrap gap-2">
                  {TYPE_OPTIONS.map(({ value, labelKey, icon }) => (
                    <button
                      key={value}
                      onClick={() => {
                        handleTypeChange(value);
                        setIsSidebarOpen(false);
                      }}
                      aria-pressed={isTypeActive(typeParam, value)}
                      className={`
                        flex items-center gap-1 px-3 py-1.5 rounded-full text-sm
                        focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
                        transition-colors
                        ${
                          isTypeActive(typeParam, value)
                            ? 'bg-brand-primary text-white font-medium'
                            : 'bg-background border border-border text-foreground hover:bg-underground-lighter'
                        }
                      `}
                    >
                      <span aria-hidden="true">{icon}</span>
                      {t(labelKey)}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Main results area */}
          <main className="flex-1 min-w-0" id="search-results-main">
            {/* Loading state */}
            {loading && (
              <div
                className="flex flex-col items-center justify-center py-16"
                role="status"
                aria-live="polite"
              >
                <div className="inline-block animate-spin rounded-full h-8 w-8 border-2 border-neon-purple border-t-transparent mb-3" />
                <p className="text-sm text-foreground-secondary">{t('search.results.searching')}</p>
              </div>
            )}

            {/* Error state */}
            {!loading && error && (
              <div className="py-16 text-center" role="alert">
                <p className="text-sm text-status-error mb-2">{t('errors.generic')}</p>
                <p className="text-xs text-foreground-tertiary">{error}</p>
              </div>
            )}

            {/* Empty state – no query */}
            {!loading && !error && !query.trim() && (
              <div className="flex flex-col items-center justify-center py-24 text-center">
                <span className="text-5xl mb-4" aria-hidden="true">
                  {ICONS.EMPTY}
                </span>
                <p className="text-base font-medium text-foreground mb-1">
                  {t('search.results.emptyQueryHeading')}
                </p>
                <p className="text-sm text-foreground-secondary">
                  {t('search.results.emptyQueryHint')}
                </p>
              </div>
            )}

            {/* Empty state – query with no results */}
            {!loading && !error && query.trim() && !hasResults && (
              <div className="flex flex-col items-center justify-center py-24 text-center">
                <span className="text-5xl mb-4" aria-hidden="true">
                  {ICONS.EMPTY}
                </span>
                <p className="text-base font-medium text-foreground mb-1">
                  {t('search.noResults', { query })}
                </p>
                <p className="text-sm text-foreground-secondary">
                  {t('search.results.noResultsHint')}
                </p>
              </div>
            )}

            {/* Results */}
            {!loading && !error && hasResults && (
              <>
                {showGrouped ? (
                  <div className="space-y-6">
                    <ResultsSection
                      titleKey="search.sections.scenes"
                      icon={ICONS.SCENE}
                      items={groupedScenes}
                    />
                    <ResultsSection
                      titleKey="search.sections.events"
                      icon={ICONS.EVENT}
                      items={groupedEvents}
                    />
                    <ResultsSection
                      titleKey="search.sections.posts"
                      icon={ICONS.POST}
                      items={groupedPosts}
                    />
                  </div>
                ) : (
                  <ul className="space-y-2" role="list">
                    {pagedItems.map((item, idx) => (
                      <li key={`${item.type}-${idx}`}>
                        <ResultCard item={item} />
                      </li>
                    ))}
                  </ul>
                )}

                {/* Pagination */}
                {totalPages > 1 && (
                  <nav
                    className="mt-8 flex items-center justify-center gap-2 flex-wrap"
                    aria-label={t('search.results.pagination.label')}
                  >
                    <button
                      onClick={() => handlePageChange(safePage - 1)}
                      disabled={safePage <= 1}
                      aria-label={t('search.results.pagination.previous')}
                      className="
                        px-3 py-1.5 rounded-lg text-sm border border-border
                        bg-background-secondary text-foreground
                        hover:bg-underground-lighter disabled:opacity-40 disabled:cursor-not-allowed
                        focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
                        transition-colors
                      "
                    >
                      ‹ {t('search.results.pagination.previous')}
                    </button>

                    {getPaginationPages(safePage, totalPages).map((item, idx) =>
                      item === '…' ? (
                        <span
                          key={`ellipsis-${idx}`}
                          className="px-1 text-foreground-tertiary select-none"
                          aria-hidden="true"
                        >
                          …
                        </span>
                      ) : (
                        <button
                          key={item}
                          onClick={() => handlePageChange(item as number)}
                          aria-current={(item as number) === safePage ? 'page' : undefined}
                          aria-label={t('search.results.pagination.page', { page: item })}
                          className={`
                            min-w-[2rem] px-2 py-1.5 rounded-none text-sm border
                            focus:outline-none focus-visible:ring-2 focus-visible:ring-neon-purple
                            transition-colors
                            ${
                              (item as number) === safePage
                                ? 'bg-neon-purple border-neon-purple text-white font-medium'
                                : 'bg-background-secondary border-border text-foreground hover:bg-background-hover'
                            }
                          `}
                        >
                          {item}
                        </button>
                      )
                    )}

                    <button
                      onClick={() => handlePageChange(safePage + 1)}
                      disabled={safePage >= totalPages}
                      aria-label={t('search.results.pagination.next')}
                      className="
                        px-3 py-1.5 rounded-lg text-sm border border-border
                        bg-background-secondary text-foreground
                        hover:bg-underground-lighter disabled:opacity-40 disabled:cursor-not-allowed
                        focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
                        transition-colors
                      "
                    >
                      {t('search.results.pagination.next')} ›
                    </button>
                  </nav>
                )}
              </>
            )}
          </main>
        </div>
      </div>
    </div>
  );
};
