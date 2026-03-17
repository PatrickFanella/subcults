/**
 * LoadingSpinner Component Tests
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { LoadingSpinner, FullPageLoader } from './LoadingSpinner';

describe('LoadingSpinner', () => {
  describe('Rendering', () => {
    it('renders with default props', () => {
      render(<LoadingSpinner />);
      expect(screen.getByRole('status', { name: 'Loading' })).toBeInTheDocument();
    });

    it('renders with custom label', () => {
      render(<LoadingSpinner label="Processing" />);
      expect(screen.getByRole('status', { name: 'Processing' })).toBeInTheDocument();
    });

    it('renders all sizes', () => {
      const { rerender } = render(<LoadingSpinner size="sm" />);
      expect(screen.getByRole('status')).toHaveClass('h-4', 'w-4');

      rerender(<LoadingSpinner size="md" />);
      expect(screen.getByRole('status')).toHaveClass('h-6', 'w-6');

      rerender(<LoadingSpinner size="lg" />);
      expect(screen.getByRole('status')).toHaveClass('h-8', 'w-8');

      rerender(<LoadingSpinner size="xl" />);
      expect(screen.getByRole('status')).toHaveClass('h-12', 'w-12');
    });
  });

  describe('Accessibility', () => {
    it('has status role', () => {
      render(<LoadingSpinner />);
      expect(screen.getByRole('status')).toBeInTheDocument();
    });

    it('has accessible label', () => {
      render(<LoadingSpinner label="Custom loading" />);
      expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Custom loading');
    });

    it('has visually hidden text for screen readers', () => {
      const { container } = render(<LoadingSpinner label="Test" />);
      const srOnly = container.querySelector('.sr-only');
      expect(srOnly).toHaveTextContent('Test');
    });
  });

  describe('Styling', () => {
    it('applies custom className', () => {
      render(<LoadingSpinner className="custom-spinner" />);
      expect(screen.getByRole('status')).toHaveClass('custom-spinner');
    });

    it('uses Tailwind classes only (no inline styles)', () => {
      const { container } = render(<LoadingSpinner />);
      const spinner = container.querySelector('[role="status"]');
      expect(spinner?.getAttribute('style')).toBeNull();
    });

    it('has animate-spin class', () => {
      render(<LoadingSpinner />);
      expect(screen.getByRole('status')).toHaveClass('animate-spin');
    });
  });
});

describe('FullPageLoader', () => {
  describe('Rendering', () => {
    it('renders with default props', () => {
      render(<FullPageLoader />);
      expect(screen.getByRole('status', { name: 'Loading content' })).toBeInTheDocument();
      expect(screen.getByText('Loading content...')).toBeInTheDocument();
    });

    it('renders without text when showText is false', () => {
      render(<FullPageLoader showText={false} />);
      expect(screen.queryByText('Loading content...')).not.toBeInTheDocument();
    });

    it('renders with custom label', () => {
      render(<FullPageLoader label="Fetching data" />);
      expect(screen.getByText('Fetching data...')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('has status role', () => {
      render(<FullPageLoader />);
      expect(screen.getByRole('status')).toBeInTheDocument();
    });

    it('has aria-live and aria-busy attributes', () => {
      const { container } = render(<FullPageLoader />);
      const status = container.querySelector('[role="status"]');
      expect(status).toHaveAttribute('aria-live', 'polite');
      expect(status).toHaveAttribute('aria-busy', 'true');
    });
  });

  describe('Styling', () => {
    it('uses Tailwind classes only (no inline styles)', () => {
      const { container } = render(<FullPageLoader />);
      const wrapper = container.querySelector('[role="status"]');
      expect(wrapper?.getAttribute('style')).toBeNull();
    });

    it('centers content on full page', () => {
      const { container } = render(<FullPageLoader />);
      const wrapper = container.querySelector('[role="status"]');
      expect(wrapper).toHaveClass('flex', 'items-center', 'justify-center', 'h-screen');
    });
  });
});
