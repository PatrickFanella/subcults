/**
 * Error Boundary Demo
 * Demonstrates error boundary functionality
 */

import { useState } from 'react';
import { ErrorBoundary } from './components/ErrorBoundary';

function BuggyComponent({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error('This is a simulated rendering error');
  }
  return <div>Component rendered successfully</div>;
}

export function ErrorBoundaryDemo() {
  const [shouldThrow, setShouldThrow] = useState(false);

  return (
    <div style={{ 
      padding: '2rem', 
      maxWidth: '600px', 
      margin: '0 auto',
      fontFamily: 'system-ui, sans-serif',
    }}>
      <h1 style={{ marginBottom: '2rem', fontSize: '2rem' }}>
        Error Boundary Demo
      </h1>
      
      <button
        onClick={() => setShouldThrow(true)}
        style={{
          padding: '0.75rem 1.5rem',
          fontSize: '1rem',
          backgroundColor: '#ef4444',
          color: 'white',
          border: 'none',
          borderRadius: '0.5rem',
          cursor: 'pointer',
          fontWeight: '600',
          marginBottom: '2rem',
        }}
      >
        Trigger Rendering Error
      </button>
      
      <ErrorBoundary>
        <div style={{
          padding: '1.5rem',
          backgroundColor: '#f3f4f6',
          borderRadius: '0.5rem',
          marginBottom: '1rem',
        }}>
          <BuggyComponent shouldThrow={shouldThrow} />
        </div>
      </ErrorBoundary>
      
      <div style={{
        padding: '1.5rem',
        backgroundColor: '#f3f4f6',
        borderRadius: '0.5rem',
        fontSize: '0.875rem',
        lineHeight: '1.5',
      }}>
        <h2 style={{ marginBottom: '0.5rem', fontSize: '1.125rem', fontWeight: '600' }}>
          About Error Boundaries
        </h2>
        <p style={{ marginBottom: '0.5rem' }}>
          Error boundaries catch rendering errors and display a fallback UI instead of crashing the entire app.
        </p>
        <p>
          Click the button above to see the error boundary in action.
        </p>
      </div>
    </div>
  );
}
