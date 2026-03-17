# Theming System Documentation

## Overview

The Subcults frontend uses Tailwind CSS as the single source of truth for design tokens, with support for dark mode toggling. The theming system combines Tailwind's utility classes with CSS variables for runtime theme switching.

## Architecture

### 1. Tailwind Configuration (`tailwind.config.js`)

The Tailwind config defines all design tokens:

- **Brand Colors**: Underground music aesthetic with electric blue and cyan accents
- **Semantic Colors**: Referenced via CSS variables for runtime switching
- **Extended Spacing**: Additional spacing utilities (18, 88, 128)
- **Typography**: Custom font sizes and families
- **Animations**: Fade-in and slide-up keyframes
- **Dark Mode**: Class-based strategy (`class` mode)

### 2. CSS Variables (`index.css`)

CSS variables enable runtime theme switching without rebuilding:

```css
:root {
  --color-background: #ffffff;
  --color-foreground: #213547;
  /* ... more light mode colors */
}

.dark {
  --color-background: #242424;
  --color-foreground: rgba(255, 255, 255, 0.87);
  /* ... more dark mode colors */
}
```

**Theme Transition Utility**: Use the `.theme-transition` class on elements that need smooth color transitions when toggling dark mode. This avoids performance issues from applying transitions to all elements.

```tsx
<div className="bg-background text-foreground theme-transition">
  Smooth theme transitions
</div>
```

### 3. Theme Store (`stores/themeStore.ts`)

Zustand store managing theme state:

- `theme`: Current theme ('light' | 'dark')
- `setTheme()`: Set theme explicitly (persists to localStorage)
- `toggleTheme()`: Toggle between light/dark (persists to localStorage)
- `initializeTheme()`: Initialize from localStorage or system preference (only persists if theme was previously manually set)

**Key Behavior**: The store distinguishes between:
- **Manual theme selection**: When user explicitly sets theme via toggle (persisted to localStorage)
- **System preference**: Initial theme from `prefers-color-scheme` (not persisted, allows auto-switching)

This ensures system preference auto-switching works correctly while respecting user's manual choices.

### 4. ThemeProvider Component

React component that:
- Initializes theme on mount
- Applies `dark` class to `<html>` element
- Listens for system preference changes
- Auto-switches only when user hasn't set manual preference

### 5. DarkModeToggle Component

Accessible button for theme switching:
- Shows moon icon in light mode, sun in dark mode
- ARIA labels for accessibility
- Optional text label
- Keyboard accessible

## Usage Patterns

### Using Tailwind Utility Classes

For static colors and styles:

```tsx
<div className="bg-brand-primary text-white rounded-lg p-4">
  Static styled element
</div>
```

### Using CSS Variables for Theme-Aware Colors

For colors that change with dark mode, add the `theme-transition` class for smooth transitions:

```tsx
<div className="bg-background text-foreground border border-border theme-transition">
  Theme-aware element with smooth transitions
</div>
```

### Responsive Dark Mode Variants

Use Tailwind's `dark:` prefix for dark mode overrides:

```tsx
<button className="
  bg-white dark:bg-gray-800
  text-gray-900 dark:text-gray-100
  border-gray-300 dark:border-gray-700
">
  Responsive button
</button>
```

### Accessing Theme in Components

```tsx
import { useTheme, useThemeActions } from '@/stores/themeStore';

function MyComponent() {
  const theme = useTheme(); // 'light' | 'dark'
  const { toggleTheme } = useThemeActions();
  
  return (
    <button onClick={toggleTheme}>
      Current theme: {theme}
    </button>
  );
}
```

### Adding the Theme Toggle

```tsx
import { DarkModeToggle } from '@/components/DarkModeToggle';

function Header() {
  return (
    <nav>
      {/* Other nav items */}
      <DarkModeToggle showLabel={true} />
    </nav>
  );
}
```

## Design Tokens Reference

### Brand Colors

