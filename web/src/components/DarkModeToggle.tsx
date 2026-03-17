/**
 * DarkModeToggle Component
 * Accessible button for toggling between light and dark mode
 */

import { useTheme, useThemeActions } from '../stores/themeStore';

interface DarkModeToggleProps {
  /**
   * Show text label alongside icon (default: false)
   */
  showLabel?: boolean;
  /**
   * Additional CSS classes
   */
  className?: string;
}

/**
 * DarkModeToggle provides an accessible control for theme switching
 */
export function DarkModeToggle({ showLabel = false, className = '' }: DarkModeToggleProps) {
  const theme = useTheme();
  const { toggleTheme } = useThemeActions();

  const isDark = theme === 'dark';
  const icon = isDark ? '☀️' : '🌙';
  const label = isDark ? 'Light mode' : 'Dark mode';

  return (
    <button
      onClick={toggleTheme}
      aria-label={`Switch to ${label}`}
      title={`Switch to ${label}`}
      className={`
        inline-flex items-center gap-2 px-3 py-2 rounded-none
        bg-background-secondary hover:bg-underground-lighter
        border border-border hover:border-border-hover
        transition-none
        theme-transition
        focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
        ${className}
      `.trim()}
    >
      <span className="text-xl" role="img" aria-hidden="true">
        {icon}
      </span>
      {showLabel && <span className="text-sm font-medium text-foreground">{label}</span>}
    </button>
  );
}
