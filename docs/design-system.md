# Subcults Design System

**Neo-Brutalist Terminal Aesthetic**

**Style**: Retro/Cyberpunk/Terminal + Underground Music Platform
**Vibe**: Raw, high-contrast, edgy underground music discovery with digital grit
**Last Updated**: 2026-03-01

---

## Design Philosophy

**Raw. Unpolished. Neon.** Subcults rejects polished SaaS aesthetics. This design embraces neo-brutalism's sharp corners, high contrast, and raw typography—infused with cyberpunk terminal aesthetics (neon glows, dark backgrounds, visible scanlines). Every design decision treats the interface like an underground music zine from the '90s remixed through a cyberpunk terminal.

### Core Principles

1. **Sharp Corners Everywhere**: No rounded corners except where physically necessary (avatars, pills). Brutalism demands precision; curves soften power.

2. **Visible Structure**: Borders, dividers, and grid lines are visible. Hide nothing. The frame is as important as the content.

3. **High Contrast is Justice**: Color isolation on pure dark backgrounds. Neon primary colors on `#000000` or `#0D1117`. Accessibility meets anarchism.

4. **Monospace Typography**: `Space Mono` for all contexts (headings, body, UI). Monospace evokes terminal culture, old-school computing, DIY authenticity.

5. **Neon Accents Sparingly**: Neon green, magenta, cyan are highlights—not noise. Use glow effects only on interactive elements, live indicators, errors.

6. **No Smooth Animations**: Transitions are instant by default. Movement is functional, not decorative. Terminal culture has no time for easing functions.

---

## Color Palette

### Neutrals (Required)

| Name                  | Hex     | RGB           | Purpose                                |
| --------------------- | ------- | ------------- | -------------------------------------- |
| **bg-terminal-black** | #000000 | 0, 0, 0       | Pure black; primary background         |
| **bg-dark-charcoal**  | #0D1117 | 13, 17, 23    | Alternative dark (for elevation/cards) |
| **border-dark**       | #1F1F1F | 31, 31, 31    | Dark borders, subtle dividers          |
| **text-white**        | #FFFFFF | 255, 255, 255 | Primary text, high contrast            |
| **text-light-gray**   | #E2E8F0 | 226, 232, 240 | Secondary text                         |

### Neon Accent Colors

| Name               | Hex     | RGB          | Purpose                  | Best Use                                    |
| ------------------ | ------- | ------------ | ------------------------ | ------------------------------------------- |
| **neon-green**     | #00FF41 | 0, 255, 65   | Success, primary accents | Live indicators, check marks, active states |
| **purple-primary** | #7C3AED | 124, 58, 237 | Links, primary actions   | Buttons, links, interactive elements        |
| **magenta**        | #FF00FF | 255, 0, 255  | Secondary accent, energy | Scene badges, secondary calls-to-action     |
| **cyan**           | #00FFFF | 0, 255, 255  | Info, tertiary accent    | Information states, hover effects           |

### Intent Colors

| State       | Primary     | Hex     | Secondary   | Hex     | Notes                                                                 |
| ----------- | ----------- | ------- | ----------- | ------- | --------------------------------------------------------------------- |
| **Success** | Neon Green  | #00FF41 | Dark Green  | #008F11 | Does **not** meet WCAG AA for normal text on #000000 (2.8:1); reserve for large UI elements/icons or place text on a compliant background box |
| **Error**   | Neon Red    | #FF3333 | Dark Red    | #CC0000 | High visibility; use with caution for WCAG compliance                 |
| **Warning** | Neon Orange | #FFB000 | Dark Orange | #CC8800 | Eye-catching but readable                                             |
| **Info**    | Cyan        | #00FFFF | Deep Cyan   | #0088FF | Informational, non-blocking                                           |

### Contrast Guarantees

All critical text combinations:

- **#FFFFFF on #000000**: 21:1 ✓ **AAA**
- **#E2E8F0 on #0D1117**: 11.3:1 ✓ **AAA**
- **#00FFFF on #000000**: 6.3:1 ✓ **AA**
- **#7C3AED on #000000**: 5.6:1 ✓ **AA**
- **#FF3333 on #000000**: 3.9:1 ⚠ Limited use; add box background if critical text
- **#00FF41 on #000000**: 2.8:1 ⚠ Use only for non-text UI (icons, dots; add background box for text)

---

## Typography

**Primary Font**: `'Space Mono', monospace` (all contexts—headings, body, labels, data)

**Google Fonts Import**:

```css
@import url('https://fonts.googleapis.com/css2?family=Space+Mono:wght@400;700&display=swap');
```

