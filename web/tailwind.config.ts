import type { Config } from 'tailwindcss';

export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      colors: {
        bg: '#020617',
        surface: '#0F172A',
        'surface-alt': '#111B35',
        border: '#1E293B',
        'border-light': '#334155',
        dim: '#64748B',
        'dim-light': '#94A3B8',
        accent: '#38BDF8',
        'accent-glow': '#38BDF820',
        green: '#22C55E',
        'green-glow': '#22C55E20',
        red: '#EF4444',
        'red-glow': '#EF444420',
        text: '#F8FAFC',
      },
      fontFamily: {
        sans: ['"Fira Sans"', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'Roboto', 'sans-serif'],
        mono: ['"Fira Code"', 'SFMono-Regular', 'Consolas', '"Liberation Mono"', 'Menlo', 'monospace'],
      },
      boxShadow: {
        'glow-sm': '0 0 10px var(--tw-shadow-color, rgb(56 189 248 / 0.15))',
        'glow-md': '0 0 20px var(--tw-shadow-color, rgb(56 189 248 / 0.15))',
        card: '0 1px 2px rgb(0 0 0 / 0.4)',
      },
      animation: {
        'pulse-dot': 'pulse-dot 2s ease-in-out infinite',
        'fade-in': 'fade-in 0.3s ease-out',
        'slide-up': 'slide-up 0.3s ease-out',
      },
      keyframes: {
        'pulse-dot': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.4' },
        },
        'fade-in': {
          from: { opacity: '0' },
          to: { opacity: '1' },
        },
        'slide-up': {
          from: { opacity: '0', transform: 'translateY(8px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
      },
    },
  },
} satisfies Config;
