# Accessibility Quick Start Guide

## For Developers: 5-Minute A11y Primer

This guide gets you started with accessibility in the Subcults codebase quickly.

## TL;DR - The Essentials

1. ✅ Use semantic HTML (`<button>` not `<div onClick>`)
2. ✅ Add `aria-label` to icon-only buttons
3. ✅ Test with keyboard (Tab, Enter, Escape)
4. ✅ Run `npm test -- src/a11y-audit.test.tsx` before committing

## Common Patterns (Copy-Paste Ready)

### Button (Icon Only)
```tsx
<button 
  onClick={handleClick}
  aria-label="Close dialog"
  className="focus:outline-none focus-visible:ring-2"
>
  ×
</button>
```

### Button (With Icon + Text)
```tsx
<button onClick={handleClick}>
  <span aria-hidden="true">🔍</span>
  <span>Search</span>
</button>
```

### Form Input
```tsx
<label htmlFor="email">Email</label>
<input 
  id="email"
  type="email"
  aria-describedby="email-error"
  aria-invalid={hasError}
/>
{hasError && (
  <div id="email-error" role="alert">
    Please enter a valid email
  </div>
)}
```

### Modal/Dialog
```tsx
<div
  role="dialog"
  aria-modal="true"
  aria-labelledby="dialog-title"
>
  <h2 id="dialog-title">Dialog Title</h2>
  <button onClick={onClose} aria-label="Close">×</button>
  {/* Content */}
</div>
```

### Loading State
```tsx
<div role="status" aria-live="polite">
  {loading ? 'Loading...' : 'Content loaded'}
</div>
```

### Error Message
```tsx
<div role="alert" aria-live="assertive">
  Error: Something went wrong
</div>
```

### Image
```tsx
{/* Informative image */}
<img src="/photo.jpg" alt="Band performing on stage" />

{/* Decorative image */}
<img src="/decorative.svg" alt="" aria-hidden="true" />
```

## Pre-Commit Checklist

Before pushing code with UI changes:

```bash
# 1. Run accessibility tests
npm test -- src/a11y-audit.test.tsx

# 2. Test with keyboard only
# - Tab through all interactive elements
# - Enter/Space to activate buttons
# - Escape to close modals

# 3. Check focus indicators are visible
# - Every interactive element should show focus ring
```

## Common Mistakes to Avoid

❌ **DON'T:**
```tsx
// Missing aria-label on icon button
<button onClick={close}>×</button>

// Using div as button
<div onClick={handleClick}>Click me</div>

// Missing alt text
<img src="/photo.jpg" />

// Color as only indicator
<span style={{ color: 'red' }}>Error</span>
```

✅ **DO:**
```tsx
// Proper aria-label
<button onClick={close} aria-label="Close">×</button>

// Use semantic button
<button onClick={handleClick}>Click me</button>

// Descriptive alt text
<img src="/photo.jpg" alt="Concert crowd" />

// Icon + text for errors
<span className="text-red-500">
  <ErrorIcon aria-hidden="true" />
  <span>Error: Invalid input</span>
</span>
```

## Keyboard Navigation Rules

| Key | Action |
|-----|--------|
| Tab | Move to next focusable element |
| Shift+Tab | Move to previous focusable element |
| Enter/Space | Activate button/link |
| Escape | Close modal/menu/dropdown |
| Arrow keys | Navigate within custom widgets |

## When to Use What

### role="button" vs `<button>`
**Always use `<button>`** - Semantic HTML is better than ARIA

### aria-label vs aria-labelledby
- `aria-label`: Simple text label
- `aria-labelledby`: Reference to another element's ID

### role="alert" vs role="status"
- `alert`: Important, immediate (errors)
- `status`: Less urgent (loading, success)

### aria-live="assertive" vs "polite"
- `assertive`: Interrupts screen reader (errors, warnings)
- `polite`: Waits for pause (status updates, success)

## Testing Tools

### Browser DevTools
1. Open DevTools → Accessibility tab
2. Inspect element tree
3. Check computed properties

### Keyboard Testing
1. Close your trackpad/mouse
2. Navigate site with Tab only
3. Can you reach everything?

### Screen Reader Testing

**Mac (VoiceOver)**:
```bash
Cmd+F5          # Toggle VoiceOver
VO+Right Arrow  # Next item
VO+Space        # Activate
```

**Windows (NVDA - Free)**:
```bash
Ctrl+Alt+N      # Start NVDA
Down Arrow      # Next item
Enter           # Activate
```

## Resources

### Quick References
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/) - Patterns for complex widgets
- [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/) - Check color contrast

### In-Depth Guides
- [ACCESSIBILITY.md](ACCESSIBILITY.md) - Full accessibility guide
- [A11Y_CHECKLIST.md](A11Y_CHECKLIST.md) - Complete checklist

### Code Examples
- Search component tests for ARIA combobox pattern
- DetailPanel for modal focus management
- ToastContainer for live regions

## Getting Help

1. **Check existing components** - Many patterns already implemented
2. **Review test files** - See how we test accessibility
3. **Ask questions** - Better to ask than ship inaccessible code

## Remember

🎯 **Accessibility is not optional** - It's a core requirement  
⚡ **Build it in from the start** - Easier than retrofitting  
🧪 **Test early and often** - Catch issues before production  
📚 **Learn patterns once** - Apply them everywhere

---

**Next Steps**: Read [ACCESSIBILITY.md](ACCESSIBILITY.md) for comprehensive documentation.
