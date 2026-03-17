/**
 * Button Component
 * Unified button component with variants following design system
 *
 * Features:
 * - Four variants: primary, secondary, danger, ghost
 * - Consistent sizing and spacing
 * - Accessible focus indicators (WCAG AA compliant)
 * - Loading state support
 * - Disabled state handling
 * - Touch-friendly (44px minimum)
 */

import React from 'react';

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost';
export type ButtonSize = 'sm' | 'md' | 'lg';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /**
   * Visual style variant
   * @default 'primary'
   */
  variant?: ButtonVariant;

  /**
   * Button size
   * @default 'md'
   */
  size?: ButtonSize;

  /**
   * Show loading spinner and disable interaction
   * @default false
   */
  isLoading?: boolean;

  /**
   * Full width button
   * @default false
   */
  fullWidth?: boolean;

  /**
   * Additional CSS classes
   */
  className?: string;

  /**
   * Button content
   */
  children: React.ReactNode;
}

/**
 * Get variant-specific styles
 */
function getVariantStyles(variant: ButtonVariant): string {
  const variants = {
    primary: `
      bg-brand-primary hover:bg-brand-primary-dark
      text-white border-brand-primary hover:border-brand-primary-dark
      focus-visible:ring-brand-primary
    `,
    secondary: `
      bg-background-secondary hover:bg-brand-underground-lighter
      text-foreground border-border hover:border-border-hover
      focus-visible:ring-brand-primary
    `,
    danger: `
      bg-status-error hover:opacity-90
      text-white border-status-error
      focus-visible:ring-status-error
    `,
    ghost: `
      bg-transparent hover:bg-brand-underground-lighter
      text-foreground border-transparent hover:border-border-hover
      focus-visible:ring-brand-primary
    `,
  };

  return variants[variant].replace(/\s+/g, ' ').trim();
}

/**
 * Get size-specific styles
 */
function getSizeStyles(size: ButtonSize): string {
  const sizes = {
    sm: 'px-3 py-1.5 text-sm min-h-[36px]',
    md: 'px-5 py-2.5 text-base min-h-touch', // 44px minimum for touch
    lg: 'px-6 py-3 text-lg min-h-[52px]',
  };

  return sizes[size];
}

/**
 * Button component with design system variants
 */
export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = 'primary',
      size = 'md',
      isLoading = false,
      fullWidth = false,
      className = '',
      disabled,
      children,
      ...props
    },
    ref
  ) => {
    const baseStyles = `
      inline-flex items-center justify-center gap-2
      rounded-none border font-bold uppercase tracking-[0.05em]
      cursor-pointer transition-none
      focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2
      disabled:opacity-50 disabled:cursor-not-allowed
      theme-transition
      min-w-touch
    `;

    const variantStyles = getVariantStyles(variant);
    const sizeStyles = getSizeStyles(size);
    const widthStyle = fullWidth ? 'w-full' : '';

    const combinedClassName =
      `${baseStyles} ${variantStyles} ${sizeStyles} ${widthStyle} ${className}`.trim();

    return (
      <button ref={ref} className={combinedClassName} disabled={disabled || isLoading} {...props}>
        {isLoading && (
          <span
            className="inline-block animate-spin rounded-full h-4 w-4 border-2 border-current border-t-transparent"
            role="status"
            aria-label="Loading"
          />
        )}
        {children}
      </button>
    );
  }
);

Button.displayName = 'Button';
