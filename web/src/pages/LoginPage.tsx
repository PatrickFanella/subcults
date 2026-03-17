/**
 * LoginPage Component
 * User login form with email/password authentication
 */

import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { login } from '../lib/auth-service';
import { useAuth } from '../stores/authStore';

export const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { isAuthenticated } = useAuth();

  // Form state
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);

  // Get the page they tried to visit, or default to home
  const from = (location.state as { from?: { pathname: string } } | null)?.from?.pathname || '/';

  // Redirect if already authenticated
  useEffect(() => {
    if (isAuthenticated) {
      navigate(from, { replace: true });
    }
  }, [isAuthenticated, navigate, from]);

  // Load saved username if "remember me" was checked previously
  useEffect(() => {
    const savedUsername = localStorage.getItem('subcults-remembered-username');
    if (savedUsername) {
      setUsername(savedUsername);
      setRememberMe(true);
    }
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsLoading(true);

    try {
      // Call login API
      await login({ username, password });

      // Save username if "remember me" is checked
      if (rememberMe) {
        localStorage.setItem('subcults-remembered-username', username);
      } else {
        localStorage.removeItem('subcults-remembered-username');
      }

      // Navigate to the page they tried to visit, or home
      navigate(from, { replace: true });
    } catch (err) {
      // Handle login errors
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Login failed. Please check your credentials and try again.');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center px-4 py-12 bg-background">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-foreground mb-2">Welcome to Subcults</h1>
          <p className="text-foreground-secondary">Sign in to discover underground music scenes</p>
        </div>

        {/* Login Form */}
        <div className="bg-background-secondary border border-border rounded-none shadow-none p-6">
          <form onSubmit={handleSubmit} className="space-y-5">
            {/* Error Message */}
            {error && (
              <div
                role="alert"
                className="px-4 py-3 rounded-none bg-background border border-status-error text-status-error"
              >
                <p className="text-sm">{error}</p>
              </div>
            )}

            {/* Username Field */}
            <div>
              <label htmlFor="username" className="block text-sm font-medium text-foreground mb-2">
                Username or Email
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                autoComplete="username"
                disabled={isLoading}
                className="
                  w-full px-4 py-2 rounded-lg
                  bg-background border border-border
                  text-foreground placeholder:text-foreground-tertiary
                  focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
                  focus:border-brand-primary
                  disabled:opacity-50 disabled:cursor-not-allowed
                  transition-colors
                "
                placeholder="Enter your username or email"
              />
            </div>

            {/* Password Field */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label htmlFor="password" className="block text-sm font-medium text-foreground">
                  Password
                </label>
                <a
                  href="#"
                  onClick={(e) => e.preventDefault()}
                  className="text-sm text-brand-primary hover:text-brand-accent transition-colors opacity-50 cursor-not-allowed"
                  tabIndex={-1}
                  title="Coming soon"
                >
                  Forgot password?
                </a>
              </div>
              <div className="relative">
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  autoComplete="current-password"
                  disabled={isLoading}
                  className="
                    w-full px-4 py-2 pr-12 rounded-lg
                    bg-background border border-border
                    text-foreground placeholder:text-foreground-tertiary
                    focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary
                    focus:border-brand-primary
                    disabled:opacity-50 disabled:cursor-not-allowed
                    transition-colors
                  "
                  placeholder="Enter your password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  disabled={isLoading}
                  className="
                    absolute inset-y-0 right-0 pr-3 flex items-center
                    text-foreground-tertiary hover:text-foreground
                    focus:outline-none focus-visible:text-brand-primary
                    disabled:opacity-50 disabled:cursor-not-allowed
                  "
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                >
                  {showPassword ? (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
                      />
                    </svg>
                  ) : (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                      />
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                      />
                    </svg>
                  )}
                </button>
              </div>
            </div>

            {/* Remember Me Checkbox */}
            <div className="flex items-center">
              <input
                id="remember-me"
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                disabled={isLoading}
                className="
                  w-4 h-4 rounded
                  border-border bg-background
                  text-brand-primary
                  focus:ring-2 focus:ring-brand-primary focus:ring-offset-0
                  disabled:opacity-50 disabled:cursor-not-allowed
                "
              />
              <label htmlFor="remember-me" className="ml-2 block text-sm text-foreground">
                Remember me
              </label>
            </div>

            {/* Submit Button */}
            <button
              type="submit"
              disabled={isLoading || !username || !password}
              className="
                w-full px-4 py-2.5 rounded-lg
                bg-brand-primary text-white font-medium
                hover:bg-brand-accent
                focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary focus-visible:ring-offset-2
                disabled:opacity-50 disabled:cursor-not-allowed
                transition-colors
              "
            >
              {isLoading ? (
                <span className="flex items-center justify-center gap-2">
                  <svg className="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    />
                  </svg>
                  Signing in...
                </span>
              ) : (
                'Sign in'
              )}
            </button>
          </form>

          {/* Sign Up Link */}
          <div className="mt-6 text-center">
            <p className="text-sm text-foreground-secondary">
              Don't have an account?{' '}
              <a
                href="#"
                onClick={(e) => e.preventDefault()}
                className="text-brand-primary hover:text-brand-accent font-medium transition-colors opacity-50 cursor-not-allowed"
                tabIndex={-1}
                title="Coming soon"
              >
                Sign up
              </a>
            </p>
          </div>
        </div>

        {/* Additional Help */}
        <div className="mt-6 text-center">
          <p className="text-xs text-foreground-tertiary">
            By signing in, you agree to our{' '}
            <a
              href="#"
              onClick={(e) => e.preventDefault()}
              className="underline hover:text-foreground-secondary opacity-50 cursor-not-allowed"
              title="Coming soon"
            >
              Terms of Service
            </a>{' '}
            and{' '}
            <a
              href="#"
              onClick={(e) => e.preventDefault()}
              className="underline hover:text-foreground-secondary opacity-50 cursor-not-allowed"
              title="Coming soon"
            >
              Privacy Policy
            </a>
          </p>
        </div>
      </div>
    </div>
  );
};