- **Primary**: `brand-primary` - Electric blue (#646cff)
- **Primary Light**: `brand-primary-light` - Lighter blue (#747bff)
- **Primary Dark**: `brand-primary-dark` - Darker blue (#535bf2)
- **Accent**: `brand-accent` - React cyan (#61dafb)
- **Underground**: `brand-underground` - Deep dark (#1a1a1a)

### Semantic Colors (CSS Variables)

- **Background**: `bg-background` - Main background color
- **Background Secondary**: `bg-background-secondary` - Secondary background
- **Foreground**: `text-foreground` - Main text color
- **Foreground Secondary**: `text-foreground-secondary` - Secondary text
- **Foreground Muted**: `text-foreground-muted` - Muted/disabled text
- **Border**: `border-border` - Border color
- **Border Hover**: `border-border-hover` - Border on hover

### Custom Spacing

- `space-18`: 4.5rem (72px)
- `space-88`: 22rem (352px)
- `space-128`: 32rem (512px)

### Animations

- `animate-fade-in`: Fade in over 0.2s
- `animate-slide-up`: Slide up with fade over 0.3s

## Best Practices

### 1. Single Source of Truth

**DO**: Define design tokens in `tailwind.config.js`

```js
// tailwind.config.js
theme: {
  extend: {
    colors: {
      primary: '#646cff',
    },
  },
}
```

**DON'T**: Hardcode colors in components

```tsx
// ❌ Bad
<div style={{ color: '#646cff' }}>Text</div>

// ✅ Good
<div className="text-brand-primary">Text</div>
```

### 2. Use CSS Variables for Dynamic Values

**DO**: Use CSS variables with theme-transition class for theme-aware colors

```tsx
// ✅ Good - Automatically adapts to dark mode with smooth transition
<div className="bg-background text-foreground theme-transition">Content</div>
```

**DON'T**: Use conditional classes based on theme

```tsx
// ❌ Bad - Causes re-renders and complexity
<div className={theme === 'dark' ? 'bg-gray-900' : 'bg-white'}>
  Content
</div>
```

### 3. Optimize Performance with Selective Transitions

**DO**: Apply `theme-transition` class only to elements that change colors

```tsx
<section className="bg-background-secondary theme-transition">
  <h2>Section Title</h2>
  <p className="text-foreground">Content</p>
</section>
```

**DON'T**: Apply transitions globally with universal selector

```css
/* ❌ Bad - Causes performance issues */
* {
  transition: all 250ms;
}
```

### 3. Leverage Tailwind's Dark Mode

**DO**: Use `dark:` prefix for overrides with theme-transition

```tsx
<button className="bg-blue-500 dark:bg-blue-700 hover:bg-blue-600 dark:hover:bg-blue-800 theme-transition">
  Button with hover states
</button>
```

### 4. Optimize for Re-renders

**DO**: Use selector hooks for specific values

```tsx
// ✅ Good - Only re-renders when theme changes
const theme = useTheme();
```

**DON'T**: Use entire store if only need actions

```tsx
// ❌ Bad - Re-renders on all store changes
const { theme, setTheme, toggleTheme } = useThemeStore();
```

### 5. System Preference Auto-Switching

The theme system distinguishes between:
- **Manual selection**: User explicitly toggled theme (persisted to localStorage)
- **System preference**: Initial theme from `prefers-color-scheme` (not persisted)

When no manual selection exists, the app auto-switches with system preference changes. Once user manually sets a preference, auto-switching is disabled until localStorage is cleared.

### 5. Accessibility First

- Always provide ARIA labels on theme toggle
- Ensure sufficient color contrast in both modes
- Test keyboard navigation
- Respect `prefers-reduced-motion`
- Use `theme-transition` class sparingly to avoid performance issues
- System preference auto-switching respects user's manual choices

## Testing

### Testing Components with Theme

```tsx
import { useThemeStore } from '@/stores/themeStore';

describe('MyComponent', () => {
  beforeEach(() => {
    useThemeStore.setState({ theme: 'light' });
  });

  it('renders in dark mode', () => {
    useThemeStore.setState({ theme: 'dark' });
    render(<MyComponent />);
    // assertions
  });
});
```

### Testing Theme Toggle

```tsx
it('toggles theme when clicked', async () => {
  const user = userEvent.setup();
  useThemeStore.setState({ theme: 'light' });
  
  render(<DarkModeToggle />);
  await user.click(screen.getByRole('button'));
  
  expect(useThemeStore.getState().theme).toBe('dark');
});
```

## Troubleshooting

### Dark Mode Not Applying

1. Check that `ThemeProvider` wraps your app
2. Verify `dark` class is on `<html>` element
3. Ensure CSS variables are defined in `index.css`
4. Check Tailwind purge settings in `tailwind.config.js`

### Colors Not Updating

1. Verify you're using CSS variable-based colors (e.g., `bg-background`)
2. Check transition property in `index.css` is present
3. Ensure PostCSS is processing Tailwind directives

### LocalStorage Not Persisting

1. Check browser privacy settings
2. Verify key name matches: `'subcults-theme'`
3. Test in non-incognito window

## Future Enhancements

- [ ] More color schemes (e.g., high contrast mode)
- [ ] Per-component theme overrides
- [ ] Animation preferences (respect prefers-reduced-motion)
- [ ] Custom brand colors for scene organizers
- [ ] Color scheme generator for events
