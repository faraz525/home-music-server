/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        spotify: {
          green: "#1DB954",
          black: "#121212",
          dark: "#0B0B0B",
          gray: {
            100: "#F1F1F1",
            200: "#E1E1E1",
            300: "#C1C1C1",
            400: "#A1A1A1",
            500: "#7A7A7A",
            600: "#535353",
            700: "#3E3E3E",
            800: "#2A2A2A",
            900: "#181818"
          }
        }
      },
      borderRadius: {
        xl: "14px",
      },
      boxShadow: {
        glow: "0 0 0 1px rgba(29,185,84,0.2), 0 8px 24px rgba(0,0,0,0.25)",
      }
    },
  },
  plugins: [],
}

