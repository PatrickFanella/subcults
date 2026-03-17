/**
 * Input Component Tests
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Input } from './Input';

describe('Input', () => {
  describe('Rendering', () => {
    it('renders basic input', () => {
      render(<Input placeholder="Enter text" />);
      expect(screen.getByPlaceholderText('Enter text')).toBeInTheDocument();
    });

    it('renders with label', () => {
      render(<Input label="Username" />);
      expect(screen.getByLabelText('Username')).toBeInTheDocument();
    });

    it('renders with required indicator', () => {
      render(<Input label="Email" required />);
      expect(screen.getByLabelText('required')).toBeInTheDocument();
      expect(screen.getByLabelText(/Email/)).toBeRequired();
    });

    it('renders with helper text', () => {
      render(<Input label="Password" helperText="Must be at least 8 characters" />);
      expect(screen.getByText('Must be at least 8 characters')).toBeInTheDocument();
    });

    it('renders with error message', () => {
      render(<Input label="Email" error="Invalid email format" />);
      const errorMessage = screen.getByRole('alert');
      expect(errorMessage).toHaveTextContent('Invalid email format');
    });

    it('shows error message instead of helper text when both provided', () => {
      render(<Input label="Email" helperText="Enter your email" error="Invalid email" />);
      expect(screen.getByRole('alert')).toHaveTextContent('Invalid email');
      expect(screen.queryByText('Enter your email')).not.toBeInTheDocument();
    });

    it('renders full width when specified', () => {
      render(<Input fullWidth />);
      const wrapper = screen.getByRole('textbox').parentElement;
      expect(wrapper).toHaveClass('w-full');
    });
  });

  describe('States', () => {
    it('applies error state styles', () => {
      render(<Input error="Error message" />);
      const input = screen.getByRole('textbox');
      expect(input).toHaveClass('border-status-error');
      expect(input).toHaveAttribute('aria-invalid', 'true');
    });

    it('applies success state styles', () => {
      render(<Input success />);
      const input = screen.getByRole('textbox');
      expect(input).toHaveClass('border-status-success');
    });

    it('applies disabled state', () => {
      render(<Input disabled />);
      expect(screen.getByRole('textbox')).toBeDisabled();
    });
  });

  describe('Interaction', () => {
    it('accepts user input', async () => {
      const user = userEvent.setup();
      render(<Input placeholder="Type here" />);

      const input = screen.getByPlaceholderText('Type here');
      await user.type(input, 'Hello');

      expect(input).toHaveValue('Hello');
    });

    it('calls onChange handler', async () => {
      const handleChange = vi.fn();
      const user = userEvent.setup();

      render(<Input onChange={handleChange} />);
      const input = screen.getByRole('textbox');

      await user.type(input, 'A');
      expect(handleChange).toHaveBeenCalled();
    });

    it('does not accept input when disabled', async () => {
      const user = userEvent.setup();
      render(<Input disabled value="" />);

      const input = screen.getByRole('textbox');
      await user.type(input, 'Test').catch(() => {
        // Expected to fail for disabled input
      });

      expect(input).toHaveValue('');
    });
  });

  describe('Accessibility', () => {
    it('associates label with input', () => {
      render(<Input label="Username" id="username" />);
      const input = screen.getByLabelText('Username');
      expect(input).toHaveAttribute('id', 'username');
    });

    it('generates unique ID when not provided', () => {
      const { container } = render(<Input label="Test" />);
      const input = container.querySelector('input');
      expect(input?.getAttribute('id')).toBeTruthy();
    });

    it('associates error message with input via aria-describedby', () => {
      render(<Input label="Email" error="Invalid email" />);
      const input = screen.getByLabelText('Email');
      const errorId = screen.getByRole('alert').getAttribute('id');

      expect(input.getAttribute('aria-describedby')).toContain(errorId as string);
    });

    it('associates helper text with input via aria-describedby', () => {
      render(<Input label="Password" helperText="Helper text" />);
      const input = screen.getByLabelText('Password');
      const helperText = screen.getByText('Helper text');
      const helperId = helperText.getAttribute('id');

      expect(input.getAttribute('aria-describedby')).toContain(helperId as string);
    });

    it('has visible focus indicator', () => {
      render(<Input />);
      expect(screen.getByRole('textbox')).toHaveClass('focus-visible:ring-2');
    });
  });

  describe('Styling', () => {
    it('applies custom className to input', () => {
      render(<Input className="custom-input" />);
      expect(screen.getByRole('textbox')).toHaveClass('custom-input');
    });

    it('applies custom className to wrapper', () => {
      render(<Input wrapperClassName="custom-wrapper" />);
      const wrapper = screen.getByRole('textbox').parentElement;
      expect(wrapper).toHaveClass('custom-wrapper');
    });

    it('uses Tailwind classes only (no inline styles)', () => {
      const { container } = render(<Input label="Test" />);
      const input = container.querySelector('input');
      expect(input?.getAttribute('style')).toBeNull();
    });

    it('has theme transition class', () => {
      render(<Input />);
      expect(screen.getByRole('textbox')).toHaveClass('theme-transition');
    });
  });

  describe('Forward Ref', () => {
    it('forwards ref correctly', () => {
      const ref = vi.fn();
      render(<Input ref={ref} />);
      expect(ref).toHaveBeenCalled();
    });
  });
});
