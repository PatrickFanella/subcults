# Accessibility Testing Guide

## Overview

This document outlines our approach to accessibility testing and compliance with WCAG 2.1 Level AA standards.

## Automated Testing

### Tools

We use [axe-core](https://github.com/dequelabs/axe-core) via [vitest-axe](https://github.com/chaance/vitest-axe) for automated accessibility testing. This provides:

- **Comprehensive coverage**: Tests for WCAG 2.1 Level A and AA violations
- **Integration with Vitest**: Seamlessly works with our existing test infrastructure
- **CI/CD support**: Runs automatically on every PR and push to main branches

### Running Tests

```bash
# Run all accessibility tests
cd web
npm test -- --run src/pages/*.a11y.test.tsx src/test/*.a11y.test.tsx

# Run tests for a specific page
npm test -- --run src/pages/HomePage.a11y.test.tsx

# Run tests in watch mode during development
npm test src/pages/*.a11y.test.tsx
```

### Test Coverage

We have automated accessibility tests for all critical pages:

1. **HomePage (Map View)** - `src/pages/HomePage.a11y.test.tsx`
   - Tests map application ARIA labels
   - Validates keyboard navigation
   - Checks for axe-core violations

2. **SceneDetailPage** - `src/pages/SceneDetailPage.a11y.test.tsx`
   - Tests heading hierarchy
   - Validates content readability
   - Checks for axe-core violations

3. **StreamPage** - `src/pages/StreamPage.a11y.test.tsx`
   - Tests streaming interface accessibility
   - Validates button accessibility
   - Tests error alert roles
   - Checks for axe-core violations

4. **AccountPage** - `src/pages/AccountPage.a11y.test.tsx`
   - Tests account management interface
   - Validates content structure
   - Checks for axe-core violations

5. **Focus Outline** - `src/test/focus-outline.a11y.test.tsx`
   - Documents focus-visible pattern usage
   - Validates keyboard navigation support
   - Ensures no global outline removal

## Key Accessibility Features

### 1. Focus Indicators

We use the **focus-visible pattern** to provide clear focus indicators for keyboard users while avoiding them for mouse users:

```css
/* From index.css */
button {
  focus:outline-none              /* Remove default outline */
  focus-visible:outline           /* Show outline for keyboard navigation */
  focus-visible:outline-4
  focus-visible:outline-brand-primary
}
```

**Why focus-visible?**
- Improves UX for mouse users (no outline on click)
- Maintains accessibility for keyboard users (outline on Tab navigation)
- Follows modern web accessibility best practices

### 2. Skip to Content Link

The `AppLayout` component includes a skip-to-content link that becomes visible when focused:

```tsx
<a
  href="#main-content"
  style={{
    position: 'absolute',
    top: '-100px',  // Off-screen by default
  }}
  onFocus={(e) => {
    e.currentTarget.style.top = '0';  // Visible on focus
  }}
  onBlur={(e) => {
    e.currentTarget.style.top = '-100px';  // Hide again
  }}
>
  Skip to content
</a>
```

**Benefits:**
- Allows keyboard users to skip navigation
- Jumps directly to main content area
- Improves efficiency for screen reader users

### 3. Semantic HTML

We use semantic HTML elements with appropriate ARIA roles:

- `<header role="banner">` - Site header
- `<main role="main" id="main-content">` - Primary content area
- `<nav role="navigation" aria-label="...">` - Navigation sections
- Proper heading hierarchy (h1 → h2 → h3)
- Form labels associated with inputs
- Button elements for interactive controls

### 4. ARIA Labels

We provide descriptive ARIA labels for:
- Interactive map: `role="application" aria-label="Interactive map showing scenes and events"`
- Volume sliders: `aria-label="Volume slider" aria-valuemin="0" aria-valuemax="100"`
- Navigation menus: `aria-label="Main navigation"`, `aria-label="Mobile navigation"`
- Mobile menu toggle: `aria-expanded="true|false"`

### 5. Color Contrast

All text and interactive elements meet WCAG AA color contrast requirements (4.5:1 for normal text, 3:1 for large text):

- Light mode: Dark text on light backgrounds
- Dark mode: Light text on dark backgrounds
- Brand colors chosen for sufficient contrast

## Writing New Accessibility Tests

### 1. Create a test file

Create a `.a11y.test.tsx` file next to your component or in `src/pages/`:

```typescript
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { YourComponent } from './YourComponent';
import { expectNoA11yViolations } from '../test/a11y-helpers';

describe('YourComponent - Accessibility', () => {
  it('should not have any accessibility violations', async () => {
    const { container } = render(<YourComponent />);
    await expectNoA11yViolations(container);
  });

  it('should have proper ARIA labels', () => {
    const { getByRole } = render(<YourComponent />);
    const button = getByRole('button', { name: 'Submit' });
    expect(button).toBeInTheDocument();
  });
});
```

### 2. Use the a11y-helpers

The `src/test/a11y-helpers.ts` file provides utilities:

- `expectNoA11yViolations(container)` - Runs axe-core and expects no violations
- `runAxeTest(container, config?)` - Runs axe-core with optional custom config
- `axeConfig` - Our standard WCAG 2.1 AA configuration

### 3. Test specific accessibility features

Beyond automated testing, manually verify:

- **Keyboard navigation**: Tab through all interactive elements
- **Screen reader compatibility**: Test with NVDA/JAWS/VoiceOver
- **Focus indicators**: Ensure visible focus on all interactive elements
- **Color contrast**: Use browser DevTools or online checkers
- **Semantic HTML**: Verify proper heading hierarchy and landmarks

## CI/CD Integration

Accessibility tests run automatically in CI via `.github/workflows/accessibility.yml`:

- **On PR**: Tests run on all changes to `web/**`
- **On push to main/develop**: Tests run to catch regressions
- **Artifacts**: Test results are uploaded for 7 days
- **PR comments**: Failures trigger automatic comments on PRs

## Manual Testing Checklist

For new features, manually verify:

- [ ] All interactive elements are keyboard accessible (Tab, Enter, Space)
- [ ] Focus indicators are visible when navigating with Tab
- [ ] No keyboard traps (can Tab out of all components)
- [ ] Screen reader announces all important content and state changes
- [ ] Color is not the only way to convey information
- [ ] Text has sufficient contrast (use browser DevTools)
- [ ] Images have alt text
- [ ] Forms have proper labels
- [ ] Error messages are associated with form fields
- [ ] Modals and dialogs trap focus appropriately
- [ ] Skip-to-content link is functional

## Resources

- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [axe-core Rules](https://github.com/dequelabs/axe-core/blob/develop/doc/rule-descriptions.md)
- [MDN Accessibility](https://developer.mozilla.org/en-US/docs/Web/Accessibility)
- [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/)
- [ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/)

## Common Issues and Solutions

### Issue: Form inputs without labels

**Problem:**
```tsx
<input type="text" name="username" />
```

**Solution:**
```tsx
<label htmlFor="username">Username</label>
<input id="username" type="text" name="username" />
```

### Issue: Missing alt text

**Problem:**
```tsx
<img src="logo.png" />
```

**Solution:**
```tsx
<img src="logo.png" alt="Subcults logo" />
```

### Issue: Buttons without accessible names

**Problem:**
```tsx
<button><span className="icon-x" /></button>
```

**Solution:**
```tsx
<button aria-label="Close">
  <span className="icon-x" aria-hidden="true" />
</button>
```

### Issue: Custom focus styles removed globally

**Problem:**
```css
* {
  outline: none;
}
```

**Solution:**
```css
button {
  outline: none; /* Remove default outline */
}
button:focus-visible {
  outline: 2px solid var(--brand-primary); /* Show for keyboard users */
}
```

## Getting Help

If you have questions about accessibility:

1. Check this guide and the resources above
2. Run axe-core tests to identify specific violations
3. Review the existing a11y test files for examples
4. Consult the WCAG guidelines for specific requirements
5. Ask in the team chat or create a discussion issue

## Future Improvements

Potential enhancements to our accessibility testing:

- [ ] Add E2E tests with real screen readers (e.g., Playwright with accessibility tree)
- [ ] Integrate Pa11y or Lighthouse CI for additional checks
- [ ] Add visual regression testing for focus states
- [ ] Create custom axe-core rules for project-specific patterns
- [ ] Add accessibility linting in code editor (eslint-plugin-jsx-a11y)
- [ ] Set up automated color contrast checking in CI
