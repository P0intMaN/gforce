import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // Backgrounds
        base: '#0a0b0d',
        surface: '#111318',
        elevated: '#1a1d24',
        overlay: '#21262d',
        // Borders
        line: '#2a2d35',
        'line-muted': '#1e2128',
        // Accents
        'accent-green': '#3fb950',
        'accent-blue': '#58a6ff',
        'accent-orange': '#d29922',
        'accent-red': '#f85149',
        'accent-purple': '#bc8cff',
        // Syntax
        'syn-green': '#7ee787',
        'syn-red': '#ff7b72',
      },
      textColor: {
        primary: '#e6edf3',
        secondary: '#8b949e',
        muted: '#484f58',
      },
      fontFamily: {
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'monospace'],
        sans: ['Geist', 'system-ui', '-apple-system', 'sans-serif'],
      },
      fontSize: {
        '2xs': ['0.625rem', { lineHeight: '1rem' }],
      },
      animation: {
        'cursor-blink': 'cursor-blink 1.2s step-end infinite',
        'fade-in': 'fade-in 0.15s ease-out',
        'slide-down': 'slide-down 0.15s ease-out',
      },
      keyframes: {
        'cursor-blink': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0' },
        },
        'fade-in': {
          from: { opacity: '0' },
          to: { opacity: '1' },
        },
        'slide-down': {
          from: { opacity: '0', transform: 'translateY(-4px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
      },
      boxShadow: {
        panel: '0 0 0 1px #2a2d35',
        dropdown: '0 8px 24px rgba(0,0,0,0.5), 0 0 0 1px #2a2d35',
      },
      // Disable Tailwind's default blue ring so it can never override our green focus style
      ringColor: {
        DEFAULT: 'transparent',
      },
      ringWidth: {
        DEFAULT: '0px',
      },
    },
  },
  plugins: [],
} satisfies Config
