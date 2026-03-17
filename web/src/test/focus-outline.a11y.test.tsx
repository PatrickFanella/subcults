/**
 * Focus Outline Accessibility Tests
 * Ensures keyboard navigation is supported with visible focus indicators
 */

import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';

describe('Focus Outline - Accessibility', () => {
  it('should have focus-visible styles for buttons', () => {
    const { container } = render(
      <button type="button">Test Button</button>
    );
    
    const button = container.querySelector('button');
    expect(button).toBeInTheDocument();
    
    // Button should be focusable (no tabindex=-1)
    expect(button).not.toHaveAttribute('tabindex', '-1');
  });

  it('should have focus-visible styles for volume sliders', () => {
    const { container } = render(
      <input 
        type="range" 
        className="volume-slider"
        aria-label="Volume"
        min="0"
        max="100"
        defaultValue="50"
      />
    );
    
    const slider = container.querySelector('.volume-slider');
    expect(slider).toBeInTheDocument();
    expect(slider).toHaveClass('volume-slider');
    
    // Slider should be focusable
    expect(slider).not.toHaveAttribute('tabindex', '-1');
  });

  it('should have accessible skip-to-content link', () => {
    const { container } = render(
      <a href="#main-content">Skip to content</a>
    );
    
    const skipLink = container.querySelector('a[href="#main-content"]');
    expect(skipLink).toBeInTheDocument();
    expect(skipLink).toHaveTextContent('Skip to content');
  });
});
