/**
 * AppLayout Component
 * Main application layout shell with header, sidebar, and content outlet
 */

import React, { useState } from 'react';
import { Outlet, Link } from 'react-router-dom';
import { useAuth } from '../stores/authStore';
import { MiniPlayer } from '../components/MiniPlayer';
import { SearchBar } from '../components/SearchBar';
import { NotificationBadge } from '../components/NotificationBadge';
import { ProfileDropdown } from '../components/ProfileDropdown';
import { Sidebar } from '../components/Sidebar';
import { DarkModeToggle } from '../components/DarkModeToggle';

export const AppLayout: React.FC = () => {
  const { isAuthenticated } = useAuth();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  return (
    <div className="flex flex-col h-screen w-full">
      {/* Skip to content link for accessibility */}
      <a
        href="#main-content"
        className="
          absolute -top-24 left-0 z-[9999]
          px-4 py-2 bg-white text-underground
          focus:top-0
          transition-all
        "
      >
        Skip to content
      </a>

      {/* Header */}
      <header
        role="banner"
        className="
          flex-shrink-0
          bg-underground border-b border-border
          px-3 py-2 sm:px-4 sm:py-3
          theme-transition
        "
      >
        <div className="flex items-center justify-between gap-2 sm:gap-4 max-w-[1600px] mx-auto">
          {/* Mobile Menu Toggle + Logo */}
          <div className="flex items-center gap-2 sm:gap-3">
            <button
              onClick={() => setIsSidebarOpen(!isSidebarOpen)}
              aria-label="Toggle sidebar"
              aria-expanded={isSidebarOpen}
              className="
                lg:hidden p-2 sm:p-2.5 rounded-none min-h-touch min-w-touch
                text-white hover:bg-background-secondary
                focus:outline-none focus-visible:ring-2 focus-visible:ring-neon-purple
                touch-manipulation
              "
            >
              <svg
                aria-hidden="true"
                className="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 6h16M4 12h16M4 18h16"
                />
              </svg>
            </button>

            <Link
              to="/"
              className="text-xl sm:text-2xl font-bold text-white hover:text-brand-primary transition-colors min-h-touch"
            >
              Subcults
            </Link>
          </div>

          {/* Search Bar - Desktop */}
          <div className="hidden md:block flex-1 max-w-xl">
            <SearchBar placeholder="Search scenes, events, posts..." />
          </div>

          {/* Right Actions */}
          <div className="flex items-center gap-1 sm:gap-2">
            <DarkModeToggle />
            {isAuthenticated ? (
              <>
                <NotificationBadge />
                <ProfileDropdown />
              </>
            ) : (
              <Link
                to="/account/login"
                className="
                  px-3 py-1 sm:px-4 sm:py-1 rounded-none min-h-touch
                  bg-white text-foreground
                  font-medium text-sm sm:text-base
                  hover:bg-gray-100
                  transition-colors
                  touch-manipulation
                "
              >
                Login
              </Link>
            )}
          </div>
        </div>

        {/* Search Bar - Mobile */}
        <div className="md:hidden mt-2 sm:mt-3">
          <SearchBar placeholder="Search..." />
        </div>
      </header>

      {/* Main Content Area with Sidebar */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <Sidebar isOpen={isSidebarOpen} onClose={() => setIsSidebarOpen(false)} />

        {/* Main Content */}
        <main
          id="main-content"
          role="main"
          className="flex-1 overflow-auto relative bg-background theme-transition"
        >
          <Outlet />
        </main>
      </div>

      {/* Mini Player (persistent across routes) */}
      <MiniPlayer />
    </div>
  );
};
