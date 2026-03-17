/**
 * SearchBar Demo
 * Example usage of the SearchBar component
 */

import { SearchBar } from './components/SearchBar';
import type { SearchResultItem } from './types/search';

function SearchBarDemo() {
  const handleSelect = (item: SearchResultItem) => {
    console.log('Selected:', item.type, item.data);
  };

  return (
    <div className="min-h-screen bg-background p-8">
      <div className="max-w-4xl mx-auto space-y-12">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold text-foreground mb-2">
            SearchBar Component Demo
          </h1>
          <p className="text-foreground-secondary">
            Global search with typeahead, keyboard navigation, and ARIA compliance
          </p>
        </div>

        {/* Basic Example */}
        <section className="space-y-4">
          <h2 className="text-xl font-semibold text-foreground">Basic Usage</h2>
          <SearchBar />
        </section>

        {/* With Selection Callback */}
        <section className="space-y-4">
          <h2 className="text-xl font-semibold text-foreground">With Selection Callback</h2>
          <SearchBar onSelect={handleSelect} />
        </section>

        {/* Auto-focus */}
        <section className="space-y-4">
          <h2 className="text-xl font-semibold text-foreground">Auto-focus</h2>
          <SearchBar autoFocus placeholder="I'm auto-focused!" />
        </section>

        {/* Custom Styling */}
        <section className="space-y-4">
          <h2 className="text-xl font-semibold text-foreground">Custom Styling</h2>
          <div className="flex justify-center">
            <SearchBar className="max-w-md" placeholder="Centered search..." />
          </div>
        </section>

        {/* Usage Instructions */}
        <section className="space-y-4">
          <h2 className="text-xl font-semibold text-foreground">Keyboard Navigation</h2>
          <div className="bg-background-secondary rounded-lg p-6 space-y-3">
            <div className="flex items-center gap-3">
              <kbd className="px-2 py-1 bg-background border border-border rounded text-sm">↓</kbd>
              <span className="text-foreground-secondary">Move selection down</span>
            </div>
            <div className="flex items-center gap-3">
              <kbd className="px-2 py-1 bg-background border border-border rounded text-sm">↑</kbd>
              <span className="text-foreground-secondary">Move selection up</span>
            </div>
            <div className="flex items-center gap-3">
              <kbd className="px-2 py-1 bg-background border border-border rounded text-sm">Enter</kbd>
              <span className="text-foreground-secondary">Navigate to selected result</span>
            </div>
            <div className="flex items-center gap-3">
              <kbd className="px-2 py-1 bg-background border border-border rounded text-sm">Esc</kbd>
              <span className="text-foreground-secondary">Close dropdown</span>
            </div>
          </div>
        </section>

        {/* Features */}
        <section className="space-y-4">
          <h2 className="text-xl font-semibold text-foreground">Features</h2>
          <ul className="space-y-2 text-foreground-secondary">
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>300ms debounce to reduce API load</span>
            </li>
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>Automatic request cancellation on new queries</span>
            </li>
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>Parallel search across scenes, events, and posts</span>
            </li>
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>Grouped results with icons</span>
            </li>
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>Loading, empty, and error states</span>
            </li>
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>Full keyboard navigation</span>
            </li>
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>Search history with localStorage persistence (last 5 searches)</span>
            </li>
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>Global keyboard shortcut (Cmd/Ctrl+K) to focus search</span>
            </li>
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>ARIA combobox pattern compliance</span>
            </li>
            <li className="flex items-start gap-2">
              <span>✓</span>
              <span>Dark mode support</span>
            </li>
          </ul>
        </section>

        {/* New Features Info */}
        <section className="space-y-4">
          <h2 className="text-xl font-semibold text-foreground">Recent Enhancements</h2>
          <div className="bg-background-secondary rounded-lg p-6 space-y-4">
            <div>
              <h3 className="font-medium text-foreground mb-2">Search History</h3>
              <p className="text-foreground-secondary text-sm">
                Focus the search bar with an empty input to view your recent searches. 
                The last 5 searches are saved automatically and persist across sessions.
              </p>
            </div>
            <div>
              <h3 className="font-medium text-foreground mb-2">Keyboard Shortcut</h3>
              <p className="text-foreground-secondary text-sm">
                Press <kbd className="px-2 py-1 bg-background border border-border rounded text-sm">Cmd/Ctrl+K</kbd> anywhere 
                on the page to quickly focus the search bar. Perfect for power users!
              </p>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}

export default SearchBarDemo;
