/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class', // Enable class-based dark mode
  theme: {
    extend: {
      screens: {
        xs: '375px', // iPhone SE and similar
      },
      colors: {
        // Terminal/Neon accent colors
        neon: {
          green: 'var(--color-neon-green)',
          purple: 'var(--color-neon-purple)',
          magenta: 'var(--color-neon-magenta)',
          cyan: 'var(--color-neon-cyan)',
        },
        // Status colors
        status: {
          error: 'var(--color-status-error)',
          success: 'var(--color-status-success)',
          warning: 'var(--color-status-warning)',
          info: 'var(--color-status-info)',
        },
        // Semantic colors with dark mode variants
        background: {
          DEFAULT: 'var(--color-background)',
          secondary: 'var(--color-background-secondary)',
          hover: 'var(--color-background-hover)',
        },
        foreground: {
          DEFAULT: 'var(--color-foreground)',
          secondary: 'var(--color-foreground-secondary)',
          muted: 'var(--color-foreground-muted)',
        },
        border: {
          DEFAULT: 'var(--color-border)',
          hover: 'var(--color-border-hover)',
        },
        // Backward-compatible aliases for existing components
        brand: {
          primary: 'var(--color-neon-purple)',
          'primary-dark': 'var(--color-neon-purple-dark)',
          accent: 'var(--color-neon-cyan)',
          underground: 'var(--color-background-secondary)',
          'underground-lighter': 'var(--color-background-hover)',
        },
        underground: {
          DEFAULT: 'var(--color-background-secondary)',
          lighter: 'var(--color-background-hover)',
        },
      },
      spacing: {
        11: '2.75rem', // 44px - minimum touch target
        18: '4.5rem',
        88: '22rem',
        128: '32rem',
      },
      minHeight: {
        touch: '44px', // Minimum touch target height
      },
      minWidth: {
        touch: '44px', // Minimum touch target width
      },
      fontSize: {
        '2xs': ['0.625rem', { lineHeight: '0.875rem' }],
      },
      fontFamily: {
        display: ['"Space Mono"', 'monospace'],
        sans: ['"Space Mono"', 'monospace'],
        mono: ['"Space Mono"', 'monospace'],
      },
      borderRadius: {
        // No rounded corners in neo-brutalist design
        none: '0px',
      },
      transitionDuration: {
        0: '0ms',
        250: '250ms',
      },
      animation: {
        // Pulse for live indicators
        pulse: 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        // Entry animations for overlays and panels
        'fade-in': 'fade-in 0.15s ease-out',
        'slide-up': 'slide-up 0.2s ease-out',
        'slide-in': 'slide-in 0.2s ease-out',
      },
      keyframes: {
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'slide-up': {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        'slide-in': {
          '0%': { transform: 'translateX(100%)' },
          '100%': { transform: 'translateX(0)' },
        },
      },
    },
  },
  plugins: [],
};
