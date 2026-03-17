/**
 * EventDetailPage Component
 * Displays details for a specific event
 */

import React from 'react';
import { useParams } from 'react-router-dom';

export const EventDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();

  return (
    <div style={{ padding: '2rem' }}>
      <h1>Event Detail</h1>
      <p>Event ID: {id}</p>
      <p>Event details will be displayed here.</p>
    </div>
  );
};