**Tailwind Config**:

```js
fontFamily: {
  display: ['"Space Mono"', 'monospace'],
  sans: ['"Space Mono"', 'monospace'],
  mono: ['"Space Mono"', 'monospace'],
}
```

**Why Space Mono everywhere**: Monospace is synonymous with terminal culture, early internet, and DIY computing—the spiritual core of underground music communities. Mixing fonts dilutes the aesthetic.

### Type Scale

| Role        | Size | Weight  | Line Height | Letter Spacing | Use Case                      |
| ----------- | ---- | ------- | ----------- | -------------- | ----------------------------- |
| **h1**      | 32px | **700** | 1.25 (40px) | -0.02em        | Page titles, hero sections    |
| **h2**      | 24px | **700** | 1.33 (32px) | -0.01em        | Section headings              |
| **h3**      | 20px | **700** | 1.4 (28px)  | -0.01em        | Subsection titles             |
| **h4**      | 16px | **700** | 1.5 (24px)  | 0em            | Smaller headings, scene names |
| **body-lg** | 16px | **400** | 1.5 (24px)  | 0em            | Large body text               |
| **body**    | 14px | **400** | 1.5 (21px)  | 0em            | Default paragraph             |
| **body-sm** | 12px | **400** | 1.5 (18px)  | 0em            | Small text, captions          |
| **label**   | 12px | **700** | 1.33 (16px) | 0.05em         | Form labels, badges           |
| **code**    | 13px | **400** | 1.4 (18px)  | 0em            | Code blocks                   |

**Weight Constraint**: Space Mono provides only **400** and **700**. Use 700 for all headings/emphasis; 400 for body/secondary.

---

## Iconography

