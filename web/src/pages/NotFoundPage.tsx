/**
 * NotFoundPage Component
 * 404 page for invalid routes
 */

import React from 'react';
import { Link } from 'react-router-dom';

export const NotFoundPage: React.FC = () => {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        padding: '2rem',
        textAlign: 'center',
      }}
    >
      <h1 style={{ fontSize: '4rem', marginBottom: '1rem' }}>404</h1>
      <h2 style={{ marginBottom: '1rem' }}>Page Not Found</h2>
      <p style={{ marginBottom: '2rem', maxWidth: '600px' }}>
        The page you're looking for doesn't exist or has been moved.
      </p>
      <Link
        to="/"
        style={{
          padding: '0.75rem 1.5rem',
          fontSize: '1rem',
          textDecoration: 'none',
          backgroundColor: 'white',
          color: '#1a1a1a',
          borderRadius: '4px',
        }}
      >
        Go Home
      </Link>
    </div>
  );
};
