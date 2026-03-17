/**
 * DarkModeToggle Tests
 * Validates theme toggle button behavior and accessibility
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DarkModeToggle } from './DarkModeToggle';
import { useThemeStore } from '../stores/themeStore';

describe('DarkModeToggle', () => {
  beforeEach(() => {
    localStorage.clear();
    useThemeStore.setState({ theme: 'light' });
    document.documentElement.classList.remove('dark');
  });

  it('renders toggle button', () => {
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    expect(button).toBeInTheDocument();
  });

  it('shows moon icon in light mode', () => {
    useThemeStore.setState({ theme: 'light' });
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    expect(button.textContent).toContain('🌙');
  });

  it('shows sun icon in dark mode', () => {
    useThemeStore.setState({ theme: 'dark' });
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    expect(button.textContent).toContain('☀️');
  });

  it('has correct aria-label in light mode', () => {
    useThemeStore.setState({ theme: 'light' });
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('aria-label', 'Switch to Dark mode');
  });

  it('has correct aria-label in dark mode', () => {
    useThemeStore.setState({ theme: 'dark' });
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('aria-label', 'Switch to Light mode');
  });

  it('has title attribute for tooltip', () => {
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('title');
  });

  it('toggles theme when clicked', async () => {
    const user = userEvent.setup();
    useThemeStore.setState({ theme: 'light' });
    
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    await user.click(button);
    
    expect(useThemeStore.getState().theme).toBe('dark');
  });

  it('toggles back to light when clicked again', async () => {
    const user = userEvent.setup();
    useThemeStore.setState({ theme: 'dark' });
    
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    await user.click(button);
    
    expect(useThemeStore.getState().theme).toBe('light');
  });

  it('shows label text when showLabel is true', () => {
    useThemeStore.setState({ theme: 'light' });
    render(<DarkModeToggle showLabel={true} />);
    
    expect(screen.getByText('Dark mode')).toBeInTheDocument();
  });

  it('hides label text when showLabel is false', () => {
    useThemeStore.setState({ theme: 'light' });
    render(<DarkModeToggle showLabel={false} />);
    
    expect(screen.queryByText('Dark mode')).not.toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<DarkModeToggle className="custom-class" />);
    
    const button = screen.getByRole('button');
    expect(button.className).toContain('custom-class');
  });

  it('has proper keyboard accessibility', async () => {
    const user = userEvent.setup();
    useThemeStore.setState({ theme: 'light' });
    
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    button.focus();
    
    expect(button).toHaveFocus();
    
    await user.keyboard('{Enter}');
    expect(useThemeStore.getState().theme).toBe('dark');
  });

  it('icon has aria-hidden to avoid screen reader duplication', () => {
    render(<DarkModeToggle />);
    
    const icon = screen.getByRole('img', { hidden: true });
    expect(icon).toHaveAttribute('aria-hidden', 'true');
  });

  it('updates document class when toggled', async () => {
    const user = userEvent.setup();
    useThemeStore.setState({ theme: 'light' });
    
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    await user.click(button);
    
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });

  it('persists theme to localStorage when toggled', async () => {
    const user = userEvent.setup();
    useThemeStore.setState({ theme: 'light' });
    
    render(<DarkModeToggle />);
    
    const button = screen.getByRole('button');
    await user.click(button);
    
    expect(localStorage.getItem('subcults-theme')).toBe('dark');
  });
});
