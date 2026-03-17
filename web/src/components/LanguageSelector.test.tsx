/**
 * Language Selector Component Tests
 * Validates language switching UI functionality
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { LanguageSelector } from './LanguageSelector';
import { useLanguageStore } from '../stores/languageStore';

describe('LanguageSelector', () => {
  beforeEach(() => {
    // Reset language store
    useLanguageStore.setState({ language: 'en' });
  });

  it('renders language selector', () => {
    render(<LanguageSelector />);
    
    const select = screen.getByRole('combobox');
    expect(select).toBeInTheDocument();
  });

  it('shows current language as selected', () => {
    useLanguageStore.setState({ language: 'es' });
    
    render(<LanguageSelector />);
    
    const select = screen.getByRole('combobox') as HTMLSelectElement;
    expect(select.value).toBe('es');
  });

  it('displays all available languages', () => {
    render(<LanguageSelector />);
    
    const options = screen.getAllByRole('option');
    expect(options).toHaveLength(4);
    expect(options[0]).toHaveValue('en');
    expect(options[1]).toHaveValue('es');
    expect(options[2]).toHaveValue('fr');
    expect(options[3]).toHaveValue('de');
  });

  it('changes language when selection changes', async () => {
    const user = userEvent.setup();
    render(<LanguageSelector />);
    
    const select = screen.getByRole('combobox');
    await user.selectOptions(select, 'es');
    
    expect(useLanguageStore.getState().language).toBe('es');
  });

  it('persists language to localStorage', async () => {
    const user = userEvent.setup();
    render(<LanguageSelector />);
    
    const select = screen.getByRole('combobox');
    await user.selectOptions(select, 'es');
    
    expect(localStorage.getItem('subcults-language')).toBe('es');
  });
});
