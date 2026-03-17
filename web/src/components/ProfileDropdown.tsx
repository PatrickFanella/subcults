/**
 * ProfileDropdown Component
 * User profile menu with account actions
 */

import { useState, useRef, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth, authStore } from '../stores/authStore';

export interface ProfileDropdownProps {
  /**
   * Additional CSS classes
   */
  className?: string;
}

/**
 * ProfileDropdown shows user info and account actions
 */
export function ProfileDropdown({ className = '' }: ProfileDropdownProps) {
  const { user, isAdmin } = useAuth();
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  // Close on Escape key
  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      return () => document.removeEventListener('keydown', handleEscape);
    }
  }, [isOpen]);

  const handleLogout = () => {
    authStore.logout();
    setIsOpen(false);
    navigate('/');
  };

  if (!user) {
    return null;
  }

  // Truncate DID for display
  const displayDid = user.did.length > 30 ? `${user.did.slice(0, 30)}...` : user.did;

  return (
    <div className={`relative ${className}`} ref={dropdownRef}>
      {/* Profile Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
        aria-haspopup="true"
        aria-label={t('profile.menu')}
        className="
          flex items-center gap-1 sm:gap-2 p-2 rounded-none min-h-touch min-w-touch
          text-foreground hover:bg-underground-lighter
          focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
          transition-none touch-manipulation
        "
      >
        {/* Avatar */}
        <div
          className="
            w-8 h-8 sm:w-9 sm:h-9 rounded-full
            bg-brand-primary text-white
            flex items-center justify-center
            text-xs sm:text-sm font-bold
          "
          aria-hidden="true"
        >
          {user.did.length >= 6 ? user.did.slice(4, 6).toUpperCase() : '??'}
        </div>

        {/* Chevron - Hidden on very small screens */}
        <svg
          className={`w-4 h-4 transition-transform hidden xs:block ${isOpen ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Dropdown Menu */}
      {isOpen && (
        <div
          className="
            absolute right-0 mt-2 w-64 sm:w-72
            bg-background-secondary border border-border
            rounded-none shadow-none
            py-2
            z-50
          "
          role="menu"
        >
          {/* User Info */}
          <div className="px-4 py-3 border-b border-border">
            <p className="text-sm text-foreground-secondary">Signed in as</p>
            <p className="text-sm font-medium text-foreground truncate" title={user.did}>
              {displayDid}
            </p>
            {isAdmin && (
              <span className="inline-block mt-1 px-2 py-0.5 text-xs font-medium text-white bg-brand-accent rounded-full">
                {t('navigation.admin')}
              </span>
            )}
          </div>

          {/* Menu Items */}
          <div className="py-1">
            <Link
              to="/account"
              onClick={() => setIsOpen(false)}
              className="
                block px-4 py-2.5 min-h-touch
                text-sm text-foreground
                hover:bg-underground-lighter
                focus:outline-none focus-visible:bg-underground-lighter
                touch-manipulation
              "
              role="menuitem"
            >
              {t('profile.account')}
            </Link>
            <Link
              to="/settings"
              onClick={() => setIsOpen(false)}
              className="
                block px-4 py-2.5 min-h-touch
                text-sm text-foreground
                hover:bg-underground-lighter
                focus:outline-none focus-visible:bg-underground-lighter
                touch-manipulation
              "
              role="menuitem"
            >
              {t('profile.settings')}
            </Link>
            {isAdmin && (
              <Link
                to="/admin"
                onClick={() => setIsOpen(false)}
                className="
                  block px-4 py-2.5 min-h-touch
                  text-sm text-foreground
                  hover:bg-underground-lighter
                  focus:outline-none focus-visible:bg-underground-lighter
                  touch-manipulation
                "
                role="menuitem"
              >
                {t('navigation.admin')}
              </Link>
            )}
          </div>

          {/* Logout */}
          <div className="py-1 border-t border-border">
            <button
              onClick={handleLogout}
              className="
                w-full text-left px-4 py-2.5 min-h-touch
                text-sm text-status-error
                hover:bg-underground-lighter
                focus:outline-none focus-visible:bg-underground-lighter
                touch-manipulation
              "
              role="menuitem"
            >
              {t('profile.logout')}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
