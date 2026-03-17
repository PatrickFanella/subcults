/**
 * AccountPage Accessibility Tests
 * Validates WCAG compliance for account management
 */

import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AccountPage } from './AccountPage';
import { expectNoA11yViolations } from '../test/a11y-helpers';

describe('AccountPage - Accessibility', () => {
  it('should not have any accessibility violations', async () => {
    const { container } = render(
      <MemoryRouter>
        <AccountPage />
      </MemoryRouter>
    );

    await expectNoA11yViolations(container);
  });

  it('should have proper heading hierarchy', () => {
    const { container } = render(
      <MemoryRouter>
        <AccountPage />
      </MemoryRouter>
    );

    const h1 = container.querySelector('h1');
    expect(h1).toBeInTheDocument();
    expect(h1?.textContent).toBe('Account');
  });

  it('should have readable content', () => {
    const { getByText } = render(
      <MemoryRouter>
        <AccountPage />
      </MemoryRouter>
    );

    expect(getByText(/Account management features/)).toBeInTheDocument();
    expect(getByText(/login, profile settings/)).toBeInTheDocument();
  });

  it('should have sufficient padding for readability', () => {
    const { container } = render(
      <MemoryRouter>
        <AccountPage />
      </MemoryRouter>
    );

    const wrapper = container.querySelector('div');
    expect(wrapper).toHaveStyle({ padding: '2rem' });
  });
});
