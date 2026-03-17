# Component Accessibility Checklist

Use this checklist when developing new components to ensure WCAG 2.1 Level AA compliance.

## ✅ Semantic HTML

- [ ] Use semantic HTML5 elements (`<nav>`, `<main>`, `<header>`, `<button>`, etc.)
- [ ] Proper heading hierarchy (h1 → h2 → h3, no skipping levels)
- [ ] Lists use `<ul>`/`<ol>` and `<li>` elements
- [ ] Tables use proper structure (`<table>`, `<thead>`, `<tbody>`, `<th>`, `<td>`)

## ✅ ARIA Attributes

- [ ] Icon-only buttons have `aria-label`
- [ ] Dynamic content has `aria-live` regions
- [ ] Expandable elements have `aria-expanded`
- [ ] Modal dialogs have `role="dialog"` and `aria-modal="true"`
- [ ] Form errors use `aria-invalid` and `aria-describedby`
- [ ] Current page/state indicated with `aria-current`
- [ ] Hidden elements use `aria-hidden="true"` or CSS `display: none`

## ✅ Keyboard Navigation

- [ ] All interactive elements keyboard accessible (Tab, Enter, Space)
- [ ] Custom widgets support arrow key navigation where appropriate
- [ ] Focus visible on all interactive elements
- [ ] Focus order logical (matches visual order)
- [ ] No keyboard traps (Escape works to exit modals/menus)
- [ ] Skip links available for bypassing navigation

## ✅ Focus Management

- [ ] Modal/dialog focus trapped inside when open
- [ ] Focus returns to trigger element on modal close
- [ ] Initial focus set appropriately (usually close button for modals)
- [ ] Focus indicators visible (outline or ring)
- [ ] `:focus-visible` used instead of `:focus` where appropriate

## ✅ Forms

- [ ] All inputs have associated `<label>` elements
- [ ] Labels use `htmlFor` matching input `id`
- [ ] Required fields indicated (not just color)
- [ ] Error messages linked via `aria-describedby`
- [ ] Errors use `role="alert"` for immediate announcement
- [ ] Field hints/help text associated with input
- [ ] Autocomplete attributes used where appropriate

## ✅ Images & Media

- [ ] All `<img>` elements have `alt` attribute
- [ ] Decorative images use `alt=""` and/or `aria-hidden="true"`
- [ ] Complex images have extended descriptions
- [ ] Icons in buttons supplemented with text or `aria-label`
- [ ] SVGs use `aria-hidden="true"` when decorative
- [ ] Videos have captions/transcripts
- [ ] Audio has transcripts

## ✅ Color & Contrast

- [ ] Text contrast meets WCAG AA (4.5:1 for normal, 3:1 for large)
- [ ] UI components contrast meets WCAG AA (3:1)
- [ ] Color not sole means of conveying information
- [ ] Links distinguishable from plain text (not just color)
- [ ] Error states not just red (use icons + text)

## ✅ Responsive & Mobile

- [ ] Touch targets minimum 44x44px
- [ ] `touch-action: manipulation` on buttons to prevent delay
- [ ] Works with 200% zoom
- [ ] Content reflows at narrow widths
- [ ] No horizontal scrolling (except data tables)
- [ ] Text resizable without breaking layout

## ✅ Dynamic Content

- [ ] Loading states announced via `role="status"`
- [ ] Error states announced via `role="alert"`
- [ ] Toast notifications use `aria-live="polite"` or `"assertive"`
- [ ] Client-side routing announces page changes
- [ ] Infinite scroll has load more button alternative

## ✅ Testing

- [ ] Component has accessibility test using axe-core
- [ ] Test with keyboard only (no mouse/trackpad)
- [ ] Test with screen reader (VoiceOver or NVDA)
- [ ] Test at 200% browser zoom
- [ ] Test with color blindness simulator
- [ ] Passes automated axe-core scan (0 violations)

## ✅ Documentation

- [ ] Props/API documented (especially accessibility-related)
- [ ] Keyboard shortcuts documented
- [ ] Complex patterns explained
- [ ] Examples show accessible usage

---

## Quick Reference

### Common ARIA Patterns

**Button (icon-only)**
```tsx
<button aria-label="Close dialog">×</button>
```

**Toggle Button**
```tsx
<button aria-pressed={isPressed}>Toggle</button>
```

**Disclosure (Expandable)**
```tsx
<button aria-expanded={isOpen} aria-controls="content-id">
  Expand
</button>
<div id="content-id">Content</div>
```

**Dialog/Modal**
```tsx
<div role="dialog" aria-modal="true" aria-labelledby="title-id">
  <h2 id="title-id">Dialog Title</h2>
  ...
</div>
```

**Combobox (Search/Autocomplete)**
```tsx
<input
  role="combobox"
  aria-expanded={isOpen}
  aria-controls="listbox-id"
  aria-activedescendant={selectedId}
/>
<div id="listbox-id" role="listbox">
  <div role="option" id="option-1">Option 1</div>
</div>
```

**Live Region**
```tsx
<div role="status" aria-live="polite">
  Loading...
</div>
```

**Alert**
```tsx
<div role="alert" aria-live="assertive">
  Error occurred
</div>
```

### Focus Management

**Focus Trap**
```tsx
useEffect(() => {
  if (!isOpen) return;
  
  const handleTab = (e: KeyboardEvent) => {
    if (e.key !== 'Tab') return;
    
    const focusable = panel.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  };
  
  panel.addEventListener('keydown', handleTab);
  return () => panel.removeEventListener('keydown', handleTab);
}, [isOpen]);
```

**Return Focus**
```tsx
const previousFocusRef = useRef<HTMLElement | null>(null);

useEffect(() => {
  if (isOpen) {
    previousFocusRef.current = document.activeElement as HTMLElement;
  } else if (previousFocusRef.current) {
    previousFocusRef.current.focus();
  }
}, [isOpen]);
```

### Tailwind Utilities

```tsx
// Screen reader only
className="sr-only"

// Visible on focus
className="sr-only focus:not-sr-only"

// Focus ring
className="focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary"

// Touch targets
className="min-h-touch min-w-touch"

// Touch optimization
className="touch-manipulation"
```

---

## Running Accessibility Tests

```bash
# Run full accessibility test suite
npm test -- src/a11y-audit.test.tsx

# Run specific component test
npm test -- SearchBar.test.tsx

# Run all tests with coverage
npm test -- --coverage
```

---

**Remember**: Accessibility is not a feature, it's a requirement. Build it in from the start, not as an afterthought.
