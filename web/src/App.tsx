/**
 * App Component
 * Root application component with routing
 */

import { useEffect } from 'react';
import { ErrorBoundary } from './components/ErrorBoundary';
import { ToastContainer } from './components/ToastContainer';
import { ThemeProvider } from './components/ThemeProvider';
import { AppRouter } from './routes';
import { authStore } from './stores/authStore';
import { useStreamingStore } from './stores/streamingStore';
import { useSettingsStore } from './stores/settingsStore';
import { useLanguageStore } from './stores/languageStore';
import { sessionReplay } from './lib/session-replay';
import { initPerformanceMonitoring } from './lib/performance-metrics';
import { useWebVitals, useNavigationTiming } from './hooks/useWebVitals';

function App() {
  // Track Core Web Vitals metrics
  useWebVitals();

  // Track navigation and resource timing
  useNavigationTiming();

  // Initialize auth on app startup
  useEffect(() => {
    authStore.initialize();
  }, []);

  // Initialize streaming store on app startup
  useEffect(() => {
    const streamingStore = useStreamingStore.getState();
    streamingStore.initialize();
  }, []);

  // Initialize language store on app startup
  useEffect(() => {
    useLanguageStore.getState().initializeLanguage();
  }, []);

  // Initialize settings store and session replay
  useEffect(() => {
    // Load settings from localStorage
    useSettingsStore.getState().initializeSettings();

    // Initialize performance monitoring after settings are loaded
    const { telemetryOptOut } = useSettingsStore.getState();
    initPerformanceMonitoring(telemetryOptOut);

    // Start session replay if user has opted in
    // (will check opt-in status internally)
    sessionReplay.start();

    // Cleanup on unmount
    return () => {
      sessionReplay.destroy();
    };
  }, []);

  return (
    <ThemeProvider>
      <ErrorBoundary>
        <AppRouter />
        <ToastContainer />
      </ErrorBoundary>
    </ThemeProvider>
  );
}

export default App;
