/**
 * SettingsPage Component Tests
 * Tests settings page functionality and user interactions
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { SettingsPage } from './SettingsPage';
import { useThemeStore } from '../stores/themeStore';

// Mock auth store
vi.mock('../stores/authStore', () => ({
  useAuth: () => ({
    user: { did: 'did:plc:test123', role: 'user' },
    logout: vi.fn(),
  }),
}));

// Mock toast store
vi.mock('../stores/toastStore', () => ({
  useToasts: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  }),
}));

// Mock API client
vi.mock('../lib/api-client', () => ({
  apiClient: {
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('SettingsPage', () => {
  beforeEach(() => {
    localStorage.clear();
    useThemeStore.setState({ theme: 'light' });
    document.documentElement.classList.remove('dark');
  });

  const renderSettingsPage = () => {
    return render(
      <BrowserRouter>
        <SettingsPage />
      </BrowserRouter>
    );
  };

  describe('Rendering', () => {
    it('should render settings heading', () => {
      renderSettingsPage();
      expect(screen.getByRole('heading', { name: /^Settings$/i })).toBeInTheDocument();
    });

    it('should render profile section', () => {
      renderSettingsPage();
      expect(screen.getByRole('heading', { name: /^Profile$/i })).toBeInTheDocument();
    });

    it('should render privacy section', () => {
      renderSettingsPage();
      expect(screen.getByRole('heading', { name: /Privacy/i })).toBeInTheDocument();
    });

    it('should render appearance section', () => {
      renderSettingsPage();
      expect(screen.getByRole('heading', { name: /Appearance/i })).toBeInTheDocument();
    });

    it('should render notifications section', () => {
      renderSettingsPage();
      expect(screen.getByRole('heading', { name: /Notifications/i })).toBeInTheDocument();
    });

    it('should render linked accounts section', () => {
      renderSettingsPage();
      expect(screen.getByRole('heading', { name: /Linked Accounts/i })).toBeInTheDocument();
    });

    it('should render session management section', () => {
      renderSettingsPage();
      expect(screen.getByRole('heading', { name: /Session Management/i })).toBeInTheDocument();
    });

    it('should render danger zone section', () => {
      renderSettingsPage();
      expect(screen.getByRole('heading', { name: /Danger Zone/i })).toBeInTheDocument();
    });
  });

  describe('Profile Section', () => {
    it('should display avatar', () => {
      renderSettingsPage();
      // Check for avatar by role and aria-label
      const avatarContainer = screen.getByLabelText(/test123's avatar \(initials\)/i);
      expect(avatarContainer).toBeInTheDocument();
    });

    it('should have display name input', () => {
      renderSettingsPage();
      expect(screen.getByLabelText(/Display Name/i)).toBeInTheDocument();
    });

    it('should have bio textarea', () => {
      renderSettingsPage();
      expect(screen.getByLabelText(/Bio/i)).toBeInTheDocument();
    });

    it('should have change avatar button', () => {
      renderSettingsPage();
      expect(screen.getByRole('button', { name: /Change Avatar/i })).toBeInTheDocument();
    });

    it('should have save profile button', () => {
      renderSettingsPage();
      expect(screen.getByRole('button', { name: /Save Profile/i })).toBeInTheDocument();
    });

    it('should allow typing in display name field', () => {
      renderSettingsPage();
      const input = screen.getByLabelText(/Display Name/i) as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'Test User' } });
      expect(input.value).toBe('Test User');
    });

    it('should allow typing in bio field', () => {
      renderSettingsPage();
      const textarea = screen.getByLabelText(/Bio/i) as HTMLTextAreaElement;
      fireEvent.change(textarea, { target: { value: 'Test bio' } });
      expect(textarea.value).toBe('Test bio');
    });
  });

  describe('Privacy Settings', () => {
    it('should display precise location toggle', () => {
      renderSettingsPage();
      // Use heading with exact text
      expect(screen.getByRole('heading', { name: /^Precise Location$/i })).toBeInTheDocument();
    });

    it('should have location consent checkbox', () => {
      renderSettingsPage();
      const checkbox = screen.getByRole('checkbox', { name: /Allow precise location/i });
      expect(checkbox).toBeInTheDocument();
    });

    it('should display privacy information', () => {
      renderSettingsPage();
      expect(screen.getByText(/Privacy First:/i)).toBeInTheDocument();
    });

    it('should allow toggling location consent', () => {
      renderSettingsPage();
      const checkbox = screen.getByRole('checkbox', {
        name: /Allow precise location/i,
      }) as HTMLInputElement;
      expect(checkbox.checked).toBe(false);

      fireEvent.click(checkbox);
      expect(checkbox.checked).toBe(true);
    });
  });

  describe('Appearance Settings', () => {
    it('should display theme label', () => {
      renderSettingsPage();
      expect(screen.getByText('Theme')).toBeInTheDocument();
    });

    it('should display current theme', () => {
      renderSettingsPage();
      expect(screen.getByText(/Current theme:/i)).toBeInTheDocument();
      expect(screen.getByText('light')).toBeInTheDocument();
    });

    it('should render dark mode toggle component', () => {
      renderSettingsPage();
      const toggle = screen.getByLabelText(/dark mode/i);
      expect(toggle).toBeInTheDocument();
    });
  });

  describe('Linked Accounts', () => {
    it('should display Stripe Connect option', () => {
      renderSettingsPage();
      expect(screen.getByText('Stripe Connect')).toBeInTheDocument();
    });

    it('should have Stripe connect button', () => {
      renderSettingsPage();
      expect(screen.getByRole('button', { name: /Connect/i })).toBeInTheDocument();
    });

    it('should display Artist Profile option', () => {
      renderSettingsPage();
      expect(screen.getByText('Artist Profile')).toBeInTheDocument();
    });

    it('should have artist profile create button', () => {
      renderSettingsPage();
      expect(screen.getByRole('button', { name: /Create/i })).toBeInTheDocument();
    });
  });

  describe('Session Management', () => {
    it('should display logout other devices button', () => {
      renderSettingsPage();
      expect(screen.getByRole('button', { name: /Logout Other Devices/i })).toBeInTheDocument();
    });

    it('should display session management description', () => {
      renderSettingsPage();
      expect(screen.getByText(/Sign out of all other devices/i)).toBeInTheDocument();
    });
  });

  describe('Danger Zone', () => {
    it('should display delete account button', () => {
      renderSettingsPage();
      expect(screen.getByRole('button', { name: /Delete Account/i })).toBeInTheDocument();
    });

    it('should display delete warning', () => {
      renderSettingsPage();
      expect(screen.getByText(/Permanently delete your account/i)).toBeInTheDocument();
    });

    it('should show confirmation modal when delete is clicked', async () => {
      renderSettingsPage();
      const deleteButton = screen.getByRole('button', { name: /Delete Account/i });

      fireEvent.click(deleteButton);

      await waitFor(() => {
        expect(screen.getByText(/Are you absolutely sure/i)).toBeInTheDocument();
      });
    });

    it('should close confirmation modal when cancel is clicked', async () => {
      renderSettingsPage();
      const deleteButton = screen.getByRole('button', { name: /Delete Account/i });

      fireEvent.click(deleteButton);

      await waitFor(() => {
        expect(screen.getByText(/Are you absolutely sure/i)).toBeInTheDocument();
      });

      const cancelButton = screen.getByRole('button', { name: /Cancel/i });
      fireEvent.click(cancelButton);

      await waitFor(() => {
        expect(screen.queryByText(/Are you absolutely sure/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper heading hierarchy', () => {
      renderSettingsPage();

      const mainHeading = screen.getByRole('heading', { name: /^Settings$/i });
      expect(mainHeading.tagName).toBe('H1');

      const sectionHeadings = screen.getAllByRole('heading', { level: 2 });
      expect(sectionHeadings.length).toBeGreaterThan(0);
    });

    it('should have accessible form labels', () => {
      renderSettingsPage();

      expect(screen.getByLabelText(/Display Name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Bio/i)).toBeInTheDocument();
    });

    it('should have accessible avatar upload', () => {
      renderSettingsPage();
      expect(screen.getByLabelText(/Upload avatar/i)).toBeInTheDocument();
    });
  });

  describe('Layout and Styling', () => {
    it('should apply theme-aware classes', () => {
      const { container } = renderSettingsPage();

      const mainContainer = container.querySelector('.min-h-screen');
      expect(mainContainer).toBeInTheDocument();
      expect(mainContainer).toHaveClass('bg-background', 'text-foreground');
    });

    it('should have proper spacing between sections', () => {
      const { container } = renderSettingsPage();

      const sectionsContainer = container.querySelector('.space-y-6');
      expect(sectionsContainer).toBeInTheDocument();
    });

    it('should render sections with borders and sharp corners', () => {
      const { container } = renderSettingsPage();

      const sections = container.querySelectorAll('section');
      sections.forEach((section) => {
        expect(section).toHaveClass('rounded-none');
      });
    });
  });

  describe('Notifications Settings', () => {
    it('should render notification settings component', () => {
      renderSettingsPage();

      // NotificationSettings component should be rendered
      // It has its own heading
      expect(screen.getByRole('heading', { name: /Notifications/i })).toBeInTheDocument();
    });
  });

  describe('Theme State', () => {
    it('should display dark theme when store has dark theme', () => {
      useThemeStore.setState({ theme: 'dark' });

      renderSettingsPage();
      expect(screen.getByText('dark')).toBeInTheDocument();
    });
  });
});
