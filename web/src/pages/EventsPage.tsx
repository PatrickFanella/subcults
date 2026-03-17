/**
 * EventsPage Component
 * Redirects to search results filtered by events type
 */

import { Navigate } from 'react-router-dom';

export function EventsPage() {
  return <Navigate replace to="/search?type=events" />;
}
