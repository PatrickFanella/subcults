# Accessibility Guide - WCAG 2.1 Level AA Compliance

## Overview

The Subcults frontend is designed to meet **WCAG 2.1 Level AA** accessibility standards, ensuring the platform is usable by people with diverse abilities and assistive technologies.

## Testing Infrastructure

### Automated Testing
- **Tool**: axe-core via vitest-axe
- **Test Suite**: `src/a11y-audit.test.tsx` - Comprehensive accessibility audit
- **Integration**: All components automatically tested for violations
- **Command**: `npm test -- src/a11y-audit.test.tsx`

### Test Coverage Areas
1. **Semantic HTML Structure** - Proper use of HTML5 landmarks
2. **ARIA Labels** - All interactive elements properly labeled
3. **Keyboard Navigation** - Full keyboard accessibility
4. **Focus Management** - Proper focus handling in modals/dialogs
5. **Form Labels** - All form controls have associated labels
6. **Image Alt Text** - Descriptive alt text for all images
7. **Color Contrast** - WCAG AA contrast ratios (4.5:1 for text)
8. **Live Regions** - Dynamic content announces to screen readers
9. **Touch Targets** - Minimum 44x44px for mobile accessibility

## Core Accessibility Features

### 1. Landmark Regions
All pages include proper ARIA landmarks:
- `<header role="banner">` - Site header
- `<main role="main">` - Main content area  
- `<nav>` - Navigation sections
- `<aside aria-label="Sidebar navigation">` - Sidebar navigation

**Skip to Content Link**:
```tsx
<a href="#main-content" className="sr-only focus:not-sr-only">
  Skip to content
</a>
```

### 2. Keyboard Navigation

#### Global Shortcuts
- **Cmd/Ctrl+K** - Focus search bar
- **Tab/Shift+Tab** - Navigate between interactive elements
- **Enter** - Activate buttons/links
- **Escape** - Close modals/dropdowns

#### Component-Specific
- **SearchBar**: Arrow keys navigate results, Enter selects, Escape closes
- **DetailPanel**: Tab cycles through focusable elements, focus trap active
- **Sidebar**: Standard link navigation
- **MiniPlayer**: Spacebar toggles mute, Escape closes volume slider

### 3. ARIA Attributes

#### Interactive Elements
All buttons have descriptive `aria-label` or visible text:
```tsx
<button aria-label="Close detail panel">×</button>
<button aria-label="Switch to Light mode">☀️</button>
<button aria-label="Notifications, 5 unread">🔔</button>
```

#### Combobox Pattern (SearchBar)
```tsx
<input
  role="combobox"
  aria-expanded={isOpen}
  aria-controls="search-results"
  aria-autocomplete="list"
  aria-activedescendant={selectedId}
/>
<div id="search-results" role="listbox">
  <button role="option" aria-selected={isSelected}>...</button>
</div>
```

#### Modal Pattern (DetailPanel)
```tsx
<div
  role="dialog"
  aria-modal="true"
  aria-labelledby="detail-panel-title"
>
  <h2 id="detail-panel-title">Scene Name</h2>
  ...
</div>
```

### 4. Focus Management

#### Modals/Dialogs
- **Focus Trap**: Focus stays within modal when open
- **Initial Focus**: Close button receives focus on open
- **Return Focus**: Returns to trigger element on close
- **Escape Key**: Always closes modal

```tsx
useEffect(() => {
  if (isOpen) {
    previousFocusRef.current = document.activeElement;
    closeButtonRef.current?.focus();
  } else {
    previousFocusRef.current?.focus();
  }
}, [isOpen]);
```

#### Visible Focus Indicators
All interactive elements have visible focus rings:
```css
.focus-visible:ring-2 
.focus-visible:ring-brand-primary
```

### 5. Form Accessibility

#### Label Association
All form controls have associated labels:
```tsx
<label htmlFor="language-select" className="sr-only">
  Select language
</label>
<select id="language-select" aria-label="Select language">
  <option value="en">English</option>
</select>
```

#### Input Descriptions
Error messages and help text properly linked:
```tsx
<input
  aria-describedby="error-message"
  aria-invalid={hasError}
/>
<div id="error-message" role="alert">
  Error message here
</div>
```

### 6. Images & Media

#### Informative Images
All content images have descriptive alt text:
```tsx
<OptimizedImage
  src="/scene-cover.jpg"
  alt="Live music performance at underground venue"
/>
```

