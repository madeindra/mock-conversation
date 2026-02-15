/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#2563EB',
        'primary-hover': '#1D4ED8',
        surface: '#F9FAFB',
        border: '#E5E7EB',
        muted: '#6B7280',
        error: '#EF4444',
      },
    },
  },
  plugins: [],
}
