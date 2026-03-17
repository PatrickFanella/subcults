/**
 * UserSearchBar Component
 * Search for users with trust-based filtering
 */

import React, { useState, useCallback } from 'react';
import { useUserSearch, type UserSearchFilters } from '../hooks/useUserSearch';

interface UserSearchBarProps {
  /** Callback when a user is selected */
  onSelectUser?: (user: { id: string; name?: string }) => void;
  
  /** Placeholder text */
  placeholder?: string;
  
  /** Show trust filters */
  showTrustFilters?: boolean;
}

export const UserSearchBar: React.FC<UserSearchBarProps> = ({
  onSelectUser,
  placeholder = 'Search users...',
  showTrustFilters = true,
}) => {
  const [query, setQuery] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [filters, setFilters] = useState<UserSearchFilters>({});
  const [selectedIndex, setSelectedIndex] = useState(-1);
  
  const { results, loading, error, search, clear } = useUserSearch();

  const handleInputChange = useCallback(
    (value: string) => {
      setQuery(value);
      setSelectedIndex(-1);
      
      if (value.trim()) {
        search(value, filters);
      } else {
        clear();
      }
    },
    [search, clear, filters]
  );

  const handleSelectUser = useCallback(
    (user: typeof results[0]) => {
      setQuery('');
      clear();
      setSelectedIndex(-1);
      
      if (onSelectUser) {
        onSelectUser({ id: user.id, name: user.name });
      }
    },
    [clear, onSelectUser]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'ArrowDown' && selectedIndex < results.length - 1) {
        setSelectedIndex(selectedIndex + 1);
      } else if (e.key === 'ArrowUp' && selectedIndex > 0) {
        setSelectedIndex(selectedIndex - 1);
      } else if (e.key === 'Enter' && selectedIndex >= 0) {
        handleSelectUser(results[selectedIndex]);
      } else if (e.key === 'Escape') {
        clear();
        setSelectedIndex(-1);
      }
    },
    [selectedIndex, results, handleSelectUser, clear]
  );

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      <div style={{ display: 'flex', gap: '0.5rem' }}>
        <input
          type="text"
          value={query}
          onChange={(e) => handleInputChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => query && results.length > 0 && setSelectedIndex(0)}
          placeholder={placeholder}
          style={{
            flex: 1,
            padding: '0.75rem',
            border: '1px solid #262626',
            backgroundColor: '#1A1A1A',
            color: '#FAFAFA',
            fontSize: '0.875rem',
          }}
        />
        
        {showTrustFilters && (
          <button
            onClick={() => setShowFilters(!showFilters)}
            style={{
              padding: '0.75rem 1rem',
              backgroundColor: showFilters ? '#FF3D00' : '#1A1A1A',
              color: showFilters ? '#0A0A0A' : '#FAFAFA',
              border: '1px solid #262626',
              cursor: 'pointer',
              fontWeight: 500,
            }}
            title="Toggle trust filters"
          >
            🔍 Filters
          </button>
        )}
      </div>

      {/* Trust Filters */}
      {showFilters && showTrustFilters && (
        <div style={{
          marginTop: '0.5rem',
          padding: '1rem',
          backgroundColor: '#0F0F0F',
          border: '1px solid #262626',
          display: 'grid',
          gridTemplateColumns: 'repeat(2, 1fr)',
          gap: '1rem',
        }}>
          <div>
            <label style={{ fontSize: '0.875rem', fontWeight: 500, color: '#FAFAFA' }}>
              Min Trust Score
            </label>
            <input
              type="range"
              min="0"
              max="1"
              step="0.1"
              value={filters.minTrustScore ?? 0}
              onChange={(e) => {
                const newFilters = { ...filters, minTrustScore: parseFloat(e.target.value) };
                setFilters(newFilters);
                search(query, newFilters);
              }}
              style={{ width: '100%', marginTop: '0.25rem' }}
            />
            <span style={{ fontSize: '0.75rem', color: '#737373' }}>
              {((filters.minTrustScore ?? 0) * 100).toFixed(0)}%
            </span>
          </div>

          <div>
            <label style={{ fontSize: '0.875rem', fontWeight: 500, color: '#FAFAFA' }}>
              Role
            </label>
            <select
              value={filters.role ?? ''}
              onChange={(e) => {
                const newFilters = { ...filters, role: (e.target.value || undefined) as any };
                setFilters(newFilters);
                search(query, newFilters);
              }}
              style={{
                width: '100%',
                padding: '0.5rem',
                marginTop: '0.25rem',
                border: '1px solid #262626',
                backgroundColor: '#1A1A1A',
                color: '#FAFAFA',
                fontSize: '0.875rem',
              }}
            >
              <option value="">All roles</option>
              <option value="organizer">Organizer</option>
              <option value="artist">Artist</option>
              <option value="promoter">Promoter</option>
              <option value="venue">Venue</option>
            </select>
          </div>

          <div>
            <label style={{ fontSize: '0.875rem', fontWeight: 500, color: '#FAFAFA' }}>
              <input
                type="checkbox"
                checked={filters.verified ?? false}
                onChange={(e) => {
                  const newFilters = { ...filters, verified: e.target.checked || undefined };
                  setFilters(newFilters);
                  search(query, newFilters);
                }}
                style={{ marginRight: '0.5rem' }}
              />
              Verified Only
            </label>
          </div>

          <div>
            <label style={{ fontSize: '0.875rem', fontWeight: 500, color: '#FAFAFA' }}>
              Min Followers
            </label>
            <input
              type="number"
              min="0"
              value={filters.minFollowers ?? 0}
              onChange={(e) => {
                const newFilters = { ...filters, minFollowers: e.target.value ? parseInt(e.target.value) : undefined };
                setFilters(newFilters);
                search(query, newFilters);
              }}
              style={{
                width: '100%',
                padding: '0.5rem',
                marginTop: '0.25rem',
                border: '1px solid #262626',
                backgroundColor: '#1A1A1A',
                color: '#FAFAFA',
                fontSize: '0.875rem',
              }}
            />
          </div>
        </div>
      )}

      {/* Search Results Dropdown */}
      {(results.length > 0 || loading || error) && query && (
        <div
          style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            right: 0,
            marginTop: '0.25rem',
            backgroundColor: '#0F0F0F',
            border: '1px solid #262626',
            zIndex: 1000,
            maxHeight: '300px',
            overflowY: 'auto',
          }}
        >
          {loading && (
            <div style={{ padding: '1rem', textAlign: 'center', color: '#737373' }}>
              Searching...
            </div>
          )}

          {error && (
            <div style={{ padding: '1rem', color: '#FF3D00', fontSize: '0.875rem' }}>
              {error}
            </div>
          )}

          {!loading && results.map((user, index) => (
            <div
              key={user.id}
              onClick={() => handleSelectUser(user)}
              style={{
                padding: '0.75rem 1rem',
                borderBottom: '1px solid #262626',
                backgroundColor: index === selectedIndex ? '#1A1A1A' : '#0F0F0F',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '0.75rem',
                transition: 'background-color 150ms',
              }}
              onMouseEnter={() => setSelectedIndex(index)}
            >
              {user.avatar && (
                <img
                  src={user.avatar}
                  alt={user.name}
                  style={{
                    width: '32px',
                    height: '32px',
                    borderRadius: '50%',
                    objectFit: 'cover',
                  }}
                />
              )}

              <div style={{ flex: 1 }}>
                <div style={{ fontWeight: 500, fontSize: '0.875rem', color: '#FAFAFA' }}>
                  {user.name} {user.verified && '✓'}
                </div>
                {user.trustScore !== undefined && (
                  <div style={{ fontSize: '0.75rem', color: '#737373' }}>
                    Trust: {(user.trustScore * 100).toFixed(0)}% • {user.followers || 0} followers
                  </div>
                )}
              </div>
            </div>
          ))}

          {!loading && results.length === 0 && query && (
            <div style={{ padding: '1rem', textAlign: 'center', color: '#737373', fontSize: '0.875rem' }}>
              No users found
            </div>
          )}
        </div>
      )}
    </div>
  );
};

UserSearchBar.displayName = 'UserSearchBar';
