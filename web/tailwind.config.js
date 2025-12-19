/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Base colors - Brutalist dark palette
        void: '#0a0a0a',
        concrete: '#1a1a1a',
        steel: '#2a2a2a',
        iron: '#3a3a3a',
        smoke: '#8a8a8a',
        fog: '#b0b0b0',
        chalk: '#e0e0e0',
        white: '#f5f5f5',
        
        // Threat level colors
        threat: {
          critical: '#ff0844',
          high: '#ff6b35',
          medium: '#ffbe0b',
          low: '#4ecdc4',
          info: '#45b7d1',
        },
        
        // Accent colors
        terminal: '#00ff41',
        electric: '#0066ff',
        warning: '#ffaa00',
        cyber: '#ff0080',
      },
      fontFamily: {
        mono: ['"JetBrains Mono"', '"Fira Code"', '"Courier New"', 'monospace'],
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'sans-serif'],
        display: ['"Space Grotesk"', 'Inter', 'sans-serif'],
      },
      animation: {
        'glitch': 'glitch 150ms ease-in-out',
        'pulse-slow': 'pulse 2s ease-in-out infinite',
        'scan': 'scan 5s linear infinite',
      },
      keyframes: {
        glitch: {
          '0%': { transform: 'translate(0)' },
          '20%': { transform: 'translate(-2px, 2px)', opacity: '0.8' },
          '40%': { transform: 'translate(2px, -2px)' },
          '60%': { transform: 'translate(-2px, -2px)' },
          '80%': { transform: 'translate(2px, 2px)', opacity: '0.8' },
          '100%': { transform: 'translate(0)' },
        },
        scan: {
          '0%': { transform: 'translateY(-100%)' },
          '100%': { transform: 'translateY(100vh)' },
        },
      },
    },
  },
  plugins: [],
}
