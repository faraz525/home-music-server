/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        display: ['Clash Display', 'system-ui', 'sans-serif'],
        body: ['DM Sans', 'system-ui', 'sans-serif'],
      },
      colors: {
        crate: {
          // Base surfaces
          black: "#0D0A14",
          surface: "#1A171F",
          elevated: "#252130",
          border: "#332E3C",

          // Accent colors
          amber: "#E5A000",
          amberLight: "#FFB82E",
          amberDark: "#B37D00",
          cyan: "#00D4FF",
          cyanDark: "#00A8CC",

          // Text
          cream: "#F5F0E8",
          muted: "#9B95A3",
          subtle: "#6B6573",

          // Semantic
          danger: "#FF6B6B",
          success: "#4ADE80",
        }
      },
      borderRadius: {
        'xl': "12px",
        '2xl': "16px",
        '3xl': "24px",
      },
      boxShadow: {
        'glow': '0 0 20px rgba(229, 160, 0, 0.15)',
        'glow-strong': '0 0 30px rgba(229, 160, 0, 0.25)',
        'glow-cyan': '0 0 20px rgba(0, 212, 255, 0.15)',
        'elevated': '0 8px 32px rgba(0, 0, 0, 0.4)',
        'card': '0 4px 24px rgba(0, 0, 0, 0.3), inset 0 1px 0 rgba(255, 255, 255, 0.03)',
      },
      animation: {
        'spin-slow': 'spin 3s linear infinite',
        'spin-slower': 'spin 8s linear infinite',
        'pulse-glow': 'pulse-glow 2s ease-in-out infinite',
        'fade-in': 'fade-in 0.3s ease-out',
        'slide-up': 'slide-up 0.4s ease-out',
      },
      keyframes: {
        'pulse-glow': {
          '0%, 100%': { opacity: '0.6' },
          '50%': { opacity: '1' },
        },
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'slide-up': {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
      backgroundImage: {
        'grain': "url(\"data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)'/%3E%3C/svg%3E\")",
      },
    },
  },
  plugins: [],
}
