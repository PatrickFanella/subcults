/**
 * ToastContainer Component Tests
 * Validates toast rendering, dismissal, and accessibility
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { ToastContainer } from './ToastContainer';
import { useToastStore } from '../stores/toastStore';

describe('ToastContainer', () => {
  beforeEach(() => {
    // Clear toasts before each test
    useToastStore.setState({ toasts: [] });
  });

  it('renders nothing when no toasts exist', () => {
    const { container } = render(<ToastContainer />);
    expect(container.firstChild).toBeNull();
  });

  it('renders toasts when they exist', () => {
    useToastStore.setState({
      toasts: [
        { id: '1', type: 'success', message: 'Success message', duration: 5000, dismissible: true },
        { id: '2', type: 'error', message: 'Error message', duration: 5000, dismissible: true },
      ],
    });

    render(<ToastContainer />);

    expect(screen.getByText('Success message')).toBeInTheDocument();
    expect(screen.getByText('Error message')).toBeInTheDocument();
  });

  it('displays different icons for different toast types', () => {
    useToastStore.setState({
      toasts: [
        { id: '1', type: 'success', message: 'Success', duration: 5000, dismissible: true },
        { id: '2', type: 'error', message: 'Error', duration: 5000, dismissible: true },
        { id: '3', type: 'info', message: 'Info', duration: 5000, dismissible: true },
      ],
    });

    const { container } = render(<ToastContainer />);

    // Check for icons (using text content)
    expect(container.textContent).toContain('✓'); // success
    expect(container.textContent).toContain('✕'); // error
    expect(container.textContent).toContain('ℹ'); // info
  });

  it('shows dismiss button when dismissible is true', () => {
    useToastStore.setState({
      toasts: [
        { id: '1', type: 'info', message: 'Dismissible toast', duration: 5000, dismissible: true },
      ],
    });

    render(<ToastContainer />);

    const dismissButton = screen.getByRole('button', { name: /dismiss notification/i });
    expect(dismissButton).toBeInTheDocument();
  });

  it('hides dismiss button when dismissible is false', () => {
    useToastStore.setState({
      toasts: [
        {
          id: '1',
          type: 'info',
          message: 'Non-dismissible toast',
          duration: 5000,
          dismissible: false,
        },
      ],
    });

    render(<ToastContainer />);

    expect(screen.queryByRole('button', { name: /dismiss notification/i })).not.toBeInTheDocument();
  });

  it('removes toast when dismiss button is clicked', async () => {
    const user = userEvent.setup();

    useToastStore.setState({
      toasts: [{ id: '1', type: 'info', message: 'Test toast', duration: 5000, dismissible: true }],
    });

    render(<ToastContainer />);

    expect(screen.getByText('Test toast')).toBeInTheDocument();

    const dismissButton = screen.getByRole('button', { name: /dismiss notification/i });
    await user.click(dismissButton);

    await waitFor(() => {
      expect(screen.queryByText('Test toast')).not.toBeInTheDocument();
    });
  });

  it('has proper accessibility attributes', () => {
    useToastStore.setState({
      toasts: [
        {
          id: '1',
          type: 'success',
          message: 'Accessible toast',
          duration: 5000,
          dismissible: true,
        },
      ],
    });

    render(<ToastContainer />);

    // Container should have region role and aria-live
    const region = screen.getByRole('region', { name: /notifications/i });
    expect(region).toHaveAttribute('aria-live', 'polite');

    // Individual toast should have status role and aria-live
    const toastStatus = screen.getByRole('status');
    expect(toastStatus).toHaveAttribute('aria-live', 'polite');
    expect(toastStatus).toHaveAttribute('aria-atomic', 'true');
  });

  it('renders multiple toasts in order', () => {
    useToastStore.setState({
      toasts: [
        { id: '1', type: 'info', message: 'First toast', duration: 5000, dismissible: true },
        { id: '2', type: 'info', message: 'Second toast', duration: 5000, dismissible: true },
        { id: '3', type: 'info', message: 'Third toast', duration: 5000, dismissible: true },
      ],
    });

    render(<ToastContainer />);

    const toasts = screen.getAllByRole('status');
    expect(toasts).toHaveLength(3);
    expect(toasts[0]).toHaveTextContent('First toast');
    expect(toasts[1]).toHaveTextContent('Second toast');
    expect(toasts[2]).toHaveTextContent('Third toast');
  });

  it('updates when toasts are added dynamically', async () => {
    const { rerender } = render(<ToastContainer />);

    expect(screen.queryByRole('status')).not.toBeInTheDocument();

    // Add toast - wrap setState in act
    act(() => {
      useToastStore.setState({
        toasts: [
          { id: '1', type: 'success', message: 'Dynamic toast', duration: 5000, dismissible: true },
        ],
      });
    });

    rerender(<ToastContainer />);

    await waitFor(() => {
      expect(screen.getByText('Dynamic toast')).toBeInTheDocument();
    });
  });

  it('applies correct Tailwind classes for toast types', () => {
    useToastStore.setState({
      toasts: [{ id: '1', type: 'success', message: 'Success', duration: 5000, dismissible: true }],
    });

    render(<ToastContainer />);

    const toast = screen.getByRole('status');
    expect(toast).toHaveClass('bg-status-success');
  });

  it('uses Tailwind animation classes', () => {
    useToastStore.setState({
      toasts: [
        { id: '1', type: 'info', message: 'Animated toast', duration: 5000, dismissible: true },
      ],
    });

    const { container } = render(<ToastContainer />);

    // Check that toast has animation class
    const toast = container.querySelector('[role="status"]');
    expect(toast).toHaveClass('animate-slide-in');
  });

  it('positions toast container in top-right corner with Tailwind classes', () => {
    useToastStore.setState({
      toasts: [
        { id: '1', type: 'info', message: 'Positioned toast', duration: 5000, dismissible: true },
      ],
    });

    render(<ToastContainer />);

    const region = screen.getByRole('region', { name: /notifications/i });
    expect(region).toHaveClass('fixed', 'top-4', 'right-4');
  });
});
