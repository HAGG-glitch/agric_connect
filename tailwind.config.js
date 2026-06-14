/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#2E7D32',
          light: '#4CAF50',
        },
        accent: {
          gold: '#FFC107',
        },
        surface: {
          DEFAULT: '#FFFFFF',
          bg: '#F5F7F5',
        },
        text: {
          DEFAULT: '#1F2937',
          muted: '#6B7280',
        },
        border: '#E5E7EB',
      },
    },
  },
  plugins: [],
};
