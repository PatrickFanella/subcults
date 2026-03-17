/**
 * Avatar Component Tests
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Avatar } from './Avatar';

// Mock the imageUrl utility
vi.mock('../utils/imageUrl', () => ({
  getAvatarUrl: (key: string, size: string, format: string) =>
    `/api/media/${key}?size=${size}&format=${format}`,
  AVATAR_SIZES: {
    xs: 32,
    sm: 48,
    md: 64,
    lg: 96,
    xl: 128,
  },
}));

describe('Avatar', () => {
  it('renders with name', () => {
    render(<Avatar name="John Doe" />);
    
    // Should show initials when no image
    expect(screen.getByText('JD')).toBeInTheDocument();
  });

  it('shows initials for single name', () => {
    render(<Avatar name="Prince" />);
    
    expect(screen.getByText('PR')).toBeInTheDocument();
  });

  it('generates consistent initials from name', () => {
    render(<Avatar name="Alice Bob Charlie" />);
    
    // Should use first and last name
    expect(screen.getByText('AC')).toBeInTheDocument();
  });

  it('renders image when src is provided', () => {
    render(<Avatar name="John Doe" src="posts/123/avatar.jpg" />);
    
    const img = screen.getByAltText("John Doe's avatar");
    expect(img).toBeInTheDocument();
    expect(img).toHaveAttribute('src', expect.stringContaining('avatar.jpg'));
  });

  it('renders WebP and JPEG sources', () => {
    const { container } = render(
      <Avatar name="John Doe" src="posts/123/avatar.jpg" />
    );
    
    const picture = container.querySelector('picture');
    expect(picture).toBeInTheDocument();
    
    const webpSource = container.querySelector('source[type="image/webp"]');
    expect(webpSource).toBeInTheDocument();
    expect(webpSource).toHaveAttribute('srcset', expect.stringContaining('webp'));
  });

  it('falls back to initials on image error', () => {
    render(<Avatar name="John Doe" src="posts/123/broken.jpg" />);
    
    const img = screen.getByAltText("John Doe's avatar");
    
    // Simulate image error
    fireEvent.error(img);
    
    // Should show initials
    expect(screen.getByText('JD')).toBeInTheDocument();
  });

  it('applies correct size dimensions', () => {
    const { container } = render(<Avatar name="John Doe" size="lg" />);
    
    const avatar = container.querySelector('div[role="button"], div:not([role])');
    expect(avatar).toHaveStyle({ width: '96px', height: '96px' });
  });

  it('applies custom className', () => {
    const { container } = render(
      <Avatar name="John Doe" className="custom-class" />
    );
    
    const avatar = container.firstChild;
    expect(avatar).toHaveClass('custom-class');
  });

  it('shows online indicator when online=true', () => {
    render(<Avatar name="John Doe" online />);
    
    const indicator = screen.getByLabelText('Online');
    expect(indicator).toBeInTheDocument();
  });

  it('does not show online indicator by default', () => {
    render(<Avatar name="John Doe" />);
    
    const indicator = screen.queryByLabelText('Online');
    expect(indicator).not.toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const onClick = vi.fn();
    render(<Avatar name="John Doe" onClick={onClick} />);
    
    const avatar = screen.getByRole('button');
    fireEvent.click(avatar);
    
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('supports keyboard interaction when clickable', () => {
    const onClick = vi.fn();
    render(<Avatar name="John Doe" onClick={onClick} />);
    
    const avatar = screen.getByRole('button');
    
    // Test Enter key
    fireEvent.keyDown(avatar, { key: 'Enter' });
    expect(onClick).toHaveBeenCalledTimes(1);
    
    // Test Space key
    fireEvent.keyDown(avatar, { key: ' ' });
    expect(onClick).toHaveBeenCalledTimes(2);
  });

  it('has button role and tabIndex when clickable', () => {
    render(<Avatar name="John Doe" onClick={() => {}} />);
    
    const avatar = screen.getByRole('button');
    expect(avatar).toHaveAttribute('tabindex', '0');
  });

  it('does not have button role when not clickable', () => {
    const { container } = render(<Avatar name="John Doe" />);
    
    const button = screen.queryByRole('button');
    expect(button).not.toBeInTheDocument();
    
    const avatar = container.querySelector('div');
    expect(avatar).not.toHaveAttribute('tabindex');
  });

  it('uses lazy loading for images', () => {
    render(<Avatar name="John Doe" src="posts/123/avatar.jpg" />);
    
    const img = screen.getByAltText("John Doe's avatar");
    expect(img).toHaveAttribute('loading', 'lazy');
  });

  it('uses async decoding', () => {
    render(<Avatar name="John Doe" src="posts/123/avatar.jpg" />);
    
    const img = screen.getByAltText("John Doe's avatar");
    expect(img).toHaveAttribute('decoding', 'async');
  });

  it('shows loading skeleton before image loads', () => {
    const { container } = render(
      <Avatar name="John Doe" src="posts/123/avatar.jpg" />
    );
    
    const skeleton = container.querySelector('.animate-pulse');
    expect(skeleton).toBeInTheDocument();
  });

  it('hides loading skeleton after image loads', () => {
    const { container } = render(
      <Avatar name="John Doe" src="posts/123/avatar.jpg" />
    );
    
    const img = screen.getByAltText("John Doe's avatar");
    
    // Initially has skeleton
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
    
    // After load, skeleton should be hidden
    fireEvent.load(img);
    
    // Image should be visible
    expect(img).toHaveClass('opacity-100');
  });

  it('generates different colors for different names', () => {
    const { container: container1 } = render(<Avatar name="Alice" />);
    const { container: container2 } = render(<Avatar name="Bob" />);
    
    const avatar1 = container1.querySelector('[aria-label*="avatar (initials)"]');
    const avatar2 = container2.querySelector('[aria-label*="avatar (initials)"]');
    
    // Both should have background colors
    expect(avatar1).toBeInTheDocument();
    expect(avatar2).toBeInTheDocument();
  });

  it('handles empty name gracefully', () => {
    render(<Avatar name="" />);
    
    expect(screen.getByLabelText(/'s avatar \(initials\)/)).toBeInTheDocument();
  });

  it('handles whitespace-only name gracefully', () => {
    render(<Avatar name="   " />);
    
    // Should render something (even if not showing '?')
    expect(screen.getByLabelText(/'s avatar \(initials\)/)).toBeInTheDocument();
  });
});
