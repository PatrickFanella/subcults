/**
 * SceneDetailPage Component
 * Displays details for a specific scene
 */

import React from 'react';
import { useParams } from 'react-router-dom';

export const SceneDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();

  return (
    <div style={{ padding: '2rem' }}>
      <h1>Scene Detail</h1>
      <p>Scene ID: {id}</p>
      <p>Scene details will be displayed here.</p>
    </div>
  );
};
