/**
 * SearchBar Component
 * Global search bar with typeahead suggestions, keyboard navigation, and ARIA compliance
 */

import { useState, useRef, useEffect, type KeyboardEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useSearch } from '../hooks/useSearch';
import { useSearchHistory } from '../hooks/useSearchHistory';
import { useKeyboardShortcut } from '../hooks/useKeyboardShortcut';
import type { SearchResultItem } from '../types/search';

// Display constants
const POST_TITLE_TRUNCATE_LENGTH = 50;
const SECONDARY_INFO_TRUNCATE_LENGTH = 60;
const UNTITLED_POST_LABEL = 'Untitled Post';
const MENTION_PATTERN = /@[a-zA-Z0-9_.-]+/g;
const HASHTAG_PATTERN = /#[a-zA-Z0-9_-]+/g;

// Icon constants
const ICONS = {
  SCENE: '🎭',
  EVENT: '📅',
  POST: '📝',
  SEARCH: '🔍',
  CLEAR: '✕',
  HISTORY: '🕐',
  CLOSE: '✕',
} as const;

export interface SearchBarProps {
  /**
   * Placeholder text for the input field
   */
  placeholder?: string;
  /**
   * Additional CSS classes
   */
  className?: string;
  /**
   * Callback when a result is selected
   */
  onSelect?: (item: SearchResultItem) => void;
  /**
   * Auto-focus on mount (default: false)
   */
  autoFocus?: boolean;
}

/**
 * SearchBar provides unified search across scenes, events, and posts
 * with typeahead suggestions, keyboard navigation, and accessibility
 */
