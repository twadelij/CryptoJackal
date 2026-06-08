/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        jackal: {
          pink: '#e94560',
          red: '#ff6b6b',
          dark: '#1a1a2e',
          blue: '#16213e',
          deep: '#0f3460',
        }
      }
    },
  },
  plugins: [],
}
