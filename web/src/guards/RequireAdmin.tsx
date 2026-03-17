/**
 * RequireAdmin Guard
 * Protects admin-only routes
 * Redirects to home if user is not an admin
 */

import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '../stores/authStore';

interface RequireAdminProps {
  children: React.ReactNode;
}

export const RequireAdmin: React.FC<RequireAdminProps> = ({ children }) => {
  const { isAuthenticated, isAdmin, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>Loading...</div>;
  }

  if (!isAuthenticated) {
    // Not authenticated - redirect to login, preserving the attempted location
    return <Navigate to="/account/login" state={{ from: location }} replace />;
  }

  if (!isAdmin) {
    // Authenticated but not admin - redirect to home
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
};
