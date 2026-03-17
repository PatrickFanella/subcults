/**
 * RankingCalibrationPanel Component
 * Admin panel for adjusting ranking algorithm weights
 * 
 * Allows admins to fine-tune the balance between:
 * - Text relevance (full-text search score)
 * - Geographic proximity (how close results are to the user)
 * - Recency (how recent the content is)
 * - Trust weight (alliance-based ranking)
 */

import React, { useState, useCallback, useEffect } from 'react';
import { useTelemetry } from '../hooks/useTelemetry';

export interface RankingWeights {
  textRelevance: number;
  proximityScore: number;
  recency: number;
  trustWeight: number;
}

interface RankingCalibrationPanelProps {
  /** Initial weights to display */
  initialWeights?: RankingWeights;
  /** Callback when weights are submitted */
  onSubmit?: (weights: RankingWeights) => Promise<void>;
  /** Whether component is disabled */
  isDisabled?: boolean;
}

/**
 * Default ranking weights matching backend configuration
 */
const DEFAULT_WEIGHTS: RankingWeights = {
  textRelevance: 0.4,
  proximityScore: 0.3,
  recency: 0.2,
  trustWeight: 0.1,
};

/**
 * Validate that weights sum to 1.0 and are within bounds
 */
function validateWeights(weights: RankingWeights): boolean {
  const sum = Object.values(weights).reduce((a, b) => a + b, 0);
  const tolerance = 0.001; // Allow for floating point rounding
  
  // Check sum is approximately 1.0
  if (Math.abs(sum - 1.0) > tolerance) {
    return false;
  }
  
  // Check all weights are between 0 and 1
  return Object.values(weights).every(w => w >= 0 && w <= 1);
}

/**
 * Normalize weights to sum to 1.0
 */
function normalizeWeights(weights: RankingWeights): RankingWeights {
  const sum = Object.values(weights).reduce((a, b) => a + b, 0);
  if (sum === 0) return DEFAULT_WEIGHTS;
  
  return {
    textRelevance: weights.textRelevance / sum,
    proximityScore: weights.proximityScore / sum,
    recency: weights.recency / sum,
    trustWeight: weights.trustWeight / sum,
  };
}

/**
 * RankingCalibrationPanel - Admin UI for ranking weight adjustment
 */