**Source**: [Lucide Icons](https://lucide.dev/) or [Feather Icons](https://feathericons.com/) (simple SVG set)

**Constraints**:

- Size: 24×24 viewBox (consistent across all icons)
- Stroke width: 2px (standard), 1.5px (dense layouts)
- Fill: Never (outline/stroke only)
- Color: `currentColor` (inherits text color) by default; explicit `text-neon-green`, `text-magenta` for accent states
- No emoji as UI icons

**Examples**: Menu, Home, Search, Settings, Plus, Trash, Check, X, Edit, AlertCircle, Info

---

## Spacing System

**Base Unit**: 8px (8px increments throughout)

| Name    | px   | rem  | Tailwind  |
| ------- | ---- | ---- | --------- |
| **xs**  | 4px  | 0.25 | space-0.5 |
| **sm**  | 8px  | 0.5  | space-1   |
| **md**  | 16px | 1    | space-2   |
| **lg**  | 24px | 1.5  | space-3   |
| **xl**  | 32px | 2    | space-4   |
| **2xl** | 48px | 3    | space-6   |
| **3xl** | 64px | 4    | space-8   |

---

## Component Styles

### Buttons

**Style**: Solid, high-contrast, sharp corners, uppercase, bold

#### Primary Button

```css
background-color: #7c3aed; /* Purple primary */
color: #ffffff;
border: 1px solid #7c3aed;
border-radius: 0px; /* NO rounding */
padding: 11px 16px; /* Tight padding */
font-size: 12px;
font-weight: 700;
text-transform: uppercase;
letter-spacing: 0.05em;
cursor: pointer;

/* Hover */
background-color: #6d28d9; /* Darker purple */

/* Focus */
outline: 2px solid #7c3aed;
outline-offset: 2px;

/* Active/Pressed */
background-color: #5b21b6;
```

#### Button Variants

| Variant     | Background  | Border  | Text    | Hover BG | Notes              |
| ----------- | ----------- | ------- | ------- | -------- | ------------------ |
| **Primary** | #7C3AED     | #7C3AED | #FFFFFF | #6D28D9  | Main action        |
| **Success** | #00FF41     | #00FF41 | #000000 | #008F11  | Positive action    |
| **Danger**  | #FF3333     | #FF3333 | #FFFFFF | #CC0000  | Destructive action |
| **Ghost**   | transparent | #7C3AED | #7C3AED | #1F1F1F  | Secondary action   |

**All Buttons**:

- `border-radius: 0px`
- Focus: `outline 2px solid [color] / 2px offset`
- Touch target: `min-h-[44px] min-w-[44px]`
- `disabled:opacity-50 disabled:cursor-not-allowed`
- `whitespace-nowrap`

### Cards & Containers

**Style**: Visible borders, no shadows, sharp corners, grid-aligned

```css
border: 1px solid #1f1f1f;
background-color: #0d1117;
border-radius: 0px; /* NO rounding */
padding: 16px;
```

**Featured/Highlighted Card** (promoted scenes, live events):

```css
border: 2px solid #7c3aed; /* Thicker border, primary color */
background-color: #0d1117;
```

**Error/Alert Card**:

```css
border: 2px solid #ff3333; /* Neon red border */
background-color: #0d1117;
```

### Form Inputs

```css
border: 1px solid #1f1f1f;
background-color: #000000;
color: #ffffff;
border-radius: 0px; /* NO rounding */
padding: 10px 12px;
font-family: 'Space Mono', monospace;
font-size: 14px;
font-weight: 400;

/* Placeholder */
color: #737373; /* Muted gray */

/* Focus */
border-color: #7c3aed;
outline: 2px solid #7c3aed;
outline-offset: 0px;
```

### Links

```css
color: #00ffff; /* Cyan for visibility */
text-decoration: underline solid 1px;
cursor: pointer;

/* Hover */
color: #7c3aed; /* Shift to purple */
text-decoration: underline wavy; /* Subtle wavy underline */

/* Active */
color: #ff00ff; /* Magenta when active */
```

### Dividers & Borders

```css
border: 1px solid #1f1f1f;
margin: 16px 0;
```

---

## Effects & Animations

### Glow Effects (Neon)

Use sparingly for interactive feedback, live indicators, error states:

#### Subtle Glow (Focus, Hover)

```css
text-shadow: 0 0 8px rgba(124, 58, 237, 0.5); /* Purple glow */
```

#### Intense Glow (Error, Live Status)

```css
text-shadow:
  0 0 16px #ff3333,
  0 0 8px #ff3333; /* Red glow */

/* Or for success */
text-shadow:
  0 0 16px #00ff41,
  0 0 8px #00ff41; /* Green glow */
```

#### Box Glow

```css
box-shadow:
  0 0 16px rgba(0, 255, 65, 0.3),
  /* Outer green */ inset 0 0 16px rgba(0, 255, 65, 0.1); /* Inner glow */
```

### Scanlines (Optional Retro Effect)

Add retro CRT scanline effect to sections (disabled by default, opt-in per component):

```css
background-image: repeating-linear-gradient(
  0deg,
  rgba(0, 0, 0, 0.15),
  rgba(0, 0, 0, 0.15) 1px,
  transparent 1px,
  transparent 2px
);
```

### Transitions

**Philosophy**: Instant by default, minimal animations only for functional feedback

```css
/* Buttons, inputs: instant state change */
transition: none;

/* Or explicit instant on specific properties */
transition:
  background-color 0ms,
  border-color 0ms;

/* Hover state: instant feedback */
transition: none;

/* Focus outline: instant visibility */
transition: none;
```

**Exception**: Loading spinners, streaming indicators can use CSS `animation` (not `transition`) for continuous loops.

### Respect Motion Preferences

```css
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## Layout & Responsive Design

### Container

```css
max-width: 1280px; /* xl breakpoint */
margin: 0 auto;
padding: 0 16px; /* 16px left/right gutter */
```

### Grid System (12-column)

```css
display: grid;
grid-template-columns: repeat(12, 1fr);
gap: 16px; /* 2× base unit */
```

### Sections

```css
margin-bottom: 48px; /* 3xl spacing */
padding: 32px 0; /* Vertical breathing room */
border-top: 1px solid #1f1f1f; /* Visible separator */
```

### Breakpoints

| Name    | px     | Use               |
| ------- | ------ | ----------------- |
| **sm**  | 640px  | Tablets portrait  |
| **md**  | 768px  | Tablets landscape |
| **lg**  | 1024px | Laptops           |
| **xl**  | 1280px | Desktops          |
| **2xl** | 1536px | Large desktops    |

### Mobile-First Strategy

- Stack cards vertically on mobile
- Typography: scale headings per breakpoint
- Inputs: full-width on mobile, 50% width on tablet+
- Touch targets: minimum 44×44px everywhere
- No horizontal scroll (ever)

---

## Accessibility Requirements (WCAG 2.1 AA+)

### Contrast

All text combinations must meet minimum 4.5:1 (AA):

- ✓ White (#FFFFFF) on black (#000000): 21:1
- ✓ Light gray (#E2E8F0) on #0D1117: 11.3:1
- ✓ Cyan (#00FFFF) on black: 6.3:1
- ✓ Purple (#7C3AED) on black: 5.6:1
- ⚠ Neon green (#00FF41) on black: 2.8:1 — use for non-text UI only or add background box

### Focus & Keyboard

- [ ] All interactive elements have visible focus outlines (2px, offset 2px)
- [ ] Focus order matches visual reading order
- [ ] Tab/Shift+Tab navigation works throughout
- [ ] Escape key closes modals, dropdowns
- [ ] Enter/Space activates buttons

### Semantics

- [ ] Navigation uses `<nav>` element
- [ ] Main content in `<main>`
- [ ] Section headers use proper `<h1>`, `<h2>`, `<h3>` (not skipped levels)
- [ ] Form labels explicitly linked to inputs (`<label for="...">`)
- [ ] Buttons are `<button>` elements (not divs), links are `<a>`

### Images & Alt Text

- [ ] All meaningful images have descriptive `alt` text
- [ ] Decorative images: `alt=""` (empty)
- [ ] Alt text describes content, not "image of" or "picture of"

### Error Handling

- [ ] Error messages use `role="alert"` or `aria-live="polite"`
- [ ] Color is never the only indicator (pair with icon, text, or position)
- [ ] Error messages are concise and actionable

### Motion

- [ ] Respect `prefers-reduced-motion` media query
- [ ] No infinite decorative animations
- [ ] No auto-playing audio/video without user control

---

## Tailwind Configuration Reference

### CSS Variables (`:root`)

```css
/* Keep these names in sync with web/src/index.css and web/tailwind.config.js */
:root {
  --color-background: #000000;
  --color-background-secondary: #0d1117;
  --color-background-hover: #1a1f2b;
  --color-border: #1f1f1f;
  --color-border-hover: #404040;
  --color-foreground: #ffffff;
  --color-foreground-secondary: #e2e8f0;
  --color-neon-green: #00ff41;
  --color-neon-purple: #7c3aed;
  --color-neon-purple-dark: #5b21b6;
  --color-neon-magenta: #ff00ff;
  --color-neon-cyan: #00ffff;
  --color-status-error: #ff3333;
  --color-status-success: #00ff41;
  --color-status-warning: #ffb000;
  --color-status-info: #00ffff;
}
```

### Tailwind `config.js` Extension

```js
export default {
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        terminal: {
          black: 'var(--color-bg-black)',
          dark: 'var(--color-bg-dark)',
        },
        neon: {
          green: 'var(--color-neon-green)',
          purple: 'var(--color-purple)',
          magenta: 'var(--color-magenta)',
          cyan: 'var(--color-cyan)',
        },
        status: {
          error: 'var(--color-error)',
          success: 'var(--color-success)',
          warning: 'var(--color-warning)',
        },
      },
      fontFamily: {
        mono: ['"Space Mono"', 'monospace'],
      },
      borderRadius: {
        none: '0px', // Default everywhere
      },
    },
  },
};
```

---

## Implementation Checklist

### Visual Quality

- [ ] No emoji used as UI icons
- [ ] Icons from consistent set (Lucide/Feather)
- [ ] No rounded corners except avatars/pills (`border-radius: 0`)
- [ ] All borders visible (no hidden structure)
- [ ] Color palette follows token system (no scattered hex values)

### Interaction

- [ ] All clickable surfaces have `cursor: pointer`
- [ ] Hover/focus states provide clear visual feedback
- [ ] Transitions are instant or minimal (no easing)
- [ ] Focus outlines always visible (2px, offset 2px, #7C3AED)
- [ ] Keyboard navigation fully functional

### Contrast & Accessibility

- [ ] All text meets 4.5:1 contrast minimum
- [ ] Form inputs have associated labels
- [ ] Color not used as sole indicator
- [ ] Respects `prefers-reduced-motion`
- [ ] Images have descriptive alt text

### Layout

- [ ] Works at 320px, 640px, 1024px, 1440px
- [ ] No horizontal scroll on any breakpoint
- [ ] Touch targets: minimum 44×44px
- [ ] Fixed elements don't cover content

---

## Design Rationale

This system rejects Bootstrap's "friendly SaaS" aesthetic (rounded corners, soft shadows, pastels) in favor of **neo-brutalism's raw honesty** (sharp corners, visible borders, high contrast) combined with **terminal/cyberpunk culture** (monospace typography, neon glows, dark backgrounds).

The result feels like an underground music zine collided with the early internet—authentic, edgy, unpolished, and unapologetically digital. It signals to users that Subcults is not a polished corporate platform, but a grassroots community tool built by and for scenes.

Every design decision serves the mission: **presence over popularity, scene sovereignty with explicit consent.**