#### Decorative Images
Icons and decorative elements use `aria-hidden`:
```tsx
<span aria-hidden="true">🎭</span>
<svg aria-hidden="true">...</svg>
```

#### Error State
Failed images show accessible error message:
```tsx
<div role="img" aria-label="Failed to load image: Scene cover">
  <svg aria-hidden="true">...</svg>
</div>
```

### 7. Live Regions

#### Toast Notifications
```tsx
<div
  role="region"
  aria-label="Notifications"
  aria-live="polite"
>
  <div role="status" aria-live="polite" aria-atomic="true">
    Success message
  </div>
</div>
```

#### Search Results
Loading and empty states announce to screen readers:
```tsx
<div role="status" aria-live="polite">
  Searching...
</div>
```

### 8. Color Contrast

#### Text Contrast
- **Normal text**: 4.5:1 minimum (WCAG AA)
- **Large text**: 3:1 minimum (WCAG AA)
- **UI components**: 3:1 minimum (WCAG AA)

#### Brand Colors
Primary colors meet WCAG AA on appropriate backgrounds:
- `#646cff` (brand-primary) on dark backgrounds
- White text on `#646cff` backgrounds

#### Dark Mode
Both light and dark themes meet contrast requirements.

### 9. Mobile Accessibility

#### Touch Targets
All interactive elements meet 44x44px minimum:
```tsx
<button className="min-h-touch min-w-touch">...</button>
```

Tailwind config:
```js
minHeight: {
  touch: '44px',
},
minWidth: {
  touch: '44px',
}
```

#### Responsive Design
- Touch-friendly spacing
- No hover-only interactions
- `touch-action: manipulation` to prevent double-tap zoom delay

## Component Accessibility Checklist

When creating new components, ensure:

- [ ] Semantic HTML elements used appropriately
- [ ] All interactive elements keyboard accessible
- [ ] ARIA labels on icon-only buttons
- [ ] Form labels associated with inputs
- [ ] Images have alt text (or aria-hidden if decorative)
- [ ] Color not sole means of conveying information
- [ ] Focus indicators visible
- [ ] Touch targets minimum 44x44px
- [ ] Error messages linked via aria-describedby
- [ ] Live regions for dynamic content
- [ ] axe-core test added and passing

## Testing Checklist

### Manual Testing
- [ ] **Keyboard Only**: Can navigate entire app with keyboard
- [ ] **Screen Reader**: Test with NVDA (Windows) or VoiceOver (Mac)
- [ ] **Zoom**: Test at 200% browser zoom
- [ ] **Color Blindness**: Use color blindness simulator
- [ ] **Reduced Motion**: Test with prefers-reduced-motion

### Automated Testing
- [ ] axe-core audit shows 0 violations
- [ ] Component tests include accessibility assertions
- [ ] CI/CD runs accessibility tests

## Screen Reader Testing

### VoiceOver (macOS)
```bash
# Enable VoiceOver
Cmd+F5

# Navigate
VO+Right Arrow  # Next item
VO+Left Arrow   # Previous item
VO+Space        # Activate
```

### NVDA (Windows - Free)
```bash
# Start NVDA
Ctrl+Alt+N

# Navigate
Down Arrow      # Next item
Up Arrow        # Previous item
Enter           # Activate
```

## Common Patterns

### Button vs Link
```tsx
// Use button for actions
<button onClick={handleAction}>Save</button>

// Use link for navigation
<Link to="/scenes">View Scenes</Link>
```

### Loading States
```tsx
<div role="status" aria-live="polite">
  {loading ? 'Loading...' : 'Loaded'}
</div>
```

### Error States
```tsx
<div role="alert" aria-live="assertive">
  Error occurred
</div>
```

## Resources

- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [axe-core Rules](https://github.com/dequelabs/axe-core/blob/develop/doc/rule-descriptions.md)
- [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/)
- [Inclusive Components](https://inclusive-components.design/)

## Reporting Issues

If you discover accessibility issues:
1. Check if issue is already documented
2. Create GitHub issue with `accessibility` label
3. Include:
   - Component/page affected
   - WCAG criterion violated
   - Steps to reproduce
   - Suggested fix (if known)

## Continuous Improvement

Accessibility is an ongoing effort:
- Regular axe-core scans in CI/CD
- User testing with people using assistive technology
- Stay current with WCAG updates
- Review new components for accessibility