export const RankingCalibrationPanel: React.FC<RankingCalibrationPanelProps> = ({
  initialWeights = DEFAULT_WEIGHTS,
  onSubmit,
  isDisabled = false,
}) => {
  const emit = useTelemetry();
  const [weights, setWeights] = useState<RankingWeights>(initialWeights);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  
  // Reset success message after 3 seconds
  useEffect(() => {
    if (success) {
      const timer = setTimeout(() => setSuccess(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [success]);

  /**
   * Handle slider change for a weight
   */
  const handleWeightChange = useCallback((key: keyof RankingWeights, value: number) => {
    const newWeights = { ...weights, [key]: value };
    
    // Normalize so sum = 1.0
    const normalized = normalizeWeights(newWeights);
    setWeights(normalized);
    setError(null);
  }, [weights]);

  /**
   * Handle form submission
   */
  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Validate weights
    if (!validateWeights(weights)) {
      setError('Weights must sum to 1.0 and be between 0 and 1');
      return;
    }

    if (!onSubmit) {
      // Just emit event if no callback provided
      emit('admin.ranking.calibrate', weights as unknown as Record<string, unknown>);
      setSuccess(true);
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await onSubmit(weights);
      emit('admin.ranking.calibrate_success', weights as unknown as Record<string, unknown>);
      setSuccess(true);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to save ranking weights';
      setError(errorMsg);
      emit('admin.ranking.calibrate_error', { error: errorMsg });
    } finally {
      setIsSubmitting(false);
    }
  }, [weights, onSubmit, emit]);

  /**
   * Handle reset to defaults
   */
  const handleReset = useCallback(() => {
    setWeights(DEFAULT_WEIGHTS);
    setError(null);
    setSuccess(false);
    emit('admin.ranking.reset_defaults');
  }, [emit]);

  const sum = Object.values(weights).reduce((a, b) => a + b, 0);
  const isValid = validateWeights(weights);

  return (
    <div style={{ 
      padding: '1.5rem',
      border: '1px solid #262626',
      backgroundColor: '#0F0F0F',
    }}>
      <h2 style={{ fontSize: '1.25rem', fontWeight: 600, marginBottom: '1rem', color: '#FAFAFA' }}>
        Ranking Algorithm Calibration
      </h2>
      
      <p style={{ marginBottom: '1.5rem', color: '#737373' }}>
        Adjust the weights below to control how scenes and events are ranked in search results.
        Weights must sum to 1.0 and are automatically normalized when you adjust them.
      </p>

      <form onSubmit={handleSubmit}>
        {/* Text Relevance Weight */}
        <div style={{ marginBottom: '1.5rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
            <label style={{ fontWeight: 500, color: '#FAFAFA' }}>
              Text Relevance
            </label>
            <span style={{ fontSize: '0.875rem', color: '#737373' }}>
              {(weights.textRelevance * 100).toFixed(1)}%
            </span>
          </div>
          <input
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={weights.textRelevance}
            onChange={(e) => handleWeightChange('textRelevance', parseFloat(e.target.value))}
            disabled={isDisabled || isSubmitting}
            style={{
              width: '100%',
              cursor: isDisabled || isSubmitting ? 'not-allowed' : 'pointer',
              opacity: isDisabled || isSubmitting ? 0.6 : 1,
            }}
            title="How much to weight full-text search relevance"
          />
          <p style={{ fontSize: '0.875rem', color: '#737373', marginTop: '0.25rem' }}>
            How much to weight full-text search relevance (keyword matches)
          </p>
        </div>

        {/* Proximity Score Weight */}
        <div style={{ marginBottom: '1.5rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
            <label style={{ fontWeight: 500, color: '#FAFAFA' }}>
              Geographic Proximity
            </label>
            <span style={{ fontSize: '0.875rem', color: '#737373' }}>
              {(weights.proximityScore * 100).toFixed(1)}%
            </span>
          </div>
          <input
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={weights.proximityScore}
            onChange={(e) => handleWeightChange('proximityScore', parseFloat(e.target.value))}
            disabled={isDisabled || isSubmitting}
            style={{
              width: '100%',
              cursor: isDisabled || isSubmitting ? 'not-allowed' : 'pointer',
              opacity: isDisabled || isSubmitting ? 0.6 : 1,
            }}
            title="How much to weight geographic proximity"
          />
          <p style={{ fontSize: '0.875rem', color: '#737373', marginTop: '0.25rem' }}>
            How much to prioritize results near the user's location
          </p>
        </div>

        {/* Recency Weight */}
        <div style={{ marginBottom: '1.5rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
            <label style={{ fontWeight: 500, color: '#FAFAFA' }}>
              Recency
            </label>
            <span style={{ fontSize: '0.875rem', color: '#737373' }}>
              {(weights.recency * 100).toFixed(1)}%
            </span>
          </div>
          <input
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={weights.recency}
            onChange={(e) => handleWeightChange('recency', parseFloat(e.target.value))}
            disabled={isDisabled || isSubmitting}
            style={{
              width: '100%',
              cursor: isDisabled || isSubmitting ? 'not-allowed' : 'pointer',
              opacity: isDisabled || isSubmitting ? 0.6 : 1,
            }}
            title="How much to weight recency"
          />
          <p style={{ fontSize: '0.875rem', color: '#737373', marginTop: '0.25rem' }}>
            How much to prioritize newer content
          </p>
        </div>

        {/* Trust Weight */}
        <div style={{ marginBottom: '1.5rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
            <label style={{ fontWeight: 500, color: '#FAFAFA' }}>
              Trust Score
            </label>
            <span style={{ fontSize: '0.875rem', color: '#737373' }}>
              {(weights.trustWeight * 100).toFixed(1)}%
            </span>
          </div>
          <input
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={weights.trustWeight}
            onChange={(e) => handleWeightChange('trustWeight', parseFloat(e.target.value))}
            disabled={isDisabled || isSubmitting}
            style={{
              width: '100%',
              cursor: isDisabled || isSubmitting ? 'not-allowed' : 'pointer',
              opacity: isDisabled || isSubmitting ? 0.6 : 1,
            }}
            title="How much to weight user trust scores"
          />
          <p style={{ fontSize: '0.875rem', color: '#737373', marginTop: '0.25rem' }}>
            How much to weight alliance-based trust scores
          </p>
        </div>

        {/* Validation Status */}
        <div style={{
          padding: '1rem',
          marginBottom: '1.5rem',
          backgroundColor: '#1A1A1A',
          border: `1px solid ${isValid ? '#262626' : '#FF3D00'}`,
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontSize: '0.875rem', color: isValid ? '#737373' : '#FF3D00' }}>
              Total: {(sum * 100).toFixed(1)}%
              {isValid ? ' ✓' : ' (must equal 100%)'}
            </span>
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <div style={{
            padding: '0.75rem',
            marginBottom: '1.5rem',
            backgroundColor: '#1A1A1A',
            border: '1px solid #FF3D00',
            color: '#FF3D00',
            fontSize: '0.875rem',
          }}>
            {error}
          </div>
        )}

        {/* Success Message */}
        {success && (
          <div style={{
            padding: '0.75rem',
            marginBottom: '1.5rem',
            backgroundColor: '#1A1A1A',
            border: '1px solid #262626',
            color: '#737373',
            fontSize: '0.875rem',
          }}>
            Ranking weights saved successfully ✓
          </div>
        )}

        {/* Button Group */}
        <div style={{ display: 'flex', gap: '0.75rem' }}>
          <button
            type="submit"
            disabled={!isValid || isDisabled || isSubmitting}
            style={{
              flex: 1,
              padding: '0.75rem 1rem',
              backgroundColor: '#FF3D00',
              color: '#0A0A0A',
              border: 'none',
              fontWeight: 500,
              cursor: !isValid || isDisabled || isSubmitting ? 'not-allowed' : 'pointer',
              opacity: !isValid || isDisabled || isSubmitting ? 0.6 : 1,
              fontSize: '0.875rem',
            }}
          >
            {isSubmitting ? 'Saving...' : 'Save Weights'}
          </button>
          
          <button
            type="button"
            onClick={handleReset}
            disabled={isDisabled || isSubmitting}
            style={{
              flex: 1,
              padding: '0.75rem 1rem',
              backgroundColor: '#1A1A1A',
              color: '#FAFAFA',
              border: '1px solid #262626',
              fontWeight: 500,
              cursor: isDisabled || isSubmitting ? 'not-allowed' : 'pointer',
              opacity: isDisabled || isSubmitting ? 0.6 : 1,
              fontSize: '0.875rem',
            }}
          >
            Reset to Defaults
          </button>
        </div>
      </form>
    </div>
  );
};

RankingCalibrationPanel.displayName = 'RankingCalibrationPanel';
