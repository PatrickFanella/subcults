/**
 * Sidebar Tests
 * Validates sidebar navigation rendering and responsive behavior
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { authStore } from '../stores/authStore';

const renderSidebar = (props = {}) => {
  const router = createMemoryRouter(
    [
      {
        path: '/',
        element: <Sidebar {...props} />,
      },
      {
        path: '/scenes',
        element: <div>Scenes Page</div>,
      },
      {
        path: '/events',
        element: <div>Events Page</div>,
      },
      {
        path: '/account',
        element: <div>Account Page</div>,
      },
      {
        path: '/settings',
        element: <div>Settings Page</div>,
      },
      {
        path: '/admin',
        element: <div>Admin Page</div>,
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

describe('Sidebar', () => {
  beforeEach(() => {
    authStore.resetForTesting();
  });

  it('renders sidebar with proper ARIA label', () => {
    renderSidebar();
    
    expect(screen.getByRole('complementary', { name: 'Sidebar navigation' })).toBeInTheDocument();
  });

  it('renders Discover section with navigation links', () => {
    renderSidebar();
    
    // With i18n mock, text uses translation key
    expect(screen.getByText('navigation.discover')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /map/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /scenes/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /events/i })).toBeInTheDocument();
  });

  it('renders Account section with Settings link', () => {
    renderSidebar();
    
    // With i18n mock, text uses translation key
    expect(screen.getByText('navigation.account')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /settings/i })).toBeInTheDocument();
  });

  it('shows Account link when user is authenticated', () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    
    renderSidebar();
    
    const accountLinks = screen.getAllByRole('link', { name: /account/i });
    expect(accountLinks.length).toBeGreaterThan(0);
  });

  it('does not show Account link when user is not authenticated', () => {
    renderSidebar();
    
    // Only Settings should be present, not Account link
    const links = screen.getAllByRole('link');
    // With i18n mock, text uses translation key
    const accountLinks = links.filter(link => link.textContent?.includes('navigation.account'));
    expect(accountLinks.length).toBe(0);
  });

  it('shows Admin link when user is admin', () => {
    authStore.setUser({ did: 'did:example:admin', role: 'admin' }, 'test-token');
    
    renderSidebar();
    
    expect(screen.getByRole('link', { name: /admin/i })).toBeInTheDocument();
  });

  it('does not show Admin link for regular users', () => {
    authStore.setUser({ did: 'did:example:user', role: 'user' }, 'test-token');
    
    renderSidebar();
    
    expect(screen.queryByRole('link', { name: /admin/i })).not.toBeInTheDocument();
  });

  it('renders Featured Scenes section', () => {
    renderSidebar();
    
    // With i18n mock, text uses translation key
    expect(screen.getByText('navigation.featuredScenes')).toBeInTheDocument();
    expect(screen.getByText('navigation.noFeaturedScenes')).toBeInTheDocument();
  });

  it('renders version in footer', () => {
    renderSidebar();
    
    expect(screen.getByText(/Subcults v/)).toBeInTheDocument();
  });

  it('highlights active route', () => {
    const { container } = renderSidebar();
    
    // Map should be active by default (home route)
    const mapLink = screen.getByRole('link', { name: /map/i });
    expect(mapLink).toHaveClass('bg-brand-primary');
    expect(mapLink).toHaveClass('text-white');
  });

  it('applies correct styles to non-active links', () => {
    renderSidebar();
    
    const scenesLink = screen.getByRole('link', { name: /scenes/i });
    expect(scenesLink).not.toHaveClass('bg-brand-primary');
    expect(scenesLink).toHaveClass('text-foreground');
  });

  it('shows mobile overlay when isOpen is true', () => {
    const onClose = vi.fn();
    const { container } = renderSidebar({ isOpen: true, onClose });
    
    // Overlay should be present (note: bg-black/50 in Tailwind)
    const overlay = container.querySelector('.fixed.inset-0');
    expect(overlay).toBeInTheDocument();
    expect(overlay).toHaveClass('bg-black/50');
    expect(overlay).toHaveClass('lg:hidden');
  });

  it('does not show mobile overlay when isOpen is false', () => {
    const { container } = renderSidebar({ isOpen: false });
    
    // Overlay should not be present (it requires both isOpen and onClose)
    const overlay = container.querySelector('.fixed.inset-0.z-40');
    expect(overlay).not.toBeInTheDocument();
  });

  it('shows close button on mobile', () => {
    renderSidebar({ isOpen: true });
    
    // With i18n mock, aria-label uses translation key
    const closeButton = screen.getByRole('button', { name: 'actions.close' });
    expect(closeButton).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', async () => {
    const handleClose = vi.fn();
    const user = userEvent.setup();
    
    renderSidebar({ isOpen: true, onClose: handleClose });
    
    // With i18n mock, aria-label uses translation key
    const closeButton = screen.getByRole('button', { name: 'actions.close' });
    await user.click(closeButton);
    
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when overlay is clicked', async () => {
    const handleClose = vi.fn();
    const user = userEvent.setup();
    
    const { container } = renderSidebar({ isOpen: true, onClose: handleClose });
    
    const overlay = container.querySelector('.fixed.inset-0.bg-black\\/50');
    await user.click(overlay!);
    
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when navigation link is clicked', async () => {
    const handleClose = vi.fn();
    const user = userEvent.setup();
    
    renderSidebar({ isOpen: true, onClose: handleClose });
    
    const scenesLink = screen.getByRole('link', { name: /scenes/i });
    await user.click(scenesLink);
    
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('does not call onClose when onClose is not provided', async () => {
    const user = userEvent.setup();
    
    renderSidebar({ isOpen: true });
    
    // Should not throw error when clicking links
    const scenesLink = screen.getByRole('link', { name: /scenes/i });
    await user.click(scenesLink);
    
    // Test passes if no error is thrown
    expect(true).toBe(true);
  });

  it('applies custom className', () => {
    const { container } = renderSidebar({ className: 'custom-sidebar-class' });
    
    const sidebar = container.querySelector('aside');
    expect(sidebar).toHaveClass('custom-sidebar-class');
  });

  it('has proper responsive classes', () => {
    const { container } = renderSidebar({ isOpen: false });
    
    const sidebar = container.querySelector('aside');
    
    // Should have responsive transform classes
    expect(sidebar).toHaveClass('fixed');
    expect(sidebar).toHaveClass('lg:static');
    expect(sidebar).toHaveClass('transform');
    expect(sidebar).toHaveClass('transition-transform');
    // When closed, should have -translate-x-full
    expect(sidebar?.className).toContain('-translate-x-full');
  });

  it('shows sidebar when isOpen is true on mobile', () => {
    const { container } = renderSidebar({ isOpen: true });
    
    const sidebar = container.querySelector('aside');
    expect(sidebar?.className).toContain('translate-x-0');
    expect(sidebar?.className).not.toContain('-translate-x-full');
  });

  it('hides sidebar when isOpen is false on mobile', () => {
    const { container } = renderSidebar({ isOpen: false });
    
    const sidebar = container.querySelector('aside');
    expect(sidebar?.className).toContain('-translate-x-full');
  });

  it('renders navigation icons', () => {
    renderSidebar();
    
    // Check for emoji icons
    expect(screen.getByText('🗺️')).toBeInTheDocument();
    expect(screen.getByText('🎭')).toBeInTheDocument();
    expect(screen.getByText('📅')).toBeInTheDocument();
    expect(screen.getByText('⚙️')).toBeInTheDocument();
  });

  it('renders admin icon when user is admin', () => {
    authStore.setUser({ did: 'did:example:admin', role: 'admin' }, 'test-token');
    
    renderSidebar();
    
    expect(screen.getByText('🔧')).toBeInTheDocument();
  });

  it('shows menu heading on mobile', () => {
    renderSidebar({ isOpen: true });
    
    // With i18n mock, text uses translation key
    expect(screen.getByRole('heading', { name: 'navigation.menu', level: 2 })).toBeInTheDocument();
  });

  it('close button SVG is hidden from screen readers', () => {
    const { container } = renderSidebar({ isOpen: true });
    
    // With i18n mock, aria-label uses translation key
    const closeButton = screen.getByRole('button', { name: 'actions.close' });
    const svg = closeButton.querySelector('svg');
    
    expect(svg).toHaveAttribute('aria-hidden', 'true');
  });
});
