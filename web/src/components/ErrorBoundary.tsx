/**
 * ErrorBoundary Component
 * Catches rendering errors and displays fallback UI
 */

import React, { Component, type ReactNode } from 'react';
import { errorLogger } from '../lib/error-logger';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: React.ErrorInfo | null;
}

/**
 * Filter stack trace to remove sensitive information and noise
 * Only shown in development mode
 */
function filterStackTrace(stack: string | undefined): string {
  if (!stack) return '';
  
  // In production, don't show stack traces
  if (import.meta.env.PROD) return '';
  
  // In development, show filtered stack
  const lines = stack.split('\n');
  return lines
    .filter(line => !line.includes('node_modules'))
    .slice(0, 10) // Limit to first 10 lines
    .join('\n');
}

export class ErrorBoundary extends Component<Props, State> {
  private errorButtonRef = React.createRef<HTMLButtonElement>();

  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Store error info in state for display
    this.setState({ errorInfo });
    
    // Log error details in development only
    if (import.meta.env.DEV) {
      console.warn('[ErrorBoundary] Rendering error caught:', {
        message: error.message,
        stack: filterStackTrace(error.stack),
        componentStack: errorInfo.componentStack,
      });
    }
    
    // Send to error logging service with PII redaction
    errorLogger.logError(error, { 
      componentStack: errorInfo.componentStack || undefined 
    });
  }

  componentDidUpdate(_prevProps: Props, prevState: State) {
    // Focus the reload button when error state changes
    if (!prevState.hasError && this.state.hasError && this.errorButtonRef.current) {
      // Use requestAnimationFrame for reliable focus timing after DOM update
      requestAnimationFrame(() => {
        if (this.errorButtonRef.current) {
          this.errorButtonRef.current.focus();
        }
      });
    }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  handleReload = () => {
    window.location.reload();
  };

  handleReport = () => {
    const { error } = this.state;
    const subject = encodeURIComponent('Error Report: Application Error');
    const body = encodeURIComponent(
      `An error occurred in the application:\n\n` +
      `Error: ${error?.message || 'Unknown error'}\n\n` +
      `Please provide any additional context about what you were doing when this error occurred.`
    );
    // Open email client with pre-filled error report
    window.location.href = `mailto:support@subcults.com?subject=${subject}&body=${body}`;
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      const { error, errorInfo } = this.state;

      return (
        <div
          role="alert"
          aria-live="assertive"
          aria-atomic="true"
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            minHeight: '100vh',
            padding: '2rem',
            backgroundColor: '#1a1a1a',
            color: 'white',
            textAlign: 'center',
          }}
        >
          <h1 
            style={{ 
              fontSize: '2rem', 
              marginBottom: '1rem',
              fontWeight: 'bold',
            }}
            id="error-title"
          >
            Something went wrong
          </h1>
          <p 
            style={{ 
              marginBottom: '2rem', 
              maxWidth: '600px',
              fontSize: '1.125rem',
              lineHeight: '1.6',
            }}
          >
            An unexpected error occurred. Please try refreshing the page. If the problem persists, you can report this issue.
          </p>
          
          <div style={{ display: 'flex', gap: '1rem', marginBottom: '2rem' }}>
            <button
              ref={this.errorButtonRef}
              onClick={this.handleRetry}
              aria-describedby="error-title"
              style={{
                padding: '0.75rem 1.5rem',
                fontSize: '1rem',
                backgroundColor: 'white',
                color: '#1a1a1a',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                fontWeight: '600',
              }}
            >
              Try Again
            </button>
            <button
              onClick={this.handleReload}
              aria-describedby="error-title"
              style={{
                padding: '0.75rem 1.5rem',
                fontSize: '1rem',
                backgroundColor: 'transparent',
                color: 'white',
                border: '2px solid white',
                borderRadius: '4px',
                cursor: 'pointer',
                fontWeight: '600',
              }}
            >
              Reload Page
            </button>
            <button
              onClick={this.handleReport}
              aria-describedby="error-title"
              style={{
                padding: '0.75rem 1.5rem',
                fontSize: '1rem',
                backgroundColor: 'transparent',
                color: 'white',
                border: '2px solid white',
                borderRadius: '4px',
                cursor: 'pointer',
                fontWeight: '600',
              }}
            >
              Report Issue
            </button>
          </div>

          {import.meta.env.DEV && error && (
            <details style={{ maxWidth: '800px', textAlign: 'left', width: '100%' }}>
              <summary 
                style={{ 
                  cursor: 'pointer', 
                  marginBottom: '0.5rem',
                  fontSize: '0.875rem',
                  opacity: 0.8,
                }}
              >
                Error details (development only)
              </summary>
              <div
                style={{
                  padding: '1rem',
                  backgroundColor: '#2a2a2a',
                  borderRadius: '4px',
                  overflow: 'auto',
                  fontSize: '0.75rem',
                }}
              >
                <div style={{ marginBottom: '1rem' }}>
                  <strong>Message:</strong>
                  <pre style={{ margin: '0.5rem 0 0 0', whiteSpace: 'pre-wrap' }}>
                    {error.message}
                  </pre>
                </div>
                {filterStackTrace(error.stack) && (
                  <div style={{ marginBottom: '1rem' }}>
                    <strong>Stack Trace:</strong>
                    <pre style={{ margin: '0.5rem 0 0 0', whiteSpace: 'pre-wrap' }}>
                      {filterStackTrace(error.stack)}
                    </pre>
                  </div>
                )}
                {errorInfo?.componentStack && (
                  <div>
                    <strong>Component Stack:</strong>
                    <pre style={{ margin: '0.5rem 0 0 0', whiteSpace: 'pre-wrap' }}>
                      {errorInfo.componentStack}
                    </pre>
                  </div>
                )}
              </div>
            </details>
          )}
        </div>
      );
    }

    return this.props.children;
  }
}
