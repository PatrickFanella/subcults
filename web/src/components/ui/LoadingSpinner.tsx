/**
 * LoadingSpinner Component
 * Reusable loading indicator for async operations
 * 
 * Features:
 * - Multiple size variants
 * - Accessible with aria-label
 * - Consistent with design system
 */

export type SpinnerSize = 'sm' | 'md' | 'lg' | 'xl';

export interface LoadingSpinnerProps {
  /**
   * Size variant
   * @default 'md'
   */
  size?: SpinnerSize;
  
  /**
   * Accessible label for screen readers
   * @default 'Loading'
   */
  label?: string;
  
  /**
   * Additional CSS classes
   */
  className?: string;
}

/**
 * Get size-specific classes
 */
function getSizeClasses(size: SpinnerSize): string {
  const sizes = {
    sm: 'h-4 w-4 border-2',
    md: 'h-6 w-6 border-2',
    lg: 'h-8 w-8 border-4',
    xl: 'h-12 w-12 border-4',
  };
  return sizes[size];
}

/**
 * LoadingSpinner component
 */
export function LoadingSpinner({
  size = 'md',
  label = 'Loading',
  className = '',
}: LoadingSpinnerProps) {
  const sizeClasses = getSizeClasses(size);
  
  // Only add role and aria-label if label is provided
  const roleProps = label
    ? { role: 'status' as const, 'aria-label': label }
    : {};
  
  return (
    <div
      {...roleProps}
      className={`
        inline-block animate-spin rounded-full
        border-brand-primary border-t-transparent
        ${sizeClasses}
        ${className}
      `.trim()}
    >
      {label && <span className="sr-only">{label}</span>}
    </div>
  );
}

/**
 * FullPageLoader - Loading spinner centered on full page
 */
export interface FullPageLoaderProps {
  /**
   * Accessible label
   * @default 'Loading content'
   */
  label?: string;
  
  /**
   * Show loading text below spinner
   * @default true
   */
  showText?: boolean;
}

export function FullPageLoader({
  label = 'Loading content',
  showText = true,
}: FullPageLoaderProps) {
  return (
    <div
      className="flex flex-col items-center justify-center h-screen w-full bg-background"
      role="status"
      aria-live="polite"
      aria-busy="true"
      aria-label={label}
    >
      <LoadingSpinner size="xl" label="" />
      {showText && (
        <p className="mt-4 text-foreground-secondary">{label}...</p>
      )}
      <span className="sr-only">{label}</span>
    </div>
  );
}
