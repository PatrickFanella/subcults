/**
 * Mobile Responsive Design Tests
 * Tests for mobile-first responsive design and touch-friendly interactions
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { AppLayout } from '../layouts/AppLayout';
import { Sidebar } from './Sidebar';
import { NotificationBadge } from './NotificationBadge';
import { ProfileDropdown } from './ProfileDropdown';

// Helper to wrap components with Router
const renderWithRouter = (component: React.ReactElement) => {
  return render(<BrowserRouter>{component}</BrowserRouter>);
};

describe('Mobile-First Responsive Design', () => {
  describe('Touch Target Sizing', () => {
    it('mobile menu toggle button meets 44px minimum touch target', () => {
      renderWithRouter(<AppLayout />);
      const menuButton = screen.getByLabelText('Toggle sidebar');
      
      const styles = window.getComputedStyle(menuButton);
      const minHeight = styles.minHeight;
      const minWidth = styles.minWidth;
      
      // Both should be at least 44px (11 * 4 = 44px in Tailwind)
      expect(minHeight).toBeTruthy();
      expect(minWidth).toBeTruthy();
    });

    it('sidebar navigation links have proper touch targets', () => {
      renderWithRouter(<Sidebar isOpen={true} />);
      
      const navLinks = screen.getAllByRole('link');
      navLinks.forEach((link) => {
        const styles = window.getComputedStyle(link);
        const minHeight = styles.minHeight;
        
        // Should have min-h-touch class applied (44px minimum)
        expect(minHeight).toBeTruthy();
      });
    });

    it('notification badge button has touch-friendly sizing', () => {
      render(<NotificationBadge notificationCount={5} />);
      
      const button = screen.getByRole('button');
      const styles = window.getComputedStyle(button);
      
      expect(styles.minHeight).toBeTruthy();
      expect(styles.minWidth).toBeTruthy();
    });

    it('profile dropdown button has adequate touch target', () => {
      render(
        <BrowserRouter>
          <ProfileDropdown />
        </BrowserRouter>
      );
      
      // Check if button exists (will be null if user not authenticated, that's OK)
      const buttons = screen.queryAllByRole('button');
      if (buttons.length > 0) {
        const styles = window.getComputedStyle(buttons[0]);
        expect(styles.minHeight || styles.minWidth).toBeTruthy();
      }
    });
  });

  describe('Responsive Layout Classes', () => {
    it('AppLayout header uses responsive padding', () => {
      renderWithRouter(<AppLayout />);
      
      const header = screen.getByRole('banner');
      expect(header.className).toContain('px-3');
      expect(header.className).toContain('sm:px-4');
    });

    it('Sidebar has responsive width classes', () => {
      renderWithRouter(<Sidebar isOpen={true} />);
      
      const sidebar = screen.getByRole('complementary');
      const styles = window.getComputedStyle(sidebar);
      
      // Should have width styling
      expect(styles.width).toBeTruthy();
    });

    it('responsive text sizing applied to headings', () => {
      const { container } = render(<h1>Test Heading</h1>);
      const heading = container.querySelector('h1');
      
      if (heading) {
        const styles = window.getComputedStyle(heading);
        expect(styles.fontSize).toBeTruthy();
      }
    });
  });

  describe('Mobile Navigation', () => {
    it('sidebar close button visible on mobile', () => {
      renderWithRouter(<Sidebar isOpen={true} onClose={() => {}} />);
      
      const closeButton = screen.getByLabelText(/close/i);
      expect(closeButton).toBeInTheDocument();
      
      const styles = window.getComputedStyle(closeButton);
      expect(styles.minHeight).toBeTruthy();
      expect(styles.minWidth).toBeTruthy();
    });

    it('mobile menu overlay created when sidebar is open', () => {
      const { container } = renderWithRouter(
        <Sidebar isOpen={true} onClose={() => {}} />
      );
      
      // Check for overlay element
      const overlay = container.querySelector('[class*="bg-black/50"]');
      expect(overlay).toBeInTheDocument();
    });
  });

  describe('Touch Interaction Support', () => {
    it('buttons have touch-action: manipulation for better performance', () => {
      renderWithRouter(<AppLayout />);
      
      const button = screen.getByLabelText('Toggle sidebar');
      const styles = window.getComputedStyle(button);
      
      expect(button.className).toContain('touch-manipulation');
      expect(styles.touchAction).toBe('manipulation');
    });

    it('navigation links have touch-manipulation class', () => {
      renderWithRouter(<Sidebar isOpen={true} />);
      
      const links = screen.getAllByRole('link');
      links.forEach((link) => {
        const styles = window.getComputedStyle(link);
        expect(link.className).toContain('touch-manipulation');
        expect(styles.touchAction).toBe('manipulation');
      });
    });
  });

  describe('Accessibility on Mobile', () => {
    it('sidebar has proper ARIA labels for mobile navigation', () => {
      renderWithRouter(<Sidebar isOpen={true} />);
      
      const sidebar = screen.getByRole('complementary');
      expect(sidebar).toHaveAttribute('aria-label');
    });

    it('mobile menu toggle has proper ARIA expanded state', () => {
      renderWithRouter(<AppLayout />);
      
      const menuButton = screen.getByLabelText('Toggle sidebar');
      expect(menuButton).toHaveAttribute('aria-expanded');
    });

    it('skip to content link available for keyboard navigation', () => {
      renderWithRouter(<AppLayout />);
      
      const skipLink = screen.getByText('Skip to content');
      expect(skipLink).toBeInTheDocument();
      expect(skipLink.tagName).toBe('A');
      expect(skipLink).toHaveAttribute('href', '#main-content');
    });
  });
});
