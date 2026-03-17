/**
 * Auth Service
 * API calls for authentication operations
 */

import { apiClient } from './api-client';
import { authStore, type User } from '../stores/authStore';

export interface LoginCredentials {
  username: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  user: User;
}

/**
 * Login with username and password
 * Sets access token in memory and refresh token in httpOnly cookie
 */
export async function login(credentials: LoginCredentials): Promise<User> {
  const response = await apiClient.post<LoginResponse>('/auth/login', credentials, {
    skipAuth: true, // No token needed for login
  });

  // Store user and access token
  authStore.setUser(response.user, response.accessToken);

  return response.user;
}

/**
 * Logout current user
 * Clears access token from memory and refresh token cookie on backend
 */
export async function logout(): Promise<void> {
  await authStore.logout();
}

/**
 * Get current user profile
 * Example of authenticated request
 */
export async function getCurrentUser(): Promise<User> {
  return apiClient.get<User>('/auth/me');
}
