/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        primary: '#007aff',
        'primary-hover': '#0056b3',
        danger: '#ff3b30',
        success: '#34c759',
        warning: '#ff9500',
        bg: '#f2f2f7',
        card: '#ffffff',
        border: '#e5e5ea',
        'text-secondary': '#6c6c70',
      },
      borderRadius: {
        'ios': '12px',
        'ios-sm': '8px',
      },
      boxShadow: {
        'ios': '0 1px 3px rgba(0, 0, 0, 0.08)',
        'ios-lg': '0 4px 12px rgba(0, 0, 0, 0.12)',
      },
    },
  },
  plugins: [],
}
