/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./app/**/*.{js,ts,jsx,tsx,mdx}', './components/**/*.{js,ts,jsx,tsx,mdx}'],
  theme: {
    extend: {
      colors: {
        background: 'var(--background)',
        surface: 'var(--surface)',
        foreground: 'var(--foreground)',
        muted: 'var(--muted)',
        border: 'var(--border)',
        focus: 'var(--focus)',
        primary: {
          DEFAULT: 'var(--primary)',
          foreground: 'var(--primary-foreground)',
        },
        danger: 'var(--danger)',
        warning: 'var(--warning)',
        success: 'var(--success)',
        'text-body': 'var(--text-body)',
        'text-secondary': 'var(--text-secondary)',
        'text-display': 'var(--text-display)',
        'text-inverse': 'var(--text-inverse)',
        'text-inverse-muted': 'var(--text-inverse-muted)',
      },
      fontFamily: {
        display: ['var(--font-display)'],
        body: ['var(--font-body)'],
        mono: ['var(--font-mono)'],
      },
      borderRadius: {
        DEFAULT: 'var(--radius)',
      },
      minHeight: {
        touch: '2.75rem',
      },
    },
  },
  plugins: [],
}
