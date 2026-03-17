/**
 * Input Component
 * Unified input component with consistent styling and validation states
 *
 * Features:
 * - Clear focus indicators (WCAG AA compliant)
 * - Validation states (error, success)
 * - Support for labels and helper text
 * - Accessible error messages
 * - Consistent sizing with design system
 */

import React from 'react';

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  /**
   * Input label
   */
  label?: string;

  /**
   * Helper text displayed below input
   */
  helperText?: string;

  /**
   * Error message (shows error state)
   */
  error?: string;

  /**
   * Success state
   * @default false
   */
  success?: boolean;

  /**
   * Full width input
   * @default false
   */
  fullWidth?: boolean;

  /**
   * Additional CSS classes for input element
   */
  className?: string;

  /**
   * Additional CSS classes for wrapper div
   */
  wrapperClassName?: string;
}

/**
 * Input component with design system styling
 */
export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  (
    {
      label,
      helperText,
      error,
      success = false,
      fullWidth = false,
      className = '',
      wrapperClassName = '',
      id,
      disabled,
      required,
      ...props
    },
    ref
  ) => {
    // Generate unique ID if not provided (for label association)
    const inputId = id || `input-${React.useId()}`;
    const helperTextId = helperText ? `${inputId}-helper` : undefined;
    const errorId = error ? `${inputId}-error` : undefined;

    const hasError = Boolean(error);

    const baseInputStyles = `
      w-full px-3 py-2 rounded-none
      bg-background border
      text-foreground placeholder:text-foreground-muted
      transition-none
      theme-transition
      focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-1
      disabled:opacity-50 disabled:cursor-not-allowed
    `;

    const stateStyles = hasError
      ? 'border-status-error focus-visible:ring-status-error focus:border-status-error'
      : success
        ? 'border-status-success focus-visible:ring-status-success focus:border-status-success'
        : 'border-border focus-visible:ring-brand-primary focus:border-brand-primary';

    const widthStyle = fullWidth ? 'w-full' : '';
    const combinedInputClassName = `${baseInputStyles} ${stateStyles} ${className}`.trim();
    const combinedWrapperClassName = `${widthStyle} ${wrapperClassName}`.trim();

    return (
      <div className={combinedWrapperClassName}>
        {label && (
          <label htmlFor={inputId} className="block text-sm font-medium text-foreground mb-1.5">
            {label}
            {required && (
              <span className="text-status-error ml-1" aria-label="required">
                *
              </span>
            )}
          </label>
        )}

        <input
          ref={ref}
          id={inputId}
          className={combinedInputClassName}
          disabled={disabled}
          required={required}
          aria-invalid={hasError}
          aria-describedby={[helperTextId, errorId].filter(Boolean).join(' ') || undefined}
          {...props}
        />

        {helperText && !error && (
          <p id={helperTextId} className="mt-1.5 text-sm text-foreground-secondary">
            {helperText}
          </p>
        )}

        {error && (
          <p id={errorId} className="mt-1.5 text-sm text-status-error" role="alert">
            {error}
          </p>
        )}
      </div>
    );
  }
);

Input.displayName = 'Input';