export function SearchBar({
  placeholder,
  className = '',
  onSelect,
  autoFocus = false,
}: SearchBarProps) {
  const navigate = useNavigate();
  const { t } = useTranslation('common');
  const { results, loading, error, search, clear } = useSearch();
  const { history, addToHistory, removeFromHistory, clearHistory } = useSearchHistory();

  const [inputValue, setInputValue] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [showHistory, setShowHistory] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Use translated placeholder if not provided
  const searchPlaceholder = placeholder || t('search.placeholder');

  // Register keyboard shortcut (Cmd/Ctrl+K) to focus search
  useKeyboardShortcut(
    {
      key: 'k',
      ctrlKey: true,
      metaKey: true,
      preventDefault: true,
    },
    () => {
      inputRef.current?.focus();
    }
  );

  // Flatten results into a single array for keyboard navigation
  const flatResults: SearchResultItem[] = [
    ...results.scenes.map((scene) => ({ type: 'scene' as const, data: scene })),
    ...results.events.map((event) => ({ type: 'event' as const, data: event })),
    ...results.posts.map((post) => ({ type: 'post' as const, data: post })),
  ];

  const hasResults = flatResults.length > 0;
  const normalizedQuery = inputValue.trim().toLowerCase();
  const normalizedHashtagQuery = normalizedQuery.replace(/^#/, '');
  const sceneTags = results.scenes.flatMap((scene) => scene.tags ?? []);

  const genreSuggestions = Array.from(
    new Set(
      sceneTags
        .map((tag) => tag.trim())
        .filter((tag) => tag && tag.toLowerCase().includes(normalizedQuery))
    )
  ).slice(0, 5);

  const artistSuggestions = Array.from(
    new Set(
      results.posts
        .flatMap((post) => [
          ...(post.author_did ? [post.author_did] : []),
          ...(post.content?.match(MENTION_PATTERN) ?? []),
        ])
        .map((artist) => artist.trim())
        .filter((artist) => artist && artist.toLowerCase().includes(normalizedQuery))
    )
  ).slice(0, 5);

  const hashtagSuggestions = Array.from(
    new Set(
      [
        ...sceneTags.map((tag) => `#${tag.replace(/^#/, '')}`),
        ...results.posts.flatMap((post) => post.content?.match(HASHTAG_PATTERN) ?? []),
      ]
        .map((tag) => tag.trim())
        .filter((tag) => tag && tag.toLowerCase().includes(normalizedHashtagQuery))
    )
  ).slice(0, 5);
  const hasAutocompleteSuggestions =
    genreSuggestions.length > 0 || artistSuggestions.length > 0 || hashtagSuggestions.length > 0;

  /**
   * Handle input change
   */
  const handleInputChange = (value: string) => {
    setInputValue(value);
    setSelectedIndex(-1);
    search(value);

    if (value.trim()) {
      setIsOpen(true);
      setShowHistory(false);
    } else {
      // Show history when input is empty
      setIsOpen(history.length > 0);
      setShowHistory(history.length > 0);
    }
  };

  /**
   * Clear input and results
   */
  const handleClear = () => {
    setInputValue('');
    setIsOpen(false);
    setShowHistory(false);
    setSelectedIndex(-1);
    clear();
    inputRef.current?.focus();
  };

  /**
   * Navigate to result and save to history
   */
  const navigateToResult = (item: SearchResultItem) => {
    // Save to history
    const query = inputValue.trim();
    if (query) {
      addToHistory(query);
    }

    if (onSelect) {
      onSelect(item);
    }

    // Navigate based on type
    switch (item.type) {
      case 'scene':
        navigate(`/scenes/${item.data.id}`);
        break;
      case 'event':
        navigate(`/events/${item.data.id}`);
        break;
      case 'post':
        navigate(`/posts/${item.data.id}`);
        break;
    }

    // Close dropdown and clear
    handleClear();
  };

  /**
   * Handle clicking a history item
   */
  const handleHistoryClick = (historyQuery: string) => {
    setInputValue(historyQuery);
    setShowHistory(false);
    search(historyQuery);
    setIsOpen(true);
  };

  /**
   * Handle selecting an autocomplete suggestion
   */
  const handleAutocompleteClick = (suggestion: string) => {
    const query = suggestion.trim();
    if (!query) return;

    addToHistory(query);
    navigate(`/search?q=${encodeURIComponent(query)}`);
    setIsOpen(false);
    setShowHistory(false);
    setSelectedIndex(-1);
  };

  /**
   * Handle removing a history item
   */
  const handleRemoveHistory = (query: string, event: React.MouseEvent) => {
    event.stopPropagation();
    removeFromHistory(query);

    // If no more history, close dropdown
    if (history.length === 1) {
      setIsOpen(false);
      setShowHistory(false);
    }
  };

  /**
   * Handle keyboard navigation
   */
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (!isOpen) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex((prev) => (prev < flatResults.length - 1 ? prev + 1 : prev));
        break;

      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex((prev) => (prev > 0 ? prev - 1 : -1));
        break;

      case 'Enter':
        e.preventDefault();
        if (selectedIndex >= 0 && flatResults[selectedIndex]) {
          navigateToResult(flatResults[selectedIndex]);
        } else if (inputValue.trim()) {
          // Navigate to full search results page
          const query = inputValue.trim();
          addToHistory(query);
          navigate(`/search?q=${encodeURIComponent(query)}`);
          setIsOpen(false);
          setSelectedIndex(-1);
        }
        break;

      case 'Escape':
        e.preventDefault();
        setIsOpen(false);
        setSelectedIndex(-1);
        break;
    }
  };

  /**
   * Close dropdown on outside click
   */
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node) &&
        !inputRef.current?.contains(event.target as Node)
      ) {
        setIsOpen(false);
        setSelectedIndex(-1);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  /**
   * Scroll selected item into view
   */
  useEffect(() => {
    if (selectedIndex >= 0 && dropdownRef.current) {
      const selectedElement = dropdownRef.current.querySelector(`[data-index="${selectedIndex}"]`);
      // Check if scrollIntoView exists (not available in some test environments)
      if (selectedElement && typeof selectedElement.scrollIntoView === 'function') {
        selectedElement.scrollIntoView({ block: 'nearest' });
      }
    }
  }, [selectedIndex]);

  /**
   * Get icon for result type
   */
  const getIcon = (type: SearchResultItem['type']) => {
    switch (type) {
      case 'scene':
        return ICONS.SCENE;
      case 'event':
        return ICONS.EVENT;
      case 'post':
        return ICONS.POST;
    }
  };

  /**
   * Get display name for result
   */
  const getDisplayName = (item: SearchResultItem) => {
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
  };

  /**
   * Get secondary info for result
   */
  const getSecondaryInfo = (item: SearchResultItem) => {
    switch (item.type) {
      case 'scene':
        return item.data.description?.substring(0, SECONDARY_INFO_TRUNCATE_LENGTH) || null;
      case 'event':
        return item.data.description?.substring(0, SECONDARY_INFO_TRUNCATE_LENGTH) || null;
      case 'post':
        return item.data.content?.substring(0, SECONDARY_INFO_TRUNCATE_LENGTH) || null;
    }
  };

  return (
    <div className={`relative ${className}`}>
      {/* Search Input */}
      <div className="relative">
        <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
          <span className="text-foreground-tertiary" aria-hidden="true">
            {ICONS.SEARCH}
          </span>
        </div>

        <input
          ref={inputRef}
          type="text"
          role="combobox"
          aria-expanded={isOpen}
          aria-controls="search-results"
          aria-autocomplete="list"
          aria-activedescendant={selectedIndex >= 0 ? `search-result-${selectedIndex}` : undefined}
          value={inputValue}
          onChange={(e) => handleInputChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => {
            if (inputValue.trim()) {
              setIsOpen(true);
            } else if (history.length > 0) {
              setIsOpen(true);
              setShowHistory(true);
            }
          }}
          placeholder={searchPlaceholder}
          autoFocus={autoFocus}
          className="
            w-full pl-10 pr-10 py-2 rounded-none
            bg-background-secondary border border-border
            text-foreground placeholder:text-foreground-tertiary
            focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
            focus:border-brand-primary
            theme-transition
          "
        />

        {/* Clear Button */}
        {inputValue && (
          <button
            onClick={handleClear}
            aria-label={t('search.clearSearch')}
            className="
              absolute inset-y-0 right-0 pr-3 flex items-center
              text-foreground-tertiary hover:text-foreground
              focus:outline-none focus-visible:text-brand-primary
            "
          >
            <span aria-hidden="true">{ICONS.CLEAR}</span>
          </button>
        )}
      </div>

      {/* Results Dropdown */}
      {isOpen && (
        <div
          ref={dropdownRef}
          id="search-results"
          role="listbox"
          className="
            absolute z-50 w-full mt-2 rounded-none
            bg-background-secondary border border-border
            shadow-none max-h-96 overflow-y-auto
          "
        >
          {/* Show Search History when input is empty */}
          {showHistory && !inputValue.trim() && history.length > 0 && (
            <div role="group" aria-labelledby="search-history-heading">
              <div className="flex items-center justify-between px-3 py-2 bg-background">
                <h3
                  id="search-history-heading"
                  className="text-xs font-semibold text-foreground-tertiary uppercase tracking-wider"
                >
                  {t('search.recentSearches')}
                </h3>
                {history.length > 0 && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      clearHistory();
                      setIsOpen(false);
                      setShowHistory(false);
                    }}
                    className="text-xs text-foreground-tertiary hover:text-foreground"
                  >
                    {t('search.clearAll')}
                  </button>
                )}
              </div>
              {history.map((item) => (
                <div
                  key={`${item.query}-${item.timestamp}`}
                  className="
                    w-full px-3 py-2 flex items-center gap-3
                    hover:bg-underground-lighter
                    group
                  "
                >
                  <button
                    onClick={() => handleHistoryClick(item.query)}
                    className="
                      flex items-center gap-3 flex-1 min-w-0 text-left
                      focus:outline-none focus-visible:text-brand-primary
                    "
                  >
                    <span className="text-lg flex-shrink-0" aria-hidden="true">
                      {ICONS.HISTORY}
                    </span>
                    <span className="flex-1 text-sm text-foreground truncate">{item.query}</span>
                  </button>
                  <button
                    onClick={(e) => handleRemoveHistory(item.query, e)}
                    aria-label={`Remove "${item.query}" from history`}
                    className="
                      text-foreground-tertiary hover:text-foreground
                      focus:outline-none focus-visible:text-brand-primary
                      p-1 flex-shrink-0
                    "
                  >
                    <span aria-hidden="true" className="text-sm">
                      {ICONS.CLOSE}
                    </span>
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* Loading State */}
          {loading && !showHistory && (
            <div className="p-4 text-center" role="status" aria-live="polite">
              <div className="inline-block animate-spin rounded-full h-6 w-6 border-2 border-brand-primary border-t-transparent" />
              <p className="mt-2 text-sm text-foreground-secondary">{t('search.searching')}</p>
            </div>
          )}

          {/* Error State */}
          {error && !loading && !showHistory && (
            <div className="p-4 text-center">
              <p className="text-sm text-status-error" role="alert">
                {error}
              </p>
            </div>
          )}

          {/* Empty State */}
          {!loading &&
            !error &&
            !showHistory &&
            inputValue.trim() &&
            hasAutocompleteSuggestions && (
              <div role="group" aria-labelledby="search-autocomplete-heading">
                <h3
                  id="search-autocomplete-heading"
                  className="px-3 py-2 text-xs font-semibold text-foreground-tertiary uppercase tracking-wider bg-background"
                >
                  {t('search.suggestions')}
                </h3>

                {genreSuggestions.length > 0 && (
                  <div role="group" aria-labelledby="search-suggestions-genres-heading">
                    <h4
                      id="search-suggestions-genres-heading"
                      className="px-3 py-2 text-xs font-semibold text-foreground-tertiary uppercase tracking-wider"
                    >
                      {t('search.sections.genres')}
                    </h4>
                    {genreSuggestions.map((genre) => (
                      <button
                        key={`genre-${genre}`}
                        onClick={() => handleAutocompleteClick(genre)}
                        className="w-full px-3 py-2 text-left text-sm hover:bg-underground-lighter focus:outline-none focus-visible:bg-underground-lighter"
                      >
                        {genre}
                      </button>
                    ))}
                  </div>
                )}

                {artistSuggestions.length > 0 && (
                  <div role="group" aria-labelledby="search-suggestions-artists-heading">
                    <h4
                      id="search-suggestions-artists-heading"
                      className="px-3 py-2 text-xs font-semibold text-foreground-tertiary uppercase tracking-wider"
                    >
                      {t('search.sections.artists')}
                    </h4>
                    {artistSuggestions.map((artist) => (
                      <button
                        key={`artist-${artist}`}
                        onClick={() => handleAutocompleteClick(artist)}
                        className="w-full px-3 py-2 text-left text-sm hover:bg-underground-lighter focus:outline-none focus-visible:bg-underground-lighter"
                      >
                        {artist}
                      </button>
                    ))}
                  </div>
                )}

                {hashtagSuggestions.length > 0 && (
                  <div role="group" aria-labelledby="search-suggestions-hashtags-heading">
                    <h4
                      id="search-suggestions-hashtags-heading"
                      className="px-3 py-2 text-xs font-semibold text-foreground-tertiary uppercase tracking-wider"
                    >
                      {t('search.sections.hashtags')}
                    </h4>
                    {hashtagSuggestions.map((hashtag) => (
                      <button
                        key={`hashtag-${hashtag}`}
                        onClick={() => handleAutocompleteClick(hashtag)}
                        className="w-full px-3 py-2 text-left text-sm hover:bg-underground-lighter focus:outline-none focus-visible:bg-underground-lighter"
                      >
                        {hashtag}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}

          {!loading &&
            !error &&
            !hasResults &&
            !showHistory &&
            !hasAutocompleteSuggestions &&
            inputValue.trim() && (
              <div className="p-4 text-center">
                <p className="text-sm text-foreground-tertiary">
                  {t('search.noResults', { query: inputValue })}
                </p>
              </div>
            )}

          {/* Results */}
          {!loading && !error && !showHistory && hasResults && (
            <>
              {/* Scenes */}
              {results.scenes.length > 0 && (
                <div role="group" aria-labelledby="search-results-scenes-heading">
                  <h3
                    id="search-results-scenes-heading"
                    className="px-3 py-2 text-xs font-semibold text-foreground-tertiary uppercase tracking-wider bg-background"
                  >
                    {t('search.sections.scenes')}
                  </h3>
                  {results.scenes.map((scene, idx) => {
                    const flatIdx = idx;
                    const item: SearchResultItem = { type: 'scene', data: scene };
                    return (
                      <button
                        key={scene.id}
                        id={`search-result-${flatIdx}`}
                        data-index={flatIdx}
                        role="option"
                        aria-selected={selectedIndex === flatIdx}
                        onClick={() => navigateToResult(item)}
                        className={`
                          w-full px-3 py-2 text-left flex items-start gap-3
                          hover:bg-underground-lighter
                          focus:outline-none focus-visible:bg-underground-lighter
                          ${selectedIndex === flatIdx ? 'bg-underground-lighter' : ''}
                        `}
                      >
                        <span className="text-xl flex-shrink-0" aria-hidden="true">
                          {getIcon('scene')}
                        </span>
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-foreground truncate">
                            {getDisplayName(item)}
                          </div>
                          {getSecondaryInfo(item) && (
                            <div className="text-xs text-foreground-secondary truncate">
                              {getSecondaryInfo(item)}
                            </div>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}

              {/* Events */}
              {results.events.length > 0 && (
                <div role="group" aria-labelledby="search-results-events-heading">
                  <h3
                    id="search-results-events-heading"
                    className="px-3 py-2 text-xs font-semibold text-foreground-tertiary uppercase tracking-wider bg-background"
                  >
                    {t('search.sections.events')}
                  </h3>
                  {results.events.map((event, idx) => {
                    const flatIdx = results.scenes.length + idx;
                    const item: SearchResultItem = { type: 'event', data: event };
                    return (
                      <button
                        key={event.id}
                        id={`search-result-${flatIdx}`}
                        data-index={flatIdx}
                        role="option"
                        aria-selected={selectedIndex === flatIdx}
                        onClick={() => navigateToResult(item)}
                        className={`
                          w-full px-3 py-2 text-left flex items-start gap-3
                          hover:bg-underground-lighter
                          focus:outline-none focus-visible:bg-underground-lighter
                          ${selectedIndex === flatIdx ? 'bg-underground-lighter' : ''}
                        `}
                      >
                        <span className="text-xl flex-shrink-0" aria-hidden="true">
                          {getIcon('event')}
                        </span>
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-foreground truncate">
                            {getDisplayName(item)}
                          </div>
                          {getSecondaryInfo(item) && (
                            <div className="text-xs text-foreground-secondary truncate">
                              {getSecondaryInfo(item)}
                            </div>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}

              {/* Posts */}
              {results.posts.length > 0 && (
                <div role="group" aria-labelledby="search-results-posts-heading">
                  <h3
                    id="search-results-posts-heading"
                    className="px-3 py-2 text-xs font-semibold text-foreground-tertiary uppercase tracking-wider bg-background"
                  >
                    {t('search.sections.posts')}
                  </h3>
                  {results.posts.map((post, idx) => {
                    const flatIdx = results.scenes.length + results.events.length + idx;
                    const item: SearchResultItem = { type: 'post', data: post };
                    return (
                      <button
                        key={post.id}
                        id={`search-result-${flatIdx}`}
                        data-index={flatIdx}
                        role="option"
                        aria-selected={selectedIndex === flatIdx}
                        onClick={() => navigateToResult(item)}
                        className={`
                          w-full px-3 py-2 text-left flex items-start gap-3
                          hover:bg-underground-lighter
                          focus:outline-none focus-visible:bg-underground-lighter
                          ${selectedIndex === flatIdx ? 'bg-underground-lighter' : ''}
                        `}
                      >
                        <span className="text-xl flex-shrink-0" aria-hidden="true">
                          {getIcon('post')}
                        </span>
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-foreground truncate">
                            {getDisplayName(item)}
                          </div>
                          {getSecondaryInfo(item) && (
                            <div className="text-xs text-foreground-secondary truncate">
                              {getSecondaryInfo(item)}
                            </div>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
