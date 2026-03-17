/**
 * LoadingSkeleton Component Tests
 * Validates loading state rendering and accessibility
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { LoadingSkeleton } from './LoadingSkeleton';

describe('LoadingSkeleton', () => {
  it('renders loading message', () => {
    render(<LoadingSkeleton />);

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('has proper ARIA attributes for accessibility', () => {
    render(<LoadingSkeleton />);

    const skeleton = screen.getByRole('status');
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute('aria-live', 'polite');
    expect(skeleton).toHaveAttribute('aria-busy', 'true');
    expect(skeleton).toHaveAttribute('aria-label', 'Loading content');
  });

  it('renders spinner element with Tailwind animation', () => {
    const { container } = render(<LoadingSkeleton />);

    // Check for spinner div by animation class
    const spinnerDiv = container.querySelector('.animate-spin');
    expect(spinnerDiv).toBeTruthy();
  });

  it('applies correct Tailwind layout classes', () => {
    render(<LoadingSkeleton />);

    const loadingStatus = screen.getByRole('status');
    expect(loadingStatus).toBeInTheDocument();
    expect(loadingStatus).toHaveClass('flex', 'items-center', 'justify-center', 'h-full');
  });

  it('has dark background via Tailwind', () => {
    render(<LoadingSkeleton />);

    const loadingSkeleton = screen.getByRole('status');
    expect(loadingSkeleton).toHaveClass('bg-background-secondary', 'text-foreground');
  });

  it('uses Tailwind animate-spin class', () => {
    const { container } = render(<LoadingSkeleton />);

    const spinner = container.querySelector('.animate-spin');
    expect(spinner).toHaveClass('animate-spin');
  });

  it('centers content both horizontally and vertically', () => {
    render(<LoadingSkeleton />);

    const loadingStatus = screen.getByRole('status');
    expect(loadingStatus).toHaveClass('items-center', 'justify-center');

    // Check the loading text exists
    const loadingText = screen.getByText('Loading...');
    expect(loadingText).toBeInTheDocument();
  });

  it('renders spinner with circular border via Tailwind', () => {
    const { container } = render(<LoadingSkeleton />);

    const spinner = container.querySelector('.animate-spin');
    expect(spinner).toBeTruthy();
    expect(spinner).toHaveClass('rounded-full');
    expect(spinner).toHaveClass('w-[50px]', 'h-[50px]');
  });

  it('spinner has visible border with Tailwind classes', () => {
    const { container } = render(<LoadingSkeleton />);

    const spinner = container.querySelector('.animate-spin');
    expect(spinner).toHaveClass('border-4');
    expect(spinner).toHaveClass('border-border', 'border-t-neon-purple');
  });

  it('spinner has margin below it via Tailwind', () => {
    const { container } = render(<LoadingSkeleton />);

    const spinner = container.querySelector('.animate-spin');
    expect(spinner).toHaveClass('mb-4', 'mx-auto');
  });

  it('uses Tailwind classes only (no inline styles)', () => {
    const { container } = render(<LoadingSkeleton />);

    // Check that no inline styles are used
    const elementsWithStyle = container.querySelectorAll('[style]');
    expect(elementsWithStyle.length).toBe(0);
  });
});
