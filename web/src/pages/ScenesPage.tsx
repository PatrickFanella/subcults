/**
 * ScenesPage Component
 * Redirects to search results filtered by scenes type
 */

import { Navigate } from 'react-router-dom';

export function ScenesPage() {
  return <Navigate replace to="/search?type=scenes" />;
}
