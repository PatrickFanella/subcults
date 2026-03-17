/**
 * Toast Example Page
 * Demonstrates toast notification system usage
 */

import { useToasts } from '../stores/toastStore';

export function ToastExamplePage() {
  const toasts = useToasts();

  return (
    <div style={{ 
      padding: '2rem', 
      maxWidth: '600px', 
      margin: '0 auto',
      fontFamily: 'system-ui, sans-serif',
    }}>
      <h1 style={{ marginBottom: '2rem', fontSize: '2rem' }}>
        Toast Notification Examples
      </h1>
      
      <div style={{ 
        display: 'flex', 
        flexDirection: 'column', 
        gap: '1rem',
        marginBottom: '2rem',
      }}>
        <button
          onClick={() => toasts.success('Scene created successfully!')}
          style={{
            padding: '0.75rem 1.5rem',
            fontSize: '1rem',
            backgroundColor: '#10b981',
            color: 'white',
            border: 'none',
            borderRadius: '0.5rem',
            cursor: 'pointer',
            fontWeight: '600',
          }}
        >
          Show Success Toast
        </button>
        
        <button
          onClick={() => toasts.error('Failed to save changes. Please try again.')}
          style={{
            padding: '0.75rem 1.5rem',
            fontSize: '1rem',
            backgroundColor: '#ef4444',
            color: 'white',
            border: 'none',
            borderRadius: '0.5rem',
            cursor: 'pointer',
            fontWeight: '600',
          }}
        >
          Show Error Toast
        </button>
        
        <button
          onClick={() => toasts.info('Loading your scenes...')}
          style={{
            padding: '0.75rem 1.5rem',
            fontSize: '1rem',
            backgroundColor: '#3b82f6',
            color: 'white',
            border: 'none',
            borderRadius: '0.5rem',
            cursor: 'pointer',
            fontWeight: '600',
          }}
        >
          Show Info Toast
        </button>
        
        <button
          onClick={() => {
            toasts.custom({
              type: 'error',
              message: 'This toast will not auto-dismiss',
              duration: 0,
              dismissible: true,
            });
          }}
          style={{
            padding: '0.75rem 1.5rem',
            fontSize: '1rem',
            backgroundColor: '#6b7280',
            color: 'white',
            border: 'none',
            borderRadius: '0.5rem',
            cursor: 'pointer',
            fontWeight: '600',
          }}
        >
          Show Persistent Toast
        </button>
        
        <button
          onClick={() => {
            toasts.success('First notification');
            setTimeout(() => toasts.error('Second notification'), 200);
            setTimeout(() => toasts.info('Third notification'), 400);
          }}
          style={{
            padding: '0.75rem 1.5rem',
            fontSize: '1rem',
            backgroundColor: '#8b5cf6',
            color: 'white',
            border: 'none',
            borderRadius: '0.5rem',
            cursor: 'pointer',
            fontWeight: '600',
          }}
        >
          Show Multiple Toasts
        </button>
        
        <button
          onClick={() => toasts.clearAll()}
          style={{
            padding: '0.75rem 1.5rem',
            fontSize: '1rem',
            backgroundColor: 'white',
            color: '#1a1a1a',
            border: '2px solid #1a1a1a',
            borderRadius: '0.5rem',
            cursor: 'pointer',
            fontWeight: '600',
          }}
        >
          Clear All Toasts
        </button>
      </div>
      
      <div style={{
        padding: '1.5rem',
        backgroundColor: '#f3f4f6',
        borderRadius: '0.5rem',
        fontSize: '0.875rem',
        lineHeight: '1.5',
      }}>
        <h2 style={{ marginBottom: '0.5rem', fontSize: '1.125rem', fontWeight: '600' }}>
          About Toast Notifications
        </h2>
        <p style={{ marginBottom: '0.5rem' }}>
          Toasts appear in the top-right corner and automatically dismiss after 5 seconds.
        </p>
        <p style={{ marginBottom: '0.5rem' }}>
          They are fully accessible with ARIA live regions for screen readers.
        </p>
        <p>
          See <code style={{ 
            padding: '0.125rem 0.25rem', 
            backgroundColor: '#e5e7eb',
            borderRadius: '0.25rem',
            fontFamily: 'monospace',
          }}>ERROR_HANDLING.md</code> for usage documentation.
        </p>
      </div>
    </div>
  );
}
