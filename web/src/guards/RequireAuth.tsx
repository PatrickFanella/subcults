/**
 * RequireAuth Guard
 * Protects routes that require authentication
 * Redirects to /account/login if not authenticated
 */

import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '../stores/authStore';

interface RequireAuthProps {
  children: React.ReactNode;
}

export const RequireAuth: React.FC<RequireAuthProps> = ({ children }) => {
  const { isAuthenticated, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>Loading...</div>;
  }

  if (!isAuthenticated) {
    // Redirect to login, preserving the attempted location
    return <Navigate to="/account/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
};
