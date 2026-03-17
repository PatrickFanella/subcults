/**
 * ProfileDropdown Tests
 * Validates profile dropdown rendering, user interactions, and accessibility
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { ProfileDropdown } from './ProfileDropdown';
import { authStore } from '../stores/authStore';

// Mock useNavigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const renderProfileDropdown = () => {
  const router = createMemoryRouter(
    [
      {
        path: '/',
        element: <ProfileDropdown />,
      },
    ],
    {
      future: {
        v7_startTransition: true,
        v7_relativeSplatPath: true,
      },
    }
  );

  return render(<RouterProvider router={router} />);
};

describe('ProfileDropdown', () => {
  beforeEach(() => {
    authStore.resetForTesting();
    mockNavigate.mockClear();
  });

  afterEach(() => {
    authStore.resetForTesting();
  });

  it('renders nothing when user is not authenticated', () => {
    const { container } = renderProfileDropdown();

    expect(container.firstChild).toBeNull();
  });

  it('renders profile button when user is authenticated', () => {
    authStore.setUser({ did: 'did:example:test-user', role: 'user' }, 'test-token');

    const { container } = renderProfileDropdown();

    const button = screen.getByRole('button', { expanded: false });
    expect(button).toBeInTheDocument();

    // Should show avatar with initials
    expect(container.querySelector('.bg-brand-primary')).toBeInTheDocument();
  });

  it('displays DID initials in avatar', () => {
    authStore.setUser({ did: 'did:example:test-user', role: 'user' }, 'test-token');

    renderProfileDropdown();

    // DID: "did:example:test-user" -> slice(4, 6) -> "ex" -> "EX"
    expect(screen.getByText('EX')).toBeInTheDocument();
  });

  it('handles malformed DID with fallback initials', () => {
    authStore.setUser({ did: 'did:', role: 'user' }, 'test-token');

    renderProfileDropdown();

    // Should show fallback initials
    expect(screen.getByText('??')).toBeInTheDocument();
  });

  it('opens dropdown menu when button is clicked', async () => {
    authStore.setUser({ did: 'did:example:test-user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button', { expanded: false });
    await user.click(button);

    // Menu should be open
    expect(screen.getByRole('menu')).toBeInTheDocument();
    expect(screen.getByRole('button', { expanded: true })).toBeInTheDocument();
  });

  it('displays user DID in dropdown', async () => {
    authStore.setUser({ did: 'did:example:test-user-12345', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    // Should show full DID (or truncated version)
    expect(screen.getByText(/did:example:test-user-12345/)).toBeInTheDocument();
    expect(screen.getByText('Signed in as')).toBeInTheDocument();
  });

  it('truncates long DIDs in dropdown', async () => {
    const longDid = 'did:example:very-long-test-user-identifier-that-exceeds-30-characters';
    authStore.setUser({ did: longDid, role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    // Should show truncated DID with ellipsis
    const truncated = longDid.slice(0, 30) + '...';
    expect(screen.getByText(truncated)).toBeInTheDocument();
  });

  it('shows admin badge for admin users', async () => {
    authStore.setUser({ did: 'did:example:admin', role: 'admin' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    // With i18n mock, there are two matches: badge and admin menu item.
    // Ensure the badge exists with its expected styling.
    const adminLabels = screen.getAllByText('navigation.admin');
    expect(adminLabels.length).toBeGreaterThanOrEqual(2);
    expect(adminLabels[0]).toHaveClass('bg-brand-accent');
  });

  it('does not show admin badge for regular users', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    // With i18n mock, uses translation key
    expect(screen.queryByText('navigation.admin')).not.toBeInTheDocument();
  });

  it('shows Account and Settings menu items', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    // With i18n mock, uses translation keys
    expect(screen.getByRole('menuitem', { name: 'profile.account' })).toBeInTheDocument();
    expect(screen.getByRole('menuitem', { name: 'profile.settings' })).toBeInTheDocument();
  });

  it('shows Admin Panel menu item for admin users', async () => {
    authStore.setUser({ did: 'did:example:admin', role: 'admin' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    // With i18n mock, uses translation key
    expect(screen.getByRole('menuitem', { name: 'navigation.admin' })).toBeInTheDocument();
  });

  it('does not show Admin Panel for regular users', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    // With i18n mock, uses translation key
    expect(screen.queryByRole('menuitem', { name: 'navigation.admin' })).not.toBeInTheDocument();
  });

  it('closes dropdown when clicking outside', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    expect(screen.getByRole('menu')).toBeInTheDocument();

    // Click outside
    await user.click(document.body);

    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('closes dropdown when pressing Escape key', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    expect(screen.getByRole('menu')).toBeInTheDocument();

    // Press Escape
    await user.keyboard('{Escape}');

    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('logs out user and navigates to home when Sign out is clicked', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    // With i18n mock, uses translation key
    const signOutButton = screen.getByRole('menuitem', { name: 'profile.logout' });
    await user.click(signOutButton);

    // Should log out and navigate to home
    expect(authStore.getState().user).toBeNull();
    expect(mockNavigate).toHaveBeenCalledWith('/');
  });

  it('closes dropdown when clicking Account link', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    await user.click(button);

    // With i18n mock, uses translation key
    const accountLink = screen.getByRole('menuitem', { name: 'profile.account' });
    await user.click(accountLink);

    // Dropdown should close
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('has proper ARIA attributes', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    renderProfileDropdown();

    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('aria-expanded', 'false');
    expect(button).toHaveAttribute('aria-haspopup', 'true');

    await user.click(button);

    expect(button).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByRole('menu')).toBeInTheDocument();
  });

  it('chevron icon rotates when dropdown is open', async () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    const user = userEvent.setup();

    const { container } = renderProfileDropdown();

    const button = screen.getByRole('button');
    const chevron = container.querySelector('svg:last-of-type');

    // Chevron should not be rotated initially
    expect(chevron).not.toHaveClass('rotate-180');

    await user.click(button);

    // Chevron should be rotated when open
    expect(chevron).toHaveClass('rotate-180');
  });
});
