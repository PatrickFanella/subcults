/**
 * ErrorBoundary Component Tests
 * Validates error catching, fallback rendering, and accessibility
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { ErrorBoundary } from './ErrorBoundary';

// Component that throws an error
function ThrowError({ shouldThrow = false }: { shouldThrow?: boolean }) {
  if (shouldThrow) {
    throw new Error('Test error message');
  }
  return <div>No error</div>;
}

describe('ErrorBoundary', () => {
  beforeEach(() => {
    // Suppress console errors in tests
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  it('renders children when no error occurs', () => {
    render(
      <ErrorBoundary>
        <div data-testid="child">Child content</div>
      </ErrorBoundary>
    );

    expect(screen.getByTestId('child')).toBeInTheDocument();
    expect(screen.getByText('Child content')).toBeInTheDocument();
  });

  it('catches rendering errors and displays fallback UI', () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    // Check fallback UI is displayed
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText(/An unexpected error occurred/)).toBeInTheDocument();
  });

  it('displays reload and report buttons', () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    const reloadButton = screen.getByRole('button', { name: /reload page/i });
    const reportButton = screen.getByRole('button', { name: /report issue/i });

    expect(reloadButton).toBeInTheDocument();
    expect(reportButton).toBeInTheDocument();
  });

  it('has proper accessibility attributes', () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    const alertContainer = screen.getByRole('alert');
    expect(alertContainer).toHaveAttribute('aria-live', 'assertive');
    expect(alertContainer).toHaveAttribute('aria-atomic', 'true');

    // Buttons should have aria-describedby
    const reloadButton = screen.getByRole('button', { name: /reload page/i });
    expect(reloadButton).toHaveAttribute('aria-describedby', 'error-title');
  });

  // Note: Focus management is tested manually as componentDidUpdate behavior
  // is difficult to test reliably in JSDOM environment

  it('reloads page when reload button is clicked', async () => {
    const user = userEvent.setup();

    // Mock window.location.reload using Object.defineProperty
    const reloadMock = vi.fn();
    Object.defineProperty(window, 'location', {
      value: { reload: reloadMock },
      writable: true,
    });

    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    const reloadButton = screen.getByRole('button', { name: /reload page/i });
    await user.click(reloadButton);

    expect(reloadMock).toHaveBeenCalledOnce();
  });

  it('opens email client when report button is clicked', async () => {
    const user = userEvent.setup();

    // Mock window.location.href setter
    let capturedHref = '';
    Object.defineProperty(window, 'location', {
      value: {
        get href() {
          return capturedHref;
        },
        set href(value) {
          capturedHref = value;
        },
      },
      writable: true,
    });

    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    const reportButton = screen.getByRole('button', { name: /report issue/i });
    await user.click(reportButton);

    // Check that mailto: link was attempted
    expect(capturedHref).toContain('mailto:');
    expect(capturedHref).toContain('support@subcults.com');
    expect(capturedHref).toContain('Error%20Report');
  });

  it('renders custom fallback when provided', () => {
    const customFallback = <div data-testid="custom-fallback">Custom error UI</div>;

    render(
      <ErrorBoundary fallback={customFallback}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByTestId('custom-fallback')).toBeInTheDocument();
    expect(screen.getByText('Custom error UI')).toBeInTheDocument();
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
  });

  it('shows error details in development mode', () => {
    // Simulate development mode
    import.meta.env.DEV = true;
    import.meta.env.PROD = false;

    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    // Details should be present in DEV mode
    const details = screen.getByText(/error details \(development only\)/i);
    expect(details).toBeInTheDocument();
  });

  it('hides error details in production mode', () => {
    // Simulate production mode
    import.meta.env.DEV = false;
    import.meta.env.PROD = true;

    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    // Details should not be present in PROD mode
    expect(screen.queryByText(/error details/i)).not.toBeInTheDocument();
  });

  it('logs error in development mode', () => {
    const consoleWarnSpy = vi.spyOn(console, 'warn');
    import.meta.env.DEV = true;

    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(consoleWarnSpy).toHaveBeenCalledWith(
      '[ErrorBoundary] Rendering error caught:',
      expect.objectContaining({
        message: 'Test error message',
      })
    );
  });
});
